package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"

	gokrb_client "github.com/nks5295/gokrb5/client"
	gokrb_config "github.com/nks5295/gokrb5/config"
	gokrb_creds "github.com/nks5295/gokrb5/credentials"
	gokrb_crypto "github.com/nks5295/gokrb5/crypto"
	gokrb_keytab "github.com/nks5295/gokrb5/keytab"
	gokrb_msgs "github.com/nks5295/gokrb5/messages"
	gokrb_types "github.com/nks5295/gokrb5/types"

	fqdn "github.com/Showmax/go-fqdn"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl"

	"github.com/spiffe/spire/proto/agent/nodeattestor"
	"github.com/spiffe/spire/proto/common"
	spi "github.com/spiffe/spire/proto/common/plugin"

	krbc "github.com/nks5295/kerberos-attestor/common"
)

const (
	pluginName = "kerberos_attestor"
)

type KrbAttestorPlugin struct {
	realm     string
	krbConfig *gokrb_config.Config
	keytab    gokrb_keytab.Keytab
	username  string
	spireSPN  string

	mtx *sync.Mutex
}

type KrbAttestorConfig struct {
	KrbRealm      string `hcl:"krb_realm"`
	KrbConfPath   string `hcl:"krb_conf_path"`
	KrbKeytabPath string `hcl:"krb_keytab_path"`
	ServerFQDN    string `hcl:"server_fqdn"`
}

func New() *KrbAttestorPlugin {
	return &KrbAttestorPlugin{
		mtx: &sync.Mutex{},
	}
}

func (k *KrbAttestorPlugin) spiffeID(krbCreds gokrb_creds.Credentials) *url.URL {
	spiffePath := path.Join("spire", "agent", pluginName, krbCreds.Domain(), krbCreds.DisplayName())
	id := &url.URL{
		Scheme: "spiffe",
		Host:   strings.ToLower(k.realm),
		Path:   spiffePath,
	}
	return id
}

func (k *KrbAttestorPlugin) FetchAttestationData(*nodeattestor.FetchAttestationDataRequest) (*nodeattestor.FetchAttestationDataResponse, error) {
	var buf bytes.Buffer

	krbClient := gokrb_client.NewClientWithKeytab(k.username, k.realm, k.keytab)
	krbClient.WithConfig(k.krbConfig)
	krbClient.GoKrb5Conf.DisablePAFXFast = true

	err := krbClient.Login()
	if err != nil {
		err = krbc.AttestationStepError("[KRB_AS_REQ] logging in", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	serviceTkt, encryptionKey, err := krbClient.GetServiceTicket(k.spireSPN)
	if err != nil {
		err = krbc.AttestationStepError("[KRB_TGS_REQ] requesting service ticket", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	authenticator, err := gokrb_types.NewAuthenticator(krbClient.Credentials.Domain(), krbClient.Credentials.CName)
	if err != nil {
		err = krbc.AttestationStepError("[KRB_AP_REQ] building Kerberos authenticator", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	encryptionType, err := gokrb_crypto.GetEtype(encryptionKey.KeyType)
	if err != nil {
		err = krbc.AttestationStepError("[KRB_AP_REQ] getting encryption key type", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	err = authenticator.GenerateSeqNumberAndSubKey(encryptionType.GetETypeID(), encryptionType.GetKeyByteSize())
	if err != nil {
		err = krbc.AttestationStepError("[KRB_AP_REQ] generating authenticator sequence number and subkey", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	krbAPReq, err := gokrb_msgs.NewAPReq(serviceTkt, encryptionKey, authenticator)
	if err != nil {
		err = krbc.AttestationStepError("[KRB_AP_REQ] building KRB_AP_REQ", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	attestedData := &krbc.KrbAttestedData{
		KrbAPReq: krbAPReq,
	}
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(attestedData)
	if err != nil {
		err = krbc.AttestationStepError("encoding KRB_AP_REQ for transport", err)
		return &nodeattestor.FetchAttestationDataResponse{}, err
	}

	data := &common.AttestedData{
		Type: pluginName,
		Data: buf.Bytes(),
	}
	resp := &nodeattestor.FetchAttestationDataResponse{
		AttestedData: data,
		SpiffeId:     k.spiffeID(*krbClient.Credentials).String(),
	}

	return resp, nil
}

func (k *KrbAttestorPlugin) Configure(req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	resp := &spi.ConfigureResponse{}
	config := &KrbAttestorConfig{}

	hclTree, err := hcl.Parse(req.Configuration)
	if err != nil {
		err := fmt.Errorf("Error parsing Kerberos Attestor configuration: %s", err)
		return resp, err
	}
	err = hcl.DecodeObject(&config, hclTree)
	if err != nil {
		err := fmt.Errorf("Erorr decoding Kerberos Attestor configuration: %s", err)
		return resp, err
	}

	k.mtx.Lock()
	defer k.mtx.Unlock()

	krbCfg, err := gokrb_config.Load(config.KrbConfPath)
	if err != nil {
		err := fmt.Errorf("Error loading Kerberos config: %s", err)
		return resp, err
	}

	krbKt, err := gokrb_keytab.Load(config.KrbKeytabPath)
	if err != nil {
		err := fmt.Errorf("Error loading Kerberos keytab: %s", err)
		return resp, err
	}

	hostname := fqdn.Get()
	if hostname == "unknown" {
		err := fmt.Errorf("Error getting machine FQDN")
		return resp, err
	}

	spireSPN := fmt.Sprintf("%s/%s", krbc.SPIREServiceName, config.ServerFQDN)

	k.realm = config.KrbRealm
	k.krbConfig = krbCfg
	k.keytab = krbKt
	k.username = hostname
	k.spireSPN = spireSPN

	return &spi.ConfigureResponse{}, nil
}

func (k *KrbAttestorPlugin) GetPluginInfo(*spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	panic("not implemented")
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		Plugins: map[string]plugin.Plugin{
			pluginName: nodeattestor.NodeAttestorPlugin{NodeAttestorImpl: New()},
		},
		HandshakeConfig: nodeattestor.Handshake,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
