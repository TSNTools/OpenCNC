package notificationHandler

import (
	"fmt"

	"tsn-service/pkg/internalOptimizer"
	store "tsn-service/pkg/storewrapper"
	"tsn-service/pkg/structures/configuration"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"github.com/google/uuid"
)

//var log = logger.GetLogger()

// Calculates configuration and stores it as a set request in k/v store, returns ID of configuration set request
func CalculateConfiguration(ids []string) (string, error) {
	// Not yet used when calculating configuration
	var allRequestData []*configuration.Request

	// Get request from k/v store
	for _, requestId := range ids {
		reqData, err := store.GetRequestData(requestId)
		if err != nil {
			return "", err
		}
		fmt.Printf("Got request from store with id: %s\n", requestId)
		allRequestData = append(allRequestData, reqData)
	}

	// TODO: Use requests when creating configuration

	// Get topology
	topology, err := store.GetTopology()
	if err != nil {
		//log.Errorf("Failed getting topology: %v", err)
		fmt.Printf("Failed getting topology: %v\n", err)
		return "", err
	}

	//log.Info("Successfully requested topology from k/v store!")
	fmt.Println("Successfully requested topology from k/v store!")

	// NOT TESTED???
	// Get current configuration of the network
	//oldConfig, err := store.GetConfiguration("old")
	/*
		if err != nil {
			//log.Errorf("Failed getting configuration: %v", err)
			fmt.Printf("Failed getting configuration: %v\n", err)
			return "", err
		}*/
	// Calculate configuration set request
	newConfig, err := internalOptimizer.CalculateConf(topology, nil)
	if err != nil {
		//log.Errorf("Failed calculating configuration: %v", err)
		fmt.Printf("Failed calculating configuration: %v\n", err)
		return "", err
	}

	//log.Info("Successfully calculated new configuration!")
	fmt.Println("Successfully calculated new configuration!")

	// Generate an ID for configuration set request
	confId := fmt.Sprint(uuid.New())

	// Store configuration set request in k/v store
	if err := store.StoreConfiguration(newConfig, confId); err != nil {
		//log.Errorf("Failed storing new configuration: %v", err)
		fmt.Printf("Failed storing configuration: %v\n", err)

		return "", err
	}

	//log.Info("Successfully stored new configuration!")
	fmt.Println("Successfully stored new configuration!")
	return confId, nil
}
