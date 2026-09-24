package internalOptimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
var defaultSchedulePath = "tsn_service/configs/default-schedule.yaml"

// Calculates configuration set request using optimizer, if that failes build configuration set request from default schedule
func StartOptimization(
	ctx context.Context,
	obs *observability.Client,
	task *optimizer.OptimizationTask,
	allRequestData []*uni.Request,
) (*forwarding_plane.ForwardingPlaneModel, error) {

	// The GUI can replace the topology after startup. Rebuild the default
	// schedule so its node and port IDs reflect the topology used for this run.
	if err := CreateDefaultSchedule(ctx, obs); err != nil {
		return nil, fmt.Errorf("failed to refresh default schedule: %w", err)
	}

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
			streamConfig = &stream_config.StreamConfiguration{StreamId: stream.GetStreamId()}
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

func readDefaultSchedule() ([]byte, error) {
	if env := os.Getenv("DEFAULT_SCHEDULE_PATH"); env != "" {
		if b, err := os.ReadFile(env); err == nil {
			return b, nil
		}
	}
	candidates := []string{
		defaultSchedulePath,
		"configs/default-schedule.yaml",
		"../configs/default-schedule.yaml",
		"../../configs/default-schedule.yaml",
		"/home/opencnc/OpenCNC/tsn_service/configs/default-schedule.yaml",
	}
	if _, filename, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(filename), "../../configs/default-schedule.yaml"))
	}
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			return b, nil
		}
	}
	return os.ReadFile(defaultSchedulePath)
}

// Reads default schedule config file and stores configuration for schedule in k/v store
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

	type trafficClassConfig struct {
		Name            string `json:"name"`
		AssignedPortion uint64 `json:"assigned-portion"`
	}

	type portScheduleConfig struct {
		PortId         string               `json:"port-id"`
		GatingCycle    uint64               `json:"gating-cycle"`
		TrafficClasses []trafficClassConfig `json:"traffic-classes"`
	}

	// Intermediate representation of the YAML file supporting per-port or global schedule.
	var config struct {
		Ports          []portScheduleConfig `json:"ports"`
		GatingCycle    uint64               `json:"gating-cycle"`
		TrafficClasses []trafficClassConfig `json:"traffic-classes"`
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

	// Map traffic class names to gate bit positions (0 to 7).
	trafficClassBits := map[string]uint{
		"isochronous":      0,
		"cyclic-sync":      1,
		"cyclic-async":     2,
		"alarms-events":    3,
		"config-diag":      4,
		"network-control":  5,
		"best-effort":      6,
		"timecritical":     7,
		"time-critical":    7,
		"background":       7,
		"critical-control": 7,
	}

	buildGCL := func(schedID string, cycleTime uint64, tcs []trafficClassConfig) (*qbv.GateControlList, error) {
		if cycleTime == 0 {
			return nil, fmt.Errorf("gating-cycle must be greater than zero for %s", schedID)
		}

		var totalPortion uint64
		for _, tc := range tcs {
			totalPortion += tc.AssignedPortion
		}

		if totalPortion != 100 {
			return nil, fmt.Errorf(
				"traffic class assigned portions must add up to 100%% for %s, got %d%%",
				schedID,
				totalPortion,
			)
		}

		gcl := &qbv.GateControlList{
			ScheduleId:  schedID,
			BaseTime:    0,
			BaseTimeRef: qbv.TimestampReference_DEVICE_BOOT_TIME,
			CycleTime:   cycleTime * 1_000_000, // ms -> ns
			AdminState:  qbv.AdminState_ENABLED,
			Description: fmt.Sprintf("Qbv schedule for %s", schedID),
			Entries:     make([]*qbv.GateControlEntry, 0, len(tcs)),
		}

		for index, tc := range tcs {
			bit, ok := trafficClassBits[strings.ToLower(tc.Name)]
			if !ok {
				return nil, fmt.Errorf("unknown traffic class %q in schedule %s", tc.Name, schedID)
			}

			timeInterval := (cycleTime * 1_000_000 * tc.AssignedPortion) / 100
			gateStates := []byte{
				1 << bit,
			}

			gcl.Entries = append(
				gcl.Entries,
				&qbv.GateControlEntry{
					Index:        uint32(index),
					TimeInterval: timeInterval,
					GateStates:   gateStates,
					Description:  tc.Name,
				},
			)
		}

		return gcl, nil
	}

	portSchedules := make(map[string]*qbv.GateControlList)
	if len(config.Ports) > 0 {
		for _, p := range config.Ports {
			sched, err := buildGCL(p.PortId, p.GatingCycle, p.TrafficClasses)
			if err != nil {
				if obs != nil {
					_ = obs.Error(ctx, fmt.Sprintf(
						"Invalid schedule for port %s: %v",
						p.PortId,
						err,
					))
				}
				return err
			}
			portSchedules[p.PortId] = sched
		}
	} else if len(config.TrafficClasses) > 0 {
		sched, err := buildGCL("default", config.GatingCycle, config.TrafficClasses)
		if err != nil {
			if obs != nil {
				_ = obs.Error(ctx, fmt.Sprintf(
					"Invalid default schedule: %v",
					err,
				))
			}
			return err
		}
		portSchedules["*"] = sched
	} else {
		err := fmt.Errorf("no ports or traffic-classes defined in schedule file")
		if obs != nil {
			_ = obs.Error(ctx, err.Error())
		}
		return err
	}

	// Write schedule to configuration ports
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
			sched, ok := portSchedules[port.Id]
			if !ok {
				sched, ok = portSchedules["*"]
				if !ok {
					continue // Skip port if not defined in YAML
				}
			}

			portConfig := &topology_config.PortConfig{
				PortId: port.Id,
				Gcl:    sched,
			}

			nodeConfig.PortConfigs = append(nodeConfig.PortConfigs, portConfig)
		}

		if len(nodeConfig.PortConfigs) > 0 {
			topoConfig.NodeConfigs = append(topoConfig.NodeConfigs, nodeConfig)
		}
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
