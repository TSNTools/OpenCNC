package internalOptimizer

import (
	"fmt"

	"OpenCNC/common/structures/stream_config"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

// Finds the corresponding stream using its stream ID.
// Reads the PCP from the UNI VLAN tag.
// Adds it to a copy of the FPM used for the optimizer request.
// Rejects conflicting or invalid PCP values.
// incoming UNI requests. The original FPM (used for snapshot/result apply) is never modified. No traffic-class or queue priority is inferred.
func withUNIPCP(
	original *forwardingplane.ForwardingPlaneModel,
	requests []*uni.Request,
) (*forwardingplane.ForwardingPlaneModel, error) {
	if original == nil {
		return nil, fmt.Errorf("forwarding plane is nil")
	}
	if len(requests) == 0 {
		return original, nil
	}

	copy := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)
	byID := make(map[string]*forwardingplane.StreamModel, len(copy.GetStreams()))
	for _, model := range copy.GetStreams() {
		if model == nil || model.GetDefinition() == nil || model.GetDefinition().GetStreamId() == nil {
			return nil, fmt.Errorf("forwarding plane has stream without an ID")
		}
		id := model.GetDefinition().GetStreamId()
		key := requestadapter.StreamKey(id.GetMacAddress(), id.GetUniqueId())
		if _, exists := byID[key]; exists {
			return nil, fmt.Errorf("duplicate forwarding-plane stream %q", key)
		}
		byID[key] = model
	}

	seen := make(map[string]bool, len(requests))
	for _, req := range requests {
		if req == nil || req.GetTalker() == nil || req.GetTalker().GetStrId() == nil {
			return nil, fmt.Errorf("UNI request has no talker stream ID")
		}
		id := req.GetTalker().GetStrId()
		key := requestadapter.StreamKey(id.GetMacAddress(), id.GetUniqueId())
		if seen[key] {
			return nil, fmt.Errorf("duplicate UNI request for stream %q", key)
		}
		seen[key] = true

		model, ok := byID[key]
		if !ok {
			return nil, fmt.Errorf("UNI stream %q not found in forwarding plane", key)
		}

		var found bool
		var pcp uint32
		for _, spec := range req.GetTalker().GetDataFrameSpecification() {
			if spec == nil || spec.GetVlanTag() == nil {
				continue
			}
			if found {
				return nil, fmt.Errorf("stream %q has multiple UNI VLAN tag specifications", key)
			}
			found = true
			pcp = spec.GetVlanTag().GetPriorityCodePoint()
		}
		if !found {
			continue // PCP is optional in v1.4; do not invent one.
		}
		if pcp > 7 {
			return nil, fmt.Errorf("stream %q: UNI PCP %d outside 0..7", key, pcp)
		}
		if pcp == 0 {
			return nil, fmt.Errorf("stream %q: UNI proto3 cannot distinguish omitted PCP from explicit PCP 0", key)
		}
		if model.Configuration == nil {
			model.Configuration = &stream_config.StreamConfiguration{}
		}
		if existing := model.Configuration.Pcp; existing != nil && *existing != pcp {
			return nil, fmt.Errorf("stream %q: UNI PCP %d conflicts with stored stream PCP %d", key, pcp, *existing)
		}
		value := pcp
		model.Configuration.Pcp = &value
	}

	return copy, nil
}
