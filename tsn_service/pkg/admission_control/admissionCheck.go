package admissioncontrol

import (
	"OpenCNC/common/observability"
	store "OpenCNC/common/store-wrapper"
	observabilityv1 "OpenCNC/common/structures/logging"
	"context"
	"fmt"
)

// TODO: Compare config with resources for the switch
func AdmissionCheck(
	ctx context.Context,
	obs *observability.Client,
	ipAddress string,
	confId string,
) (bool, error) {
	fmt.Println("TODO: Implement actual admission check")

	// Get config
	_, err := store.GetTopologyConfiguration(confId)
	if err != nil {
		fmt.Println("Failed getting resources: %v", err)

		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed getting configuration for admission check: %v",
				err,
			))
			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"admission_control",
				"check",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"configuration",
				confId,
				fmt.Sprintf("Failed getting configuration for admission check: %v", err),
			)
		}

		return false, err
	}

	//TODO: Make check between swResource and swConf

	if obs != nil {
		_ = obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"admission_control",
			"check",
			observabilityv1.DomainResult_DOMAIN_RESULT_ACCEPTED,
			"configuration",
			confId,
			"Configuration passed admission check",
		)
	}

	return true, nil
}
