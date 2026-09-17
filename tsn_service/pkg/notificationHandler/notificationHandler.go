package notificationHandler

import (
	"context"
	"fmt"
	"time"

	"OpenCNC/common/observability"
	store "OpenCNC/common/store-wrapper"
	observabilityv1 "OpenCNC/common/structures/logging"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/internalOptimizer"
	"OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func CalculateConfiguration(
	ctx context.Context,
	obs *observability.Client,
	task *optimizer.OptimizationTask,
	allRequestData []*uni.Request,
) (*forwarding_plane.ForwardingPlaneModel, error) {

	start := time.Now()

	// TODO: Use requests when creating configuration

	// Calculate configuration set request
	newFpm, err := internalOptimizer.StartOptimization(
		ctx,
		obs,
		task,
		allRequestData,
	)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed calculating configuration: %v",
				err,
			))
			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"configuration",
				"calculate",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"forwarding_plane_model",
				"",
				fmt.Sprintf("Failed calculating configuration: %v", err),
			)
		}
		return nil, err
	}

	// Validate the new forwarding plane model before storing it
	admissionCheck := true
	if admissionCheck {
		// Store configuration set request in k/v store
		if err := store.StoreForwardingPlaneConfiguration(newFpm); err != nil {
			if obs != nil {
				_ = obs.Error(ctx, fmt.Sprintf(
					"Failed storing configuration: %v",
					err,
				))
				_ = obs.Event(
					ctx,
					observabilityv1.Severity_SEVERITY_ERROR,
					"configuration",
					"store",
					observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
					"forwarding_plane_model",
					"",
					fmt.Sprintf("Failed storing configuration: %v", err),
				)
			}
			return nil, err
		}

		if obs != nil {
			duration := time.Since(start)
			_ = obs.Metric(
				ctx,
				observabilityv1.Severity_SEVERITY_INFO,
				"configuration_calculation_duration",
				observabilityv1.MetricType_METRIC_TYPE_GAUGE,
				float64(duration.Milliseconds()),
				"ms",
				nil,
			)

			_ = obs.Info(ctx, "Successfully calculated and stored configuration")
			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_INFO,
				"configuration",
				"calculate",
				observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
				"forwarding_plane_model",
				"",
				"Successfully calculated and stored configuration",
			)
		}

		return newFpm, nil
	}

	// TODO: Handle admission denial appropriately by returning the problem to the optimizer
	// Admission denied
	err = fmt.Errorf("admission denied")

	if obs != nil {
		_ = obs.Error(ctx, "Configuration admission denied")
		_ = obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_WARN,
			"configuration",
			"calculate",
			observabilityv1.DomainResult_DOMAIN_RESULT_REJECTED,
			"forwarding_plane_model",
			"",
			"Configuration admission denied",
		)
	}

	return nil, err
}
