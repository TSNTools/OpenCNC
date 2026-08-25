package storewrapper

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"tsn-service/pkg/structures/topology"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
)

// CreateEtcdClient creates and returns an etcd client
func createEtcdClient() (*clientv3.Client, error) {
	endpoints := []string{"http://etcd.opencnc.svc.cluster.local:2379"}
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
