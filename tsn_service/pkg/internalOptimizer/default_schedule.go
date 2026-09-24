package internalOptimizer

import (
	"encoding/json"
	"fmt"
	"math"
	"net/netip"
	"strings"

	"OpenCNC/common/structures/qbv"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/common/structures/vlan"

	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/proto"
)

type trafficClassConfig struct {
	Name            string `json:"name"`
	AssignedPortion uint64 `json:"assigned-portion"`
}

type portScheduleConfig struct {
	PortID         string               `json:"port-id"`
	GatingCycle    uint64               `json:"gating-cycle"`
	TrafficClasses []trafficClassConfig `json:"traffic-classes"`
}

type scheduleConfig struct {
	Ports          []portScheduleConfig `json:"ports"`
	GatingCycle    uint64               `json:"gating-cycle"`
	TrafficClasses []trafficClassConfig `json:"traffic-classes"`
}

type switchScheduleConfig struct {
	ManagementIP string `json:"management-ip"`
	scheduleConfig
}

type defaultScheduleConfig struct {
	scheduleConfig
	Switches []switchScheduleConfig `json:"switches"`
}

// A management-IP switch schedule replaces the shared schedule in full. Only ports
// listed in that override are configured, unless it defines a global schedule.
func buildDefaultSchedule(data []byte, topo *topology.Topology) (*topology_config.TopologyConfig, error) {
	if topo == nil {
		return nil, fmt.Errorf("topology is nil")
	}
	jsonBytes, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule YAML: %w", err)
	}
	var config defaultScheduleConfig
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, fmt.Errorf("invalid schedule: %w", err)
	}
	fallback, err := buildPortSchedules("default", config.scheduleConfig)
	if err != nil {
		return nil, err
	}
	overrides := make(map[string]map[string]*qbv.GateControlList)
	for _, sw := range config.Switches {
		ip, err := netip.ParseAddr(sw.ManagementIP)
		if err != nil {
			return nil, fmt.Errorf("switch schedule requires a valid management-ip: %q", sw.ManagementIP)
		}
		key := ip.Unmap().String()
		if _, exists := overrides[key]; exists {
			return nil, fmt.Errorf("duplicate switch schedule %q", sw.ManagementIP)
		}
		schedules, err := buildPortSchedules(sw.ManagementIP, sw.scheduleConfig)
		if err != nil {
			return nil, err
		}
		if len(schedules) == 0 {
			return nil, fmt.Errorf("switch %q has no ports or traffic-classes", sw.ManagementIP)
		}
		overrides[key] = schedules
	}
	if len(fallback) == 0 && len(overrides) == 0 {
		return nil, fmt.Errorf("no switches, ports or traffic-classes defined in schedule file")
	}
	cfg := &topology_config.TopologyConfig{ConfigId: defaultSchedID}
	seenNames := make(map[string]bool)
	for _, node := range topo.GetNodes() {
		if node == nil {
			continue
		}
		// Deployment and topology storage still identify nodes by name.
		// Reject collisions instead of assigning one device another device's GCL.
		if seenNames[node.Name] {
			return nil, fmt.Errorf("duplicate topology node name %q: node IDs must remain unique; schedule overrides match management IP", node.Name)
		}
		seenNames[node.Name] = true
		ip, _ := netip.ParseAddr(node.GetManagementInfo().GetIpAddress())
		schedules, ok := overrides[ip.Unmap().String()]
		if !ok {
			schedules = fallback
		}
		nodeCfg := &topology_config.NodeConfig{NodeId: node.Name}
		for _, port := range node.GetPorts() {
			if port == nil {
				continue
			}
			gcl := schedules[port.Id]
			if gcl == nil {
				gcl = schedules["*"]
			}
			if gcl == nil {
				continue
			}
			defaultVlan := uint32(100)
			nodeCfg.PortConfigs = append(nodeCfg.PortConfigs, &topology_config.PortConfig{
				PortId:          port.Id,
				Gcl:             proto.Clone(gcl).(*qbv.GateControlList),
				DefaultVlanId:   &defaultVlan,
				VlanMemberships: []*vlan.VlanMembership{
					{
						VlanId: defaultVlan,
						Tagged: true,
					},
				},
			})
		}
		if len(nodeCfg.PortConfigs) > 0 {
			cfg.NodeConfigs = append(cfg.NodeConfigs, nodeCfg)
		}
	}
	return cfg, nil
}

func buildPortSchedules(owner string, config scheduleConfig) (map[string]*qbv.GateControlList, error) {
	result := make(map[string]*qbv.GateControlList)
	if len(config.Ports) > 0 {
		for _, port := range config.Ports {
			if port.PortID == "" || port.PortID == "*" {
				return nil, fmt.Errorf("schedule %q requires a concrete port-id", owner)
			}
			if _, exists := result[port.PortID]; exists {
				return nil, fmt.Errorf("duplicate port %q in schedule %q", port.PortID, owner)
			}
			gcl, err := buildGCL(owner+"/"+port.PortID, port.GatingCycle, port.TrafficClasses)
			if err != nil {
				return nil, err
			}
			result[port.PortID] = gcl
		}
	} else if len(config.TrafficClasses) > 0 {
		gcl, err := buildGCL(owner, config.GatingCycle, config.TrafficClasses)
		if err != nil {
			return nil, err
		}
		result["*"] = gcl
	}
	return result, nil
}

func buildGCL(id string, cycle uint64, classes []trafficClassConfig) (*qbv.GateControlList, error) {
	if cycle == 0 || cycle > math.MaxUint64/1_000_000 {
		return nil, fmt.Errorf("invalid gating-cycle for %s", id)
	}
	var total uint64
	for _, tc := range classes {
		if tc.AssignedPortion > 100 {
			return nil, fmt.Errorf("assigned portion exceeds 100%% for %s", id)
		}
		total += tc.AssignedPortion
	}
	if total != 100 {
		return nil, fmt.Errorf("traffic class assigned portions must add up to 100%% for %s, got %d%%", id, total)
	}
	bits := map[string]uint{
		"isochronous": 0, "cyclic-sync": 1, "cyclic-async": 2, "alarms-events": 3,
		"config-diag": 4, "network-control": 5, "best-effort": 6, "timecritical": 7,
		"time-critical": 7,
	}
	gcl := &qbv.GateControlList{
		ScheduleId: id, BaseTime: 0, BaseTimeRef: qbv.TimestampReference_DEVICE_BOOT_TIME,
		CycleTime: cycle * 1_000_000, AdminState: qbv.AdminState_ENABLED,
		Description: fmt.Sprintf("Qbv schedule for %s", id),
	}
	for i, tc := range classes {
		bit, ok := bits[strings.ToLower(tc.Name)]
		if !ok {
			return nil, fmt.Errorf("unknown traffic class %q in schedule %s", tc.Name, id)
		}
		gcl.Entries = append(gcl.Entries, &qbv.GateControlEntry{
			Index: uint32(i), TimeInterval: (gcl.CycleTime / 100) * tc.AssignedPortion,
			GateStates: []byte{1 << bit}, Description: tc.Name,
		})
	}
	return gcl, nil
}
