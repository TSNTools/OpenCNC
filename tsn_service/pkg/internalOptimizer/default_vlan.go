package internalOptimizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"

	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/common/structures/vlan"
	"github.com/ghodss/yaml"
)

type vlanPortConfig struct {
	PortID                string `json:"port-id"`
	Tagged                *bool  `json:"tagged"`
	RegistrarAdminControl string `json:"registrar-admin-control"`
}
type vlanSettings struct {
	RegistrarAdminControl string           `json:"registrar-admin-control"`
	VID                   uint32           `json:"vid"`
	FID                   *uint32          `json:"fid"`
	Tagged                *bool            `json:"tagged"`
	Ports                 []vlanPortConfig `json:"ports"`
}
type switchVLANConfig struct {
	ManagementIP string       `json:"management-ip"`
	DefaultVLAN  vlanSettings `json:"default-vlan"`
}
type defaultVLANConfig struct {
	DefaultVLAN *vlanSettings      `json:"default-vlan"`
	Switches    []switchVLANConfig `json:"switches"`
}

func (v vlanSettings) validate(owner string) error {
	if v.VID < 1 || v.VID > 4094 {
		return fmt.Errorf("VLAN %s: vid must be between 1 and 4094", owner)
	}
	if _, err := vlanRegistrarControl(v.RegistrarAdminControl); err != nil {
		return fmt.Errorf("VLAN %s: %w", owner, err)
	}
	seen := map[string]bool{}
	for _, port := range v.Ports {
		if port.PortID == "" || port.PortID == "*" || seen[port.PortID] {
			return fmt.Errorf("VLAN %s: empty, wildcard or duplicate port-id %q", owner, port.PortID)
		}
		if _, err := vlanRegistrarControl(port.RegistrarAdminControl); err != nil {
			return fmt.Errorf("VLAN %s port %s: %w", owner, port.PortID, err)
		}
		seen[port.PortID] = true
	}
	return nil
}

// Switch overrides replace shared VLAN settings in full, independently of Qbv.
// Omitted ports select all topology ports; ports: [] selects none.
func applyDefaultVLAN(data []byte, topo *topology.Topology, cfg *topology_config.TopologyConfig) error {
	if topo == nil || cfg == nil {
		return fmt.Errorf("default VLAN requires topology and configuration")
	}
	raw, err := yaml.YAMLToJSON(data)
	if err != nil {
		return fmt.Errorf("invalid VLAN YAML: %w", err)
	}
	var config defaultVLANConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("invalid VLAN configuration: %w", err)
	}
	if config.DefaultVLAN == nil && len(config.Switches) == 0 {
		return fmt.Errorf("VLAN configuration requires default-vlan or switches")
	}
	if config.DefaultVLAN != nil {
		if err := config.DefaultVLAN.validate("default"); err != nil {
			return err
		}
	}
	overrides := map[string]*vlanSettings{}
	for i := range config.Switches {
		sw := &config.Switches[i]
		ip, err := netip.ParseAddr(sw.ManagementIP)
		if err != nil {
			return fmt.Errorf("VLAN switch requires a valid management-ip: %q", sw.ManagementIP)
		}
		key := ip.Unmap().String()
		if _, exists := overrides[key]; exists {
			return fmt.Errorf("duplicate VLAN switch management-ip %q", sw.ManagementIP)
		}
		if err := sw.DefaultVLAN.validate(sw.ManagementIP); err != nil {
			return err
		}
		overrides[key] = &sw.DefaultVLAN
	}
	nodes := map[string]*topology.Node{}
	for _, node := range topo.GetNodes() {
		if node == nil {
			continue
		}
		if _, exists := nodes[node.Name]; exists {
			return fmt.Errorf("duplicate topology node name %q", node.Name)
		}
		nodes[node.Name] = node
	}
	for _, nc := range cfg.GetNodeConfigs() {
		if nc == nil {
			continue
		}
		node := nodes[nc.NodeId]
		if node == nil {
			return fmt.Errorf("VLAN configuration references unknown node %q", nc.NodeId)
		}
		ip, _ := netip.ParseAddr(node.GetManagementInfo().GetIpAddress())
		settings := overrides[ip.Unmap().String()]
		if settings == nil {
			settings = config.DefaultVLAN
		}
		if settings == nil {
			continue
		}
		available := map[string]bool{}
		for _, port := range node.GetPorts() {
			if port == nil {
				continue
			}
			if port.Id == "" || available[port.Id] {
				return fmt.Errorf("VLAN node %q has empty or duplicate topology port ID %q", node.Name, port.Id)
			}
			available[port.Id] = true
		}
		selected := available
		if settings.Ports != nil {
			selected = map[string]bool{}
			for _, port := range settings.Ports {
				if !available[port.PortID] {
					return fmt.Errorf("VLAN switch %q: unknown port-id %q", node.GetManagementInfo().GetIpAddress(), port.PortID)
				}
				selected[port.PortID] = true
			}
		}
		fid := uint32(1)
		if settings.FID != nil {
			fid = *settings.FID
		}
		tagged := true
		if settings.Tagged != nil {
			tagged = *settings.Tagged
		}
		control, _ := vlanRegistrarControl(settings.RegistrarAdminControl)
		portSettings := map[string]vlanPortConfig{}
		for _, p := range settings.Ports {
			portSettings[p.PortID] = p
		}
		reg := &vlan.VlanRegistrationEntry{DatabaseId: fid, VlanIds: []uint32{settings.VID}, EntryType: vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC}
		for _, pc := range nc.GetPortConfigs() {
			if pc == nil || !selected[pc.PortId] {
				continue
			}
			portTagged := tagged
			portControl := control
			options := portSettings[pc.PortId]
			if options.Tagged != nil {
				portTagged = *options.Tagged
			}
			if options.RegistrarAdminControl != "" {
				portControl, _ = vlanRegistrarControl(options.RegistrarAdminControl)
			}
			transmitted := vlan.VlanTransmitted_VLAN_TRANSMITTED_TAGGED
			if !portTagged {
				transmitted = vlan.VlanTransmitted_VLAN_TRANSMITTED_UNTAGGED
			}
			vid := settings.VID
			pc.DefaultVlanId = &vid
			pc.VlanMemberships = []*vlan.VlanMembership{{VlanId: vid, Tagged: portTagged}}
			reg.PortMaps = append(reg.PortMaps, &vlan.VlanPortMap{PortId: pc.PortId, RegistrarAdminControl: portControl, VlanTransmitted: transmitted})
		}
		if len(reg.PortMaps) == 0 {
			continue
		}
		if nc.Bridge == nil {
			nc.Bridge = &topology_config.BridgeConfig{}
		}
		nc.Bridge.VlanConfig = &vlan.BridgeVlanConfig{
			VlanRegistrationEntries: []*vlan.VlanRegistrationEntry{reg},
			VidToFidMappings:        []*vlan.VidToFidMapping{{Vid: settings.VID, Fid: fid}},
		}
	}
	return nil
}

func vlanRegistrarControl(value string) (vlan.RegistrarAdminControl, error) {
	switch value {
	case "", "fixed-new-ignored":
		return vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FIXED_NEW_IGNORED, nil
	case "normal":
		return vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_NORMAL, nil
	case "fixed-new-propagated":
		return vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FIXED_NEW_PROPAGATED, nil
	case "forbidden":
		return vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FORBIDDEN, nil
	default:
		return 0, fmt.Errorf("invalid registrar-admin-control %q", value)
	}
}
