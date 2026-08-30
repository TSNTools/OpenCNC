package main

import (
	"bytes"
	"fmt"
	"net/http"
)

//storewrapper "OpenCNC/common/store-wrapper"

func main() {

	//testUNI()

	/*
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
	*/
}

/*
func testUniAddStream() {
	conn, err := grpc.Dial(
		"localhost:9000",
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to UNI gRPC server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to UNI gRPC server")

	client := uni_grpc.NewUniServiceClient(conn)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	req := &uni_grpc.AddStreamRequest{
		Id:                   "test-stream-1",
		Name:                 "Test Stream",
		TalkerNodeId:         "D",
		ListenerNodeIds:      []string{"F", "G"},
		TrafficType:          "isochronous",
		Rank:                 "1",
		MaxFrameSize:         500,
		MaxLatencyNs:         300000,
		MaxJitterNs:          0,
		IntervalNs:           100000,
		MaxFramesPerInterval: 1,
		MinTransmitOffsetNs:  0,
		MaxTransmitOffsetNs:  50000,
		NumSeamlessTrees:     1,
	}

	resp, err := client.AddStream(ctx, req)
	if err != nil {
		log.Fatalf("AddStream failed: %v", err)
	}

	fmt.Println("Response:")
	fmt.Printf("  Success: %v\n", resp.Success)
	fmt.Printf("  Message: %s\n", resp.Message)
	fmt.Printf("  Configuration ID: %s\n", resp.ConfigurationId)
}
*/

func testUNI() {
	url := "http://localhost:8080/add_stream"

	body := []byte(`{
		"version": 1,
		"requests": []
	}`)

	fmt.Println("Sending request to UNI server...")
	fmt.Println("URL:", url)

	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		fmt.Println("UNI request failed:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("UNI server returned:", resp.Status)
}
