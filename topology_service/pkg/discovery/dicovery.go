package discovery

import (
	"context"
	"fmt"

	"OpenCNC/common/managementSessions"
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
)

type Discovery struct{}

func NewDiscovery() *Discovery {
	return &Discovery{}
}

func (d *Discovery) Discover(ctx context.Context, node *topology.Node, creds *credentials.ManagementCredentials) (*topology.Topology, error) {

	if node == nil {
		return nil, fmt.Errorf("node is required")
	}

	if creds == nil {
		return nil, fmt.Errorf("credentials are required")
	}

	factory := managementSessions.NewNetconfFactory(node, creds)

	session, err := factory.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create NETCONF session: %w", err)
	}
	defer session.Close()

	neighbors, err := GetLLDPNeighbors(session)
	if err != nil {
		return nil, fmt.Errorf("discover LLDP: %w", err)
	}

	result := &topology.Topology{
		Nodes: []*topology.Node{node},
	}

	for _, neighbor := range neighbors {
		addNeighbor(result, neighbor)
	}

	return result, nil
}

func addNeighbor(
	t *topology.Topology,
	neighbor LLDPNeighbor,
) {
	// TODO: map LLDP data into topology.Node + topology.Port + topology.Link
}
