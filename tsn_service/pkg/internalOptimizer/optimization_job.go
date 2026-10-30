// This will let us test the actual production code without starting CNC.
package internalOptimizer

import (
	"context"
	"fmt"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func submitOptimizationJob(
	ctx context.Context,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
) (*ManagerJobReceipt, string, error) {

	requestJSON, err := buildOaaSRequest(task, fpm)
	if err != nil {
		return nil, "", fmt.Errorf("request conversion: %w", err)
	}

	// Preserve original CNC data before submission.
	snapshotID, err := saveOptimizationSnapshot(fpm)
	if err != nil {
		return nil, "", fmt.Errorf("snapshot storage: %w", err)
	}

	response, err := submitOaaSRequest(ctx, requestJSON)
	if err != nil {
		return nil, snapshotID, fmt.Errorf("Manager submission: %w", err)
	}

	receipt, err := parseManagerJobReceipt(response)
	if err != nil {
		return nil, snapshotID, err
	}

	if err := saveJobSnapshotLink(receipt.JobID, snapshotID); err != nil {
		return nil, snapshotID, fmt.Errorf("job snapshot mapping: %w", err)
	}

	return receipt, snapshotID, nil
}
