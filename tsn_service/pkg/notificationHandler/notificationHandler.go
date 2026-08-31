package notificationHandler

import (
	"fmt"

	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/internalOptimizer"
	"OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

// Calculates configuration and stores it as a set request in k/v store, returns ID of configuration set request
func CalculateConfiguration(task *optimizer.OptimizationTask, allRequestData []*uni.Request) (*forwarding_plane.ForwardingPlaneModel, error) {

	// TODO: Use requests when creating configuration

	// Calculate configuration set request
	new_fpm, err := internalOptimizer.StartOptimization(task, allRequestData)
	if err != nil {
		//log.Errorf("Failed calculating configuration: %v", err)
		fmt.Printf("Failed calculating configuration: %v\n", err)
		return nil, err
	}

	// Validate the new forwarding plane model before storing it
	admissionCheck := true
	if admissionCheck {
		// Store configuration set request in k/v store
		if err := store.StoreForwardingPlaneConfiguration(new_fpm); err != nil {
			//log.Errorf("Failed storing new configuration: %v", err)
			fmt.Printf("Failed storing configuration: %v\n", err)

			return nil, err
		}

		return new_fpm, nil
	} else {
		//TODO: Handle admission denial appropriately by returning the problem to the optimizer
		// Admission denied
		return nil, fmt.Errorf("admission denied")
	}

}
