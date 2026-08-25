package storewrapper

import (
	"context"
	"fmt"
	"main-service/pkg/structures/configuration"
	"main-service/pkg/structures/streamObjects"
	"main-service/pkg/structures/topology"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
)

// CreateEtcdClient creates and returns an etcd client
func createEtcdClient() (*clientv3.Client, error) {
	endpoints := []string{"http://etcd.default.svc.cluster.local:2379"}
	//[]string{"127.0.0.1:2379"}
	username := os.Getenv("ETCD_USERNAME")
	password := os.Getenv("ETCD_PASSWORD")

	// Initialize the etcd client with provided configuration
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		Username:    username,
		Password:    password,
		DialTimeout: 10 * time.Second, // Timeout for the dial
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}

	// Return the created etcd client
	return client, nil
}

// Takes in an object as a byte slice, a URN
// in format of "storeName/Resource", and stores the structure at the URN
func sendToStore(obj []byte, urn string) error {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	// Replace all dots with slashes
	urn = strings.ReplaceAll(urn, ".", "/")

	// Put the object into etcd
	_, err = client.Put(context.Background(), urn, string(obj))
	if err != nil {
		//log.Infof("Failed storing resource \"%s\": %v", urn, err)
		return err
	}

	return nil
}

// Used when storing repeatedly like links and nodes
func sendToStoreRepeated(client *clientv3.Client, obj []byte, urn string) error {
	// Replace all dots with slashes
	urn = strings.ReplaceAll(urn, ".", "/")

	// Put the object into etcd
	_, err := client.Put(context.Background(), urn, string(obj))
	if err != nil {
		//log.Infof("Failed storing resource \"%s\": %v", urn, err)
		return err
	}
	return nil
}

// Get any data from a k/v store
func getFromStore(urn string) ([]byte, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	// Create a context with a timeout to prevent indefinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Replace all dots with slashes
	urn = strings.ReplaceAll(urn, ".", "/")

	// Get the object from etcd store
	resp, err := client.Get(ctx, urn)
	if err != nil {
		//log.Infof("Failed getting resource \"%s\": %v", urn, err)
		return nil, err
	}

	// If no value is found, return an error
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found: %s", urn)
	}

	// Return the value of the key
	return resp.Kvs[0].Value, nil
}

// Get any data from a k/v store
func getFromStoreWithPrefix(prefix string) (*clientv3.GetResponse, error) {

	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	// Replace all dots with slashes
	prefix = strings.ReplaceAll(prefix, ".", "/")

	resp, err := client.Get(context.Background(), prefix, clientv3.WithPrefix())

	if err != nil {
		return nil, fmt.Errorf("failed to get data with prefix %s: %v", prefix, err)
	}

	// Return the value of the key
	return resp, nil
}

// Get all nodes from etcd store
func getAllNodesFromStore() ([]*streamObjects.NodeObject, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all keys under "endnodes"
	resp, err := client.Get(ctx, "endnodes/", clientv3.WithPrefix())
	if err != nil {
		//log.Infof("Failed getting keys from etcd store: %v", err)
		return nil, err
	}

	var nodeList []*streamObjects.NodeObject

	// Iterate over the keys and values from etcd
	for _, kv := range resp.Kvs {
		// If the key contains "streams", skip it
		if strings.Contains(string(kv.Key), "streams") {
			continue
		}

		var node streamObjects.NodeObject

		// Unmarshal the value into the NodeObject
		if err := proto.Unmarshal(kv.Value, &node); err != nil {
			//log.Infof("Failed unmarshaling entry to node: %v", err)
			return nil, err
		}

		// Add node to the list
		nodeList = append(nodeList, &node)
	}

	return nodeList, nil
}

// Get all streams from etcd store
func getAllStreamsFromStore() (*configuration.ConfigResponse, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all keys under "endnodes"
	resp, err := client.Get(ctx, "endnodes/", clientv3.WithPrefix())
	if err != nil {
		//log.Infof("Failed getting keys from etcd store: %v", err)
		return nil, err
	}

	//log.Info("TODO: Add correct version from somewhere")
	streamList := &configuration.ConfigResponse{
		Version: 1.0,
	}

	// Iterate over the keys and values from etcd
	for _, kv := range resp.Kvs {
		// If the key doesn't contain "streams", skip it
		if !strings.Contains(string(kv.Key), "streams") {
			continue
		}

		var stream configuration.Response

		// Unmarshal the value into the Response struct
		if err := proto.Unmarshal(kv.Value, &stream); err != nil {
			//log.Infof("Failed unmarshaling entry to stream: %v", err)
			return nil, err
		}

		// Append stream to the list
		streamList.Responses = append(streamList.Responses, &stream)
	}

	return streamList, nil
}

// Get raw data from k/v store
func getRawDataFromStore(urn string) ([]byte, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a slice of maximum two URN elements
	urnElems := strings.SplitN(urn, ".", 2)
	if len(urnElems) < 2 {
		return nil, fmt.Errorf("invalid URN format")
	}

	// Get the object from etcd
	resp, err := client.Get(ctx, urnElems[0]+"/"+urnElems[1])
	if err != nil {
		//log.Infof("Failed getting resource \"%s\": %v", urnElems[1], err)
		return nil, err
	}

	// If no value is found, return an error
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found: %s", urn)
	}

	// Return the value of the key
	return resp.Kvs[0].Value, nil
}

func storeNodes(nodes []*topology.Node) error {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	for _, node := range nodes {
		// Serialize node object
		obj, err := proto.Marshal(node)
		if err != nil {
			//log.Errorf("Failed to marshal node object: %v", err)
			return err
		}
		var urn = ""

		if node.Properties != nil && node.Properties.Bridge != nil {
			urn = "bridges." + node.Name
		}

		if node.Properties != nil && node.Properties.EndStation != nil {
			urn = "endnodes." + node.Name
		}

		// Send serialized node object to it's specific path in the store
		err = sendToStoreRepeated(client, obj, urn)
		if err != nil {
			return err
		}
	}

	return nil
}

func storeLinks(links []*topology.Link) error {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	for _, link := range links {
		// Serialize link object
		obj, err := proto.Marshal(link)
		if err != nil {
			//log.Errorf("Failed to marshal node object: %v", err)
			return err
		}

		// Create a URN where the serialized link object will be stored
		urn := "links." + link.Id

		// Send serialized link object to it's specific path in the store
		err = sendToStoreRepeated(client, obj, urn)
		if err != nil {
			return err
		}
	}
	return nil
}

func getLinks(prefix string) []*topology.Link {
	var links []*topology.Link

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		//log.Errorf("Failed getting links from store: %v", err)
		return links
	}

	for _, rawLink := range rawData.Kvs {
		link := &topology.Link{}

		if err = proto.Unmarshal([]byte(rawLink.Value), link); err != nil {
			//log.Errorf("Failed unmarshaling link: %v", err)
			return links
		}
		links = append(links, link)
	}
	return links
}
