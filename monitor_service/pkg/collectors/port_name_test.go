package collectors

import (
	"OpenCNC/monitor_service/structures/monitoring"
	"testing"
)

func TestDeviceNameSelectionKeepsTopologyIdentity(t *testing.T) {
	id := "P2"
	target := &monitoring.ResourceKey{NodeId: "B&R", PortId: &id}
	collector := NewNetconfCollector(target, nil, nil, nil, 1000, "sw0p2")
	reply := `<rpc-reply><data><interfaces><interface><name>P2</name><statistics><in-octets>999</in-octets></statistics></interface><interface><name>sw0p2</name><statistics><in-octets>123</in-octets></statistics></interface><interface><name>sw0p3</name><statistics><in-octets>456</in-octets></statistics></interface></interfaces></data></rpc-reply>`
	counters := []*monitoring.Counter{{Name: "if_state_in_octets", Path: "/interfaces/interface/statistics/in-octets"}}
	values, err := parseCounters(reply, counters, collector.interfaceName)
	if err != nil {
		t.Fatal(err)
	}
	if values["if_state_in_octets"] != 123 {
		t.Fatalf("wrong interface selected: %v", values)
	}
	if collector.Target() != target || collector.Target().GetPortId() != "P2" {
		t.Fatal("sample identity changed")
	}
	legacy := NewNetconfCollector(target, nil, nil, nil, 1000)
	if legacy.interfaceName != "P2" {
		t.Fatal("legacy constructor fallback broken")
	}
}
