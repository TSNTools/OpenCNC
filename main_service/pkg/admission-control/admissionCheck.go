package admissioncontrol

import (
	store "OpenCNC/common/store-wrapper"
	"fmt"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

// TODO: Compare config with resources for the switch
func AdmissionCheck(ipAddress string, confId string) (bool, error) {
	//log.Info("TODO: Implement actual admission check")
	fmt.Println("TODO: Implement actual admission check")

	// Get config
	_, err := store.GetConfiguration(confId)
	if err != nil {
		//log.Errorf("Failed getting resources: %v", err)
		fmt.Println("Failed getting resources: %v", err)

		return false, err
	}

	//TODO: Make check between swResource and swConf
	return true, nil
}
