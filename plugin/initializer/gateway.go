package initializer

import (
	"context"

	"github.com/kisexp/xdchain/plugin/gen/proto_common"
)

type PluginGateway struct {
	client proto_common.PluginInitializerClient
}

func (g *PluginGateway) Init(ctx context.Context, nodeIdentity string, rawConfiguration []byte) error {
	_, err := g.client.Init(ctx, &proto_common.PluginInitialization_Request{
		HostIdentity:     nodeIdentity,
		RawConfiguration: rawConfiguration,
	})
	return err
}
