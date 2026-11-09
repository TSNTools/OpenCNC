// assembles the complete Optimizer Request JSON v1.4.
package requestadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	tsncontracts "github.com/Kashaf14/tsn-optimization-contracts"
)

// NOTE: The original ForwardingPlaneModel must be preserved by the caller
// before submitting the request.
// This function performs local consistency checks.
// Full JSON Schema validation is handled by the OaaS Manager.
// Network feasibility is checked separately.

// assembles entire request contract JSON
// optimization_task.go ────┐
//                          │
// forwarding_plane.go ─────┤
//                          ├── optimizer_request.go
// RequestOptions ──────────┤          ↓
//                          │    Complete JSON v1.4
// schema information ──────┘

func BuildOptimizerRequest(
	task *optimizationcontract.OptimizationTask,
	model *forwardingplane.ForwardingPlaneModel,
	schemaJSON []byte,
	options RequestOptions,
) ([]byte, error) {
	if task == nil || model == nil {
		return nil, errors.New("optimization task and forwarding-plane model are required")
	}
	operation, err := operationFromIntent(task.GetIntent())
	if err != nil {
		return nil, err
	}
	if operation == "reconfiguration" && task.GetChangePolicy() == nil {
		return nil, errors.New("reconfiguration requires changePolicy")
	}

	mechanisms, hasFRER, err := validateMechanisms(options.Mechanisms)
	if err != nil {
		return nil, err
	}
	if hasFRER && options.Reliability == nil {
		return nil, errors.New("FRER requires explicit reliability policy")
	}
	if !hasFRER && options.Reliability != nil {
		return nil, errors.New("reliability policy supplied without FRER mechanism")
	}

	taskJSON, err := ExtractOptimizationTask(task)
	if err != nil {
		return nil, err
	}
	if len(options.TrafficClasses) > 0 {
		scope, _ := taskJSON["scope"].(map[string]any)
		if scope == nil {
			scope = make(map[string]any)
		}
		scope["trafficClasses"] = options.TrafficClasses
		taskJSON["scope"] = scope
	}

	forwarding, err := ExtractForwardingPlane(model, schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("forwarding-plane extraction: %w", err)
	}
	if operation == "reconfiguration" {
		if _, ok := forwarding["configuration"].(map[string]any); !ok {
			return nil, errors.New("reconfiguration requires current forwardingPlane.configuration")
		}
	}
	if err := applyStreamOptions(forwarding, options.Streams, operation); err != nil {
		return nil, err
	}
	if err := applyEthernetOverhead(forwarding, options.Streams); err != nil {
		return nil, err
	}
	if usesApplicationEndpoints(options.Streams) && len(options.Applications) == 0 {
		return nil, errors.New("application placement requires applications definitions")
	}

	caps, err := mergeCapabilities(ExtractPortCapabilities(model), options.AdditionalCapabilities)
	if err != nil {
		return nil, err
	}

	request := map[string]any{
		"schemaVersion":    "1.4",
		"operation":        operation,
		"optimizationTask": taskJSON,
		"mechanisms":       mechanisms,
		"forwardingPlane":  forwarding,
	}
	if devices, ok := caps["devices"].([]any); ok && len(devices) > 0 {
		request["capabilities"] = caps
	}
	if len(options.Applications) > 0 {
		request["applications"] = options.Applications
	}
	if len(options.DependencyGroups) > 0 {
		request["dependencyGroups"] = options.DependencyGroups
	}
	if options.Reliability != nil {
		request["reliability"] = options.Reliability
	}
	if len(options.Metadata) > 0 {
		request["metadata"] = options.Metadata
	}
	result, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serialize optimizer request: %w", err)
	}

	if err := tsncontracts.ValidateOptimizerRequestJSON(result); err != nil {
		return nil, fmt.Errorf("Contract v1.4 validation failed: %w", err)
	}
	return result, nil
}

func operationFromIntent(intent optimizationcontract.Intent) (string, error) {
	switch intent {
	case optimizationcontract.Intent_INITIAL_CONFIGURATION:
		return "configuration", nil
	case optimizationcontract.Intent_ADD_STREAM,
		optimizationcontract.Intent_REMOVE_STREAM,
		optimizationcontract.Intent_RECONFIGURE_STREAM,
		optimizationcontract.Intent_RECOVERY,
		optimizationcontract.Intent_DEFRAGMENTATION,
		optimizationcontract.Intent_REOPTIMIZATION:
		return "reconfiguration", nil
	default:
		return "", fmt.Errorf("cannot map OpenCNC intent %s to a v1.4 operation", intent.String())
	}
}

func validateMechanisms(input []MechanismSelection) ([]MechanismSelection, bool, error) {
	if len(input) == 0 {
		return nil, false, errors.New("at least one mechanism (CBS, TAS, or FRER) is required")
	}
	seen := make(map[string]bool)
	out := make([]MechanismSelection, 0, len(input))
	for _, m := range input {
		switch m.Name {
		case "CBS", "TAS", "FRER":
		default:
			return nil, false, fmt.Errorf("unsupported mechanism %q", m.Name)
		}
		if seen[m.Name] {
			return nil, false, fmt.Errorf("duplicate mechanism %q", m.Name)
		}
		if m.Name != "CBS" && len(m.Constraints) > 0 {
			return nil, false, fmt.Errorf("%s constraints are not defined in v1.4; use optimizationTask.constraints instead", m.Name)
		}
		seen[m.Name] = true
		out = append(out, m)
	}
	return out, seen["FRER"], nil
}

// applyStreamOptions adds explicit OaaS controls while filling omitted per-stream
// decisions with the temporary CNC policy agreed for contract v1.4.
// A default fixed route is never invented: it requires an actual supplied path.
func applyStreamOptions(forwarding map[string]any, controls map[string]StreamOptions, operation string) error {
	streams, ok := forwarding["streams"].([]any)
	if !ok {
		return errors.New("forwardingPlane.streams is missing")
	}

	// Check option keys before applying defaults, so unknown stream IDs produce
	// their own actionable error instead of an unrelated missing-path error.
	known := make(map[string]bool, len(streams))
	for _, raw := range streams {
		stream, ok := raw.(map[string]any)
		if !ok {
			return errors.New("invalid forwarding-plane stream object")
		}
		definition, ok := stream["definition"].(map[string]any)
		if !ok {
			return errors.New("stream definition is missing")
		}
		key, ok := definition["id"].(string)
		if !ok || key == "" {
			return errors.New("stream definition.id is missing")
		}
		known[key] = true
	}
	for key := range controls {
		if !known[key] {
			return fmt.Errorf("stream options reference unknown derived stream ID %q", key)
		}
	}

	for _, raw := range streams {
		stream := raw.(map[string]any)
		definition := stream["definition"].(map[string]any)
		key := definition["id"].(string)
		options := controls[key] // Zero value means no explicit options were supplied.

		// Temporary CNC defaults from Hamza. Explicit values take precedence,
		// independently for each of the three decisions.
		routing, scheduling, placement := options.Routing, options.Scheduling, options.Placement
		if routing == "" {
			routing = "fixed"
		}
		if scheduling == "" {
			scheduling = "optimize"
		}
		if placement == "" {
			placement = "fixed"
		}

		decisions := map[string]any{
			"routing": routing, "scheduling": scheduling, "placement": placement,
		}
		for field, rawValue := range decisions {
			value := rawValue.(string)
			if value != "fixed" && value != "optimize" && value != "preserve" {
				return fmt.Errorf("stream %s: unsupported %s decision %q", key, field, value)
			}
		}

		if routing == "fixed" || routing == "preserve" {
			config, _ := stream["configuration"].(map[string]any)
			if config == nil || (lenArray(config["path"]) == 0 && lenArray(config["listenerPaths"]) == 0) {
				if options.Routing == "" {
					return fmt.Errorf("stream %s: default fixed routing requires an existing path; specify routing=optimize explicitly if routing must be optimized", key)
				}
				return fmt.Errorf("stream %s: %s routing requires existing path", key, routing)
			}
		}
		stream["decisionVariables"] = decisions

		if options.Change != "" {
			if operation != "reconfiguration" {
				return fmt.Errorf("stream %s: change is allowed only for reconfiguration", key)
			}
			switch options.Change {
			case "none", "add", "modify", "remove":
				stream["change"] = options.Change
			default:
				return fmt.Errorf("stream %s: unsupported change %q", key, options.Change)
			}
		}
		if options.TalkerApplicationID != "" || len(options.ListenerApplicationIDs) > 0 || placement == "optimize" {
			if options.TalkerApplicationID == "" || len(options.ListenerApplicationIDs) == 0 {
				return fmt.Errorf("stream %s: application endpoints required for placement", key)
			}
			// OaaS placement stream endpoints replace physical CNC endpoints.
			// The original CNC model is preserved separately by the caller.
			delete(definition, "talkerNodeId")
			delete(definition, "listenerNodeIds")
			definition["talkerApplicationId"] = options.TalkerApplicationID
			definition["listenerApplicationIds"] = options.ListenerApplicationIDs
		}
	}
	return nil
}

// applyEthernetOverhead implements P8 on the projected v1.4 request only.
// OpenCNC does not model these occupancy fields; OaaS callers can provide
// explicit per-stream overrides. Otherwise standard Ethernet byte-times apply.
// maxFrameSize remains unchanged: downstream optimizers use the two overhead
// fields separately and must not add them more than once.
func applyEthernetOverhead(forwarding map[string]any, controls map[string]StreamOptions) error {
	streams, ok := forwarding["streams"].([]any)
	if !ok {
		return errors.New("forwardingPlane.streams is missing")
	}
	for i, raw := range streams {
		stream, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("streams[%d]: expected stream object", i)
		}
		definition, ok := stream["definition"].(map[string]any)
		if !ok {
			return fmt.Errorf("streams[%d]: missing definition", i)
		}
		id, ok := definition["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("streams[%d]: missing stream ID", i)
		}
		spec, ok := definition["trafficSpec"].(map[string]any)
		if !ok {
			return fmt.Errorf("stream %s: missing trafficSpec", id)
		}
		option := controls[id]
		for _, field := range []struct {
			name          string
			value         *int64
			defaultAmount int64
		}{
			{"serializationOverheadBytes", option.SerializationOverheadBytes, 8},
			{"interFrameGapBytes", option.InterFrameGapBytes, 12},
		} {
			if field.value != nil {
				// Follow the same exact-integer limit used for CNC uint64 fields.
				if *field.value < 0 || *field.value > int64(maxSafeJSONInteger) {
					return fmt.Errorf("stream %s: %s must be a nonnegative safe JSON integer", id, field.name)
				}
				spec[field.name] = *field.value
			} else if _, alreadyPresent := spec[field.name]; !alreadyPresent {
				spec[field.name] = field.defaultAmount
			}
		}
	}
	return nil
}

func usesApplicationEndpoints(controls map[string]StreamOptions) bool {
	for _, stream := range controls {
		if stream.Placement == "optimize" || stream.TalkerApplicationID != "" || len(stream.ListenerApplicationIDs) > 0 {
			return true
		}
	}
	return false
}

func lenArray(value any) int {
	if values, ok := value.([]any); ok {
		return len(values)
	}
	return 0
}

// mergeCapabilities overlays externally known device/port hardware limits
// onto CNC flags. External data takes precedence, including when proto3 false
// meant "unset"; callers must provide authoritative values.
func mergeCapabilities(cncCaps map[string]any, additional map[string]any) (map[string]any, error) {
	indexed := make(map[string]map[string]any)
	for _, source := range []map[string]any{cncCaps, additional} {
		if source == nil {
			continue
		}
		items, ok := source["devices"].([]any)
		if !ok {
			return nil, errors.New("capabilities.devices must be a JSON array ([]any)")
		}
		within := make(map[string]bool)
		for _, item := range items {
			device, ok := item.(map[string]any)
			if !ok {
				return nil, errors.New("capabilities device must be an object")
			}
			name, ok := device["nodeId"].(string)
			if !ok || name == "" || within[name] {
				return nil, fmt.Errorf("capabilities contains missing or duplicate node %q", name)
			}
			within[name] = true
			if indexed[name] == nil {
				indexed[name] = map[string]any{"nodeId": name}
			}
			mergeJSONObjects(indexed[name], device)
		}
	}
	keys := make([]string, 0, len(indexed))
	for name := range indexed {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	devices := make([]any, 0, len(keys))
	for _, name := range keys {
		devices = append(devices, indexed[name])
	}
	return map[string]any{"devices": devices}, nil
}

func mergeJSONObjects(dst, src map[string]any) {
	for key, value := range src {
		child, isMap := value.(map[string]any)
		if !isMap {
			dst[key] = value
			continue
		}
		target, _ := dst[key].(map[string]any)
		if target == nil {
			target = make(map[string]any)
			dst[key] = target
		}
		mergeJSONObjects(target, child)
	}
}
