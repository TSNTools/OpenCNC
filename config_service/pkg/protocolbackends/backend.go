package protocolbackends

import (
	"OpenCNC_config_service/config_service/pkg/plugins"
	"OpenCNC_config_service/common/structures/topology"

	"github.com/golang/protobuf/proto"
)

type ProtocolBackend interface {
	Name() string
	Protocol() topology.ManagementProtocol

	AddPlugin(plugin plugins.Plugin)
	Plugins() []plugins.Plugin

	MapAndPush(msg proto.Message, target topology.Node) error
}
