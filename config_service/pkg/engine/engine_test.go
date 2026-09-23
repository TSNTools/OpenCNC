package engine

import (
	"context"
	"strings"
	"testing"

	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
)

func TestApplyConfigurationRejectsUnmanagedTopology(t *testing.T) {
	m := NewMappingEngine(nil)
	topo := &topology.Topology{Nodes: []*topology.Node{{Name: "SWITCH"}}}
	cfg := &topology_config.TopologyConfig{
		ConfigId:    "default_schedule",
		NodeConfigs: []*topology_config.NodeConfig{{NodeId: "SWITCH"}},
	}
	err := m.ApplyConfiguration(context.Background(), nil, topo, cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "no eligible device operations") {
		t.Fatalf("expected empty deployment to fail, got %v", err)
	}
	if m.GetLastTransactionId() != nil {
		t.Fatal("empty deployment must not become the last successful transaction")
	}
}
