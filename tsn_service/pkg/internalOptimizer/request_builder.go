// Combines CNC's task and forwarding plane with our configured mechanism, then calls the existing v1.4 converter.
package internalOptimizer

import (
	"OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
)

// temporary for tests passing
func buildOaaSRequest(
	task *optimizer.OptimizationTask,
	fpm *forwarding_plane.ForwardingPlaneModel,

) ([]byte, error) {
	return buildOaaSRequestWithDeviceModels(task, fpm, nil)
}
func buildOaaSRequestWithDeviceModels(
	task *optimizer.OptimizationTask,
	fpm *forwarding_plane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
	uniRequests ...*uni.Request,
) ([]byte, error) {
	if task == nil || fpm == nil {
		return nil, fmt.Errorf("optimization task and forwarding plane are required")
	}

	// Overlay UNI PCP on a request-only copy. The original FPM is snapshotted unchanged.
	requestFPM, err := withUNIPCP(fpm, uniRequests)
	if err != nil {
		return nil, fmt.Errorf("UNI PCP input: %w", err)
	}

	options, err := buildRequestOptions()
	if err != nil {
		return nil, err
	}

	// Hamza (15 November 2026): For CBS, use explicit PCP-to-queue mappings
	// from the FPM first; when absent, assume OpenCNC's default mapping.
	// Apply defaults to the request-only copy, never the preserved FPM.
	for _, mechanism := range options.Mechanisms {
		if mechanism.Name == "CBS" {
			requestFPM, err = withDefaultPCPQueueMappings(requestFPM)
			if err != nil {
				return nil, fmt.Errorf("CBS PCP-to-queue fallback: %w", err)
			}
			break
		}
	}

	// B10 - Running CNC = integration testing only.
	// Explicitly allow routing optimization for test streams.
	if os.Getenv("OAAS_B10_OPTIMIZE_ROUTING") == "true" {
		options.Streams = make(map[string]requestadapter.StreamOptions)

		for _, stream := range requestFPM.GetStreams() {
			if stream == nil || stream.GetDefinition() == nil ||
				stream.GetDefinition().GetStreamId() == nil {
				return nil, fmt.Errorf("B10: stream definition or ID missing")
			}

			id := stream.GetDefinition().GetStreamId()
			key := requestadapter.StreamKey(
				id.GetMacAddress(),
				id.GetUniqueId(),
			)

			options.Streams[key] = requestadapter.StreamOptions{
				Routing: "optimize",
			}
		}
	}

	if err := setAddStreamChanges(task, requestFPM, &options); err != nil {
		return nil, err
	}

	if len(deviceModels) > 0 {
		limitsByNode, err := collectTASGCLLimitsByNode(deviceModels)
		if err != nil {
			return nil, fmt.Errorf("device limitation extraction: %w", err)
		}

		if len(limitsByNode) > 0 {
			fmt.Printf(
				"[OAAS] Found TAS GCL limitations for %d nodes; port mapping pending\n",
				len(limitsByNode),
			)
		}
	}

	return requestadapter.BuildOptimizerRequest(
		task,
		requestFPM,
		requestadapter.SchemaJSON(),
		options,
	)

}

func setAddStreamChanges(
	task *optimizer.OptimizationTask,
	fpm *forwarding_plane.ForwardingPlaneModel,
	options *requestadapter.RequestOptions,
) error {
	if task.GetIntent() != optimizer.Intent_ADD_STREAM {
		return nil
	}

	// The ADD_STREAM scope identifies newly submitted streams.
	// Mark them "add"; mark other FPM streams "none".
	// This is OaaS reconfiguration metadata, not a Zhao CBS requirement.
	scope := task.GetScope()
	if scope == nil || len(scope.GetStreamIds()) == 0 {
		return fmt.Errorf("ADD_STREAM requires scope.streamIds")
	}

	added := make(map[string]bool)
	for _, id := range scope.GetStreamIds() {
		if id == "" || added[id] {
			return fmt.Errorf("invalid or duplicate stream ID %q", id)
		}
		added[id] = true
	}

	if options.Streams == nil {
		options.Streams = make(map[string]requestadapter.StreamOptions)
	}

	for _, stream := range fpm.GetStreams() {
		if stream == nil || stream.GetDefinition() == nil ||
			stream.GetDefinition().GetStreamId() == nil {
			return fmt.Errorf("ADD_STREAM: missing FPM stream ID")
		}

		id := stream.GetDefinition().GetStreamId()
		key := requestadapter.StreamKey(id.GetMacAddress(), id.GetUniqueId())

		change := "none"
		if added[id.AsKey()] || added[key] {
			change = "add"
			delete(added, id.AsKey())
			delete(added, key)
		}

		opt := options.Streams[key]
		if opt.Change != "" && opt.Change != change {
			return fmt.Errorf("conflicting change for stream %q", key)
		}
		opt.Change = change
		options.Streams[key] = opt
	}

	for id := range added {
		return fmt.Errorf("ADD_STREAM stream %q not found in FPM", id)
	}

	return nil
}

func prepareTASFullScopeTask(
	task *optimizer.OptimizationTask,
	fpm *forwarding_plane.ForwardingPlaneModel,
	streamOptions map[string]requestadapter.StreamOptions,
) (*optimizer.OptimizationTask, error) {
	if task.GetIntent() != optimizer.Intent_ADD_STREAM {
		return nil, fmt.Errorf("TAS pilot requires ADD_STREAM")
	}

	policy := task.GetChangePolicy()

	if policy == nil ||
		!policy.GetAllowScheduleChanges() ||
		!policy.GetPreserveExistingStreams() {
		return nil, fmt.Errorf(
			"TAS requires allowScheduleChanges=true and preserveExistingStreams=true",
		)
	}

	if policy.GetMaxConfigurationChanges() == 0 {
		return nil, fmt.Errorf(
			"TAS requires a positive maxConfigurationChanges",
		)
	}

	scope := task.GetScope()
	if scope == nil || len(scope.GetStreamIds()) == 0 {
		return nil, fmt.Errorf("ADD_STREAM requires new stream IDs")
	}
	if len(scope.GetNodeIds()) > 0 || len(scope.GetLinkIds()) > 0 {
		return nil, fmt.Errorf(
			"TAS full reconfiguration does not support restricted node/link scope",
		)
	}

	allIDs := make([]string, 0, len(fpm.GetStreams()))
	seen := make(map[string]bool)
	added, existing := 0, 0

	for _, stream := range fpm.GetStreams() {
		if stream == nil || stream.GetDefinition() == nil ||
			stream.GetDefinition().GetStreamId() == nil {
			return nil, fmt.Errorf("TAS: missing stream definition or ID")
		}

		id := stream.GetDefinition().GetStreamId()
		key := requestadapter.StreamKey(
			id.GetMacAddress(), id.GetUniqueId(),
		)

		if seen[key] {
			return nil, fmt.Errorf("TAS: duplicate stream %q", key)
		}
		seen[key] = true

		switch streamOptions[key].Change {
		case "add":
			added++
		case "none":
			existing++
		default:
			return nil, fmt.Errorf(
				"TAS: unsupported or missing change for %q", key,
			)
		}

		allIDs = append(allIDs, key)
	}

	if added == 0 || existing == 0 {
		return nil, fmt.Errorf(
			"TAS full reconfiguration requires both existing and new streams",
		)
	}

	if uint64(added) > uint64(policy.GetMaxChangedStreams()) {
		return nil, fmt.Errorf(
			"TAS: maxChangedStreams cannot cover %d new streams", added,
		)
	}

	// Never modify CNC's original optimization task.

	copyTask := proto.Clone(task).(*optimizer.OptimizationTask)
	copyTask.Scope.StreamIds = allIDs

	// TAS uses fixed routes and queues.
	// Restrict permissions only in the request copy.
	copyTask.ChangePolicy.AllowPathChanges = false
	copyTask.ChangePolicy.AllowQueueChanges = false
	copyTask.ChangePolicy.AllowVlanChanges = false
	copyTask.ChangePolicy.AllowStreamMigration = false

	return copyTask, nil

}
