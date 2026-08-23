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
	"context"
	"fmt"
	"strings"
	"time"

	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	moduleregistry "OpenCNC_config_service/common/structures/module-registry"
	"OpenCNC_config_service/common/structures/stream"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/common/structures/topology_config"

	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
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

func StoreDeviceModel(model *devicemodelregistry.DeviceModel) error {
	if model == nil {
		return fmt.Errorf("device model cannot be nil")
	}
	if model.Name == "" {
		return fmt.Errorf("device model name cannot be empty")
	}

	urn := "device-models." + model.Name

	rawResource, err := proto.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal device model: %v", err)
	}

	if err := SendToStore(rawResource, urn); err != nil {
		return fmt.Errorf("failed to store device model %s: %v", model.Name, err)
	}

	return nil
}

func DeleteDeviceModel(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("device model name cannot be empty")
	}

	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	key := strings.ReplaceAll("device-models."+name, ".", "/")
	_, err = client.Delete(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to delete device model %s: %v", name, err)
	}
	return nil
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

func GetNode(name string) (*topology.Node, string, error) {
	client, err := createEtcdClient()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	node, prefix, err := getNode(client, name)
	if err != nil {
		return nil, "", err
	}

	return node, prefix, nil
}

func GetLink(name string) (*topology.Link, string, error) {
	client, err := createEtcdClient()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	link, prefix, err := getLink(client, name)
	if err != nil {
		return nil, "", err
	}

	return link, prefix, nil
}

func GetNodes() ([]*topology.Node, error) {
	endnodes := getNodes("endnodes")
	bridges := getNodes("bridges")

	nodes := append(endnodes, bridges...)

	return nodes, nil
}

func GetLinks() ([]*topology.Link, error) {
	links := getLinks("links")

	return links, nil
}

func DeleteNode(name string) error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	_, prefix, err := getNode(client, name)
	if err != nil {
		return err
	}

	key := strings.ReplaceAll(prefix+"."+name, ".", "/")

	if _, err := client.Delete(context.Background(), key); err != nil {
		return fmt.Errorf("failed to delete node %s: %w", name, err)
	}

	return nil
}

func DeleteLink(name string) error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	_, prefix, err := getLink(client, name)
	if err != nil {
		return err
	}

	key := strings.ReplaceAll(prefix+"."+name, ".", "/")

	if _, err := client.Delete(context.Background(), key); err != nil {
		return fmt.Errorf("failed to delete link %s: %w", name, err)
	}

	return nil
}

func StoreTopology(topo *topology.Topology) error {
	if topo == nil {
		return fmt.Errorf("topology cannot be nil")
	}

	if err := storeNodes(topo.GetNodes()); err != nil {
		return fmt.Errorf("failed to store nodes: %w", err)
	}

	if err := storeLinks(topo.GetLinks()); err != nil {
		return fmt.Errorf("failed to store links: %w", err)
	}

	return nil
}

func DeleteTopology() error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prefixes := []string{
		"bridges/",
		"endnodes/",
		"links/",
	}

	var totalDeleted int64

	for _, prefix := range prefixes {
		// First get all keys under the prefix.
		resp, err := client.Get(
			ctx,
			prefix,
			clientv3.WithPrefix(),
		)
		if err != nil {
			return fmt.Errorf("failed to get keys under %q: %w", prefix, err)
		}

		log.Infof("Found %d keys under %s", len(resp.Kvs), prefix)

		// Delete each key explicitly.
		for _, kv := range resp.Kvs {
			key := string(kv.Key)

			log.Infof("Deleting topology key: %s", key)

			deleteResp, err := client.Delete(ctx, key)
			if err != nil {
				return fmt.Errorf("failed to delete key %q: %w", key, err)
			}

			totalDeleted += deleteResp.Deleted
		}
	}

	log.Infof("DeleteTopology: deleted %d keys", totalDeleted)

	return nil
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

func StoreStream(streamObj *stream.Stream, id string) error {
	if streamObj == nil {
		return fmt.Errorf("cannot store nil stream")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("stream ID cannot be empty")
	}

	obj, err := proto.Marshal(streamObj)
	if err != nil {
		return fmt.Errorf(
			"failed to marshal stream %s: %w",
			id,
			err,
		)
	}

	urn := "streams." + id

	if err := SendToStore(obj, urn); err != nil {
		return fmt.Errorf(
			"failed to store stream %s: %w",
			id,
			err,
		)
	}

	return nil
}

func GetStreamByID(id string) (*stream.Stream, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("stream ID cannot be empty")
	}

	rawData, err := getFromStoreWithPrefix("streams." + id)
	if err != nil {
		return nil, err
	}

	if len(rawData.Kvs) == 0 {
		return nil, fmt.Errorf("stream %s not found", id)
	}

	streamObj := &stream.Stream{}

	if err := proto.Unmarshal(
		[]byte(rawData.Kvs[0].Value),
		streamObj,
	); err != nil {
		return nil, fmt.Errorf(
			"failed unmarshaling stream %s: %w",
			id,
			err,
		)
	}

	return streamObj, nil
}

func DeleteStream(id string) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return fmt.Errorf("stream ID cannot be empty")
	}

	client, err := createEtcdClient()
	if err != nil {
		return err
	}
	defer client.Close()

	key := "streams/" + id

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf(
			"failed to delete stream %s: %w",
			id,
			err,
		)
	}

	return nil
}

type StoredStream struct {
	ID     string
	Stream *stream.Stream
}

func GetStreams() ([]StoredStream, error) {
	rawData, err := getFromStoreWithPrefix("streams")
	if err != nil {
		return nil, err
	}

	streams := make([]StoredStream, 0, len(rawData.Kvs))

	for _, rawStream := range rawData.Kvs {
		streamObj := &stream.Stream{}

		if err := proto.Unmarshal([]byte(rawStream.Value), streamObj); err != nil {
			log.Errorf("Failed unmarshaling stream: %v", err)
			continue
		}

		key := string(rawStream.Key)

		// Expected key:
		// streams/<stream-id>
		id := strings.TrimPrefix(key, "streams/")

		streams = append(streams, StoredStream{
			ID:     id,
			Stream: streamObj,
		})
	}

	return streams, nil
}

/*
func GetStreams(prefix string) []*stream.Stream {
	var streams []*stream.Stream

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		log.Errorf("Failed getting streams from store: %v", err)
		return streams
	}

	for _, rawStream := range rawData.Kvs {
		streamObj := &stream.Stream{}

		if err = proto.Unmarshal([]byte(rawStream.Value), streamObj); err != nil {
			log.Errorf("Failed unmarshaling stream: %v", err)
			return streams
		}

		streams = append(streams, streamObj)
	}

	return streams
}
*/
