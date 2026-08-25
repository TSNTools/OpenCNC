package admissioncontrol

import (
	"fmt"
	store "main-service/pkg/store-wrapper"
	"main-service/pkg/structures/resources"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

// TODO: Compare config with resources for the switch
func AdmissionCheck(ipAddress string, confId string) (bool, error) {
	//log.Info("TODO: Implement actual admission check")
	fmt.Println("TODO: Implement actual admission check")

	// Get available resources for the switch
	swResource, err := getNetworkResources(ipAddress)
	if err != nil {
		fmt.Println("Failed getting resources: %v", err)
		//log.Errorf("Failed getting resources: %v", err)
		return false, err
	}

	// Get config
	swConf, err := store.GetConfigurationRequest(confId)
	if err != nil {
		//log.Errorf("Failed getting resources: %v", err)
		fmt.Println("Failed getting resources: %v", err)

		return false, err
	}

	// Don't want to remove output variables, so just never run this log (THIS IS TO BE DELETED)
	if false {
		//log.Info(swResource, swConf)
		fmt.Println(swResource, swConf)

	}

	//TODO: Make check between swResource and swConf
	return true, nil
}

// Output: List of resources
func getNetworkResources(ipAddress string) (*resources.Switch, error) {
	return store.GetResources("resources." + ipAddress)
}
