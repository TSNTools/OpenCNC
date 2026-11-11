package internalOptimizer

import (
	"bytes"
	"encoding/json"
	"fmt"

	"OpenCNC/common/structures/qav"
	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

// applyValidatedOptimizerResponse maps a validated standard OptimizerResponse
// onto the PRESERVED native ForwardingPlaneModel. The mapping is driven by
// solution.changes fields, NOT by optimizer ID or algorithm type.
//
// Phase 1 supports only configuration.nodeConfigs[].portConfigs[].queueConfigs[]
// scheduling and CBS fields. Future mechanism mappings should extend the
// supported change paths here, rather than adding algorithm-specific mergers.
// Unsupported paths fail closed. No state is deployed or written to storage.
// The caller must supply a result that passed the shared v0.2.0 validator.
func applyValidatedOptimizerResponse(
	validated *validatedOptimizationResult,
) (*forwardingplane.ForwardingPlaneModel, error) {
	if validated == nil || validated.Original == nil || validated.Validation == nil {
		return nil, fmt.Errorf("response application requires a validated result and preserved original model")
	}
	if validated.Original.GetTopology() == nil || validated.Original.GetConfiguration() == nil {
		return nil, fmt.Errorf("response application requires preserved topology and configuration")
	}

	// Read the generic result envelope without imposing an algorithm-specific
	// response type. Reject changes outside the supported Phase 1 paths below.
	var result struct {
		Status   string `json:"status"`
		Solution struct {
			Changes json.RawMessage `json:"changes"`
		} `json:"solution"`
	}
	if err := json.Unmarshal(validated.OptimizerResult, &result); err != nil {
		return nil, fmt.Errorf("decode validated optimizer response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("cannot apply unsuccessful optimizer result")
	}

	clone, ok := proto.Clone(validated.Original).(*forwardingplane.ForwardingPlaneModel)
	if !ok || clone == nil {
		return nil, fmt.Errorf("clone original forwarding-plane model failed")
	}

	var changes map[string]json.RawMessage
	if len(result.Solution.Changes) == 0 ||
		bytes.Equal(bytes.TrimSpace(result.Solution.Changes), []byte("null")) {
		return nil, fmt.Errorf("optimizer response has no changes object")
	}
	if err := json.Unmarshal(result.Solution.Changes, &changes); err != nil || changes == nil {
		return nil, fmt.Errorf("invalid optimizer changes object: %v", err)
	}
	if len(changes) == 0 {
		if validated.Validation.Applicable {
			return nil, fmt.Errorf("response marked applicable but contains no changes")
		}
		return clone, nil // Validated no-op; no changes to the preserved FPM.
	}
	if !validated.Validation.Applicable {
		return nil, fmt.Errorf("nonempty optimizer changes were marked inapplicable")
	}

	for path := range changes {
		if path != "configuration" {
			return nil, fmt.Errorf("unsupported response change path %q", path)
		}
	}
	if bytes.Equal(bytes.TrimSpace(changes["configuration"]), []byte("null")) {
		return nil, fmt.Errorf("configuration change must be an object")
	}

	// DisallowUnknownFields applies recursively to nested node/port/queue/CBS
	// changes. This prevents silently discarding a valid schema field that
	// the Phase 1 native adapter cannot yet project into OpenCNC.
	var configuration struct {
		NodeConfigs []responseNodeDelta `json:"nodeConfigs"`
	}
	decoder := json.NewDecoder(bytes.NewReader(changes["configuration"]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return nil, fmt.Errorf("unsupported or invalid configuration change: %w", err)
	}
	if len(configuration.NodeConfigs) == 0 {
		return nil, fmt.Errorf("applicable response contains no node updates")
	}

	physicalPorts := make(map[string]map[string]int32)
	for _, node := range clone.GetTopology().GetNodes() {
		if node == nil || node.GetName() == "" {
			return nil, fmt.Errorf("invalid physical topology node")
		}
		if _, duplicate := physicalPorts[node.GetName()]; duplicate {
			return nil, fmt.Errorf("duplicate physical topology node %q", node.GetName())
		}
		ports := make(map[string]int32)
		for _, port := range node.GetPorts() {
			if port == nil || port.GetId() == "" {
				return nil, fmt.Errorf("invalid physical topology port on %s", node.GetName())
			}
			if _, duplicate := ports[port.GetId()]; duplicate {
				return nil, fmt.Errorf("duplicate physical topology port %s/%s", node.GetName(), port.GetId())
			}
			ports[port.GetId()] = port.GetNumberOfQueues()
		}
		physicalPorts[node.GetName()] = ports
	}

	seenNodes := make(map[string]bool)
	for _, nodeDelta := range configuration.NodeConfigs {
		if seenNodes[nodeDelta.NodeID] {
			return nil, fmt.Errorf("duplicate configuration update node %s", nodeDelta.NodeID)
		}
		seenNodes[nodeDelta.NodeID] = true
		portCounts, knownNode := physicalPorts[nodeDelta.NodeID]
		if !knownNode {
			return nil, fmt.Errorf("configuration update references unknown physical node %s", nodeDelta.NodeID)
		}
		node, err := findOrAppendNodeConfig(clone.Configuration, nodeDelta.NodeID)
		if err != nil {
			return nil, err
		}
		if len(nodeDelta.PortConfigs) == 0 {
			return nil, fmt.Errorf("configuration update node %s has no port updates", nodeDelta.NodeID)
		}
		seenPorts := make(map[string]bool)
		for _, portDelta := range nodeDelta.PortConfigs {
			if seenPorts[portDelta.PortID] {
				return nil, fmt.Errorf("duplicate configuration update port %s/%s", nodeDelta.NodeID, portDelta.PortID)
			}
			seenPorts[portDelta.PortID] = true
			queueCount, knownPort := portCounts[portDelta.PortID]
			if !knownPort {
				return nil, fmt.Errorf("configuration update references unknown physical port %s/%s", nodeDelta.NodeID, portDelta.PortID)
			}
			port, err := findOrAppendPortConfig(node, portDelta.PortID)
			if err != nil {
				return nil, err
			}
			if len(portDelta.QueueConfigs) == 0 {
				return nil, fmt.Errorf("configuration update port %s/%s has no queue updates", nodeDelta.NodeID, portDelta.PortID)
			}
			seenQueues := make(map[uint32]bool)
			for _, queueDelta := range portDelta.QueueConfigs {
				if seenQueues[queueDelta.QueueID] {
					return nil, fmt.Errorf("duplicate queue update %s/%s/%d", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
				}
				seenQueues[queueDelta.QueueID] = true
				if queueCount <= 0 || uint64(queueDelta.QueueID) >= uint64(queueCount) {
					return nil, fmt.Errorf("queue %s/%s/%d is outside the %d physical queues", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID, queueCount)
				}
				if len(queueDelta.CBS) == 0 {
					return nil, fmt.Errorf("queue update %s/%s/%d has no CBS records", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
				}
				queue, newlyAdded, err := findOrAppendQueueConfig(port, queueDelta.QueueID)
				if err != nil {
					return nil, err
				}
				if queueDelta.Scheduling != nil {
					if *queueDelta.Scheduling != "CREDIT_BASED_SHAPER" {
						return nil, fmt.Errorf("unsupported non-CBS scheduling change for %s/%s/%d", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
					}
					queue.Scheduling = topology_config.SchedulingModel_CREDIT_BASED_SHAPER.Enum()
				} else if newlyAdded || queue.GetScheduling() != topology_config.SchedulingModel_CREDIT_BASED_SHAPER {
					return nil, fmt.Errorf("queue %s/%s/%d must explicitly select CREDIT_BASED_SHAPER", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
				}
				seenCBS := make(map[uint32]bool)
				for _, delta := range queueDelta.CBS {
					if seenCBS[delta.EgressQueueID] {
						return nil, fmt.Errorf("duplicate CBS record for %s/%s/%d", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
					}
					seenCBS[delta.EgressQueueID] = true
					if delta.EgressQueueID != queueDelta.QueueID {
						return nil, fmt.Errorf("CBS egressQueueId does not match parent queueId for %s/%s/%d", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
					}
					cbs, isNew, err := findOrAppendCBSConfig(queue, delta.EgressQueueID)
					if err != nil {
						return nil, err
					}
					if isNew && (delta.AdminState == nil || delta.AdminIdleSlopeBps == nil ||
						delta.SendSlopeBps == nil || delta.HiCreditBytes == nil || delta.LoCreditBytes == nil) {
						return nil, fmt.Errorf("new CBS record %s/%s/%d requires complete parameters", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
					}
					if delta.AdminState != nil {
						switch *delta.AdminState {
						case "ENABLED":
							cbs.AdminState = qav.AdminState_ENABLED
						case "DISABLED":
							cbs.AdminState = qav.AdminState_DISABLED
						default:
							return nil, fmt.Errorf("invalid CBS adminState for %s/%s/%d", nodeDelta.NodeID, portDelta.PortID, queueDelta.QueueID)
						}
					}
					if delta.AdminIdleSlopeBps != nil {
						cbs.AdminIdleSlopeBps = *delta.AdminIdleSlopeBps
					}
					if delta.SendSlopeBps != nil {
						cbs.SendSlopeBps = *delta.SendSlopeBps
					}
					if delta.HiCreditBytes != nil {
						cbs.HiCreditBytes = *delta.HiCreditBytes
					}
					if delta.LoCreditBytes != nil {
						cbs.LoCreditBytes = *delta.LoCreditBytes
					}
					if delta.Description != nil {
						cbs.Description = *delta.Description
					}
				}
			}
		}
	}
	if proto.Equal(clone, validated.Original) {
		return nil, fmt.Errorf("response update did not change the preserved forwarding-plane model")
	}
	return clone, nil
}

type responseNodeDelta struct {
	NodeID      string              `json:"nodeId"`
	PortConfigs []responsePortDelta `json:"portConfigs"`
}
type responsePortDelta struct {
	PortID       string               `json:"portId"`
	QueueConfigs []responseQueueDelta `json:"queueConfigs"`
}
type responseQueueDelta struct {
	QueueID    uint32                   `json:"queueId"`
	Scheduling *string                  `json:"scheduling"`
	CBS        []responseCBSRecordDelta `json:"cbs"`
}
type responseCBSRecordDelta struct {
	EgressQueueID     uint32  `json:"egressQueueId"`
	AdminState        *string `json:"adminState"`
	AdminIdleSlopeBps *uint64 `json:"adminIdleSlopeBps"`
	SendSlopeBps      *uint64 `json:"sendSlopeBps"`
	HiCreditBytes     *uint64 `json:"hiCreditBytes"`
	LoCreditBytes     *uint64 `json:"loCreditBytes"`
	Description       *string `json:"description"`
}

func findOrAppendNodeConfig(config *topology_config.TopologyConfig, nodeID string) (*topology_config.NodeConfig, error) {
	var found *topology_config.NodeConfig
	for _, item := range config.GetNodeConfigs() {
		if item == nil {
			return nil, fmt.Errorf("nil original node configuration")
		}
		if item.GetNodeId() == nodeID {
			if found != nil {
				return nil, fmt.Errorf("duplicate original node configuration %s", nodeID)
			}
			found = item
		}
	}
	if found == nil {
		found = &topology_config.NodeConfig{NodeId: nodeID}
		config.NodeConfigs = append(config.NodeConfigs, found)
	}
	return found, nil
}
func findOrAppendPortConfig(node *topology_config.NodeConfig, portID string) (*topology_config.PortConfig, error) {
	var found *topology_config.PortConfig
	for _, item := range node.GetPortConfigs() {
		if item == nil {
			return nil, fmt.Errorf("nil original port configuration")
		}
		if item.GetPortId() == portID {
			if found != nil {
				return nil, fmt.Errorf("duplicate original port configuration %s", portID)
			}
			found = item
		}
	}
	if found == nil {
		found = &topology_config.PortConfig{PortId: portID}
		node.PortConfigs = append(node.PortConfigs, found)
	}
	return found, nil
}
func findOrAppendQueueConfig(port *topology_config.PortConfig, queueID uint32) (*topology_config.QueueConfig, bool, error) {
	var found *topology_config.QueueConfig
	for _, item := range port.GetQueueConfigs() {
		if item == nil {
			return nil, false, fmt.Errorf("nil original queue configuration")
		}
		if item.GetQueueId() == queueID {
			if found != nil {
				return nil, false, fmt.Errorf("duplicate original queue configuration %d", queueID)
			}
			found = item
		}
	}
	if found != nil {
		return found, false, nil
	}
	found = &topology_config.QueueConfig{QueueId: queueID}
	port.QueueConfigs = append(port.QueueConfigs, found)
	return found, true, nil
}
func findOrAppendCBSConfig(queue *topology_config.QueueConfig, queueID uint32) (*qav.CbsQueueConfig, bool, error) {
	var found *qav.CbsQueueConfig
	for _, item := range queue.GetCbs() {
		if item == nil {
			return nil, false, fmt.Errorf("nil original CBS configuration")
		}
		if item.GetEgressQueueId() == queueID {
			if found != nil {
				return nil, false, fmt.Errorf("duplicate original CBS configuration %d", queueID)
			}
			found = item
		}
	}
	if found != nil {
		return found, false, nil
	}
	found = &qav.CbsQueueConfig{EgressQueueId: queueID}
	queue.Cbs = append(queue.Cbs, found)
	return found, true, nil
}
