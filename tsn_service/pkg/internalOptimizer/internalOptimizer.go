package internalOptimizer

import (
	"bytes"
	"fmt"
	"os"
	store "tsn-service/pkg/storewrapper"
	"tsn-service/pkg/structures/schedule"
	"tsn-service/pkg/structures/topology"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"

	"google.golang.org/protobuf/proto"
)

//var log = logger.GetLogger()

var defaultSchedID = "default_schedule"

// Calculates configuration set request using optimizer, if that failes build configuration set request from default schedule
func CalculateConf(topology *topology.Topology, oldConfig *schedule.GclConfiguration) (*schedule.GclConfiguration, error) {

	// Load default schedule from k/v store
	sched, err := store.GetSchedule(defaultSchedID)
	if err != nil {
		//log.Errorf("Failed getting default schedule: %v", err)
		return nil, err
	}

	// Create configuration set request based on default schedule and topology
	configSetReq, err := createConfigurationFromSchedule(sched, topology)
	if err != nil {
		//log.Errorf("Failed creating default configuraiton set request: %v", err)
		fmt.Printf("Failed creating default configuraiton set request: %v\n", err)

		return nil, err
	}

	//log.Infof("Schedule looks like: %v", configSetReq)
	fmt.Printf("Schedule looks like: %v\n", configSetReq)

	return configSetReq, nil
}

// Reads default schedule config file and stores configuration for schedule in k/v store
func CreateDefaultSchedule() error {
	// Read default schedule from file
	schedBytes, err := os.ReadFile("configs/schedules/default-schedule.yaml")
	if err != nil {
		fmt.Println("Failed reading default schedule from file")
		//log.Errorf("Failed reading default schedule from file: %v", err)
		return err
	}

	// Convert yaml bytes to json bytes
	jsonBytes, err := yaml.YAMLToJSON(schedBytes)
	if err != nil {
		//log.Errorf("Failed converting file content from yaml to json: %v", err)
		return err
	}

	var defaultSched = &schedule.Schedule{}

	// Deserialize json bytes to schedule
	if err = jsonpb.Unmarshal(bytes.NewReader(jsonBytes), defaultSched); err != nil {
		//log.Errorf("Failed unmarshaling json to protobuf: %v", err)
		return err
	}
	fmt.Println("Successfully created default schedule with ID: ", defaultSchedID)

	// Serialize schedule
	data, err := proto.Marshal(defaultSched)
	if err != nil {
		fmt.Println("Failed marshaling default schedule: %v", err)
		//log.Errorf("Failed marshaling default schedule: %v", err)
		return err
	}

	// Store schedule in k/v store
	err = store.StoreSchedule(data, defaultSchedID)
	if err != nil {
		//log.Errorf("Failed storing default schedule: %v", err)
		return err
	}
	fmt.Println("Successfully stored default schedule with ID: ", defaultSchedID)
	//log.Infof("Successfully stored default schedule with ID: %v", defaultSchedID)

	return nil
}

// // Creates a connection to the network optimizer and requests a new configuration
// func CallOptimizer(input *optimizer.Input) (*optimizer.Output, error) {
// 	// Create gRPC connection
// 	conn, err := grpc.Dial("network-optimizer:5150", grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		//log.Fatalf("Failed dialing network-optimizer: %v", err)
// 		return &optimizer.Output{}, err
// 	}

// 	defer conn.Close()

// 	// Create gRPC client
// 	client := optimizer.NewOptimizerClient(conn)

// 	// Request config from optimizer
// 	output, err := client.GenerateConfiguration(context.Background(), input)
// 	if err != nil {
// 		//log.Errorf("Optimizer failed generating configuration: %v", err)
// 		return &optimizer.Output{}, err
// 	}

// 	return output, nil
// }
