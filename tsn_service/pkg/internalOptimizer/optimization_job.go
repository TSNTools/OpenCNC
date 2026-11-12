// handles the entire request submission
package internalOptimizer

import (
	"context"
	"fmt"
	"time"

	"OpenCNC/common/observability"
	"OpenCNC/common/structures/devicemodelregistry"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func submitOptimizationJob(
	ctx context.Context,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
) (*ManagerJobReceipt, string, error) {
	return submitOptimizationJobWithObservability(ctx, nil, task, fpm, deviceModels)
}

func submitOptimizationJobWithObservability(
	ctx context.Context,
	obs *observability.Client,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
) (*ManagerJobReceipt, string, error) {
	timings := newOAASTimingRecorder(obs)
	defer timings.publish()
	return submitOptimizationJobWithTimings(ctx, task, fpm, deviceModels, timings)
}

func submitOptimizationJobWithTimings(
	ctx context.Context,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
	timings *oaasTimingRecorder,
) (*ManagerJobReceipt, string, error) {
	started := time.Now()
	requestJSON, err := buildOaaSRequestWithDeviceModels(task, fpm, deviceModels)
	timings.record("oaas_request_build_ms", started, err == nil)
	if err != nil {
		return nil, "", fmt.Errorf("request conversion: %w", err)
	}

	// Preserve the native FPM before submission; it is never reconstructed.
	started = time.Now()
	snapshotID, err := saveOptimizationSnapshot(fpm)
	timings.record("oaas_snapshot_ms", started, err == nil)
	if err != nil {
		return nil, "", fmt.Errorf("snapshot storage: %w", err)
	}

	started = time.Now()
	response, err := submitOaaSRequest(ctx, requestJSON)
	timings.record("oaas_submission_ms", started, err == nil)
	if err != nil {
		return nil, snapshotID, fmt.Errorf("Manager submission: %w", err)
	}

	started = time.Now()
	receipt, err := parseManagerJobReceipt(response)
	timings.record("oaas_ack_parse_ms", started, err == nil)
	if err != nil {
		return nil, snapshotID, err
	}
	if timings != nil {
		timings.jobID = receipt.JobID
	}

	started = time.Now()
	if err := saveJobSnapshotLink(receipt.JobID, snapshotID); err != nil {
		timings.record("oaas_job_bookkeeping_ms", started, false)
		return nil, snapshotID, fmt.Errorf("job snapshot mapping: %w", err)
	}

	// Store the exact JSON submitted to the Manager for later validation.
	if err := saveJobRequest(receipt.JobID, requestJSON); err != nil {
		timings.record("oaas_job_bookkeeping_ms", started, false)
		return nil, snapshotID, fmt.Errorf("job request mapping: %w", err)
	}
	timings.record("oaas_job_bookkeeping_ms", started, true)
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
	return executeOptimizationJobWithObservability(ctx, nil, task, fpm, deviceModels)
}

func executeOptimizationJobWithObservability(
	ctx context.Context,
	obs *observability.Client,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
) (*forwardingplane.ForwardingPlaneModel, error) {
	timings := newOAASTimingRecorder(obs)
	defer timings.publish()
	return executeOptimizationJobWithTimings(ctx, task, fpm, deviceModels, timings)
}

func executeOptimizationJobWithTimings(
	ctx context.Context,
	task *optimizer.OptimizationTask,
	fpm *forwardingplane.ForwardingPlaneModel,
	deviceModels map[string]*devicemodelregistry.DeviceModel,
	timings *oaasTimingRecorder,
) (*forwardingplane.ForwardingPlaneModel, error) {
	receipt, _, err := submitOptimizationJobWithTimings(ctx, task, fpm, deviceModels, timings)
	if err != nil {
		return nil, fmt.Errorf("submit optimization job: %w", err)
	}
	validated, err := waitAndValidateOptimizationResultWithTimings(ctx, receipt.JobID, timings)
	if err != nil {
		return nil, fmt.Errorf("retrieve and validate optimization result for job %s: %w", receipt.JobID, err)
	}
	started := time.Now()
	updated, err := applyValidatedOptimizerResponse(validated)
	timings.record("oaas_fpm_apply_ms", started, err == nil)
	if err != nil {
		return nil, fmt.Errorf("apply validated optimization result for job %s: %w", receipt.JobID, err)
	}
	return updated, nil
}
