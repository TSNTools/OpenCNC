package storewrapper

import (
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/common/structures/uni"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func GetConfiguration(confId string) (*topology_config.TopologyConfig, error) {
	// this requires all configurations in the store to be normilized to topology_config.TopologyConfig,
	//  otherwise it will fail to unmarshal
	urn := "configurations." + confId

	rawConf, err := GetFromStore(urn)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve configuration %s: %v", confId, err)
	}

	fmt.Println("Retrieved topology configuration from k/v store")

	var config topology_config.TopologyConfig
	if err := proto.Unmarshal(rawConf, &config); err != nil {
		return nil, fmt.Errorf("failed to deserialize topology configuration: %v", err)
	}

	return &config, nil
}

func StoreConfiguration(cfg *topology_config.TopologyConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot store nil configuration")
	}

	configBytes, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	err = SendToStore(
		configBytes,
		"configurations."+cfg.GetConfigId(),
	)
	if err != nil {
		return fmt.Errorf("failed to store configuration: %w", err)
	}

	return nil
}

// Take in a config request from UNI and store it in the k/v, store "streams" with a specific path for each request
func StoreUniConfRequest(req *uni.Request) (string, error) {
	// Serialize request
	obj, err := proto.Marshal(req)
	if err != nil {
		//log.Errorf("Failed to marshal request: %v", err)
		return "", err
	}

	// Create a URN where the serialized request will be stored
	urn := "streams.requests."

	var requestId = req.GetId()
	if requestId == "" {
		requestId = uuid.New().String()
	}
	urn += fmt.Sprintf("%v", requestId)

	// Send serialized request to it's specific path in a store
	err = SendToStore(obj, urn)
	if err != nil {
		return "", err
	}

	return requestId, nil
}

// Takes in requests, stores them, and logs the events
func StoreMultipleUniConfRequest(requestList []*uni.Request) ([]string, error) {

	var requestIds []string

	// Store all requests in a k/v store
	for _, request := range requestList {
		// Store request in k/v store and get the ID for the request
		id, err := StoreUniConfRequest(request)
		requestIds = append(requestIds, id)
		if err != nil {
			fmt.Printf("Storing configuration requests failed: %v", err)
		}
	}
	return requestIds, nil
}
