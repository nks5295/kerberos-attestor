package main

import(
	"github.com/hashicorp/go-plugin"
	spi "github.com/spiffe/spire/proto/common/plugin"

	//import appropriate package corresponding to the plugin type
	//"github.com/spiffe/spire/proto/server/nodeattestor"
	//"github.com/spiffe/spire/proto/server/noderesolver"
	//"github.com/spiffe/spire/proto/server/datastore"
	//"github.com/spiffe/spire/proto/server/ca"
	//"github.com/spiffe/spire/proto/server/datastore"
	//"github.com/spiffe/spire/proto/server/upstreamca"

	//"github.com/spiffe/spire/proto/agent/nodeattestor"
	//"github.com/spiffe/spire/proto/agent/keymanager"
	//"github.com/spiffe/node-agent/plugins/workload_attestor"
)

const pluginName  = ""//Set PluginName"

type Plugin struct { // Rename Plugin Receiver Type
	config1 string
}

type PluginConfig struct { // Rename PluginConfig Type
	Config1 string `hcl:"config_1"`
}


func New() Plugin{
	//New method should return the Plugin Type being implemented
return Plugin{}
}

func (p *Plugin) Configure(*spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	return &spi.ConfigureResponse{}, nil
}

func (p *Plugin) GetPluginInfo(*spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{

		Plugins: map[string]plugin.Plugin{
			//Uncomment one of the lines below corresponding to the plugin type being implemented

			//SPIRE Server Plugin Type
			//pluginName: noderesolver.NodeResolverPlugin{NodeResolverImpl: New()},
			//pluginName: nodeattestor.NodeAttestorPlugin{NodeAttestorImpl: New()},
			//pluginName: ca.ControlPlaneCaPlugin{ControlPlaneCaImpl: New()},
			//pluginName: datastore.DataStorePlugin{DataStoreImpl: New()},
			//pluginName: upstreamca.UpstreamCaPlugin{UpstreamCaImpl: New()},


			//SPIRE Agent Plugin Type
			//pluginName: nodeattestor.NodeAttestorPlugin{NodeAttestorImpl: New()},
			//pluginName: keymanager.KeyManagerPlugin{KeyManagerImpl: New()},
			//pluginName: workloadattestor.WorkloadAttestorPlugin{WorkloadAttestorImpl: New()},

		},
		//Uncomment the line corresponding to the plugin type
		//HandshakeConfig: nodeattestor.Handshake,
		//HandshakeConfig: noderesolver.Handshake,
		//HandshakeConfig: ca.Handshake,
		//HandshakeConfig: datastore.Handshake,
		//HandshakeConfig: upstreamca.Handshake,

		//HandshakeConfig: nodeattestor.Handshake,
		//HandshakeConfig: keymanager.Handshake,
		//HandshakeConfig: workloadattestor.Handshake,

		GRPCServer: plugin.DefaultGRPCServer,
	})

}