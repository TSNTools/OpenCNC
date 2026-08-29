package pathentity

import (
	"fmt"

	pbstream "OpenCNC/common/structures/stream"
	pbstreamconfig "OpenCNC/common/structures/stream_config"
	fwdplanemodel "OpenCNC/tsn_service/pkg/structures/network_model"
)

// PathEntity manages stream path configuration in the CNC model.
// It prepares routing input for the optimizer and applies the
// optimizer's routing results to the forwarding-plane model.
type PathEntity struct {
	model *fwdplanemodel.ForwardingPlaneModel
}

// New creates a Path Entity operating on the given model.
func New(model *fwdplanemodel.ForwardingPlaneModel) *PathEntity {
	return &PathEntity{
		model: model,
	}
}

// Model returns the forwarding-plane model.
func (pe *PathEntity) Model() *fwdplanemodel.ForwardingPlaneModel {
	return pe.model
}

// SetModel changes the forwarding-plane model.
func (pe *PathEntity) SetModel(
	model *fwdplanemodel.ForwardingPlaneModel,
) {
	pe.model = model
}

// -----------------------------------------------------------------------------
// Routing input
// -----------------------------------------------------------------------------

// RoutingInput contains the information PE provides to the optimizer
// for path optimization.
//
// Keep this separate from ForwardingPlaneModel: it is an optimizer-facing
// representation, not persistent model state.
type RoutingInput struct {
	Topology *fwdplanemodel.ForwardingPlaneModel
	Streams  []*pbstream.Stream
}

// BuildRoutingInput extracts the routing-relevant information from
// the current forwarding-plane model.
func (pe *PathEntity) GetRoutingInput() (*RoutingInput, error) {
	if pe == nil || pe.model == nil {
		return nil, fmt.Errorf("forwarding-plane model is nil")
	}

	input := &RoutingInput{
		Topology: pe.model,
		Streams:  make([]*pbstream.Stream, 0, len(pe.model.Streams)),
	}

	for _, sm := range pe.model.Streams {
		if sm == nil || sm.GetDefinition() == nil {
			continue
		}

		input.Streams = append(input.Streams, sm.GetDefinition())
	}

	return input, nil
}

// -----------------------------------------------------------------------------
// Routing result
// -----------------------------------------------------------------------------

// ApplyPath applies a path selected by the optimizer to a stream's
// derived configuration.
func (pe *PathEntity) ApplyRouting(
	streamID *pbstream.StreamId,
	path []*pbstreamconfig.StreamHop,
) error {
	if pe == nil || pe.model == nil {
		return fmt.Errorf("forwarding-plane model is nil")
	}

	if streamID == nil {
		return fmt.Errorf("stream ID is nil")
	}

	stream := pe.model.Stream(streamID)
	if stream == nil {
		return fmt.Errorf("stream not found")
	}

	if stream.Configuration == nil {
		return fmt.Errorf("stream configuration is nil")
	}

	if err := pe.ValidateRouting(path); err != nil {
		return err
	}

	stream.Configuration.Path = path

	return nil
}

// -----------------------------------------------------------------------------
// Validation
// -----------------------------------------------------------------------------

func (pe *PathEntity) ValidateRouting(
	path []*pbstreamconfig.StreamHop,
) error {
	if len(path) == 0 {
		return fmt.Errorf("path is empty")
	}

	for i, hop := range path {
		if hop == nil {
			return fmt.Errorf("path contains nil hop at index %d", i)
		}

		if int(hop.GetHopNumber()) != i {
			return fmt.Errorf(
				"invalid hop number at index %d: got %d",
				i,
				hop.GetHopNumber(),
			)
		}

		if hop.GetBridgeId() == "" {
			return fmt.Errorf("path hop %d has no bridge ID", i)
		}

		if hop.GetIngressPortId() == "" {
			return fmt.Errorf("path hop %d has no ingress port", i)
		}

		if hop.GetEgressPortId() == "" {
			return fmt.Errorf("path hop %d has no egress port", i)
		}
	}

	return nil
}
