package eventhandler

import (
	"fmt"
	"time"

	store "OpenCNC/common/store-wrapper"
	storewrapper "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/topology"
	uni "OpenCNC/common/structures/uni"
	configurationHandler "OpenCNC/main_service/pkg/configuration-handler"
	"OpenCNC/tsn_service/pkg/notificationServer"

	"google.golang.org/protobuf/types/known/timestamppb"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

// Take in a configuration request, process it and once a configuration
// has been calculated, return ID of the new configuration.
func HandleAddStreamEvent(configReq *uni.ConfigRequest, timeOfReq time.Time) (string, error) {
	// Store requests in k/v store and log the events
	requestIds, err := store.StoreMultipleUniConfRequest(configReq.Requests)
	if err != nil {
		//log.Errorf("Failed storing and logging events: %v", err)
		fmt.Printf("Failed storing and logging events: %v", err)
		return "", err
	}

	//log.Info("Configuration requests stored successfully!")
	fmt.Println("Configuration requests stored successfully!")

	// Notify TSN service that it should calculate a new configuration
	event := &notificationServer.Event{
		EventId:    "...",
		Type:       notificationServer.EventType_STREAM_ADDED,
		OccurredAt: timestamppb.New(timeOfReq),
		Source:     "some-service",
		Payload: &notificationServer.Event_StreamAdded{
			StreamAdded: &notificationServer.StreamAdded{
				RequestIds: requestIds,
				StreamId:   "...",
			},
		},
	}

	configId, err := notifyTsnService(event)
	if err != nil {
		//log.Errorf("Failed to notify TSN service: %v", err)
		fmt.Printf("Failed to notify TSN service: %v", err)
		return "", err
	}

	tentativeConfiguration, err := storewrapper.GetTopologyConfiguration(configId)
	if err != nil {
		fmt.Printf("Failed getting tentative configuration: %v", err)
		return "", err
	}

	// Finalize the configuration
	if err = configurationHandler.FinalizeConfiguration(tentativeConfiguration); err != nil {
		//log.Errorf("Failed finalizing configuration: %v", err)
		fmt.Printf("Failed finalizing configuration: %v", err)
		return "", err
	}

	// Send network change to config-service to use new configuration
	if err = notifyConfigService(configId); err != nil {
		//MTODO:
		//log.Errorf("Failed notifying config-service of new configuration: %v", err)
		fmt.Printf("Failed notifying config-service of new configuration: %v", err)

		return "", err
	}

	return configId, nil
}

func RegisterNode(node *topology.Node) error {
	if node.GetType() == topology.NodeRole_END_STATION || node.GetType() == topology.NodeRole_END_STATION {
		//log.Infof("Registering node with MAC %v of type %v", node.GetStreamMAC(), node.GetNodeType())
		store.StoreNode(node)
	} else {
		//log.Errorf("Failed identifying type of end node: %v", node.GetNodeType())
	}

	return nil
}
