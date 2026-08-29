package storewrapper

import (
	"OpenCNC/common/structures/uni"

	"google.golang.org/protobuf/proto"
)

func GetRequestData(configId string) (*uni.Request, error) {
	// Build the URN for the request data
	urn := "streams.requests." + configId

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
