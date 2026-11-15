// It tests whether the function:
// - Produces the eight default mappings.
// - Preserves an existing mapping.
// - Respects explicitly disabled mapping.
// - Avoids unsupported assumptions when required data is absent.
package internalOptimizer

import (
	"testing"

	"OpenCNC/common/structures/pcp"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

func defaultPCPTestFPM() *forwardingplane.ForwardingPlaneModel {
	return &forwardingplane.ForwardingPlaneModel{
		Topology: &topology.Topology{Nodes: []*topology.Node{{
			Name: "SW1",
			Ports: []*topology.Port{
				{Id: "P1", NumberOfQueues: 8},
				{Id: "P2", NumberOfQueues: 4},
			},
		}}},
		Configuration: &topology_config.TopologyConfig{NodeConfigs: []*topology_config.NodeConfig{{
			NodeId: "SW1",
			PortConfigs: []*topology_config.PortConfig{
				{PortId: "P1"},
				{PortId: "P2"},
			},
		}}},
	}
}

func TestDefaultPCPQueueMappingIsRequestOnly(t *testing.T) {
	original := defaultPCPTestFPM()
	before := proto.Clone(original)

	got, err := withDefaultPCPQueueMappings(original)
	if err != nil {
		t.Fatal(err)
	}
	if got == original || !proto.Equal(original, before) {
		t.Fatal("request fallback mutated the original FPM")
	}
	port := got.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[0]
	if !port.GetPcpMappingEnabled() || port.PcpMappingEnabled == nil {
		t.Fatal("fallback should mark the request-only PCP mapping enabled")
	}
	entries := port.GetPcpConfig().GetTrafficClassTable()
	if len(entries) != 8 {
		t.Fatalf("expected 8 identity mappings, got %d", len(entries))
	}
	for i, entry := range entries {
		if entry.GetPriority() != uint32(i) || entry.GetEgressQueueId() != uint32(i) {
			t.Fatalf("unexpected default mapping at index %d: %+v", i, entry)
		}
	}
	if got.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[1].GetPcpConfig() != nil {
		t.Fatal("must not assume 8P0D on a four-queue port")
	}
}

func TestDefaultPCPPreservesExplicitMapping(t *testing.T) {
	original := defaultPCPTestFPM()
	port := original.Configuration.NodeConfigs[0].PortConfigs[0]
	port.PcpMappingEnabled = proto.Bool(true)
	port.PcpConfig = &pcp.PcpPortConfig{TrafficClassTable: []*pcp.TrafficClassMappingEntry{{
		Priority: 7, EgressQueueId: 0,
	}}}

	got, err := withDefaultPCPQueueMappings(original)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(original, got) {
		t.Fatal("existing PCP mapping must not be changed")
	}
}

func TestDefaultPCPDoesNotOverrideExplicitDisable(t *testing.T) {
	original := defaultPCPTestFPM()
	port := original.Configuration.NodeConfigs[0].PortConfigs[0]
	port.PcpMappingEnabled = proto.Bool(false)

	got, err := withDefaultPCPQueueMappings(original)
	if err != nil {
		t.Fatal(err)
	}
	gotPort := got.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[0]
	if gotPort.PcpMappingEnabled == nil || gotPort.GetPcpMappingEnabled() || gotPort.GetPcpConfig() != nil {
		t.Fatal("explicitly disabled PCP mapping must be preserved")
	}
}

func TestDefaultPCPSkipsAbsentTopologyOrConfiguration(t *testing.T) {
	for _, fpm := range []*forwardingplane.ForwardingPlaneModel{
		{Topology: &topology.Topology{}},
		{Configuration: &topology_config.TopologyConfig{}},
	} {
		got, err := withDefaultPCPQueueMappings(fpm)
		if err != nil || got != fpm {
			t.Fatalf("missing inputs must be left unchanged, got %v", err)
		}
	}
}
