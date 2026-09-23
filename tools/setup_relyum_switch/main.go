package main

import (
	"context"
	"fmt"
	"log"
	"os"

	storewrapper "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/topology"
	"OpenCNC/tsn_service/pkg/internalOptimizer"

	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	ctx := context.Background()

	fmt.Println("==================================================================")
	fmt.Println(" Setting up Relyum Switch (192.168.4.64) in OpenCNC Store")
	fmt.Println("==================================================================")

	// 1. Load Device Model RELY-TSN4
	modelFile := "common/deviceModels/rely-tsn4_device_model.json"
	rawModel, err := os.ReadFile(modelFile)
	if err != nil {
		log.Fatalf("Failed to read device model file %s: %v", modelFile, err)
	}

	model := &devicemodelregistry.DeviceModel{}
	if err := jsonpb.UnmarshalString(string(rawModel), model); err != nil {
		log.Fatalf("Failed to unmarshal device model: %v", err)
	}

	if err := storewrapper.StoreDeviceModel(model); err != nil {
		log.Fatalf("Failed to store device model %s: %v", model.Name, err)
	}
	fmt.Printf("[✓] Stored Device Model: %s\n", model.Name)

	// 2. Store credentials under the same name used in the topology.
	rawCredentials, err := os.ReadFile("relyum_switch_credentials.json")
	if err != nil {
		log.Fatalf("Failed to read credentials: %v", err)
	}
	creds := &credentials.ManagementCredentials{}
	if err := protojson.Unmarshal(rawCredentials, creds); err != nil {
		log.Fatalf("Failed to parse credentials: %v", err)
	}
	switchNodeName := "SWITCH"
	if err := storewrapper.StoreCredentials(switchNodeName, creds); err != nil {
		log.Fatalf("Failed to store credentials for %s: %v", switchNodeName, err)
	}
	fmt.Printf("[✓] Stored NETCONF Credentials for: %s (user: sys-admin)\n", switchNodeName)

	// 3. Store topology containing SWITCH.
	topoFile := "relyum_switch_topology.json"
	rawTopo, err := os.ReadFile(topoFile)
	if err != nil {
		log.Fatalf("Failed to read topology file %s: %v", topoFile, err)
	}

	topo := &topology.Topology{}
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshaler.Unmarshal(rawTopo, topo); err != nil {
		log.Fatalf("Failed to unmarshal topology: %v", err)
	}

	if err := storewrapper.StoreTopology(topo); err != nil {
		log.Fatalf("Failed to store topology: %v", err)
	}
	fmt.Printf("[✓] Stored Topology with %d node(s): %s at %s:%d (%s/%s)\n",
		len(topo.Nodes),
		topo.Nodes[0].Name,
		topo.Nodes[0].ManagementInfo.IpAddress,
		topo.Nodes[0].ManagementInfo.ManagementPort,
		topo.Nodes[0].ManagementInfo.Protocol.String(),
		topo.Nodes[0].ManagementInfo.Transport.String(),
	)

	// 4. Regenerate the default schedule so SWITCH has a NodeConfig.
	if err := internalOptimizer.CreateDefaultSchedule(ctx, nil); err != nil {
		log.Fatalf("Failed to create default schedule: %v", err)
	}
	fmt.Println("[✓] Generated default schedule for Relyum-switch with Gate 7 open (1000ms cycle)")

	fmt.Println("==================================================================")
	fmt.Println(" Setup Complete! config-service is now configured to reach")
	fmt.Println(" Relyum switch at 192.168.4.64 via NETCONF/SSH.")
	fmt.Println("==================================================================")
}
