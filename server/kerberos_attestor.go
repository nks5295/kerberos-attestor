package main

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl"

	spi "github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/server/nodeattestor"
)

const pluginName = "kerberos_attestor"

const (
	krbServiceName = "SPIRE"
)

type KrbAttestorPlugin struct {
	krbConfPath   string
	krbKeytabPath string
	serverFQDN    string

	mtx *sync.Mutex
}

type KrbAttestorConfig struct {
	KrbConfPath   string `hcl:"krb_conf_path"`
	KrbKeytabPath string `hcl:"krb_keytab_path"`
	ServerFQDN    string `hcl:server_fqdn`
}

func New() *KrbAttestorPlugin {
	return &KrbAttestorPlugin{}
}

func (k *KrbAttestorPlugin) Attest(req *nodeattestor.AttestRequest) (*nodeattestor.AttestResponse, error) {
	panic("not implemented")
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

	k.krbConfPath = config.KrbConfPath
	k.krbKeytabPath = config.KrbKeytabPath
	k.serverFQDN = config.ServerFQDN

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
