package storewrapper

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

import (
	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	moduleregistry "OpenCNC_config_service/common/structures/module-registry"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/common/structures/topology_config"
	"fmt"

	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
	"google.golang.org/protobuf/proto"
)

var log = logger.GetLogger()

// Changed from model plugin to device model registry
func GetDeviceModelRegistry() (*devicemodelregistry.DeviceModelRegistry, error) {

	// Build the prefix for the request data
	prefix := "device-models."

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		log.Errorf("Failed getting device model from store: %v", err)
		return &devicemodelregistry.DeviceModelRegistry{}, err
	}

	var dregistry = &devicemodelregistry.DeviceModelRegistry{}

	for _, model := range rawData.Kvs {
		var dmodel = &devicemodelregistry.DeviceModel{}
		if err = proto.Unmarshal([]byte(model.Value), dmodel); err != nil {
			log.Errorf("Failed unmarshaling device model: %v", err)
			return &devicemodelregistry.DeviceModelRegistry{}, err
		}
		dregistry.DeviceModels = append(dregistry.DeviceModels, dmodel)
	}

	return dregistry, nil
}

func GetDeviceModel(name string) (*devicemodelregistry.DeviceModel, error) {
	// Build the URN for the request data
	urn := "device-models." + name

	// Send request to specific path in k/v store "device-models"
	rawData, err := GetFromStore(urn)
	if err != nil {
		log.Errorf("Failed getting request data from store: %v", err)
		return &devicemodelregistry.DeviceModel{}, err
	}

	var model = &devicemodelregistry.DeviceModel{}
	if err = proto.Unmarshal(rawData, model); err != nil {
		log.Errorf("Failed unmarshaling schedule: %v", err)
		return &devicemodelregistry.DeviceModel{}, err
	}
	return model, nil
}

func GetTopology() (*topology.Topology, error) {
	var topo = &topology.Topology{}

	endnodes := getNodes("endnodes")
	bridges := getNodes("bridges")

	topo.Nodes = append(endnodes, bridges...)

	links := getLinks("links")

	topo.Links = append(topo.Links, links...)

	return topo, nil
}

func GetModuleRegistry() (*moduleregistry.ModuleRegistry, error) {
	// Build the URN for the request data
	urn := "yang-modules."

	rawData, err := GetFromStore(urn)
	if err != nil {
		log.Errorf("Failed getting request data from store: %v", err)
		return &moduleregistry.ModuleRegistry{}, err
	}

	var mregistry = &moduleregistry.ModuleRegistry{}
	if err = proto.Unmarshal(rawData, mregistry); err != nil {
		log.Errorf("Failed unmarshaling schedule: %v", err)
		return &moduleregistry.ModuleRegistry{}, err
	}
	return mregistry, nil
}

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

/*
	func GetLastConfiguration() (*schedule.GclConfiguration, string, error) {
		const prefix = "configurations.tsn-configuration"

		resp, err := getLastFromStoreWithPrefix(prefix)
		if err != nil {
			return nil, "", err
		}
		if len(resp.Kvs) == 0 {
			return nil, "", fmt.Errorf("no configurations found")
		}

		kv := resp.Kvs[0]

		// Convert the key back to your confId (if needed)
		confId := path.Base(string(kv.Key)) // e.g. "4d8f7c72-f9e3-4d33-8210-3a5437b8a821"

		var config schedule.GclConfiguration
		if err := proto.Unmarshal(kv.Value, &config); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal last config: %v", err)
		}

		return &config, confId, nil
	}
*/
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
