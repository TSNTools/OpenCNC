package internalOptimizer

import (
	"fmt"

	pcpmodel "OpenCNC/common/structures/pcp"
	pcpdefaults "OpenCNC/tsn_service/pkg/resource_allocation_entity/PCP"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

// withDefaultPCPQueueMappings supplies a request-only default where the
// ForwardingPlaneModel has no PCP-to-queue mapping for a configured port.
//
// TEMPORARY CBS PILOT — Hamza, 15 November 2026: The PCP-to-queue mapping
// may be configured in the FPM or optimized. Until an authoritative mapping
// is available, check the FPM first; if missing, use OpenCNC's default.
// Here we use the existing eight-priority (8P0D) identity table ONLY for
// ports advertising eight queues. This is an explicit optimization-input
// assumption, NOT a claim about the physical switch's operational state.
// Never override an existing table or explicitly disabled PCP mapping.
// The source FPM and its snapshot remain unchanged.

// Important limitation: The current helper only fills mappings for ports that already have a port-configuration entry. It does not create missing port-configuration entries.
func withDefaultPCPQueueMappings(
	original *forwardingplane.ForwardingPlaneModel,
) (*forwardingplane.ForwardingPlaneModel, error) {
	if original == nil {
		return nil, fmt.Errorf("forwarding plane is nil")
	}
	if original.GetTopology() == nil || original.GetConfiguration() == nil {
		return original, nil // Cannot locate ports and their existing configuration.
	}

	// Map topology port IDs to physical queue counts; never invent a queue
	// count from a stream's PCP or from the device-model description.
	type portKey struct{ nodeID, portID string }
	queueCounts := make(map[portKey]int32)
	for _, node := range original.GetTopology().GetNodes() {
		if node == nil {
			continue
		}
		for _, port := range node.GetPorts() {
			if port == nil {
				continue
			}
			queueCounts[portKey{node.GetName(), port.GetId()}] = port.GetNumberOfQueues()
		}
	}

	// Clone before inserting any defaults. No runtime CNC configuration or
	// preserved FPM is changed by this request-adaptation step.
	requestFPM := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)
	for _, node := range requestFPM.GetConfiguration().GetNodeConfigs() {
		if node == nil {
			continue
		}
		for _, port := range node.GetPortConfigs() {
			if port == nil {
				continue
			}

			// An explicit disable or an existing mapping is authoritative.
			if port.PcpMappingEnabled != nil && !port.GetPcpMappingEnabled() {
				continue
			}
			if current := port.GetPcpConfig(); current != nil && (len(current.GetTrafficClassTable()) > 0 ||
				len(current.GetEncodingTables()) > 0 ||
				len(current.GetDecodingTables()) > 0 ||
				current.GetPriorityRegeneration() != nil) {
				continue
			}
			if queueCounts[portKey{node.GetNodeId(), port.GetPortId()}] != 8 {
				continue // Do not assume the eight-queue default on other hardware.
			}

			// Use only the priority-to-queue table. The full default-PCP helper
			// also builds encoding/decoding tables not handled by v1.4.
			table, err := pcpdefaults.NewDefaultTrafficClassTable()
			if err != nil {
				return nil, fmt.Errorf("default PCP table for %s/%s: %w",
					node.GetNodeId(), port.GetPortId(), err)
			}
			port.PcpConfig = &pcpmodel.PcpPortConfig{TrafficClassTable: table}
			port.PcpMappingEnabled = proto.Bool(true)
		}
	}
	return requestFPM, nil
}
