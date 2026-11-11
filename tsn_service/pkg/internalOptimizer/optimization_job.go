// handles the entire request submission
package internalOptimizer

import (
	"OpenCNC/common/structures/devicemodelregistry"
	"context"
	"fmt"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func submitOptimizationJob(
	ctx context.Context,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,

) (*ManagerJobReceipt, string, error) {

	requestJSON, err := buildOaaSRequestWithDeviceModels(task, fpm, deviceModels)
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

	// Save exactly the JSON sent to the Manager. Response validation must use
	// this request plus the independently preserved native CNC snapshot.
	// A storage failure blocks later result application.
	if err := saveJobRequest(receipt.JobID, requestJSON); err != nil {
		return nil, snapshotID, fmt.Errorf("job request mapping: %w", err)
	}

	return receipt, snapshotID, nil
}

// executeOptimizationJob orchestrates the approved CNC integration boundary:
// submit a valid request, retrieve the completed result, validate against the
// preserved original, and return an updated copy of the forwarding plane.
// It does not deploy the configuration or modify other OpenCNC services.
func executeOptimizationJob(
	ctx context.Context,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
) (*forwardingplane.ForwardingPlaneModel, error) {
	receipt, _, err := submitOptimizationJob(ctx, task, fpm, deviceModels)
	if err != nil {
		return nil, fmt.Errorf("submit optimization job: %w", err)
	}
	validated, err := waitAndValidateOptimizationResult(ctx, receipt.JobID)
	if err != nil {
		return nil, fmt.Errorf("retrieve and validate optimization result for job %s: %w", receipt.JobID, err)
	}
	updated, err := applyValidatedOptimizerResponse(validated)
	if err != nil {
		return nil, fmt.Errorf("apply validated optimization result for job %s: %w", receipt.JobID, err)
	}
	return updated, nil
}
