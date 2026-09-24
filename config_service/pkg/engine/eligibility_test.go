package engine

import (
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	_ "OpenCNC/config_service/pkg/protocolbackends/netconf"
	"context"
	"strings"
	"testing"
)

func TestNoEligibleOperationsIncludesCause(t *testing.T) {
	for _, tc := range []struct {
		name         string
		info         *topology.ManagementInfo
		nodeID, want string
	}{
		{"missing management", nil, "SWITCH", "managementInfo is missing"},
		{"stale schedule", &topology.ManagementInfo{}, "old-switch", "no matching node configuration"},
		{"missing credentials", &topology.ManagementInfo{Protocol: topology.ManagementProtocol_NETCONF}, "SWITCH", "management credentials are nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			topo := &topology.Topology{Nodes: []*topology.Node{{Name: "SWITCH", Type: topology.NodeRole_BRIDGE, ManagementInfo: tc.info}}}
			cfg := &topology_config.TopologyConfig{ConfigId: "default_schedule", NodeConfigs: []*topology_config.NodeConfig{{NodeId: tc.nodeID}}}
			err := NewMappingEngine(nil).ApplyConfiguration(context.Background(), nil, topo, cfg, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "SWITCH") {
				t.Fatalf("expected node and cause %q, got %v", tc.want, err)
			}
		})
	}
}
