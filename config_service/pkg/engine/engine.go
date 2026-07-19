package engine

import (
	"fmt"

	"OpenCNC_config_service/common/observability"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/common/structures/topology_config"
	protocolbackends "OpenCNC_config_service/config_service/pkg/protocolbackends"
)

type Operation struct {
	Node      *topology.Node
	Config    *topology_config.NodeConfig
	Backend   protocolbackends.ProtocolBackend
	prepared  bool
	committed bool
}

type ConfigurationTransaction struct {
	configId   string
	Operations []Operation
}

func (t *ConfigurationTransaction) Commit() error {

	for i := range t.Operations {

		op := &t.Operations[i]

		if err := op.Backend.Commit(op.Node); err != nil {

			// Roll back everything that was already committed.
			t.Rollback()

			return fmt.Errorf(
				"commit failed for node %s, Aborted transaction and rolled back previous commits",
				op.Node.Name,
			)
		}

		op.committed = true
	}

	return nil
}

func (t *ConfigurationTransaction) Rollback() error {

	var firstErr error

	for i := len(t.Operations) - 1; i >= 0; i-- {

		op := &t.Operations[i]

		if !op.committed {
			continue
		}

		if err := op.Backend.Rollback(op.Node); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		op.committed = false
	}

	return firstErr
}

func (t *ConfigurationTransaction) Prepare() error {

	for i := range t.Operations {

		op := &t.Operations[i]

		if err := op.Backend.PrepareSnapshot(op.Config, op.Node); err != nil {

			return fmt.Errorf(
				"prepare failed for node %s: %w",
				op.Node.Name,
				err,
			)
		}

		op.prepared = true
	}

	return nil
}

func NewConfigurationTransaction(configId string) *ConfigurationTransaction {
	return &ConfigurationTransaction{
		configId: configId,
	}
}

//=================================
// definition of the MappingEngine
//=================================

// MappingEngine is the top-level orchestrator for applying a topology-wide configuration.
type MappingEngine struct {
	logger observability.Logger

	lastTransaction *ConfigurationTransaction // last applied configuration transaction

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

func (m *MappingEngine) GetLastTransactionId() *string {
	if m.lastTransaction == nil {
		return nil
	}
	return &m.lastTransaction.configId
}

func (m *MappingEngine) ApplyConfiguration(topo *topology.Topology, cfg *topology_config.TopologyConfig, secret string) error {
	if topo == nil || cfg == nil {
		return fmt.Errorf("topology and config must not be nil")
	}

	tx := NewConfigurationTransaction(cfg.GetConfigId())

	for _, node := range topo.Nodes {
		if node == nil || node.ManagementInfo == nil {
			continue
		}

		nodeCfg := findNodeConfig(cfg, node.Name)
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

		tx.Operations = append(tx.Operations, Operation{
			Node:    node,
			Config:  nodeCfg,
			Backend: backend,
		})
	}

	if err := tx.Prepare(); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// transaction promotion: update the current and previous transaction IDs
	m.lastTransaction = tx

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

	if m.lastTransaction == nil {
		return fmt.Errorf("transaction rollback is available only after a successful configuration transaction!!")
	}

	if err := m.lastTransaction.Rollback(); err != nil {
		return err
	}

	m.lastTransaction = nil

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
