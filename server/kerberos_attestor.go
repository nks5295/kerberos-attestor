package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/url"
	"os"
	"path"
	"sync"

	gokrb_config "github.com/nks5295/gokrb5/config"
	gokrb_creds "github.com/nks5295/gokrb5/credentials"
	gokrb_keytab "github.com/nks5295/gokrb5/keytab"
	gokrb_service "github.com/nks5295/gokrb5/service"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl"

	krbc "github.com/nks5295/kerberos-attestor/common"
	spi "github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/server/nodeattestor"
)

const (
	pluginName = "kerberos_attestor"
)

type KrbAttestorPlugin struct {
	realm     string
	krbConfig *gokrb_config.Config
	keytab    gokrb_keytab.Keytab
	spireSPN  string

	mtx *sync.Mutex
}

type KrbAttestorConfig struct {
	KrbRealm      string `hcl:"krb_realm"`
	KrbConfPath   string `hcl:"krb_conf_path"`
	KrbKeytabPath string `hcl:"krb_keytab_path"`
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
		Host:   k.realm,
		Path:   spiffePath,
	}
	return id
}

func (k *KrbAttestorPlugin) Attest(req *nodeattestor.AttestRequest) (*nodeattestor.AttestResponse, error) {
	var attestedData krbc.KrbAttestedData
	var buf bytes.Buffer

	buf.Write(req.AttestedData.Data)
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&attestedData)
	if err != nil {
		err = krbc.AttestationStepError("decoding KRB_AP_REQ from attestation data", err)
		return &nodeattestor.AttestResponse{Valid: false}, err
	}

	valid, creds, err := gokrb_service.ValidateAPREQ(attestedData.KrbAPReq, k.keytab, k.spireSPN, "0", false)
	if err != nil {
		err = krbc.AttestationStepError("validating KRB_AP_REQ", err)
		return &nodeattestor.AttestResponse{Valid: false}, err
	}

	if valid {
		resp := &nodeattestor.AttestResponse{
			Valid:        true,
			BaseSPIFFEID: k.spiffeID(creds).String(),
		}

		return resp, nil
	}

	err = krbc.AttestationStepError("validating KRB_AP_REQ", fmt.Errorf("failed to validate KRB_AP_REQ"))
	return &nodeattestor.AttestResponse{Valid: false}, err
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

	hostname, err := os.Hostname()
	if err != nil {
		err := fmt.Errorf("Error obtaining hostname: %s", err)
		return resp, err
	}
	spireSPN := fmt.Sprintf("%s/%s", krbc.SPIREServiceName, hostname)

	k.realm = config.KrbRealm
	k.krbConfig = krbCfg
	k.keytab = krbKt
	k.spireSPN = spireSPN

	return &spi.ConfigureResponse{}, nil
}

func (k *KrbAttestorPlugin) GetPluginInfo(*spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
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
