package eventhandler

import (
	"fmt"
	"time"

	store "main-service/pkg/store-wrapper"
	"main-service/pkg/structures/configuration"
	"main-service/pkg/structures/notification"
	"main-service/pkg/structures/streamObjects"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

// Take in a configuration request, process it and once a configuration
// has been calculated, return ID of the new configuration.
func HandleAddStreamEvent(event *configuration.ConfigRequest, timeOfReq time.Time) (*notification.UUID, error) {
	// Store requests in k/v store and log the events
	requestIds, err := storeRequestsInStore(event.Requests, timeOfReq)
	if err != nil {
		//log.Errorf("Failed storing and logging events: %v", err)
		fmt.Printf("Failed storing and logging events: %v", err)
		return nil, err
	}

	//log.Info("Configuration requests stored successfully!")
	fmt.Println("Configuration requests stored successfully!")

	// Notify TSN service that it should calculate a new configuration
	configId, err := notifyTsnService(requestIds)
	if err != nil {
		//log.Errorf("Failed to notify TSN service: %v", err)
		fmt.Printf("Failed to notify TSN service: %v", err)
		return nil, err
	}

	fmt.Println("Notified TSN service...")
	//log.Infof("Configuration calculated with ID: %s", configId.GetValue())

	// admissionCheck, err := admissioncontrol.AdmissionCheck("192.168.0.2", configId.GetValue())
	// if err != nil {
	// 	//log.Errorf("Admission failed: %v", err)
	// 	return nil, err
	// }

	admissionCheck := true

	if admissionCheck {
		//log.Info("Admission accepted")
		fmt.Println("Admission accepted")

		// Send network change to config-service to use new configuration
		/*if err = applyConfiguration(configId); err != nil {
			//MTODO:
			//log.Errorf("Failed notifying config-service of new configuration: %v", err)
			fmt.Printf("Failed notifying config-service of new configuration: %v", err)

			return nil, err
		}*/
	} else {
		//log.Info("Admission denied")
	}

	// TODO: Finalize configuration and apply it

	return configId, nil
}

func RegisterNode(node *streamObjects.NodeObject) error {
	if node.GetNodeType() == "talker" || node.GetNodeType() == "listener" {
		//log.Infof("Registering node with MAC %v of type %v", node.GetStreamMAC(), node.GetNodeType())
		store.StoreNode(node)
	} else {
		//log.Errorf("Failed identifying type of end node: %v", node.GetNodeType())
	}

	return nil
}
