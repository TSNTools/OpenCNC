package stub

import (
	storewrapper "OpenCNC/common/store-wrapper"
	streamproto "OpenCNC/common/structures/stream"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/uni"
	"OpenCNC/gui_service/internal/domain"
	"bytes"
	"fmt"
	"math"
	"net"
	"strings"
)

func buildListenerGroups(streamId *streamproto.StreamId, stream domain.Stream, interfaces []*uni.Interface) []*uni.ListenerGroup {

	listeners := make([]*uni.ListenerGroup, 0, len(interfaces))

	for i, iface := range interfaces {
		listeners = append(listeners, &uni.ListenerGroup{
			Index: uint32(i),

			StrId: streamId,

			EndStationInterfaces: []*uni.Interface{
				iface,
			},

			UserToNetReq: &uni.UserToNetworkRequirements{
				NumSeamlessTrees: uint32(stream.NumSeamlessTrees),
				MaxLatency:       uint32(stream.MaxLatencyNs),
			},
		})
	}

	return listeners
}

func buildTrafficSpecification(stream domain.Stream) *uni.TrafficSpecification {
	traffic := &uni.TrafficSpecification{
		MaxFramesPerInterval: uint32(stream.MaxFramesPerInterval),
		MaxFrameSize:         uint32(stream.MaxFrameSize),
	}

	if stream.MinTransmitOffsetNs != 0 ||
		stream.MaxTransmitOffsetNs != 0 ||
		stream.MaxJitterNs != 0 {

		traffic.TimeAware = &uni.TimeAware{
			EarliestTransmitOffset: uint32(stream.MinTransmitOffsetNs),
			LatestTransmitOffset:   uint32(stream.MaxTransmitOffsetNs),
			Jitter:                 uint32(stream.MaxJitterNs),
		}
	}

	return traffic
}

func generateTSNStreamID(talkerMAC string) (*streamproto.StreamId, error) {
	streams, err := storewrapper.GetStreams()
	if err != nil {
		return nil, err
	}

	mac, err := net.ParseMAC(talkerMAC)
	if err != nil || len(mac) != 6 {
		return nil, fmt.Errorf("invalid talker MAC: %s", talkerMAC)
	}

	used := make(map[uint16]bool)

	for _, s := range streams {
		streamMAC, uid, err := parseTSNStreamID(s.StreamId.String())
		if err != nil {
			continue
		}

		if bytes.Equal(streamMAC, mac) {
			used[uid] = true
		}
	}

	for uid := uint32(1); uid <= math.MaxUint16; uid++ {
		if !used[uint16(uid)] {
			return &streamproto.StreamId{
				MacAddress: formatMAC(mac),
				UniqueId:   uint32(uid),
			}, nil
		}
	}

	return nil, fmt.Errorf("no available TSN stream ID for talker %s", talkerMAC)
}

func parseTSNStreamID(id string) (net.HardwareAddr, uint16, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("invalid StreamID: %s", id)
	}

	mac, err := net.ParseMAC(parts[0])
	if err != nil || len(mac) != 6 {
		return nil, 0, fmt.Errorf("invalid StreamID MAC: %s", id)
	}

	var uid uint16
	if _, err := fmt.Sscanf(parts[1], "%x", &uid); err != nil {
		return nil, 0, err
	}

	return mac, uid, nil
}

func formatMAC(mac net.HardwareAddr) string {
	return strings.ToUpper(mac.String())
}

func findNode(nodes []*topology.Node, nodeName string) (*topology.Node, error) {
	for _, node := range nodes {
		if node != nil && node.Name == nodeName {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node %q not found in topology", nodeName)
}

func nodeToInterface(node *topology.Node) (*uni.Interface, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	if len(node.Ports) == 0 {
		return nil, fmt.Errorf(
			"node %q has no ports",
			node.Name,
		)
	}

	// For now, use the first port.
	// If a node can have multiple ports, we should
	// select the correct one based on your topology.
	port := node.Ports[0]

	return &uni.Interface{
		Index: 0,
		InterfaceId: &uni.InterfaceId{
			MacAddress:    port.MacAddress,
			InterfaceName: port.Name,
		},
	}, nil
}

func (b *Backend) resolveEndStations(stream domain.Stream) (*uni.Interface, []*uni.Interface, error) {
	// Get topology nodes.
	nodes, err := storewrapper.GetNodes()
	if err != nil {
		return nil, nil, err
	}

	// Resolve talker node.
	talkerNode, err := findNode(nodes, stream.TalkerNodeID)
	if err != nil {
		return nil, nil, err
	}

	talkerNode.Ports[0].MacAddress = "00:00:00:00:00:00"

	talkerInterface, err := nodeToInterface(talkerNode)
	if err != nil {
		return nil, nil, err
	}

	// Build listener interfaces.
	listenerInterfaces := make([]*uni.Interface, 0, len(stream.ListenerNodeIDs))

	for _, listenerNodeID := range stream.ListenerNodeIDs {
		listenerNode, err := findNode(nodes, listenerNodeID)
		if err != nil {
			return nil, nil, err
		}

		listenerNode.Ports[0].MacAddress = "00:00:00:00:00:01"

		listenerInterface, err := nodeToInterface(listenerNode)
		if err != nil {
			return nil, nil, err
		}

		listenerInterfaces = append(listenerInterfaces, listenerInterface)
	}

	return talkerInterface, listenerInterfaces, nil
}
