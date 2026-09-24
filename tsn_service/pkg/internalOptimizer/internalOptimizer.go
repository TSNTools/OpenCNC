package internalOptimizer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"OpenCNC/common/observability"
	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/stream_config"
	"OpenCNC/common/structures/uni"
	"OpenCNC/common/structures/vlan"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
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

	// If incoming stream requests specify a VLAN, ensure the port configs use that VLAN ID
	var reqVlanId uint32
	for _, req := range allRequestData {
		if req.GetTalker() != nil {
			for _, df := range req.GetTalker().GetDataFrameSpecification() {
				if df.GetVlanTag() != nil && df.GetVlanTag().GetVlanId() > 0 {
					reqVlanId = df.GetVlanTag().GetVlanId()
					break
				}
			}
		}
		if reqVlanId > 0 {
			break
		}
	}

	if reqVlanId > 0 && reqVlanId != 100 {
		for _, nc := range topoConfig.GetNodeConfigs() {
			for _, pc := range nc.GetPortConfigs() {
				pc.DefaultVlanId = &reqVlanId
				pc.VlanMemberships = []*vlan.VlanMembership{
					{
						VlanId: reqVlanId,
						Tagged: true,
					},
				}
			}
		}
		_ = store.StoreTopologyConfiguration(topoConfig)
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
		return os.ReadFile(env)
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

// CreateDefaultSchedule builds a schedule for the current topology and stores it.
func CreateDefaultSchedule(ctx context.Context, obs *observability.Client) error {
	schedBytes, err := readDefaultSchedule()
	if err != nil {
		return fmt.Errorf("failed reading default schedule: %w", err)
	}
	topo, err := store.GetTopology()
	if err != nil {
		return fmt.Errorf("failed retrieving topology: %w", err)
	}
	cfg, err := buildDefaultSchedule(schedBytes, topo)
	if err != nil {
		return err
	}
	if err := store.StoreTopologyConfiguration(cfg); err != nil {
		return fmt.Errorf("failed storing default schedule: %w", err)
	}
	if obs != nil {
		_ = obs.Info(ctx, fmt.Sprintf("Successfully stored default schedule with ID: %s", defaultSchedID))
	}
	return nil
}
