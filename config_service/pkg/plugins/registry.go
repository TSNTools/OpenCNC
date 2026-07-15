// plugins/registry.go
package plugins

import (
	"log"

	"OpenCNC_config_service/common/structures/topology"
)

type PluginFactory struct {
	Protocol topology.ManagementProtocol
	New      func(*log.Logger) Plugin
}

var registry = []PluginFactory{}

func Register(f PluginFactory) {
	registry = append(registry, f)
}

func ForProtocol(protocol topology.ManagementProtocol, logger *log.Logger) []Plugin {
	var result []Plugin

	for _, factory := range registry {
		if factory.Protocol == protocol {
			result = append(result, factory.New(logger))
		}
	}

	return result
}
