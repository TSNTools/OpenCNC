package forwarding_plane

import (
	"fmt"

	pbstream "OpenCNC/common/structures/stream"
	pbstreamconfig "OpenCNC/common/structures/stream_config"
	pbtopology "OpenCNC/common/structures/topology"
	pbconfig "OpenCNC/common/structures/topology_config"

	"google.golang.org/protobuf/proto"
)

// -----------------------------------------------------------------------------
// Construction
// -----------------------------------------------------------------------------

func NewForwardingPlaneModel() *ForwardingPlaneModel {
	return &ForwardingPlaneModel{
		Metadata:      &ModelMetadata{},
		Topology:      &pbtopology.Topology{},
		Configuration: &pbconfig.TopologyConfig{},
		Streams:       []*StreamModel{},
	}
}

func (m *ForwardingPlaneModel) Clone() *ForwardingPlaneModel {
	if m == nil {
		return nil
	}

	return proto.Clone(m).(*ForwardingPlaneModel)
}

// -----------------------------------------------------------------------------
// Topology - navigation
// -----------------------------------------------------------------------------

func (m *ForwardingPlaneModel) Node(id string) *pbtopology.Node {
	if m == nil || m.Topology == nil {
		return nil
	}

	for _, n := range m.Topology.Nodes {
		if n.GetName() == id {
			return n
		}
	}

	return nil
}

func (m *ForwardingPlaneModel) Port(nodeID, portID string) *pbtopology.Port {
	node := m.Node(nodeID)
	if node == nil {
		return nil
	}

	for _, p := range node.Ports {
		if p.GetId() == portID {
			return p
		}
	}

	return nil
}

func (m *ForwardingPlaneModel) Link(id string) *pbtopology.Link {
	if m == nil || m.Topology == nil {
		return nil
	}

	for _, l := range m.Topology.Links {
		if l.GetId() == id {
			return l
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Topology - mutation
// -----------------------------------------------------------------------------

func (m *ForwardingPlaneModel) AddNode(node *pbtopology.Node) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	if node.GetName() == "" {
		return fmt.Errorf("node name is empty")
	}

	if m.Node(node.GetName()) != nil {
		return fmt.Errorf("node %q already exists", node.GetName())
	}

	m.ensureTopology()
	m.Topology.Nodes = append(m.Topology.Nodes, node)

	return nil
}

func (m *ForwardingPlaneModel) RemoveNode(id string) error {
	if m.Topology == nil {
		return fmt.Errorf("topology is nil")
	}

	for i, n := range m.Topology.Nodes {
		if n.GetName() == id {
			m.Topology.Nodes = append(
				m.Topology.Nodes[:i],
				m.Topology.Nodes[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf("node %q not found", id)
}

func (m *ForwardingPlaneModel) AddPort(
	nodeID string,
	port *pbtopology.Port,
) error {
	if port == nil {
		return fmt.Errorf("port is nil")
	}

	node := m.Node(nodeID)
	if node == nil {
		return fmt.Errorf("node %q not found", nodeID)
	}

	if port.GetId() == "" {
		return fmt.Errorf("port id is empty")
	}

	if m.Port(nodeID, port.GetId()) != nil {
		return fmt.Errorf(
			"port %q already exists on node %q",
			port.GetId(),
			nodeID,
		)
	}

	node.Ports = append(node.Ports, port)

	return nil
}

func (m *ForwardingPlaneModel) RemovePort(
	nodeID, portID string,
) error {
	node := m.Node(nodeID)
	if node == nil {
		return fmt.Errorf("node %q not found", nodeID)
	}

	for i, p := range node.Ports {
		if p.GetId() == portID {
			node.Ports = append(
				node.Ports[:i],
				node.Ports[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf(
		"port %q on node %q not found",
		portID,
		nodeID,
	)
}

func (m *ForwardingPlaneModel) AddLink(
	link *pbtopology.Link,
) error {
	if link == nil {
		return fmt.Errorf("link is nil")
	}

	if link.GetId() == "" {
		return fmt.Errorf("link id is empty")
	}

	if m.Link(link.GetId()) != nil {
		return fmt.Errorf("link %q already exists", link.GetId())
	}

	m.ensureTopology()
	m.Topology.Links = append(m.Topology.Links, link)

	return nil
}

func (m *ForwardingPlaneModel) RemoveLink(id string) error {
	if m.Topology == nil {
		return fmt.Errorf("topology is nil")
	}

	for i, l := range m.Topology.Links {
		if l.GetId() == id {
			m.Topology.Links = append(
				m.Topology.Links[:i],
				m.Topology.Links[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf("link %q not found", id)
}

// -----------------------------------------------------------------------------
// Configuration - navigation
// -----------------------------------------------------------------------------

func (m *ForwardingPlaneModel) NodeConfig(
	nodeID string,
) *pbconfig.NodeConfig {
	if m == nil || m.Configuration == nil {
		return nil
	}

	for _, c := range m.Configuration.NodeConfigs {
		if c.GetNodeId() == nodeID {
			return c
		}
	}

	return nil
}

func (m *ForwardingPlaneModel) PortConfig(
	nodeID, portID string,
) *pbconfig.PortConfig {
	node := m.NodeConfig(nodeID)
	if node == nil {
		return nil
	}

	for _, c := range node.PortConfigs {
		if c.GetPortId() == portID {
			return c
		}
	}

	return nil
}

func (m *ForwardingPlaneModel) LinkConfig(
	linkID string,
) *pbconfig.LinkConfig {
	if m == nil || m.Configuration == nil {
		return nil
	}

	for _, c := range m.Configuration.LinkConfigs {
		if c.GetLinkId() == linkID {
			return c
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Configuration - mutation
// -----------------------------------------------------------------------------

func (m *ForwardingPlaneModel) SetNodeConfig(
	config *pbconfig.NodeConfig,
) error {
	if config == nil {
		return fmt.Errorf("node config is nil")
	}

	m.ensureConfiguration()

	for i, existing := range m.Configuration.NodeConfigs {
		if existing.GetNodeId() == config.GetNodeId() {
			m.Configuration.NodeConfigs[i] = config
			return nil
		}
	}

	m.Configuration.NodeConfigs =
		append(m.Configuration.NodeConfigs, config)

	return nil
}

func (m *ForwardingPlaneModel) RemoveNodeConfig(
	nodeID string,
) error {
	if m.Configuration == nil {
		return fmt.Errorf("configuration is nil")
	}

	for i, c := range m.Configuration.NodeConfigs {
		if c.GetNodeId() == nodeID {
			m.Configuration.NodeConfigs = append(
				m.Configuration.NodeConfigs[:i],
				m.Configuration.NodeConfigs[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf("node config %q not found", nodeID)
}

func (m *ForwardingPlaneModel) SetPortConfig(
	nodeID string,
	config *pbconfig.PortConfig,
) error {
	if config == nil {
		return fmt.Errorf("port config is nil")
	}

	node := m.NodeConfig(nodeID)
	if node == nil {
		node = &pbconfig.NodeConfig{
			NodeId: nodeID,
		}

		if err := m.SetNodeConfig(node); err != nil {
			return err
		}
	}

	for i, existing := range node.PortConfigs {
		if existing.GetPortId() == config.GetPortId() {
			node.PortConfigs[i] = config
			return nil
		}
	}

	node.PortConfigs = append(node.PortConfigs, config)

	return nil
}

func (m *ForwardingPlaneModel) RemovePortConfig(
	nodeID, portID string,
) error {
	node := m.NodeConfig(nodeID)
	if node == nil {
		return fmt.Errorf("node config %q not found", nodeID)
	}

	for i, c := range node.PortConfigs {
		if c.GetPortId() == portID {
			node.PortConfigs = append(
				node.PortConfigs[:i],
				node.PortConfigs[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf(
		"port config %q on node %q not found",
		portID,
		nodeID,
	)
}

func (m *ForwardingPlaneModel) SetLinkConfig(
	config *pbconfig.LinkConfig,
) error {
	if config == nil {
		return fmt.Errorf("link config is nil")
	}

	m.ensureConfiguration()

	for i, existing := range m.Configuration.LinkConfigs {
		if existing.GetLinkId() == config.GetLinkId() {
			m.Configuration.LinkConfigs[i] = config
			return nil
		}
	}

	m.Configuration.LinkConfigs =
		append(m.Configuration.LinkConfigs, config)

	return nil
}

func (m *ForwardingPlaneModel) RemoveLinkConfig(
	linkID string,
) error {
	if m.Configuration == nil {
		return fmt.Errorf("configuration is nil")
	}

	for i, c := range m.Configuration.LinkConfigs {
		if c.GetLinkId() == linkID {
			m.Configuration.LinkConfigs = append(
				m.Configuration.LinkConfigs[:i],
				m.Configuration.LinkConfigs[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf("link config %q not found", linkID)
}

// -----------------------------------------------------------------------------
// Streams
// -----------------------------------------------------------------------------

func streamIDEqual(a, b *pbstream.StreamId) bool {
	if a == nil || b == nil {
		return false
	}

	return proto.Equal(a, b)
}

func (m *ForwardingPlaneModel) Stream(
	id *pbstream.StreamId,
) *StreamModel {
	if m == nil || id == nil {
		return nil
	}

	for _, s := range m.Streams {
		if s == nil || s.GetDefinition() == nil {
			continue
		}

		if streamIDEqual(s.GetDefinition().GetStreamId(), id) {
			return s
		}
	}

	return nil
}

func (m *ForwardingPlaneModel) StreamConfiguration(
	id *pbstream.StreamId,
) *pbstreamconfig.StreamConfiguration {
	s := m.Stream(id)
	if s == nil {
		return nil
	}

	return s.GetConfiguration()
}

func (m *ForwardingPlaneModel) AddStream(
	s *StreamModel,
) error {
	if s == nil {
		return fmt.Errorf("stream is nil")
	}

	if s.GetDefinition() == nil {
		return fmt.Errorf("stream definition is nil")
	}

	id := s.GetDefinition().GetStreamId()
	if id == nil {
		return fmt.Errorf("stream ID is nil")
	}

	if m.Stream(id) != nil {
		return fmt.Errorf("stream already exists")
	}

	m.Streams = append(m.Streams, s)

	return nil
}

func (m *ForwardingPlaneModel) RemoveStream(
	id *pbstream.StreamId,
) error {
	if id == nil {
		return fmt.Errorf("stream ID is nil")
	}

	for i, s := range m.Streams {
		if s == nil || s.GetDefinition() == nil {
			continue
		}

		if streamIDEqual(s.GetDefinition().GetStreamId(), id) {
			m.Streams = append(
				m.Streams[:i],
				m.Streams[i+1:]...,
			)
			return nil
		}
	}

	return fmt.Errorf("stream not found")
}

func (m *ForwardingPlaneModel) SetStreamConfiguration(
	id *pbstream.StreamId,
	config *pbstreamconfig.StreamConfiguration,
) error {
	if id == nil {
		return fmt.Errorf("stream ID is nil")
	}

	if config == nil {
		return fmt.Errorf("stream configuration is nil")
	}

	s := m.Stream(id)
	if s == nil {
		return fmt.Errorf("stream not found")
	}

	s.Configuration = config

	return nil
}

func (m *ForwardingPlaneModel) RemoveStreamConfiguration(
	id *pbstream.StreamId,
) error {
	if id == nil {
		return fmt.Errorf("stream ID is nil")
	}

	s := m.Stream(id)
	if s == nil {
		return fmt.Errorf("stream not found")
	}

	s.Configuration = nil

	return nil
}

// -----------------------------------------------------------------------------
// Internal initialization
// -----------------------------------------------------------------------------

func (m *ForwardingPlaneModel) ensureTopology() {
	if m.Topology == nil {
		m.Topology = &pbtopology.Topology{}
	}
}

func (m *ForwardingPlaneModel) ensureConfiguration() {
	if m.Configuration == nil {
		m.Configuration = &pbconfig.TopologyConfig{}
	}
}
