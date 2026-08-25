package internalOptimizer

import (
	"fmt"
	"strings"
	"tsn-service/pkg/structures/schedule"
	"tsn-service/pkg/structures/topology"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// TODO: Make generic paths for all the updates, they are currently built statically for our switches

// Creates configuration set request from given schedule and network topology
func createConfigurationFromSchedule(sched *schedule.Schedule, topo *topology.Topology) (*schedule.GclConfiguration, error) {

	gclConfig := &schedule.GclConfiguration{
		Configs: []*schedule.ConfigMap{},
	}

	for _, node := range topo.Nodes {
		nodeID := node.Name

		for _, port := range node.Ports {
			configMap := &schedule.ConfigMap{
				NodePort: fmt.Sprintf("%s.%s", nodeID, port.Name),
				Sched:    sched,
			}
			gclConfig.Configs = append(gclConfig.Configs, configMap)
		}
	}

	// For every object in the topology
	//for _, node := range topology.Nodes {

	// Add GCL and gating cycle for all port interfaces on device
	//for _, port := range devicePortMap[string(node.Name)] {
	//configMap:= schedule.ConfigMap{node.Id+"."+port.Id, sched}
	//gclConfig.Update(configMap)

	/*
		TODO: this is to use in the config service when we talk to gnmi switch
		// Create initial gate status updates
		statusChange := getStatusChangeElems(port, string(node.Name), len(sched.TrafficClasses))

		// Create GCL entries for a specific port
		gcl := getGclElems(sched, port, string(node.Name))

		// Create gating cycle entry for a specific port
		adminCycle := getAdminCycleTimeElems(sched.GatingCycle, port, string(node.Name))

		// Create extra time information and set config change to true
		timeInfoAndConfigChange := getFinalElems(port, string(node.Name))

		// Add all updates to set request
		req.Update = append(req.Update, statusChange...)
		req.Update = append(req.Update, gcl...)
		req.Update = append(req.Update, adminCycle...)
		req.Update = append(req.Update, timeInfoAndConfigChange...)
	*/
	//	}
	return gclConfig, nil
}

// Finds every port on the devices
func findAllPortsOnDevices(topology *topology.Topology) (map[string][]string, error) {
	var devicePortMap = map[string][]string{}

	// Find all ports on all devices
	for _, link := range topology.Links {

		//"source": "switch-c4.Port3"
		source := strings.Split(link.SourcePort, ".")
		srcDevice := source[0]
		srcPort := source[1]

		dest := strings.Split(link.TargetPort, ".")
		dstDevice := dest[0]
		dstPort := dest[1]

		// Append src device and dst device with their ports
		devicePortMap[srcDevice] = append(devicePortMap[srcDevice], srcPort)
		devicePortMap[dstDevice] = append(devicePortMap[dstDevice], dstPort)
	}

	return devicePortMap, nil
}

// Cycle time extension is statically 0 now, not sure what to do about it yet
// Base time is statically 0 on both seconds and fractional seconds now, not sure what to do about it yet
// Create updates for admin-cycle-time-extension, admin-base-time, and config-change
func getFinalElems(port string, deviceIp string) []*pb.Update {
	// Build update for admin cycle time extension
	cycleTimeExtUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-cycle-time-extension",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_UintVal{
				UintVal: 0, // This should maybe be calculated???
			},
		},
	}

	// Build update for admin base time seconds
	baseTimeSecondsUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-base-time",
					Key:  map[string]string{},
				},
				{
					Name: "seconds",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_StringVal{
				StringVal: "0", // This should maybe be calculated???
			},
		},
	}

	// Build update for admin base time fractional seconds
	baseTimeFractionalSecondsUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-base-time",
					Key:  map[string]string{},
				},
				{
					Name: "fractional-seconds",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_StringVal{
				StringVal: "0", // This should maybe be calculated???
			},
		},
	}

	// Build update for config change
	confChangeUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "config-change",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_BoolVal{
				BoolVal: true,
			},
		},
	}

	return []*pb.Update{cycleTimeExtUpd, baseTimeSecondsUpd, baseTimeFractionalSecondsUpd, confChangeUpd}
}

// Create updates for gate-enabled, admin-gate-states, and admin-control-list-length
func getStatusChangeElems(port string, deviceIp string, numOfTrafficClassEntries int) []*pb.Update {
	// Build update for gate enabled
	gateEnabledUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "gate-enabled",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_BoolVal{
				BoolVal: true,
			},
		},
	}

	// Build update for admin gate states
	gateStatesUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-gate-states",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_UintVal{
				UintVal: 255, // Statically set all gates to be open for their initial state
			},
		},
	}

	// Build update for admin control list length
	controlListLenUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-control-list-length",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_UintVal{
				UintVal: uint64(numOfTrafficClassEntries),
			},
		},
	}

	return []*pb.Update{gateEnabledUpd, gateStatesUpd, controlListLenUpd}
}

// Create updates for operation-name, gate-states-value, and time-interval-value, for every traffic class in schedule
func getGclElems(sched *schedule.Schedule, port string, deviceIp string) []*pb.Update {
	var updates []*pb.Update
	// For every traffic class, create an entry in the admin-control-list
	for index, trafficClass := range sched.TrafficClasses {
		// Build update for type of operation
		operationUpd := &pb.Update{
			Path: &pb.Path{
				Elem: []*pb.PathElem{
					{
						Name: "interfaces",
						Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
					},
					{
						Name: "interface",
						Key:  map[string]string{"name": port},
					},
					{
						Name: "gate-parameters",
						Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
					},
					{
						Name: "admin-control-list",
						Key:  map[string]string{"index": fmt.Sprint(index)},
					},
					{
						Name: "operation-name",
						Key:  map[string]string{},
					},
				},
				Target: deviceIp,
			},
			Val: &pb.TypedValue{
				Value: &pb.TypedValue_StringVal{
					StringVal: "set-gate-states",
				},
			},
		}

		// Build update for gate states
		gateStateUpd := &pb.Update{
			Path: &pb.Path{
				Elem: []*pb.PathElem{
					{
						Name: "interfaces",
						Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
					},
					{
						Name: "interface",
						Key:  map[string]string{"name": port},
					},
					{
						Name: "gate-parameters",
						Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
					},
					{
						Name: "admin-control-list",
						Key:  map[string]string{"index": fmt.Sprint(index)},
					},
					{
						Name: "sgs-params",
						Key:  map[string]string{},
					},
					{
						Name: "gate-states-value",
						Key:  map[string]string{},
					},
				},
				Target: deviceIp,
			},
			Val: &pb.TypedValue{
				Value: &pb.TypedValue_UintVal{
					UintVal: getGateStatesValue(trafficClass.Name),
				},
			},
		}

		// Build update for time interval
		timeIntervalUpd := &pb.Update{
			Path: &pb.Path{
				Elem: []*pb.PathElem{
					{
						Name: "interfaces",
						Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
					},
					{
						Name: "interface",
						Key:  map[string]string{"name": port},
					},
					{
						Name: "gate-parameters",
						Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
					},
					{
						Name: "admin-control-list",
						Key:  map[string]string{"index": fmt.Sprint(index)},
					},
					{
						Name: "sgs-params",
						Key:  map[string]string{},
					},
					{
						Name: "time-interval-value",
						Key:  map[string]string{},
					},
				},
				Target: deviceIp,
			},
			Val: &pb.TypedValue{
				Value: &pb.TypedValue_UintVal{
					UintVal: getInterval(trafficClass.AssignedPortion, sched.GatingCycle),
				},
			},
		}

		updates = append(updates, operationUpd)
		updates = append(updates, gateStateUpd)
		updates = append(updates, timeIntervalUpd)
	}

	return updates
}

// Get gate state values if traffic class matches a predefined traffic class (best effort will open two gates)
func getGateStatesValue(trafficClassName string) uint64 {
	switch trafficClassName {
	case "isochronous":
		return 128
	case "cyclic-sync":
		return 64
	case "cyclic-async":
		return 32
	case "alarms-events":
		return 16
	case "config-diag":
		return 8
	case "network-control":
		return 4
	case "best-effort":
		return 3
	default:
		//log.Errorf("Traffic class was not found: %v", errors.New("traffic class did not match any of the predefined traffic classes, all gates will be closed"))
	}

	return 0
}

// Get interval in nanoseconds from scheduled percentage of gatingcycle
func getInterval(assignedPercentage int32, gatingCycle float32) uint64 {
	return uint64(gatingCycle * 1000000 * (float32(assignedPercentage) / 100))
}

// Create updates for admin-cycle-time (numerator and denominator)
func getAdminCycleTimeElems(gatingCycle float32, port string, deviceIp string) []*pb.Update {
	// Create update element for numerator
	numeratorUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-cycle-time",
					Key:  map[string]string{},
				},
				{
					Name: "numerator",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_IntVal{
				IntVal: int64(gatingCycle),
			},
		},
	}

	// Create update element for denominator
	denominatorUpd := &pb.Update{
		Path: &pb.Path{
			Elem: []*pb.PathElem{
				{
					Name: "interfaces",
					Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
				},
				{
					Name: "interface",
					Key:  map[string]string{"name": port},
				},
				{
					Name: "gate-parameters",
					Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"},
				},
				{
					Name: "admin-cycle-time",
					Key:  map[string]string{},
				},
				{
					Name: "denominator",
					Key:  map[string]string{},
				},
			},
			Target: deviceIp,
		},
		Val: &pb.TypedValue{
			Value: &pb.TypedValue_IntVal{
				IntVal: 1000, // 1000 ensures the gating cycle is in milliseconds
			},
		},
	}

	return []*pb.Update{numeratorUpd, denominatorUpd}
}
