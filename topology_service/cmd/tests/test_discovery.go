package tests

import (
	"context"
	"fmt"
	"log"

	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	"OpenCNC/topology_service/pkg/discovery"
)

func Test() {
	node := &topology.Node{
		Name: "device-1",
		ManagementInfo: &topology.ManagementInfo{
			IpAddress:      "192.168.0.1",
			ManagementPort: 830,
			Protocol:       topology.ManagementProtocol_NETCONF,
			Transport:      topology.ManagementTransport_SSH,
		},
	}

	creds := &credentials.ManagementCredentials{
		Authentication: &credentials.ManagementCredentials_UsernamePassword{
			UsernamePassword: &credentials.UsernamePassword{
				Username: "root",
				Password: "",
			},
		},
	}

	discovery := discovery.NewDiscovery()

	result, err := discovery.Discover(
		context.Background(),
		node,
		creds,
	)
	if err != nil {
		log.Fatalf("discovery failed: %v", err)
	}

	fmt.Printf("Topology:\n")
	fmt.Printf("Nodes: %d\n", len(result.GetNodes()))
	fmt.Printf("Links: %d\n", len(result.GetLinks()))

	for _, n := range result.GetNodes() {
		fmt.Printf("\nNode: %s\n", n.GetName())

		for _, p := range n.GetPorts() {
			fmt.Printf(
				"  Port: id=%s name=%s mac=%s\n",
				p.GetId(),
				p.GetName(),
				p.GetMacAddress(),
			)
		}
	}

	for _, l := range result.GetLinks() {
		fmt.Printf(
			"\nLink: %s:%s <-> %s:%s\n",
			l.GetSourceNode(),
			l.GetSourcePort(),
			l.GetDestinationNode(),
			l.GetDestinationPort(),
		)
	}
}
