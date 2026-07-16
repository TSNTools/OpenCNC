package engine

import (
	"fmt"

	"OpenCNC_config_service/common/observability"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/common/structures/topology_config"
	protocolbackends "OpenCNC_config_service/config_service/pkg/protocolbackends"
)

// MappingEngine is the top-level orchestrator for applying a topology-wide configuration.
type MappingEngine struct {
	logger   observability.Logger
	backends map[topology.ManagementProtocol]protocolbackends.ProtocolBackend
}

func NewMappingEngine(logger observability.Logger) *MappingEngine {
	return &MappingEngine{
		logger:   observability.NormalizeLogger(logger),
		backends: make(map[topology.ManagementProtocol]protocolbackends.ProtocolBackend),
	}
}

func (m *MappingEngine) RegisterBackend(backend protocolbackends.ProtocolBackend) {
	m.backends[backend.Protocol()] = backend
}

func (m *MappingEngine) ApplyConfiguration(
	topo *topology.Topology,
	cfg *topology_config.TopologyConfig,
	secret string,
) error {
	if topo == nil || cfg == nil {
		return fmt.Errorf("topology and config must not be nil")
	}

	for _, node := range topo.Nodes {
		if node == nil || node.ManagementInfo == nil {
			continue
		}

		nodeCfg := m.findNodeConfig(cfg, node.Name)
		if nodeCfg == nil {
			continue
		}

		backend, ok := m.backends[node.ManagementInfo.Protocol]
		if !ok {
			if m.logger != nil {
				m.logger.Printf("no backend registered for protocol %v", node.ManagementInfo.Protocol)
			}
			continue
		}
		backend.MapAndPush(nodeCfg, node)
	}

	return nil
}

func (m *MappingEngine) findNodeConfig(
	cfg *topology_config.TopologyConfig,
	nodeName string,
) *topology_config.NodeConfig {
	for _, nodeCfg := range cfg.GetNodeConfigs() {
		if nodeCfg != nil && nodeCfg.GetNodeId() == nodeName {
			return nodeCfg
		}
	}
	return nil
}

/*
func (m *MappingEngine) findPortConfig(
	nodeCfg *topology_config.NodeConfig,
	portID string,
) *topology_config.PortConfig {
	if nodeCfg == nil {
		return nil
	}

	for _, portCfg := range nodeCfg.GetPortConfigs() {
		if portCfg != nil && portCfg.GetPortId() == portID {
			return portCfg
		}
	}

	return nil
}


		if err := backend.MapAndPush(portCfg, target); err != nil {
			if m.logger != nil {
				m.logger.Printf(
					"failed to apply config to %s/%s: %v",
					node.Name,
					port.Name,
					err,
				)
			}
			return err
		}
	}

*/
