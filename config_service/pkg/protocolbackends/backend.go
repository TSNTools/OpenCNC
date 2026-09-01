package protocolbackends

import (
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	"OpenCNC/config_service/pkg/plugins"
	"context"
)

type ProtocolBackend interface {
	Name() string
	Protocol() topology.ManagementProtocol

	AddPlugin(plugin plugins.Plugin)
	Plugins() []plugins.Plugin

	PrepareSnapshot(ctx context.Context, cmsg *topology_config.NodeConfig) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	UpdateCredentials(cred *credentials.ManagementCredentials)
}

type Snapshot interface {
	Clone() Snapshot
	Update(featureXML *plugins.FeatureXML, node *topology.Node, portId string) error
}

type SnapshotSet[T any] struct {
	Current    T
	Working    T
	LastStable T
}
