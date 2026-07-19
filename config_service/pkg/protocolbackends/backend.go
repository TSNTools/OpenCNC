package protocolbackends

import (
	"OpenCNC_config_service/common/structures/topology"
	topology_config "OpenCNC_config_service/common/structures/topology_config"
	"OpenCNC_config_service/config_service/pkg/managementSessions"
	"OpenCNC_config_service/config_service/pkg/plugins"
)

type ProtocolBackend interface {
	Name() string
	Protocol() topology.ManagementProtocol

	AddPlugin(plugin plugins.Plugin)
	Plugins() []plugins.Plugin

	PrepareSnapshot(msg *topology_config.NodeConfig, node *topology.Node) error
	Commit(target *topology.Node) error
	Rollback(target *topology.Node) error
}

type Snapshot interface {
	Clone() Snapshot
	Update(featureXML *plugins.FeatureXML, target managementSessions.DeviceTarget) error
}

type SnapshotSet[T any] struct {
	Current    T
	Working    T
	LastStable T
}
