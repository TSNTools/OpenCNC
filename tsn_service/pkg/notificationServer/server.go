package notificationServer

import (
	"context"
	"fmt"

	handler "OpenCNC/tsn_service/pkg/notificationHandler"

	"OpenCNC/tsn_service/pkg/structures/notification"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	notification.UnimplementedNotificationServer
}

// Notify is the single gRPC entry point for notifications received by
// the TSN service.
func (s *Server) Notify(ctx context.Context, event *notification.Event) (*notification.NotifyResponse, error) {
	if event == nil {
		return nil, status.Error(codes.InvalidArgument, "event is nil")
	}

	fmt.Printf(
		"Received notification: id=%s type=%s source=%s\n",
		event.GetEventId(),
		event.GetType().String(),
		event.GetSource(),
	)

	switch event.GetType() {

	case notification.EventType_STREAM_ADDED:
		return s.handleStreamAdded(ctx, event)

	case notification.EventType_STREAM_REMOVED:
		return s.handleStreamRemoved(ctx, event)

	case notification.EventType_NETWORK_EVENT:
		return s.handleNetworkEvent(ctx, event)

	case notification.EventType_EVENT_TYPE_UNSPECIFIED:
		return nil, status.Error(
			codes.InvalidArgument,
			"event type is unspecified",
		)

	default:
		return nil, status.Error(
			codes.InvalidArgument,
			"unknown event type",
		)
	}
}

func (s *Server) handleStreamAdded(
	ctx context.Context,
	event *notification.Event,
) (*notification.NotifyResponse, error) {

	payload := event.GetStreamAdded()
	if payload == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"STREAM_ADDED event has no payload",
		)
	}

	requestIDs := payload.GetRequestIds()

	fmt.Printf(
		"Stream added: stream_id=%s request_ids=%v\n",
		payload.GetStreamId(),
		requestIDs,
	)

	// Existing TSN configuration calculation.
	configID, err := handler.CalculateConfiguration(requestIDs)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"failed to calculate configuration: %v",
			err,
		)
	}

	return &notification.NotifyResponse{
		Accepted:   true,
		WorkflowId: configID,
		Message:    "stream addition accepted and configuration calculated",
	}, nil
}

func (s *Server) handleStreamRemoved(
	ctx context.Context,
	event *notification.Event,
) (*notification.NotifyResponse, error) {

	payload := event.GetStreamRemoved()
	if payload == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"STREAM_REMOVED event has no payload",
		)
	}

	fmt.Printf(
		"Stream removed: stream_id=%s request_ids=%v reason=%s\n",
		payload.GetStreamId(),
		payload.GetRequestIds(),
		payload.GetReason(),
	)

	// TODO:
	// Call the stream removal / resource release workflow here.
	//
	// For now we only acknowledge the event.

	return &notification.NotifyResponse{
		Accepted: true,
		Message:  "stream removal accepted",
	}, nil
}

func (s *Server) handleNetworkEvent(
	ctx context.Context,
	event *notification.Event,
) (*notification.NotifyResponse, error) {

	payload := event.GetNetworkEvent()
	if payload == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"NETWORK_EVENT event has no payload",
		)
	}

	fmt.Printf(
		"Network event: trigger=%s nodes=%v links=%v reason=%s\n",
		payload.GetTrigger().String(),
		payload.GetAffectedNodeIds(),
		payload.GetAffectedLinkIds(),
		payload.GetReason(),
	)

	// Network events should trigger the TSN reconfiguration workflow.
	//
	// The workflow itself decides which components need updating:
	//
	//   STP
	//   VLAN
	//   routing
	//   scheduling
	//   QoS
	//
	// The notification server should not contain those implementation
	// details.

	// TODO:
	// workflowID, err := handler.HandleNetworkEvent(payload)
	//
	// For now we acknowledge the event.

	return &notification.NotifyResponse{
		Accepted: true,
		Message:  "network event accepted",
	}, nil
}
