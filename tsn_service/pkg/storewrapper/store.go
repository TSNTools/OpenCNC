package storewrapper

import (
	"fmt"
	"tsn-service/pkg/structures/configuration"
	"tsn-service/pkg/structures/schedule"
	"tsn-service/pkg/structures/topology"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/protobuf/proto"
)

//var log = logger.GetLogger()

func GetKeyValueStore(urn string) (*configuration.Request, error) {
	// Send request to specific path in k/v store "streams"
	rawData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &configuration.Request{}, err
	}

	// Unmarshal the byte slice from the store into request data
	var req = &configuration.Request{}
	err = proto.Unmarshal(rawData, req)
	if err != nil {
		//log.Errorf("Failed to unmarshal request data from store: %v", err)
		return nil, err
	}

	return req, nil
}

func GetRequestData(configId string) (*configuration.Request, error) {
	// Build the URN for the request data
	urn := "streams.requests." + configId

	// Send request to specific path in k/v store "streams"
	rawData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &configuration.Request{}, err
	}

	// Unmarshal the byte slice from the store into request data
	var req = &configuration.Request{}
	err = proto.Unmarshal(rawData, req)
	if err != nil {
		//log.Errorf("Failed to unmarshal request data from store: %v", err)
		return nil, err
	}

	return req, nil
}

func StoreConfiguration(config *schedule.GclConfiguration, confId string) error {
	// Create a URN where the serialized request will be stored
	urn := "configurations.tsn-configuration." + confId

	// Serialize config
	rawConf, err := proto.Marshal(config)
	if err != nil {
		//log.Errorf("Failed marshaling config: %v", err)
		return err
	}

	// Send serialized request to it's specific path in a store
	if err = sendToStore(rawConf, urn); err != nil {
		//log.Errorf("Failed storing configuration: %v", err)
		return err
	}

	return nil
}

func GetConfiguration(confId string) (*schedule.GclConfiguration, error) {
	// Construct the URN where the config is stored
	urn := "configurations.tsn-configuration." + confId

	// Get the raw bytes from the store
	rawConf, err := getFromStore(urn)
	if err != nil {
		// log.Errorf("Failed to retrieve configuration: %v", err)
		return nil, err
	}

	fmt.Println("Retrieved configuration from k/v store")

	// Unmarshal the bytes into a SetRequest
	var config schedule.GclConfiguration
	if err := proto.Unmarshal(rawConf, &config); err != nil {
		// log.Errorf("Failed to unmarshal configuration: %v", err)
		return nil, err
	}

	return &config, nil
}

func GetAllConfigurations() ([]*pb.SetRequest, error) {
	const prefix = "configurations.tsn-configuration"

	// Get all KVs with this prefix from etcd
	resp, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get configurations: %w", err)
	}

	var configs []*pb.SetRequest
	for _, kv := range resp.Kvs {
		var config pb.SetRequest
		if err := proto.Unmarshal(kv.Value, &config); err != nil {
			fmt.Printf("Failed to unmarshal config at key %s: %v\n", string(kv.Key), err)
			continue // skip invalid entries
		}
		configs = append(configs, &config)
	}

	return configs, nil
}

func StoreSchedule(sched []byte, schedId string) error {
	// Create a URN where the serialized request will be stored
	urn := "configurations.schedules." + schedId

	// Send serialized request to it's specific path in a store
	err := sendToStore(sched, urn)
	if err != nil {
		//log.Errorf("Failed storing schedule: %v", err)
		return err
	}

	return nil
}

func GetSchedule(schedId string) (*schedule.Schedule, error) {
	// Build the URN for the request data
	urn := "configurations.schedules." + schedId

	// Send request to specific path in k/v store "configurations"
	rawData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &schedule.Schedule{}, err
	}

	var sched = &schedule.Schedule{}
	if err = proto.Unmarshal(rawData, sched); err != nil {
		//log.Errorf("Failed unmarshaling schedule: %v", err)
		return &schedule.Schedule{}, err
	}

	return sched, nil
}

func GetNodes(prefix string) ([]*topology.Node, error) {
	var nodes []*topology.Node

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		//log.Errorf("Failed getting nodes from store: %v", err)
		return nodes, err
	}

	for _, rawNode := range rawData.Kvs {
		node := &topology.Node{}

		if err = proto.Unmarshal([]byte(rawNode.Value), node); err != nil {
			//log.Errorf("Failed unmarshaling node: %v", err)
			return nodes, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func GetTopology() (*topology.Topology, error) {
	var topo = &topology.Topology{}

	endnodes, _ := GetNodes("endnodes")
	bridges, _ := GetNodes("bridges")

	topo.Nodes = append(endnodes, bridges...)

	links := getLinks("links")

	topo.Links = append(topo.Links, links...)

	return topo, nil
}

//////////////////////////////////////////////////
/*                   TEMPLATES                  */
//////////////////////////////////////////////////
/*

func PublicFunctionName(client *clientv3.Client, req structureType) error {
	// Serialize request
	obj, err := proto.Marshal(req)
	if err != nil {
		//log.Errorf("Failed to marshal request: %v", err)
		return err
	}

	// Create a URN where the serialized request will be stored
	urn := "store.type."

	// TODO: Generate or use some ID to keep track of the specific stream request
	urn += fmt.Sprintf("%v", uuid.New())

	// Send serialized request to it's specific path in a store
	err = sendToStore(clienct, obj, urn)
	if err != nil {
		return err
	}

	return nil
}

*/
