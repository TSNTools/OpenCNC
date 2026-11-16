// Staged ADD_STREAM native projection for the approved CNC pilot.
// Only supported response fields are mapped; all other proposed changes fail
// closed rather than being silently discarded. No deployment/storage write.
package internalOptimizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
)

type nativeAdditionEntry struct {
	Definition    json.RawMessage `json:"definition"`
	Configuration json.RawMessage `json:"configuration"`
}

// applyStagedStreamAdditions RECONCILES entries already staged in the preserved
// native FPM. It never appends a new stream; that would duplicate staged data.
// The shared validator must previously have verified change=add, full
// definition equivalence, scope, request authorization, and route topology.
func applyStagedStreamAdditions(
	model *forwardingplane.ForwardingPlaneModel, raw json.RawMessage,
) error {
	var additions []nativeAdditionEntry
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&additions); err != nil {
		return fmt.Errorf("decode stream additions: %w", err)
	}
	if len(additions) == 0 {
		return fmt.Errorf("streamAdditions must contain at least one staged stream")
	}

	originalByKey := make(map[nativeStreamKey]*forwardingplane.StreamModel)
	for index, entry := range model.GetStreams() {
		if entry == nil || entry.GetDefinition() == nil ||
			entry.GetDefinition().GetStreamId() == nil {
			return fmt.Errorf("staged original streams[%d] lacks composite StreamId", index)
		}
		id := entry.GetDefinition().GetStreamId()
		if id.GetMacAddress() == "" {
			return fmt.Errorf("staged original streams[%d] has empty MAC address", index)
		}
		key := nativeStreamKey{MacAddress: id.GetMacAddress(), UniqueID: id.GetUniqueId()}
		if _, duplicate := originalByKey[key]; duplicate {
			return fmt.Errorf("duplicate staged original stream %s/%d", key.MacAddress, key.UniqueID)
		}
		originalByKey[key] = entry
	}

	routes := make([]nativeRouteUpdate, 0, len(additions))
	pcpValues := make(map[nativeStreamKey]uint32)
	seen := make(map[nativeStreamKey]bool)
	for index, addition := range additions {
		if len(addition.Definition) == 0 || len(addition.Configuration) == 0 {
			return fmt.Errorf("stream addition %d requires full definition and configuration", index)
		}
		var definition map[string]json.RawMessage
		if err := json.Unmarshal(addition.Definition, &definition); err != nil || definition == nil {
			return fmt.Errorf("stream addition %d has invalid definition: %v", index, err)
		}
		rawID, hasID := definition["streamId"]
		if !hasID {
			return fmt.Errorf("stream addition %d lacks definition.streamId", index)
		}
		var id nativeRouteID
		if err := json.Unmarshal(rawID, &id); err != nil ||
			id.UniqueID == nil || id.MacAddress == "" {
			return fmt.Errorf("stream addition %d has invalid composite StreamId", index)
		}
		key := nativeStreamKey{MacAddress: id.MacAddress, UniqueID: *id.UniqueID}
		if seen[key] {
			return fmt.Errorf("duplicate stream addition for %s/%d", key.MacAddress, key.UniqueID)
		}
		seen[key] = true
		staged, found := originalByKey[key]
		if !found {
			return fmt.Errorf("stream addition %s/%d was not staged in original FPM", key.MacAddress, key.UniqueID)
		}

		// This native pilot supports only physical unicast endpoints.
		// Application placement and additional OaaS-only definition fields
		// remain valid in the generic contract, but have no approved CNC mapping.
		if _, present := definition["talkerApplicationId"]; present {
			return fmt.Errorf("stream addition %s/%d has unsupported logical talker placement", key.MacAddress, key.UniqueID)
		}
		if _, present := definition["listenerApplicationIds"]; present {
			return fmt.Errorf("stream addition %s/%d has unsupported logical listener placement", key.MacAddress, key.UniqueID)
		}
		if _, present := definition["intraClassPriority"]; present {
			return fmt.Errorf("stream addition %s/%d has unsupported intra-class priority mapping", key.MacAddress, key.UniqueID)
		}
		if _, present := definition["talkerNodeId"]; !present {
			return fmt.Errorf("stream addition %s/%d lacks physical talker endpoint", key.MacAddress, key.UniqueID)
		}
		if _, present := definition["listenerNodeIds"]; !present || len(staged.GetDefinition().GetListenerNodeIds()) != 1 {
			return fmt.Errorf("stream addition %s/%d requires one physical listener", key.MacAddress, key.UniqueID)
		}

		// These identity/endpoint checks complement (not replace) the shared
		// contract's full-definition comparison with the stored request.
		var stringID string
		if err := json.Unmarshal(definition["id"], &stringID); err != nil ||
			stringID != requestadapter.StreamKey(key.MacAddress, key.UniqueID) {
			return fmt.Errorf("stream addition %s/%d has mismatched OaaS string ID", key.MacAddress, key.UniqueID)
		}
		if field, present := definition["talkerNodeId"]; present {
			var node string
			if err := json.Unmarshal(field, &node); err != nil || node != staged.GetDefinition().GetTalkerNodeId() {
				return fmt.Errorf("stream addition %s/%d would alter talker endpoint", key.MacAddress, key.UniqueID)
			}
		}
		if field, present := definition["listenerNodeIds"]; present {
			var nodes []string
			if err := json.Unmarshal(field, &nodes); err != nil ||
				!reflect.DeepEqual(nodes, staged.GetDefinition().GetListenerNodeIds()) {
				return fmt.Errorf("stream addition %s/%d would alter listener endpoints", key.MacAddress, key.UniqueID)
			}
		}
		if field, present := definition["trafficClassId"]; present {
			var trafficClass string
			if err := json.Unmarshal(field, &trafficClass); err != nil ||
				trafficClass != staged.GetDefinition().GetTrafficClassId() {
				return fmt.Errorf("stream addition %s/%d would alter traffic class", key.MacAddress, key.UniqueID)
			}
		}

		var cfg map[string]json.RawMessage
		if err := json.Unmarshal(addition.Configuration, &cfg); err != nil || cfg == nil {
			return fmt.Errorf("stream addition %s/%d has invalid configuration", key.MacAddress, key.UniqueID)
		}
		// The response contains a complete StreamModel. This native pilot
		// applies the generated path and the request-authorized PCP. The PCP
		// can be absent in the preserved original because CNC's UNI overlay
		// sets it on a REQUEST-ONLY copy; do not lose that requested priority.
		// Other native/OaaS changes fail closed until explicitly supported.
		for field, value := range cfg {
			switch field {
			case "path":
				// Parsed below.
			case "pcp":
				var pcp uint32
				if err := json.Unmarshal(value, &pcp); err != nil || pcp > 7 {
					return fmt.Errorf("stream addition %s/%d has invalid PCP", key.MacAddress, key.UniqueID)
				}
				var current *uint32
				if staged.GetConfiguration() != nil {
					current = staged.GetConfiguration().Pcp
				}
				if current != nil && *current != pcp {
					return fmt.Errorf("stream addition %s/%d conflicts with preserved PCP", key.MacAddress, key.UniqueID)
				}
				pcpValues[key] = pcp
			case "enabled":
				var enabled bool
				if err := json.Unmarshal(value, &enabled); err != nil ||
					enabled != staged.GetConfiguration().GetEnabled() {
					return fmt.Errorf("stream addition %s/%d contains unsupported enabled change", key.MacAddress, key.UniqueID)
				}
			case "queueAssignments":
				var queues []json.RawMessage
				if err := json.Unmarshal(value, &queues); err != nil ||
					len(queues) != 0 || len(staged.GetConfiguration().GetQueueAssignments()) != 0 {
					return fmt.Errorf("stream addition %s/%d contains unsupported queue assignment", key.MacAddress, key.UniqueID)
				}
			case "streamId":
				var configID nativeRouteID
				if err := json.Unmarshal(value, &configID); err != nil ||
					configID.UniqueID == nil || configID.MacAddress != key.MacAddress ||
					*configID.UniqueID != key.UniqueID {
					return fmt.Errorf("stream addition %s/%d has conflicting configuration StreamId", key.MacAddress, key.UniqueID)
				}
			default:
				return fmt.Errorf("stream addition %s/%d has unsupported configuration field %q",
					key.MacAddress, key.UniqueID, field)
			}
		}
		if field, exists := cfg["path"]; !exists {
			return fmt.Errorf("stream addition %s/%d lacks generated path", key.MacAddress, key.UniqueID)
		} else {
			var hops []nativeRouteHop
			if err := json.Unmarshal(field, &hops); err != nil || len(hops) == 0 {
				return fmt.Errorf("stream addition %s/%d requires nonempty path", key.MacAddress, key.UniqueID)
			}
			routes = append(routes, nativeRouteUpdate{
				StreamID:      &nativeRouteID{MacAddress: key.MacAddress, UniqueID: id.UniqueID},
				Configuration: &nativeRouteConfiguration{Path: hops},
			})
		}
	}

	serializedRoutes, err := json.Marshal(routes)
	if err != nil {
		return fmt.Errorf("marshal staged routes: %w", err)
	}
	// Use the existing, tested native path translator. This mutates only the
	// caller's cloned FPM, and enforces missing-path and hop constraints.
	if err := applyGeneratedStreamPaths(model, serializedRoutes); err != nil {
		return err
	}
	for key, pcp := range pcpValues {
		item := originalByKey[key]
		if item.Configuration == nil {
			return fmt.Errorf("stream addition %s/%d missing configuration after route application", key.MacAddress, key.UniqueID)
		}
		priority := pcp
		item.Configuration.Pcp = &priority
	}
	return nil
}
