package main

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	//storewrapper "OpenCNC_config_service/common/store-wrapper"
	"OpenCNC_config_service/common/structures/topology"
)

func main() {

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prefixes := []string{
		"bridges/",
		"endnodes/",
	}

	for _, prefix := range prefixes {
		resp, err := client.Get(
			ctx,
			prefix,
			clientv3.WithPrefix(),
		)
		if err != nil {
			log.Fatalf("failed to get nodes from %s: %v", prefix, err)
		}

		fmt.Printf("\n========================================\n")
		fmt.Printf("  %s\n", prefix)
		fmt.Printf("========================================\n")

		if len(resp.Kvs) == 0 {
			fmt.Println("No nodes found.")
			continue
		}

		for _, kv := range resp.Kvs {
			node := &topology.Node{}

			if err := proto.Unmarshal(kv.Value, node); err != nil {
				log.Printf(
					"failed to unmarshal node %s: %v",
					string(kv.Key),
					err,
				)
				continue
			}

			jsonBytes, err := protojson.MarshalOptions{
				Multiline:       true,
				Indent:          "  ",
				EmitUnpopulated: true,
			}.Marshal(node)

			if err != nil {
				log.Printf(
					"failed to convert node %s to JSON: %v",
					node.GetName(),
					err,
				)
				continue
			}

			fmt.Printf("\nETCD key: %s\n", string(kv.Key))
			fmt.Println("----------------------------------------")
			fmt.Println(string(jsonBytes))
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Done.")
	fmt.Println("========================================")
}
