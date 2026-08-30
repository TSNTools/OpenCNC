package storewrapper

import (
	"OpenCNC/common/structures/topology"
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
		return err
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

func nodeStorePrefix(node *topology.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("node cannot be nil")
	}
	if strings.TrimSpace(node.GetName()) == "" {
		return "", fmt.Errorf("node name cannot be empty")
	}

	if props := node.GetProperties(); props != nil {
		if props != nil {
			return "bridges", nil
		}
		if props != nil {
			return "endnodes", nil
		}
	}

	switch node.GetType() {
	case topology.NodeRole_BRIDGE:
		return "bridges", nil
	case topology.NodeRole_END_STATION,
		topology.NodeRole_BRIDGED_END_STATION,
		topology.NodeRole_UNKNOWN:
		return "endnodes", nil
	default:
		return "", fmt.Errorf("unsupported node role: %v", node.GetType())
	}
}

func getNode(client *clientv3.Client, name string) (*topology.Node, string, error) {
	for _, prefix := range []string{"endnodes", "bridges"} {
		key := strings.ReplaceAll(prefix+"."+name, ".", "/")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Get(ctx, key)
		cancel()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get node %s from store: %w", name, err)
		}
		if len(resp.Kvs) == 0 {
			continue
		}

		node := &topology.Node{}
		if err := proto.Unmarshal(resp.Kvs[0].Value, node); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal node %s: %w", name, err)
		}

		return node, prefix, nil
	}

	return nil, "", fmt.Errorf("node %s not found", name)
}

func storeNodeWithClient(client *clientv3.Client, node *topology.Node, requireExisting bool) error {
	prefix, err := nodeStorePrefix(node)
	if err != nil {
		return err
	}

	obj, err := proto.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node object: %w", err)
	}

	if _, existingPrefix, err := getNode(client, node.GetName()); err == nil {
		if existingPrefix != prefix {
			oldKey := strings.ReplaceAll(existingPrefix+"."+node.GetName(), ".", "/")
			if _, err := client.Delete(context.Background(), oldKey); err != nil {
				return fmt.Errorf("failed to delete previous node entry %s: %w", node.GetName(), err)
			}
		}
	} else {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
		if requireExisting {
			return err
		}
	}

	return sendToStoreRepeated(client, obj, prefix+"."+node.GetName())
}

func StoreNode(node *topology.Node) error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	return storeNodeWithClient(client, node, false)
}

func EditNode(node *topology.Node) error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	return storeNodeWithClient(client, node, true)
}

func StoreLink(link *topology.Link) error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	return storeLinkWithClient(client, link, false)
}

func EditLink(link *topology.Link) error {
	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	return storeLinkWithClient(client, link, true)
}

func storeLinkWithClient(client *clientv3.Client, link *topology.Link, requireExisting bool) error {
	if link == nil {
		return fmt.Errorf("link cannot be nil")
	}
	if strings.TrimSpace(link.GetId()) == "" {
		return fmt.Errorf("link ID cannot be empty")
	}

	key := "links/" + link.GetId()

	if requireExisting {
		resp, err := client.Get(context.Background(), key)
		if err != nil {
			return fmt.Errorf("failed to get link %s from store: %w", link.GetId(), err)
		}
		if len(resp.Kvs) == 0 {
			return fmt.Errorf("link %s not found", link.GetId())
		}
	}

	obj, err := proto.Marshal(link)
	if err != nil {
		return fmt.Errorf("failed to marshal link object: %w", err)
	}

	return sendToStoreRepeated(client, obj, key)
}

func getLink(client *clientv3.Client, name string) (*topology.Link, string, error) {
	key := strings.ReplaceAll("links."+name, ".", "/")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Get(ctx, key)
	cancel()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get link %s from store: %w", name, err)
	}
	if len(resp.Kvs) == 0 {
		return nil, "", fmt.Errorf("link %s not found", name)
	}

	link := &topology.Link{}
	if err := proto.Unmarshal(resp.Kvs[0].Value, link); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal link %s: %w", name, err)
	}

	return link, "links", nil
}

func storeNodes(nodes []*topology.Node) error {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
	}
	defer client.Close()

	for _, node := range nodes {
		if err := storeNodeWithClient(client, node, false); err != nil {
			return err
		}
	}

	return nil
}

func storeLinks(links []*topology.Link) error {
	client, err := createEtcdClient()
	if err != nil {
		return err
	}
	defer client.Close()

	for _, link := range links {
		if link == nil {
			continue
		}

		// Generate an ID if the imported link doesn't have one.
		if strings.TrimSpace(link.GetId()) == "" {
			link.Id = fmt.Sprintf(
				"%s-%s",
				link.GetSourceNode(),
				link.GetDestinationNode(),
			)
		}

		obj, err := proto.Marshal(link)
		if err != nil {
			return fmt.Errorf(
				"failed to marshal link %s: %w",
				link.GetId(),
				err,
			)
		}

		urn := "links." + link.GetId()

		if err := sendToStoreRepeated(client, obj, urn); err != nil {
			return fmt.Errorf(
				"failed to store link %s: %w",
				link.GetId(),
				err,
			)
		}
	}

	return nil
}

// Get raw data from k/v store
func getRawDataFromStore(urn string) ([]byte, error) {
	// Connect to ETCD
	client, err := createEtcdClient()
	if err != nil {
		//log.Fatal(err)
		return nil, err
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
