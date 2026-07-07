package storewrapper

import (
	"OpenCNC_config_service/pkg/structures/topology"
	"context"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
)

// createEtcdClient creates and returns an etcd client
func createEtcdClient() (*clientv3.Client, error) {
	// Initialize the etcd client with provided configuration
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://127.0.0.1:2379"}, // List of etcd endpoints
		DialTimeout: 10 * time.Second,                  // Timeout for the dial
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}

	// Return the created etcd client
	return client, nil
}

// Takes in an object as a byte slice, a URN in the format of "storeName.Resource",
// //and stores the structure at the URN
func SendToStore(obj []byte, urn string) error {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Replace all dots with slashes
	urn = strings.ReplaceAll(urn, ".", "/")

	// Put the object into etcd
	_, err = client.Put(context.Background(), urn, string(obj))
	if err != nil {
		log.Infof("Failed storing resource \"%s\": %v", urn, err)
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
		log.Infof("Failed storing resource \"%s\": %v", urn, err)
		return err
	}

	return nil
}

// Get any data from a k/v store
func GetFromStore(urn string) ([]byte, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		log.Fatal(err)
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
		log.Infof("Failed getting resource \"%s\": %v", urn, err)
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
		log.Fatal(err)
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

// Get the most recently created entry under a prefix
func getLastFromStoreWithPrefix(prefix string) (*clientv3.GetResponse, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	// Replace all dots with slashes to match your storage scheme
	prefix = strings.ReplaceAll(prefix, ".", "/")

	// Query etcd: sort by creation revision, descending, limit to 1
	resp, err := client.Get(
		context.Background(),
		prefix,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortDescend),
		clientv3.WithLimit(1),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get last entry for prefix %s: %v", prefix, err)
	}

	return resp, nil
}

func getLinks(prefix string) []*topology.Link {
	var links []*topology.Link

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		log.Errorf("Failed getting links from store: %v", err)
		return links
	}

	for _, rawLink := range rawData.Kvs {
		link := &topology.Link{}

		if err = proto.Unmarshal([]byte(rawLink.Value), link); err != nil {
			log.Errorf("Failed unmarshaling link: %v", err)
			return links
		}
		links = append(links, link)
	}
	return links
}

func getNodes(prefix string) []*topology.Node {
	var nodes []*topology.Node

	rawData, err := getFromStoreWithPrefix(prefix)
	if err != nil {
		log.Errorf("Failed getting nodes from store: %v", err)
		return nodes
	}

	for _, rawNode := range rawData.Kvs {
		node := &topology.Node{}

		if err = proto.Unmarshal([]byte(rawNode.Value), node); err != nil {
			log.Errorf("Failed unmarshaling node: %v", err)
			return nodes
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// WatchKeys watches for changes on keys with a given prefix and calls the callback on new keys
func WatchKeys(prefix string, callback func(outerKey, innerKey, value string), client *clientv3.Client) {
	watcher := clientv3.NewWatcher(client)
	defer watcher.Close()

	// Watch for changes to keys with the provided prefix
	watchChan := watcher.Watch(context.Background(), prefix, clientv3.WithPrefix())

	// Handle incoming watch events
	for resp := range watchChan {
		for _, ev := range resp.Events {
			// Process only EventTypePut (added or updated keys)
			if ev.Type == clientv3.EventTypePut {
				// Split the key into outer and inner parts (this assumes keys are in a "outer/inner" format)
				keyParts := splitKey(string(ev.Kv.Key))
				outerKey := keyParts[0]
				innerKey := keyParts[1]

				// Call the callback function when a new key is added
				callback(outerKey, innerKey, string(ev.Kv.Value))
			}
		}
	}
}

// Helper function to split the key (assuming it's in "outer/inner" format)
func splitKey(key string) []string {
	parts := strings.Split(key, "/") // Split the key on '/'
	if len(parts) < 2 {
		return []string{key} // Return the original key if it does not contain '/'
	}
	return parts
}
