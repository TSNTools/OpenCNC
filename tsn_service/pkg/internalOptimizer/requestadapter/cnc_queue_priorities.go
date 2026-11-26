package requestadapter

import (
	"fmt"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
)

// ApplyCNCQueuePriorities enriches the projected request only. Hamza's
// November 2026 CNC model interpretation orders TC7 highest through TC0
// lowest. Limit this pilot policy to existing configurations of CBS-capable
// eight-queue ports; it is not evidence of physical scheduling behavior.
// PCP mappings are independent and are never changed by this helper.
// The caller supplies the already projected map, not the native protobuf.
func ApplyCNCQueuePriorities(forwarding map[string]any, model *forwardingplane.ForwardingPlaneModel) error {
	type portKey struct{ node, port string }
	eligible := make(map[portKey]bool)
	for _, node := range model.GetTopology().GetNodes() {
		for _, port := range node.GetPorts() {
			if port.GetNumberOfQueues() == 8 && port.GetCapabilities().GetSupportsCbs() {
				eligible[portKey{node.GetName(), port.GetId()}] = true
			}
		}
	}
	config, _ := forwarding["configuration"].(map[string]any)
	nodes, _ := config["nodeConfigs"].([]any)
	for _, item := range nodes {
		node, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid projected node configuration")
		}
		nodeID, _ := node["nodeId"].(string)
		ports, _ := node["portConfigs"].([]any)
		for _, item := range ports {
			port, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid projected port configuration")
			}
			portID, _ := port["portId"].(string)
			if !eligible[portKey{nodeID, portID}] {
				continue
			}
			if _, exists := port["queuePriorityTable"]; exists {
				return fmt.Errorf("unexpected existing priority table at %s/%s", nodeID, portID)
			}
			table := make([]any, 0, 8)
			for queue := 0; queue < 8; queue++ {
				table = append(table, map[string]any{
					"queueId": queue, "schedulingPriority": 8 - queue,
				})
			}
			port["queuePriorityTable"] = table
		}
	}
	return nil
}
