package engine

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"OpenCNC/common/observability"
	"OpenCNC/common/structures/credentials"
	observabilityv1 "OpenCNC/common/structures/logging"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/config_service/pkg/managementSessions"
	"OpenCNC/config_service/pkg/metrics"
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

func (m *MappingEngine) ApplyConfiguration(ctx context.Context, obs *observability.Client, topo *topology.Topology, cfg *topology_config.TopologyConfig, creds []managementSessions.NodeCredentials) error {
	if topo == nil || cfg == nil {
		return fmt.Errorf("topology and config must not be nil")
	}

	// create a new configuration transaction
	tx := NewConfigurationTransaction(cfg.GetConfigId())
	var skipped []string

	for _, node := range topo.Nodes {
		if node == nil {
			continue
		}
		if node.ManagementInfo == nil {
			if node.GetType() != topology.NodeRole_END_STATION {
				m.logger.Printf("Skipping node %q: managementInfo is missing", node.Name)
				skipped = append(skipped, fmt.Sprintf("node %q: managementInfo is missing", node.Name))
			}
			continue
		}

		nodeCfg := findNodeConfig(cfg, node.Name)
		if nodeCfg == nil {
			skipped = append(skipped, fmt.Sprintf("node %q: no matching node configuration", node.Name))
			m.logger.Printf("Skipping node %q: no matching node configuration in %q", node.Name, cfg.GetConfigId())
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
				skipped = append(skipped, fmt.Sprintf("node %q: %v", node.Name, err))
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
	if len(tx.Operations) == 0 {
		if len(skipped) == 0 {
			skipped = append(skipped, "topology has no managed nodes")
		}
		return fmt.Errorf("configuration %q has no eligible device operations: %s", cfg.GetConfigId(), strings.Join(skipped, "; "))
	}
	m.logger.Printf("Applying configuration %q to %d device(s)", cfg.GetConfigId(), len(tx.Operations))

	//set concurrency limit based on available workers
	tx.MAX_CONCURRENT_CONFIGS, _ = getAvailableWorkers()

	preparationStart := time.Now()

	if err := tx.Prepare(); err != nil {
		if obs != nil {
			_ = obs.Metric(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"configuration_preparation_failures",
				observabilityv1.MetricType_METRIC_TYPE_COUNTER,
				1,
				"",
				nil,
			)
		}
		return err
	}

	if obs != nil {
		_ = obs.Metric(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"configuration_preparation_duration",
			observabilityv1.MetricType_METRIC_TYPE_GAUGE,
			float64(time.Since(preparationStart).Milliseconds()),
			"ms",
			nil,
		)
	}

	commitStart := time.Now()

	if err := tx.Commit(ctx, obs); err != nil {
		if obs != nil {
			_ = obs.Metric(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"configuration_commit_failures",
				observabilityv1.MetricType_METRIC_TYPE_COUNTER,
				1,
				"",
				nil,
			)
		}
		return err
	}

	if obs != nil {
		_ = obs.Metric(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"configuration_commit_duration",
			observabilityv1.MetricType_METRIC_TYPE_GAUGE,
			float64(time.Since(commitStart).Milliseconds()),
			"ms",
			nil,
		)
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

func (m *MappingEngine) Rollback(ctx context.Context, obs *observability.Client, reqStart time.Time) error {
	m.lastTransactionMu.Lock()
	defer m.lastTransactionMu.Unlock()

	if reqStart.IsZero() {
		reqStart = time.Now()
	}

	if m.lastTransaction == nil {
		if obs != nil {
			_ = obs.Metric(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"configuration_rollback_failures",
				observabilityv1.MetricType_METRIC_TYPE_COUNTER,
				1,
				"",
				nil,
			)
		}
		return fmt.Errorf("transaction rollback is available only after a successful configuration transaction!!")
	}

	tx := m.lastTransaction

	// Phase 1: Prepare rollback (validate snapshots and generate rollback payloads for switches)
	if err := tx.PrepareRollback(ctx); err != nil {
		if obs != nil {
			_ = obs.Metric(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"configuration_rollback_failures",
				observabilityv1.MetricType_METRIC_TYPE_COUNTER,
				1,
				"",
				nil,
			)
		}
		return err
	}

	// The rollback process is done and ready just before it is sent to the switches!
	readyTime := time.Now()
	rollbackLatencyMs := float64(readyTime.Sub(reqStart).Nanoseconds()) / 1e6
	metrics.RecordRollbackLatency(tx.ConfigId, reqStart, readyTime)

	if obs != nil {
		_ = obs.Metric(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"configuration_rollback_duration",
			observabilityv1.MetricType_METRIC_TYPE_GAUGE,
			rollbackLatencyMs,
			"ms",
			nil,
		)
	}

	// Phase 2: Commit rollback to switches (pushing restored configuration)
	if err := tx.CommitRollback(ctx, obs); err != nil {
		if obs != nil {
			_ = obs.Metric(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"configuration_rollback_failures",
				observabilityv1.MetricType_METRIC_TYPE_COUNTER,
				1,
				"",
				nil,
			)
		}
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
