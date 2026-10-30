// Combines CNC's task and forwarding plane with our configured mechanism, then calls the existing v1.4 converter.
package internalOptimizer

import (
	"fmt"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func buildOaaSRequest(
	task *optimizer.OptimizationTask,
	fpm *forwarding_plane.ForwardingPlaneModel,
) ([]byte, error) {
	if task == nil || fpm == nil {
		return nil, fmt.Errorf("optimization task and forwarding plane are required")
	}

	options, err := buildRequestOptions()
	if err != nil {
		return nil, err
	}

	return requestadapter.BuildOptimizerRequest(
		task,
		fpm,
		requestadapter.SchemaJSON(),
		options,
	)
}
