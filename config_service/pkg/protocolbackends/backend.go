package protocolbackends

import (
	credentials "OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	"OpenCNC/config_service/pkg/plugins"
)

type ProtocolBackend interface {
	Name() string
	Protocol() topology.ManagementProtocol

	AddPlugin(plugin plugins.Plugin)
	Plugins() []plugins.Plugin

	PrepareSnapshot(msg *topology_config.NodeConfig, node *topology.Node) error
	Commit(target *topology.Node, cred *credentials.ManagementCredentials) error
	Rollback(target *topology.Node, cred *credentials.ManagementCredentials) error
}

type Snapshot interface {
	Clone() Snapshot
	Update(featureXML *plugins.FeatureXML, node *topology.Node) error
}

type SnapshotSet[T any] struct {
	Current    T
	Working    T
	LastStable T
}
