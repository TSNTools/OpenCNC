// plugins/registry.go
package plugins

import (
	"OpenCNC_config_service/common/observability"
	"OpenCNC_config_service/common/structures/topology"
)

type PluginFactory struct {
	Protocol topology.ManagementProtocol
	New      func(observability.Logger) Plugin
}

var registry = []PluginFactory{}

func Register(f PluginFactory) {
	registry = append(registry, f)
}

func ForProtocol(protocol topology.ManagementProtocol, logger observability.Logger) []Plugin {
	var result []Plugin
	logger = observability.NormalizeLogger(logger)

	for _, factory := range registry {
		if factory.Protocol == protocol {
			result = append(result, factory.New(logger))
		}
	}

	return result
}
