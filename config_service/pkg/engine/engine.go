package engine

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"

	"OpenCNC/common/observability"
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/config_service/pkg/managementSessions"
	protocolbackends "OpenCNC/config_service/pkg/protocolbackends"
)

//=================================
// definition of the MappingEngine
//=================================

// MappingEngine is the top-level orchestrator for applying a topology-wide configuration.
type MappingEngine struct {
	logger observability.Logger

	lastTransaction   *ConfigurationTransaction // last applied configuration transaction
	lastTransactionMu sync.RWMutex

	backends map[string]protocolbackends.ProtocolBackend // this is extra now
}

func NewMappingEngine(logger observability.Logger) *MappingEngine {
	return &MappingEngine{
		logger:   observability.NormalizeLogger(logger),
		backends: make(map[string]protocolbackends.ProtocolBackend),
	}
}

func (m *MappingEngine) RegisterBackend(target string, backend protocolbackends.ProtocolBackend) {
	m.backends[target] = backend
}

func (m *MappingEngine) GetLastTransactionId() *string {
	m.lastTransactionMu.RLock()
	defer m.lastTransactionMu.RUnlock()

	if m.lastTransaction == nil {
		return nil
	}
	return &m.lastTransaction.ConfigId
}

func (m *MappingEngine) ApplyConfiguration(topo *topology.Topology, cfg *topology_config.TopologyConfig, creds []managementSessions.NodeCredentials) error {
	if topo == nil || cfg == nil {
		return fmt.Errorf("topology and config must not be nil")
	}

	// create a new configuration transaction
	tx := NewConfigurationTransaction(cfg.GetConfigId())

	for _, node := range topo.Nodes {
		if node == nil || node.ManagementInfo == nil {
			continue
		}

		nodeCfg := findNodeConfig(cfg, node.Name)
		if nodeCfg == nil {
			continue
		}

		var nodeCredentials *credentials.ManagementCredentials

		for _, cred := range creds {
			if cred.ID == node.Name {
				nodeCredentials = cred.Credentials
				break
			}
		}

		// Get backend for the target node
		key := node.Name + "-" + node.ManagementInfo.Protocol.String()
		backend, ok := m.backends[key]
		if ok {
			backend.UpdateCredentials(nodeCredentials)
		} else {
			var err error
			backend, err = protocolbackends.ForProtocol(
				node.ManagementInfo.Protocol,
				node,
				nodeCredentials,
				m.logger,
			)
			if err != nil {
				m.logger.Printf(
					"failed to create backend for %s: %v",
					key,
					err,
				)
				continue
			}
			m.RegisterBackend(key, backend)
		}

		tx.Operations = append(tx.Operations, Operation{Config: nodeCfg, Backend: backend})
	}

	//set concurrency limit based on available workers
	tx.MAX_CONCURRENT_CONFIGS, _ = getAvailableWorkers()

	if err := tx.Prepare(); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// transaction promotion: update the current and previous transaction IDs
	m.lastTransactionMu.Lock()
	m.lastTransaction = tx
	m.lastTransactionMu.Unlock()

	// TODO:
	// Persist the new configuration in the KV store only after all
	// backends have successfully committed.

	return nil
}

func findNodeConfig(cfg *topology_config.TopologyConfig, nodeName string) *topology_config.NodeConfig {
	for _, nodeCfg := range cfg.GetNodeConfigs() {
		if nodeCfg != nil && nodeCfg.GetNodeId() == nodeName {
			return nodeCfg
		}
	}
	return nil
}

func (m *MappingEngine) Rollback() error {
	m.lastTransactionMu.Lock()
	defer m.lastTransactionMu.Unlock()

	if m.lastTransaction == nil {
		return fmt.Errorf("transaction rollback is available only after a successful configuration transaction!!")
	}

	if err := m.lastTransaction.Rollback(); err != nil {
		return err
	}

	m.lastTransaction = nil
	return nil
}

func getAvailableWorkers() (int, error) {
	maxWorkers := runtime.NumCPU() * 2

	if envVal := os.Getenv("MAX_WORKERS"); envVal != "" {
		parsedVal, err := strconv.Atoi(envVal)
		if err != nil || parsedVal <= 0 {
			return 0, fmt.Errorf("invalid MAX_WORKERS value: %q", envVal)
		}
		maxWorkers = parsedVal
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	return maxWorkers, nil
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
