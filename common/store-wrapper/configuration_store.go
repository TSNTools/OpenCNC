package storewrapper

import (
	stream "OpenCNC/common/structures/stream_config"
	"OpenCNC/common/structures/topology_config"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

////////////////////////////////////
// store forwarding plane models
//////////////////////////////////////

func StoreForwardingPlaneConfiguration(model *forwarding_plane.ForwardingPlaneModel) error {
	if model == nil {
		return fmt.Errorf("cannot store nil forwarding plane model")
	}

	// Store topology configuration.
	if model.GetConfiguration() != nil {
		if err := StoreTopologyConfiguration(model.GetConfiguration()); err != nil {
			return fmt.Errorf("failed to store topology configuration: %w", err)
		}
	}

	// Store each stream definition and configuration separately.
	for _, streamModel := range model.GetStreams() {
		if streamModel == nil {
			continue
		}

		if streamModel.GetConfiguration() != nil {
			if err := StoreStreamConfiguration(streamModel.GetConfiguration()); err != nil {
				return fmt.Errorf("failed to store stream configuration: %w", err)
			}
		}
	}

	return nil
}

//////////////////////////////////////
// Topology Configuration Functions //
//////////////////////////////////////

func GetTopologyConfiguration(configID string) (*topology_config.TopologyConfig, error) {
	data, err := GetFromStore("configurations.topology." + configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topology configuration: %w", err)
	}

	cfg := &topology_config.TopologyConfig{}
	if err := proto.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to deserialize topology configuration: %w", err)
	}

	return cfg, nil
}

func StoreTopologyConfiguration(cfg *topology_config.TopologyConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot store nil configuration")
	}

	data, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize topology configuration: %w", err)
	}

	return SendToStore(data, "configurations.topology."+cfg.GetConfigId())
}

//////////////////////////////////////
// Stream Configuration Functions 	//
//////////////////////////////////////

func GetStreamConfiguration(streamID string) (*stream.StreamConfiguration, error) {
	data, err := GetFromStore("configurations.streams." + streamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream configuration: %w", err)
	}

	cfg := &stream.StreamConfiguration{}
	if err := proto.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to deserialize stream configuration: %w", err)
	}

	return cfg, nil
}

func StoreStreamConfiguration(cfg *stream.StreamConfiguration) error {
	if cfg == nil {
		return fmt.Errorf("cannot store nil stream configuration")
	}

	data, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize stream configuration: %w", err)
	}

	return SendToStore(data, "configurations.streams."+cfg.GetStreamId().String())
}

////////////////////////////
// UNI Request  Functions //
////////////////////////////

func GetUniRequestData(configId string) (*uni.Request, error) {
	// Build the URN for the request data
	urn := "configurations.requests." + configId

	// Send request to specific path in k/v store "streams"
	rawData, err := GetFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &uni.Request{}, err
	}

	// Unmarshal the byte slice from the store into request data
	var req = &uni.Request{}
	err = proto.Unmarshal(rawData, req)
	if err != nil {
		//log.Errorf("Failed to unmarshal request data from store: %v", err)
		return nil, err
	}

	return req, nil
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
	urn := "configurations.requests."

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
