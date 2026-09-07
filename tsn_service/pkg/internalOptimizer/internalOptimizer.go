package internalOptimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"OpenCNC/common/observability"
	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/qbv"
	"OpenCNC/common/structures/stream_config"
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/common/structures/uni"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"github.com/ghodss/yaml"
)

var defaultSchedID = "default_schedule"
var defaultSchedulePath = "/home/opencnc/OpenCNC/tsn_service/configs/default-schedule.yaml"

// Calculates configuration set request using optimizer, if that failes build configuration set request from default schedule
func StartOptimization(
	ctx context.Context,
	obs *observability.Client,
	task *optimizer.OptimizationTask,
	allRequestData []*uni.Request,
) (*forwarding_plane.ForwardingPlaneModel, error) {

	fpm := createForwardingPlaneModel(ctx, obs)
	// TODO: Implement the optimization logic here.

	// If optimization fails, fall back to the default schedule.
	topoConfig, err := store.GetTopologyConfiguration(defaultSchedID)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed getting current configuration: %v",
				err,
			))
		}
		return nil, err
	}

	fpm.Configuration = topoConfig
	//fpm.Metadata.ModelId = defaultSchedID

	return fpm, nil
}

func createForwardingPlaneModel(
	ctx context.Context,
	obs *observability.Client,
) *forwarding_plane.ForwardingPlaneModel {
	//TODO: optimizer is not supposed to directly access the store,
	// it ask  "PE" and "RAE" for the necessary data as input

	// Get topology
	topology, err := store.GetTopology()
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed getting topology: %v",
				err,
			))
		}
		return nil
	}

	// Get current active configuration
	//TODO: this should be the current active configuration
	topoConfig, err := store.GetTopologyConfiguration(defaultSchedID)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed getting current configuration: %v",
				err,
			))
		}
		return nil
	}

	//Get all streams from k/v store
	streams, err := store.GetStreams()
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed getting streams: %v",
				err,
			))
		}
		return nil
	}

	// Get all streamconfigurations from k/v store
	streamModels := []*forwarding_plane.StreamModel{}
	for _, stream := range streams {
		streamConfig, err := store.GetStreamConfiguration(stream.StreamId.AsKey())
		if err != nil {
			//fmt.Printf("Failed getting stream configuration for stream %v: %v\n", stream.StreamId.AsKey(), err)
			if obs != nil {
				_ = obs.Error(ctx, fmt.Sprintf(
					"Failed getting stream configuration for stream %v: %v",
					stream.StreamId.AsKey(),
					err,
				))
			}
			streamConfig = &stream_config.StreamConfiguration{}
		}

		streamModels = append(streamModels, &forwarding_plane.StreamModel{
			Definition:    stream,
			Configuration: streamConfig,
		})
	}

	return &forwarding_plane.ForwardingPlaneModel{
		Topology:      topology,
		Configuration: topoConfig,
		Streams:       streamModels,
	}
}

// Reads default schedule config file and stores configuration for schedule in k/v store
func CreateDefaultSchedule(
	ctx context.Context,
	obs *observability.Client,
) error {
	// Read default schedule from file
	schedBytes, err := os.ReadFile(defaultSchedulePath)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed reading default schedule from file: %v",
				err,
			))
		}
		return err
	}

	// Convert yaml bytes to json bytes
	jsonBytes, err := yaml.YAMLToJSON(schedBytes)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed converting default schedule from YAML to JSON: %v",
				err,
			))
		}
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
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed parsing schedule JSON: %v",
				err,
			))
		}
		return err
	}

	// Validate the configuration.
	if config.GatingCycle == 0 {
		err := fmt.Errorf("gating-cycle must be greater than zero")
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Invalid default schedule: %v",
				err,
			))
		}
		return err
	}

	var totalPortion uint64
	for _, tc := range config.TrafficClasses {
		totalPortion += tc.AssignedPortion
	}

	if totalPortion != 100 {
		err := fmt.Errorf(
			"traffic class assigned portions must add up to 100%%, got %d%%",
			totalPortion,
		)
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Invalid default schedule: %v",
				err,
			))
		}
		return err
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
			err := fmt.Errorf("unknown traffic class %q", tc.Name)
			if obs != nil {
				_ = obs.Error(ctx, fmt.Sprintf(
					"Invalid default schedule: %v",
					err,
				))
			}
			return err
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
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to retrieve topology for default schedule: %v",
				err,
			))
		}
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
	err = store.StoreTopologyConfiguration(topoConfig)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed storing default schedule with ID %s: %v",
				defaultSchedID,
				err,
			))
		}
		return err
	}

	if obs != nil {
		_ = obs.Info(ctx, fmt.Sprintf(
			"Successfully stored default schedule with ID: %s",
			defaultSchedID,
		))
	}

	return nil
}