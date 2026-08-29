package internalOptimizer

import (
	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/qbv"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	"encoding/json"
	"fmt"
	"os"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"github.com/ghodss/yaml"
)

//var log = logger.GetLogger()

var defaultSchedID = "default_schedule"
var defaultSchedulePath = "/home/opencnc/OpenCNC/tsn_service/configs/default-schedule.yaml"

func StartOptimization() {
	// Implementation for starting the optimization process goes here.
}

// Calculates configuration set request using optimizer, if that failes build configuration set request from default schedule
func RequestOptimization(topology *topology.Topology, oldConfig *qbv.GateControlList) (*topology_config.TopologyConfig, error) {

	// Load default schedule from k/v store
	topoConfig, err := store.GetConfiguration(defaultSchedID)
	if err != nil {
		//log.Errorf("Failed getting default schedule: %v", err)
		return nil, err
	}
	return topoConfig, nil
}

// Reads default schedule config file and stores configuration for schedule in k/v store
func CreateDefaultSchedule() error {
	// Read default schedule from file
	schedBytes, err := os.ReadFile(defaultSchedulePath)
	if err != nil {
		fmt.Println("Failed reading default schedule from file")
		// log.Errorf("Failed reading default schedule from file: %v", err)
		return err
	}

	// Convert yaml bytes to json bytes
	jsonBytes, err := yaml.YAMLToJSON(schedBytes)
	if err != nil {
		// log.Errorf("Failed converting file content from yaml to json: %v", err)
		return err
	}

	// Intermediate representation of the YAML file.
	var config struct {
		GatingCycle    uint64 `json:"gating-cycle"`
		TrafficClasses []struct {
			Name            string `json:"name"`
			AssignedPortion uint64 `json:"assigned-portion"`
		} `json:"traffic-classes"`
	}

	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		// log.Errorf("Failed parsing schedule JSON: %v", err)
		return err
	}

	// Validate the configuration.
	if config.GatingCycle == 0 {
		return fmt.Errorf("gating-cycle must be greater than zero")
	}

	var totalPortion uint64
	for _, tc := range config.TrafficClasses {
		totalPortion += tc.AssignedPortion
	}

	if totalPortion != 100 {
		return fmt.Errorf(
			"traffic class assigned portions must add up to 100%%, got %d%%",
			totalPortion,
		)
	}

	// Map traffic class names to gate bit positions.
	trafficClassBits := map[string]uint{
		"isochronous":     0,
		"cyclic-sync":     1,
		"cyclic-async":    2,
		"alarms-events":   3,
		"config-diag":     4,
		"network-control": 5,
		"best-effort":     6,
	}

	defaultSched := &qbv.GateControlList{
		ScheduleId:  "default",
		BaseTime:    0,
		BaseTimeRef: qbv.TimestampReference_DEVICE_BOOT_TIME,
		CycleTime:   config.GatingCycle * 1_000_000, // ms -> ns
		AdminState:  qbv.AdminState_ENABLED,
		Description: "Default Qbv schedule",
		Entries:     make([]*qbv.GateControlEntry, 0, len(config.TrafficClasses)),
	}

	for index, tc := range config.TrafficClasses {
		bit, ok := trafficClassBits[tc.Name]
		if !ok {
			return fmt.Errorf("unknown traffic class %q", tc.Name)
		}

		// assigned-portion is a percentage of the gating cycle.
		//
		// Example:
		//   1 ms cycle * 30% = 300 us = 300,000 ns
		timeInterval := (config.GatingCycle * 1_000_000 * tc.AssignedPortion) / 100

		// Open only the gate belonging to this traffic class.
		//
		// Bit 0 = isochronous
		// Bit 1 = cyclic-sync
		// ...
		// Bit 6 = best-effort
		gateStates := []byte{
			1 << bit,
		}

		defaultSched.Entries = append(
			defaultSched.Entries,
			&qbv.GateControlEntry{
				Index:        uint32(index),
				TimeInterval: timeInterval,
				GateStates:   gateStates,
				Description:  tc.Name,
			},
		)
	}

	//write schedule to all configuration ports
	topoConfig := &topology_config.TopologyConfig{ConfigId: defaultSchedID}
	topology, err := store.GetTopology()
	if err != nil {
		return fmt.Errorf("failed to retrieve topology: %v", err)
	}
	for _, node := range topology.GetNodes() {
		nodeConfig := &topology_config.NodeConfig{NodeId: node.Name}
		for _, port := range node.GetPorts() {
			portConfig := &topology_config.PortConfig{
				PortId: port.Id,
				Gcl:    defaultSched,
			}
			nodeConfig.PortConfigs = append(nodeConfig.PortConfigs, portConfig)
		}
		topoConfig.NodeConfigs = append(topoConfig.NodeConfigs, nodeConfig)
	}

	// Store schedule in k/v store
	err = store.StoreConfiguration(topoConfig)
	if err != nil {
		//log.Errorf("Failed storing default schedule: %v", err)
		return err
	}
	fmt.Println("Successfully stored default schedule with ID: ", defaultSchedID)
	//log.Infof("Successfully stored default schedule with ID: %v", defaultSchedID)

	return nil
}

// // Creates a connection to the network optimizer and requests a new configuration
// func CallOptimizer(input *optimizer.Input) (*optimizer.Output, error) {
// 	// Create gRPC connection
// 	conn, err := grpc.Dial("network-optimizer:5150", grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		//log.Fatalf("Failed dialing network-optimizer: %v", err)
// 		return &optimizer.Output{}, err
// 	}

// 	defer conn.Close()

// 	// Create gRPC client
// 	client := optimizer.NewOptimizerClient(conn)

// 	// Request config from optimizer
// 	output, err := client.GenerateConfiguration(context.Background(), input)
// 	if err != nil {
// 		//log.Errorf("Optimizer failed generating configuration: %v", err)
// 		return &optimizer.Output{}, err
// 	}

// 	return output, nil
// }
