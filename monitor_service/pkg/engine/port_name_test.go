package engine

import (
	"OpenCNC/common/structures/topology"
	"OpenCNC/monitor_service/structures/monitoring"
	"testing"
)

func TestResolveInterfaceName(t *testing.T) {
	for _, tt := range []struct {
		name, id, portName, want string
		missing                  bool
	}{
		{name: "different topology ID", id: "P2", portName: "sw0p2", want: "sw0p2"},
		{name: "legacy interface ID", id: "sw0p2", want: "sw0p2"},
		{name: "trim name", id: "P2", portName: " sw0p2 ", want: "sw0p2"},
		{name: "blank name falls back", id: "sw0p2", portName: " ", want: "sw0p2"},
		{name: "node target", want: ""},
		{name: "unknown port", id: "missing", missing: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			node := &topology.Node{Name: "B&R", Ports: []*topology.Port{{Id: tt.id, Name: tt.portName}}}
			if tt.missing {
				node.Ports = nil
			}
			resource := &monitoring.ResourceKey{NodeId: "B&R", PortId: &tt.id}
			got, err := resolveInterfaceName(node, resource)
			if tt.missing {
				if err == nil {
					t.Fatal("unknown port accepted")
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
			if resource.GetPortId() != tt.id {
				t.Fatal("topology ID was changed")
			}
		})
	}
}
