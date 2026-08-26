/*
Running etcd in the Background
If you want to run etcd in the background as a service, you can add the & at the end of the command:
etcd --name node1 \
     --data-dir /var/lib/etcd \
     --listen-client-urls http://localhost:2379 \
     --advertise-client-urls http://localhost:2379

To stop etcd:
	sudo systemctl stop etcd

To read using etcdctl:
	etcdctl get "" --prefix
	etcdctl put $KEY $VALUE
	etcdctl del $KEY

*/

package storewrapper

import (
	"context"
	"fmt"
	"OpenCNC/main_service/pkg/structures/configuration"
	devicemodelregistry "OpenCNC/main_service/pkg/structures/device-model-registry"
	"OpenCNC/main_service/pkg/structures/event"
	moduleregistry "OpenCNC/main_service/pkg/structures/module-registry"
	"OpenCNC/main_service/pkg/structures/notification"
	"OpenCNC/main_service/pkg/structures/schedule"
	"OpenCNC/main_service/pkg/structures/streamObjects"
	"OpenCNC/main_service/pkg/structures/topology"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	resources "OpenCNC/main_service/pkg/structures/resources"
	monitor "OpenCNC/main_service/pkg/structures/temp-monitor-conf"

	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

var stores = []string{
	"configurations",
	"streams",
	"events",
	"endnodes",
	"bridges",
	"links",
}

//var log = logger.GetLogger()

// Generates all stores defined in the global variable "stores"
func CreateStores() {
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	var successful = true
	for _, name := range stores {
		//log.Infof("About to create store %v", name)

		// Per-call context for Get
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Get(ctx, name)
		cancel()
		if err != nil {
			//log.Infof("Failed to check store \"%s\": %v", name, err)
			successful = false
			continue
		}

		if resp.Count == 0 {
			// Per-call context for Put
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := client.Put(ctx, name, "")
			cancel()
			if err != nil {
				//log.Infof("Failed to create store \"%s\": %v", name, err)
				successful = false
			}
		} else {
			//log.Infof("Store \"%s\" already exists", name)
		}
	}

	if successful {
		//log.Infof("All stores created!")
	}
}

// Log an event to k/v store
func LogEvent(ev *event.Event) error {
	// Serialize event
	obj, err := proto.Marshal(ev)
	if err != nil {
		//log.Errorf("Failed to marshal request: %v", err)
		return err
	}

	urn := ""

	// Create a URN where the serialized request will be stored
	if ev.EventType == event.EventType_ADD_STREAM {
		urn += "events.addStream."
	} else if ev.EventType == event.EventType_REMOVE_STREAM {
		urn += "events.removeStream."
	}

	urn += fmt.Sprintf("%v", ev.EventId)

	// Send serialized event to it's specific path in a store
	err = sendToStore(obj, urn)
	if err != nil {
		return err
	}

	return nil
}

// Take in a config request from UNI and store it in the k/v, store "streams" with a specific path for each request
func StoreUniConfRequest(req *configuration.Request) (*notification.UUID, error) {
	// Serialize request
	obj, err := proto.Marshal(req)
	if err != nil {
		//log.Errorf("Failed to marshal request: %v", err)
		return nil, err
	}

	// Create a URN where the serialized request will be stored
	urn := "streams.requests."

	var requestId = notification.UUID{
		Value: fmt.Sprintf("%v", uuid.New()),
	}
	urn += fmt.Sprintf("%v", requestId.Value)

	// Send serialized request to it's specific path in a store
	err = sendToStore(obj, urn)
	if err != nil {
		return nil, err
	}

	return &requestId, nil
}

// Get configuration response from k/v store
func GetResponseData(configId string) (*configuration.ConfigResponse, error) {

	// Build the URN for the request data
	urn := "configurations.tsn-configuration." + configId

	//log.Info(urn)

	// Send request to specific path in k/v store "streams"
	respData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return nil, err
	}

	// Unmarshal the byte slice from the store into request data
	var req = &configuration.ConfigResponse{}
	err = proto.Unmarshal(respData, req)
	if err != nil {
		//log.Errorf("Failed to unmarshal request data from store: %v", err)
		return &configuration.ConfigResponse{}, nil
	}

	return req, nil
}

// Get configuration as set request (network change) from k/v store
func GetConfigurationRequest(configId string) (*pb.SetRequest, error) {
	// Build the URN for the request data
	urn := "configurations.tsn-configuration." + configId

	// Send request to specific path in k/v store "configurations"
	rawData, err := getRawDataFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return nil, err
	}

	var confReq = &pb.SetRequest{}
	if err = proto.Unmarshal(rawData, confReq); err != nil {
		//log.Errorf("Failed unmarshaling schedule: %v", err)
		return nil, err
	}

	return confReq, nil
}

func GetResources(urn string) (*resources.Switch, error) {
	// Send request to specific path in k/v store "configurations"
	rawData, err := getRawDataFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &resources.Switch{}, err
	}

	var switchResource = &resources.Switch{}
	if err = proto.Unmarshal(rawData, switchResource); err != nil {
		//log.Errorf("Failed unmarshaling schedule: %v", err)
		return &resources.Switch{}, err
	}

	return switchResource, nil
}

func StoreMonitorConfig(conf *monitor.Config, deviceIP string) error {
	obj, err := proto.Marshal(conf)
	if err != nil {
		//log.Errorf("Failed to marshal request: %v", err)
		return err
	}

	urn := "configurations.monitor-config." + deviceIP

	// Send serialized request to it's specific path in a store
	err = sendToStore(obj, urn)
	if err != nil {
		return err
	}

	return nil
}

func StoreAdapter(protocol string, adapterName string) error {
	data, err := proto.Marshal(&monitor.Adapter{
		Protocol: protocol,
		Address:  adapterName,
	})
	if err != nil {
		//log.Errorf("Failed marshaling adapter: %v", err)
		return err
	}

	if err := sendToStore(data, "configurations.adapter."+protocol); err != nil {
		////log.Errorf("Failed storing adapter: %v", err)
		return err
	}

	return nil
}

func StoreResource(res *resources.Switch) error {
	rawRes, err := proto.Marshal(res)
	if err != nil {
		//log.Errorf("Failed to marshall resource", err)
		return err
	}

	if err := sendToStore(rawRes, "resources."+res.IpAddress); err != nil {
		//log.Errorf("Failed storing resource: %v", err)
		return err
	}

	return nil
}

func StoreNode(node *streamObjects.NodeObject) error {
	// Serialize node object
	obj, err := proto.Marshal(node)
	if err != nil {
		//log.Errorf("Failed to marshal node object: %v", err)
		return err
	}

	// Create a URN where the serialized node object will be stored
	urn := "endnodes." + node.GetNodeType() + "." + node.GetStreamMAC()

	// Send serialized node object to it's specific path in the store
	err = sendToStore(obj, urn)
	if err != nil {
		return err
	}

	return nil
}

func StoreStream(stream *configuration.Response) error {
	// Serialize stream object
	obj, err := proto.Marshal(stream)
	if err != nil {
		//log.Errorf("Failed to marshal stream object: %v", err)
		return err
	}

	// Create a URN where the serialized stream object will be stored
	urn := "endnodes.streams" + "." + stream.GetStatusGroup().GetStrId().GetMacAddress() + "." + stream.GetStatusGroup().GetStrId().GetUniqueId()

	//log.Infof("Storing stream at %v", urn)

	// Send serialized stream object to it's specific path in the store
	err = sendToStore(obj, urn)
	if err != nil {
		return err
	}

	return nil
}

func GetAllNodesOfType(nodeType string) (*streamObjects.NodeList, error) {
	// Get all objects
	nodes, err := getAllNodesFromStore()
	if err != nil {
		//log.Errorf("Failed getting all nodes from store: %v", err)
		return &streamObjects.NodeList{}, err
	}

	nodesOfCorrectType := &streamObjects.NodeList{}

	// Sort through objects
	for _, node := range nodes {
		if node.GetNodeType() == nodeType {
			nodesOfCorrectType.Nodes = append(nodesOfCorrectType.Nodes, node)
		}
	}

	return nodesOfCorrectType, nil
}

// Gets all streams from k/v store
func GetAllStreams() (*configuration.ConfigResponse, error) {
	streams, err := getAllStreamsFromStore()
	if err != nil {
		//log.Errorf("Failed getting all streams from store: %v", err)
		return &configuration.ConfigResponse{}, err
	}

	return streams, nil
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

func GetModuleRegistry() (*moduleregistry.ModuleRegistry, error) {
	// Build the URN for the request data
	urn := "yang-modules"

	rawData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &moduleregistry.ModuleRegistry{}, err
	}

	var mregistry = &moduleregistry.ModuleRegistry{}
	if err = proto.Unmarshal(rawData, mregistry); err != nil {
		//log.Errorf("Failed unmarshaling schedule: %v", err)
		return &moduleregistry.ModuleRegistry{}, err
	}
	return mregistry, nil
}

func StoreModuleRegistry() error {

	registry, err := moduleregistry.CreateRegistry("pkg/structures/yang_modules")
	if err != nil {
		//log.Errorf("Failed to create module registry", err)
		return err
	}

	rawResource, err := proto.Marshal(registry)
	if err != nil {
		//log.Errorf("Failed to marshall resource", err)
		return err
	}

	// Create a URN where the serialized request will be stored
	urn := "yang-modules"

	//log.Infof("Storing module registry at %v", urn)

	if err := sendToStore(rawResource, urn); err != nil {
		//log.Errorf("Failed storing resource: %v", err)
		return err
	}

	return nil
}

func StoreDeviceModel(model *devicemodelregistry.DeviceModel) error {
	registry, err := GetModuleRegistry()
	if err != nil {
		return err
	}

	yangModulesMap := make(map[string]*moduleregistry.YangModule)
	for _, rmodule := range registry.YangModules {
		yangModulesMap[rmodule.Name] = rmodule
	}

	// Remove YangFiles with empty name
	validYangFiles := []*devicemodelregistry.YangFile{}
	for _, ymodule := range model.YangFiles {
		if ymodule.Name == "" {
			fmt.Printf("Skipping YangFile with empty name in model %s\n", model.Name)
			continue
		}
		if ymodule.Revision == "" {
			ymodule.Revision = ""
		}
		if rmodule, ok := yangModulesMap[ymodule.Name]; ok {
			ymodule.Structure = rmodule.Structure
		} else {
			ymodule.Structure = "No structure found."
		}
		validYangFiles = append(validYangFiles, ymodule)
	}
	model.YangFiles = validYangFiles

	urn := "device-models." + model.Name

	// Overwrite existing models
	if _, err := getFromStore(urn); err == nil {
		fmt.Printf("Device model %s exists, overwriting.\n", model.Name)
	}

	rawResource, err := proto.Marshal(model)
	if err != nil {
		return err
	}

	if err := sendToStore(rawResource, urn); err != nil {
		return err
	}

	fmt.Printf("Device model %s stored successfully.\n", model.Name)
	return nil
}

func GetDeviceModel(name string) (*devicemodelregistry.DeviceModel, error) {
	// Build the URN for the request data
	urn := "device-models." + name

	// Send request to specific path in k/v store "device-models"
	rawData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting request data from store: %v", err)
		return &devicemodelregistry.DeviceModel{}, err
	}

	var model = &devicemodelregistry.DeviceModel{}
	if err = proto.Unmarshal(rawData, model); err != nil {
		//log.Errorf("Failed unmarshaling schedule: %v", err)
		return &devicemodelregistry.DeviceModel{}, err
	}
	return model, nil
}

func GetDeviceModelRegistry() (*devicemodelregistry.DeviceModelRegistry, error) {

	// Build the prefix for the request data
	prefix := "device-models."

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		//log.Errorf("Failed getting device model from store: %v", err)
		return &devicemodelregistry.DeviceModelRegistry{}, err
	}

	var dregistry = &devicemodelregistry.DeviceModelRegistry{}

	for _, model := range rawData.Kvs {
		var dmodel = &devicemodelregistry.DeviceModel{}
		if err = proto.Unmarshal([]byte(model.Value), dmodel); err != nil {
			//log.Errorf("Failed unmarshaling device model: %v", err)
			return &devicemodelregistry.DeviceModelRegistry{}, err
		}
		dregistry.DeviceModels = append(dregistry.DeviceModels, dmodel)
	}

	return dregistry, nil
}

func StoreTopology(topo *topology.Topology) error {

	//log.Infof("Storing topology...")

	storeNodes(topo.GetNodes())
	storeLinks(topo.GetLinks())

	return nil
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

//
/*                   TEMPLATE                   */
//

/*
func PublicFunctionName(req structureType) error {
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
	err = sendToStore(client, obj, urn)
	if err != nil {
		return err
	}

	return nil
} */
