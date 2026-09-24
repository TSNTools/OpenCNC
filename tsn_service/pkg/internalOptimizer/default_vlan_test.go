package internalOptimizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"OpenCNC/common/structures/topology_config"
)

const testSchedule = "gating-cycle: 1\ntraffic-classes: [{name: best-effort, assigned-portion: 100}]"

func TestDefaultVLANManagementIPSelection(t *testing.T) {
	topo := scheduleTopology("SWITCH", "B&R", "other")
	topo.Nodes[0].Name = "renamed"
	cfg, err := buildDefaultSchedule([]byte(testSchedule), topo)
	if err != nil {
		t.Fatal(err)
	}
	err = applyDefaultVLAN([]byte(`default-vlan: {vid: 100}
switches:
- management-ip: "::ffff:192.168.4.64"
  default-vlan:
    vid: 234
    fid: 9
    tagged: false
    ports: [{port-id: P2}]
- management-ip: 192.168.4.65
  default-vlan: {vid: 300, ports: []}
`), topo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	node := cfg.NodeConfigs[0]
	if node.PortConfigs[0].DefaultVlanId != nil || len(node.PortConfigs[0].VlanMemberships) != 0 {
		t.Fatal("unselected port received VLAN")
	}
	if node.PortConfigs[0].Gcl == nil {
		t.Fatal("VLAN selection removed schedule")
	}
	pc := node.PortConfigs[1]
	if pc.GetDefaultVlanId() != 234 || pc.VlanMemberships[0].Tagged {
		t.Fatal("incorrect port VLAN")
	}
	reg := node.Bridge.VlanConfig.VlanRegistrationEntries[0]
	if reg.DatabaseId != 9 || len(reg.PortMaps) != 1 || reg.PortMaps[0].PortId != "P2" || reg.PortMaps[0].VlanTransmitted.String() != "VLAN_TRANSMITTED_UNTAGGED" {
		t.Fatalf("incorrect registration: %v", reg)
	}
	if cfg.NodeConfigs[1].Bridge != nil || cfg.NodeConfigs[1].PortConfigs[0].DefaultVlanId != nil {
		t.Fatal("empty port list received VLAN")
	}
	if len(cfg.NodeConfigs[2].Bridge.VlanConfig.VlanRegistrationEntries[0].PortMaps) != 2 {
		t.Fatal("shared fallback did not cover all ports")
	}

}

func TestDefaultVLANNoSharedFallback(t *testing.T) {
	topo := scheduleTopology("SWITCH", "B&R")
	cfg, err := buildDefaultSchedule([]byte(testSchedule), topo)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDefaultVLAN([]byte("switches: [{management-ip: 192.168.4.64, default-vlan: {vid: 200, ports: [{port-id: P1}]}}]"), topo, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.NodeConfigs[0].Bridge == nil || cfg.NodeConfigs[1].Bridge != nil {
		t.Fatal("switch-only configuration did not isolate VLAN defaults")
	}
}

func TestInvalidDefaultVLAN(t *testing.T) {
	for _, tc := range []struct{ data, want string }{
		{"{}", "requires default-vlan"},
		{"default-vlan: {vid: 0}", "between 1 and 4094"},
		{"default-vlan: {vid: 4095}", "between 1 and 4094"},
		{"default-vlan: {vid: 100, port: P1}", "unknown field"},
		{"default-vlan: {vid: 100, ports: [{port-id: missing}]}", "unknown port-id"},
		{"default-vlan: {vid: 100, ports: [{port-id: P1}, {port-id: P1}]}", "duplicate port-id"},
		{"default-vlan: {vid: 100, ports: [{}]}", "empty"},
		{"switches: [{management-ip: invalid, default-vlan: {vid: 100}}]", "valid management-ip"},
		{"switches: [{management-ip: 192.168.4.64, default-vlan: {vid: 100}}, {management-ip: '::ffff:192.168.4.64', default-vlan: {vid: 200}}]", "duplicate VLAN switch"},
	} {
		t.Run(tc.want+tc.data, func(t *testing.T) {
			topo := scheduleTopology("SWITCH")
			cfg, err := buildDefaultSchedule([]byte(testSchedule), topo)
			if err != nil {
				t.Fatal(err)
			}
			err = applyDefaultVLAN([]byte(tc.data), topo, cfg)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReadDefaultVLAN(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vlan.yaml")
	data := []byte("default-vlan: {vid: 222}")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEFAULT_VLAN_PATH", path)
	got, err := readDefaultVLAN()
	if err != nil || string(got) != string(data) {
		t.Fatalf("read override: %q, %v", got, err)
	}
	t.Setenv("DEFAULT_VLAN_PATH", path+".missing")
	if _, err := readDefaultVLAN(); err == nil {
		t.Fatal("missing explicit file must fail")
	}
	t.Setenv("DEFAULT_VLAN_PATH", "")
	got, err = readDefaultVLAN()
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDefaultVLAN(got, scheduleTopology(), &topology_config.TopologyConfig{}); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultVLANMixedPortRegistration(t *testing.T) {
	topo := scheduleTopology("SWITCH")
	cfg, err := buildDefaultSchedule([]byte(testSchedule), topo)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte(`switches:
- management-ip: 192.168.4.64
  default-vlan:
    vid: 10
    fid: 1
    registrar-admin-control: normal
    ports:
    - port-id: P1
      tagged: false
    - port-id: P2
      tagged: true
`)
	if err := applyDefaultVLAN(data, topo, cfg); err != nil {
		t.Fatal(err)
	}
	node := cfg.NodeConfigs[0]
	reg := node.Bridge.VlanConfig.VlanRegistrationEntries[0]
	if reg.VlanIds[0] != 10 || reg.DatabaseId != 1 || len(reg.PortMaps) != 2 {
		t.Fatalf("wrong registration: %v", reg)
	}
	for i, want := range []string{"VLAN_TRANSMITTED_UNTAGGED", "VLAN_TRANSMITTED_TAGGED"} {
		pm := reg.PortMaps[i]
		if pm.PortId != topo.Nodes[0].Ports[i].Id || pm.VlanTransmitted.String() != want || pm.RegistrarAdminControl.String() != "REGISTRAR_ADMIN_CONTROL_NORMAL" {
			t.Fatalf("wrong port map: %v", pm)
		}
		if node.PortConfigs[i].VlanMemberships[0].Tagged != (i == 1) {
			t.Fatal("membership tagging disagrees with registration")
		}
	}
	for _, bad := range []string{
		"default-vlan: {vid: 10, registrar-admin-control: typo}",
		"default-vlan: {vid: 10, ports: [{port-id: P1, registrar-admin-control: typo}]}",
	} {
		if err := applyDefaultVLAN([]byte(bad), topo, cfg); err == nil {
			t.Fatal("invalid registrar control accepted")
		}
	}
}
