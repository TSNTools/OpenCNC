// Extracts optimization-relevant information from the CNC forwarding plane, including topology, streams, configuration and available capabilities.
package requestadapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/encoding/protojson"
)

// Maximum exact integer for typical JSON consumers (js/ts).
const maxSafeJSONInteger uint64 = 9007199254740991

// StreamKey derives the required OaaS stream definition.id from OpenCNC's
// composite StreamId. All request adapter cross-references must use this same function.
func StreamKey(macAddress string, uniqueID uint32) string {
	return strings.ToLower(macAddress) + "/" + strconv.FormatUint(uint64(uniqueID), 10)
}

// ExtractForwardingPlane converts OpenCNC's ForwardingPlaneModel
// into the subset supported by OaaS Contract v1.4.
//
// The original CNC model is preserved separately by the caller
// before submitting the optimization request.
//
// CNC-specific fields not supported by the contract are excluded.
// This function does not modify the original protobuf.
func ExtractForwardingPlane(
	model *forwardingplane.ForwardingPlaneModel,
	schemaJSON []byte,
) (map[string]any, error) {
	if model == nil || model.GetTopology() == nil {
		return nil, errors.New("forwarding plane and topology are required")
	}
	if err := rejectUnrepresentableState(model); err != nil {
		return nil, err
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("parse v1.4 JSON schema: %w", err)
	}
	rootProperties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("schema has no root properties")
	}
	versionProperty, _ := rootProperties["schemaVersion"].(map[string]any)
	if versionProperty["const"] != "1.4" {
		return nil, errors.New("expected optimizer request schemaVersion 1.4")
	}
	forwardSpec, ok := rootProperties["forwardingPlane"].(map[string]any)
	if !ok {
		return nil, errors.New("schema has no forwardingPlane property")
	}
	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		return nil, errors.New("schema has no $defs")
	}

	// ProtoJSON honors field names/enum spellings from OpenCNC and emits uint64
	// values as decimal strings; the recursive projection below converts those
	// numbers to JSON integers according to the OaaS schema.
	data, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(model)
	if err != nil {
		return nil, fmt.Errorf("marshal OpenCNC forwarding plane: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var source map[string]any
	if err := decoder.Decode(&source); err != nil {
		return nil, err
	}

	// OpenCNC identifies streams by (MAC, uniqueId). The OaaS v1.4 contract
	// also requires a stable string id. Derive it rather than guessing f1/f2.
	rawStreams, ok := source["streams"].([]any)
	if !ok || len(rawStreams) != len(model.GetStreams()) {
		return nil, errors.New("unexpected protobuf streams representation")
	}
	seen := make(map[string]bool, len(rawStreams))
	for i, item := range model.GetStreams() {
		if item == nil || item.GetDefinition() == nil ||
			item.GetDefinition().GetStreamId() == nil {
			return nil, fmt.Errorf("streams[%d]: missing stream definition or streamId", i)
		}
		streamID := item.GetDefinition().GetStreamId()
		if streamID.GetMacAddress() == "" {
			return nil, fmt.Errorf("streams[%d]: empty stream MAC address", i)
		}
		key := StreamKey(streamID.GetMacAddress(), streamID.GetUniqueId())
		if seen[key] {
			return nil, fmt.Errorf("duplicate OpenCNC stream identity: %s", key)
		}
		seen[key] = true
		streamObj, ok := rawStreams[i].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("streams[%d]: expected object", i)
		}
		definition, ok := streamObj["definition"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("streams[%d]: missing definition object", i)
		}
		definition["id"] = key

		// OpenCNC main defines StreamRank directly as an enum.
		// Convert its value to the string expected by Contract v1.4.
		rankName := item.GetDefinition().GetStreamRank().String()

		switch rankName {
		case "RANK_EMERGENCY", "RANK_NON_EMERGENCY":
			definition["streamRank"] = rankName
		default:
			return nil, fmt.Errorf(
				"streams[%d]: unsupported OpenCNC streamRank %q",
				i, rankName,
			)
		}
	}

	projected, err := projectToContract(source, forwardSpec, defs, "forwardingPlane")
	if err != nil {
		return nil, err
	}
	// This handles if a future schema change produces an unexpected structure.
	result, ok := projected.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("projected forwarding plane is not an object")
	}

	return result, nil
}

// ExtractPortCapabilities converts the *known* per-port mechanism flags
// from OpenCNC into v1.4 top-level capabilities.devices[].ports.
// Hardware-specific limits (e.g. supportedQueues, maxGclEntries) MUST be
// obtained from device inventory/Config Service and merged separately.
// False flags are carried as reported by proto3, but proto3 cannot tell an
// absent flag from an explicitly false flag.
func ExtractPortCapabilities(model *forwardingplane.ForwardingPlaneModel) map[string]any {
	devices := make([]any, 0)
	if model == nil || model.GetTopology() == nil {
		return map[string]any{"devices": devices}
	}
	for _, node := range model.GetTopology().GetNodes() {
		if node == nil {
			continue
		}
		ports := make(map[string]any)
		for _, port := range node.GetPorts() {
			if port == nil || port.GetCapabilities() == nil {
				continue
			}
			c := port.GetCapabilities()
			// End stations may perform FRER replication/elimination too. Keep
			// their advertised TSN capabilities when any flag is enabled.
			if node.GetType().String() == "END_STATION" &&
				!c.GetSupportsFrer() && !c.GetSupportsTas() && !c.GetSupportsCbs() {
				continue
			}
			ports[port.GetId()] = map[string]any{
				"mechanisms": map[string]any{
					"CBS":  map[string]any{"supported": c.GetSupportsCbs()},
					"TAS":  map[string]any{"supported": c.GetSupportsTas(), "framePreemptionSupported": c.GetSupportsFramePreemption()},
					"FRER": map[string]any{"supported": c.GetSupportsFrer()},
				},
			}
		}
		if len(ports) != 0 {
			devices = append(devices, map[string]any{
				"nodeId": node.GetName(),
				"ports":  ports,
			})
		}
	}
	return map[string]any{"devices": devices}
}

// Reject CNC state that would make the simplified optimization view unsafe.
// This is deliberately conservative: filtering inactive topology links,
// STP paths and advanced VLAN/PCP state needs a dedicated verified adapter.
func rejectUnrepresentableState(model *forwardingplane.ForwardingPlaneModel) error {
	configuration := model.GetConfiguration()
	if configuration == nil {
		return nil
	}
	for _, link := range configuration.GetLinkConfigs() {
		if link != nil && link.GetStateOverride().String() == "FORCE_DOWN" {
			return fmt.Errorf("link %s: disabled link must be excluded from effective topology", link.GetLinkId())
		}
	}
	for _, node := range configuration.GetNodeConfigs() {
		if node == nil {
			continue
		}
		if bridge := node.GetBridge(); bridge != nil &&
			(bridge.GetStpConfiguration() != nil || (bridge.StpEnabled != nil && bridge.GetStpEnabled())) {
			return fmt.Errorf("node %s: enabled STP/configuration needs effective-topology translation", node.GetNodeId())
		}
		for _, port := range node.GetPortConfigs() {
			if port == nil {
				continue
			}
			portName := node.GetNodeId() + "/" + port.GetPortId()
			if port.GetAdminState().String() == "ADMIN_STATE_DISABLED" ||
				(port.IngressEnabled != nil && !port.GetIngressEnabled()) ||
				(port.EgressEnabled != nil && !port.GetEgressEnabled()) {
				return fmt.Errorf("port %s: disabled port must be excluded from effective topology", portName)
			}
			if port.GetStp() != nil || len(port.GetVlanMemberships()) > 0 || port.GetVlanAdvanced() != nil {
				return fmt.Errorf("port %s: STP/VLAN state needs topology-feasibility translation", portName)
			}
			if pcp := port.GetPcpConfig(); pcp != nil &&
				(len(pcp.GetEncodingTables()) > 0 || len(pcp.GetDecodingTables()) > 0 || pcp.GetPriorityRegeneration() != nil) {
				return fmt.Errorf("port %s: advanced PCP mappings are not represented by v1.4", portName)
			}
			if gcl := port.GetGcl(); gcl != nil {
				for i, entry := range gcl.GetEntries() {
					if entry == nil || entry.GetIndex() != uint32(i) {
						return fmt.Errorf("port %s: GCL entries must be ordered by contiguous index (entry %d)", portName, i)
					}
				}
			}
		}
	}
	return nil
}

// projectToContract recursively follows the actual v1.4 schema. It strips
// CNC-only properties even at deeply nested levels and preserves any fields
// which the schema explicitly permits. It is NOT a replacement for validation.
func projectToContract(value any, spec map[string]any, defs map[string]any, path string) (any, error) {
	if value == nil {
		return nil, nil
	}
	if ref, ok := spec["$ref"].(string); ok {
		const prefix = "#/$defs/"
		if !strings.HasPrefix(ref, prefix) {
			return nil, fmt.Errorf("%s: unsupported schema reference %q", path, ref)
		}
		definition, ok := defs[strings.TrimPrefix(ref, prefix)].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: missing schema definition %q", path, ref)
		}
		return projectToContract(value, definition, defs, path)
	}

	schemaType, _ := spec["type"].(string)
	if schemaType == "" {
		if _, ok := spec["properties"]; ok {
			schemaType = "object"
		} else if _, ok := spec["items"]; ok {
			schemaType = "array"
		} else {
			return value, nil
		}
	}

	switch schemaType {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: expected object", path)
		}
		props, _ := spec["properties"].(map[string]any)
		result := make(map[string]any)
		for key, child := range obj {
			if child == nil {
				continue
			}
			if p, ok := props[key].(map[string]any); ok {
				v, err := projectToContract(child, p, defs, path+"."+key)
				if err != nil {
					return nil, err
				}
				result[key] = v
				continue
			}
			// If unknown properties are permitted, keep them. Otherwise strip
			// CNC-only fields (including original revision, management state,
			// and obsolete v1.2 fields).
			switch additional := spec["additionalProperties"].(type) {
			case bool:
				if additional {
					result[key] = child
				}
			case map[string]any:
				v, err := projectToContract(child, additional, defs, path+"."+key)
				if err != nil {
					return nil, err
				}
				result[key] = v
			}
		}
		if required, ok := spec["required"].([]any); ok {
			for _, item := range required {
				key, _ := item.(string)
				if _, present := result[key]; !present {
					return nil, fmt.Errorf("%s: missing required %s", path, key)
				}
			}
		}
		return result, nil
	case "array":
		items, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("%s: expected array", path)
		}
		itemSpec, _ := spec["items"].(map[string]any)
		out := make([]any, 0, len(items))
		for i, item := range items {
			if itemSpec == nil {
				out = append(out, item)
				continue
			}
			v, err := projectToContract(item, itemSpec, defs, fmt.Sprintf("%s.%d", path, i))
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case "integer":
		return safeJSONInteger(value, path)
	case "number":
		if number, ok := value.(json.Number); ok {
			f, err := number.Float64()
			if err != nil {
				return nil, fmt.Errorf("%s: invalid number: %w", path, err)
			}
			return f, nil
		}
		return value, nil
	default:
		return value, nil
	}
}

// JSON Schema defines many OaaS nanosecond/rate fields as JSON integers while
// ProtoJSON encodes protobuf uint64/int64 as decimal strings. Reject values
// beyond 2^53-1 instead of silently losing precision in TypeScript consumers.
func safeJSONInteger(value any, path string) (any, error) {
	var s string
	switch n := value.(type) {
	case string:
		s = n
	case json.Number:
		s = n.String()
	default:
		return nil, fmt.Errorf("%s: expected integer, got %T", path, value)
	}
	if strings.ContainsAny(s, ".eE") {
		return nil, fmt.Errorf("%s: expected an integer, got %q", path, s)
	}
	if strings.HasPrefix(s, "-") {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < -int64(maxSafeJSONInteger) {
			return nil, fmt.Errorf("%s: integer outside safe JSON range: %s", path, s)
		}
		return n, nil
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n > maxSafeJSONInteger {
		return nil, fmt.Errorf("%s: integer outside safe JSON range: %s", path, s)
	}
	return n, nil
}
