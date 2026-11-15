// Combines CNC's task and forwarding plane with our configured mechanism, then calls the existing v1.4 converter.
package internalOptimizer

import (
	"fmt"
	"os"

	"OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
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
