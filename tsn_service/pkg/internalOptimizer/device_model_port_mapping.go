package internalOptimizer

import (
	"fmt"

	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
)

// Unprefixed bridge-port path derived from the scheduling YANG modules.
// The registered DeviceModel XPath convention still needs confirmation.
const bridgeAdminGCLPath = "/interfaces/interface/bridge-port/gate-parameter-table/admin-control-list/gate-control-entry"

// mapTASGCLLimitsToPorts prepares additional request capabilities without
// changing the FPM or inventing capability flags. This first mapping covers
// ordinary bridges with a common device-model limit and TAS-capable ports.
func mapTASGCLLimitsToPorts(
	fpm *forwarding_plane.ForwardingPlaneModel,
	limitsByNode map[string][]tasGCLLimit,
) (map[string]any, error) {
	if fpm == nil || fpm.GetTopology() == nil {
		return nil, fmt.Errorf("forwarding-plane topology is missing")
	}
	devices := make([]any, 0)
	seen := make(map[string]bool)
	for _, node := range fpm.GetTopology().GetNodes() {
		if node == nil {
			return nil, fmt.Errorf("topology contains a nil node")
		}
		nodeID := node.GetName()
		if nodeID == "" || seen[nodeID] {
			return nil, fmt.Errorf("missing or duplicate node ID %q", nodeID)
		}
		seen[nodeID] = true
		limits := limitsByNode[nodeID]
		if len(limits) == 0 || node.GetType().String() != "BRIDGE" {
			continue
		}
		var maxEntries uint32
		for _, limit := range limits {
			// Do not guess the meaning of a different or port-specific XPath.
			if limit.TargetPath != bridgeAdminGCLPath {
				return nil, fmt.Errorf("node %q: unconfirmed GCL limitation path %q", nodeID, limit.TargetPath)
			}
			if limit.MaxEntries == 0 {
				return nil, fmt.Errorf("node %q: invalid zero GCL limit", nodeID)
			}
			if maxEntries != 0 && maxEntries != limit.MaxEntries {
				return nil, fmt.Errorf("node %q: conflicting GCL limitations", nodeID)
			}
			maxEntries = limit.MaxEntries
		}
		ports := make(map[string]any)
		for _, port := range node.GetPorts() {
			if port == nil || !port.GetCapabilities().GetSupportsTas() {
				continue
			}
			// A distinct port-level DeviceModel requires a separately verified mapping.
			if model := port.GetDeviceModel(); model != "" && model != node.GetDeviceInfo().GetDeviceModel() {
				return nil, fmt.Errorf("node %q port %q: separate port DeviceModel is not supported by this mapping", nodeID, port.GetId())
			}
			id := port.GetId()
			if _, exists := ports[id]; id == "" || exists {
				return nil, fmt.Errorf("node %q: missing or duplicate TAS port ID %q", nodeID, id)
			}
			ports[id] = map[string]any{
				"mechanisms": map[string]any{
					"TAS": map[string]any{"maxGclEntries": maxEntries},
				},
			}
		}
		if len(ports) > 0 {
			devices = append(devices, map[string]any{"nodeId": nodeID, "ports": ports})
		}
	}
	for nodeID := range limitsByNode {
		if !seen[nodeID] {
			return nil, fmt.Errorf("GCL limitations reference unknown node %q", nodeID)
		}
	}
	return map[string]any{"devices": devices}, nil
}
