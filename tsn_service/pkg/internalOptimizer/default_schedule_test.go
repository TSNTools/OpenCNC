package internalOptimizer

import (
	"OpenCNC/common/structures/topology"
	"os"
	"strings"
	"testing"
)

func scheduleTopology(names ...string) *topology.Topology {
	topo := &topology.Topology{}
	for _, name := range names {
		topo.Nodes = append(topo.Nodes, &topology.Node{Name: name, ManagementInfo: &topology.ManagementInfo{IpAddress: map[string]string{"SWITCH": "192.168.4.64", "B&R": "192.168.4.65"}[name]}, Ports: []*topology.Port{{Id: "P1"}, {Id: "P2"}}})
	}
	return topo
}

func TestSwitchSchedulesAndFallback(t *testing.T) {
	data := []byte(`gating-cycle: 4
traffic-classes:
  - name: best-effort
    assigned-portion: 100
switches:
  - management-ip: 192.168.4.64
    ports:
      - port-id: P1
        gating-cycle: 1
        traffic-classes:
          - name: isochronous
            assigned-portion: 60
          - name: best-effort
            assigned-portion: 40
  - management-ip: 192.168.4.65
    gating-cycle: 2
    traffic-classes:
      - name: Timecritical
        assigned-portion: 100
`)
	cfg, err := buildDefaultSchedule(data, scheduleTopology("SWITCH", "B&R", "other"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.NodeConfigs) != 3 {
		t.Fatalf("got %d node configs", len(cfg.NodeConfigs))
	}
	a, b, fallback := cfg.NodeConfigs[0], cfg.NodeConfigs[1], cfg.NodeConfigs[2]
	if len(a.PortConfigs) != 1 {
		t.Fatal("switch override must not inherit unspecified ports")
	}
	if a.PortConfigs[0].Gcl.CycleTime != 1_000_000 || a.PortConfigs[0].Gcl.Entries[0].TimeInterval != 600_000 || a.PortConfigs[0].Gcl.Entries[0].GateStates[0] != 1 {
		t.Fatal("incorrect SWITCH schedule")
	}
	if len(b.PortConfigs) != 2 || b.PortConfigs[0].Gcl.CycleTime != 2_000_000 || b.PortConfigs[0].Gcl.Entries[0].GateStates[0] != 128 {
		t.Fatal("incorrect B&R schedule")
	}
	if fallback.PortConfigs[0].Gcl.CycleTime != 4_000_000 {
		t.Fatal("shared fallback not applied")
	}
	b.PortConfigs[0].Gcl.Entries[0].GateStates[0] = 0
	if b.PortConfigs[1].Gcl.Entries[0].GateStates[0] != 128 {
		t.Fatal("ports share mutable GCL state")
	}
}

func TestLegacyPortSchedule(t *testing.T) {
	data := []byte(`ports:
  - port-id: P1
    gating-cycle: 3
    traffic-classes:
      - name: cyclic-sync
        assigned-portion: 100
`)
	cfg, err := buildDefaultSchedule(data, scheduleTopology("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range cfg.NodeConfigs {
		if len(node.PortConfigs) != 1 || node.PortConfigs[0].Gcl.CycleTime != 3_000_000 {
			t.Fatalf("legacy schedule changed: %v", node)
		}
	}
}

func TestInvalidSwitchSchedules(t *testing.T) {
	for _, tc := range []struct{ name, data, want string }{
		{"missing name", "switches: [{gating-cycle: 1}]", "valid management-ip"},
		{"empty override", "switches: [{management-ip: 192.168.4.64}]", "no ports or traffic-classes"},
		{"bad total", "switches: [{management-ip: 192.168.4.64, gating-cycle: 1, traffic-classes: [{name: best-effort, assigned-portion: 80}]}]", "100%"},
		{"duplicate node", "switches: [{management-ip: 192.168.4.64, gating-cycle: 1, traffic-classes: [{name: best-effort, assigned-portion: 100}]}, {management-ip: 192.168.4.64}]", "duplicate switch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildDefaultSchedule([]byte(tc.data), scheduleTopology("SWITCH"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestRepositoryDefaultSchedule(t *testing.T) {
	data, err := os.ReadFile("../../configs/default-schedule.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := buildDefaultSchedule(data, scheduleTopology("SWITCH", "B&R"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.NodeConfigs) != 2 {
		t.Fatalf("expected both switches, got %d", len(cfg.NodeConfigs))
	}
}

func TestScheduleSelectionIgnoresNodeName(t *testing.T) {
	data, err := os.ReadFile("../../configs/default-schedule.yaml")
	if err != nil {
		t.Fatal(err)
	}
	topo := scheduleTopology("SWITCH", "B&R")
	topo.Nodes[0].Name = "renamed-relyum"
	topo.Nodes[1].Name = "renamed-bandr"
	cfg, err := buildDefaultSchedule(data, topo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NodeConfigs[0].PortConfigs[0].Gcl.Entries[0].GateStates[0] != 1 || cfg.NodeConfigs[1].PortConfigs[0].Gcl.Entries[0].GateStates[0] != 128 {
		t.Fatal("IP schedules were lost after renaming nodes")
	}
	topo.Nodes[1].Name = topo.Nodes[0].Name
	if _, err := buildDefaultSchedule(data, topo); err == nil {
		t.Fatal("duplicate deployment IDs must be rejected")
	}
}

func TestEquivalentIPv6OverridesRejected(t *testing.T) {
	data := []byte(`switches:
  - management-ip: "2001:db8::1"
    gating-cycle: 1
    traffic-classes: [{name: best-effort, assigned-portion: 100}]
  - management-ip: "2001:0db8:0:0:0:0:0:1"
    gating-cycle: 2
    traffic-classes: [{name: best-effort, assigned-portion: 100}]
`)
	if _, err := buildDefaultSchedule(data, scheduleTopology("SWITCH")); err == nil || !strings.Contains(err.Error(), "duplicate switch") {
		t.Fatalf("expected duplicate IP error, got %v", err)
	}
}

// Exercise the shipped YAML with identical port IDs on two switches. Verify
// every port's cycle, gate order, and intervals survive in the generated config.
func TestPerIPPerPortDefaultSchedules(t *testing.T) {
	data, err := os.ReadFile("../../configs/default-schedule.yaml")
	if err != nil {
		t.Fatal(err)
	}
	topo := scheduleTopology("SWITCH", "B&R")
	for _, node := range topo.Nodes {
		node.Ports = []*topology.Port{{Id: "P1"}, {Id: "P2"}, {Id: "P3"}, {Id: "P4"}}
	}
	cfg, err := buildDefaultSchedule(data, topo)
	if err != nil {
		t.Fatal(err)
	}
	expected := [][]struct {
		cycle    uint64
		gates    []byte
		portions []uint64
	}{
		{
			{1, []byte{1, 2, 4, 32, 16, 8, 64, 128}, []uint64{20, 10, 10, 10, 10, 10, 20, 10}},
			{2, []byte{1, 2, 4, 8, 16, 32, 64, 128}, []uint64{20, 10, 10, 10, 10, 10, 20, 10}},
			{1, []byte{1, 2, 4, 1, 16, 32, 64, 128}, []uint64{20, 10, 10, 10, 10, 10, 20, 10}},
			{4, []byte{1, 32, 4, 8, 16, 32, 64, 128}, []uint64{20, 10, 10, 10, 10, 10, 20, 10}},
		},
		{
			{2, []byte{128, 32, 64}, []uint64{60, 10, 30}},
			{3, []byte{2, 8, 64}, []uint64{50, 20, 30}},
			{4, []byte{4, 64}, []uint64{30, 70}},
			{5, []byte{32, 16, 64}, []uint64{40, 40, 20}},
		},
	}
	if len(cfg.NodeConfigs) != len(expected) {
		t.Fatalf("got %d nodes", len(cfg.NodeConfigs))
	}
	for i, node := range cfg.NodeConfigs {
		if len(node.PortConfigs) != 4 {
			t.Fatalf("%s: got %d ports", node.NodeId, len(node.PortConfigs))
		}
		for j, port := range node.PortConfigs {
			want := expected[i][j]
			gcl := port.Gcl
			if port.PortId != topo.Nodes[i].Ports[j].Id || gcl.CycleTime != want.cycle*1_000_000 || len(gcl.Entries) != len(want.gates) {
				t.Fatalf("%s/%s: unexpected cycle or gate count: %v", node.NodeId, port.PortId, gcl)
			}
			var total uint64
			for k, entry := range gcl.Entries {
				if len(entry.GateStates) != 1 || entry.GateStates[0] != want.gates[k] || entry.TimeInterval != want.cycle*10_000*want.portions[k] {
					t.Fatalf("%s/%s entry %d: incorrect gate or interval: %v", node.NodeId, port.PortId, k, entry)
				}
				total += entry.TimeInterval
			}
			if total != gcl.CycleTime {
				t.Fatalf("%s/%s: intervals do not fill cycle", node.NodeId, port.PortId)
			}
		}
	}
}
