package notificationServer

import (
	"context"
	"fmt"

	storewrapper "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/uni"
	handler "OpenCNC/tsn_service/pkg/notificationHandler"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	UnimplementedNotificationServer
}

// Notify is the single gRPC entry point for notifications received by
// the TSN service.
func (s *Server) Notify(ctx context.Context, event *Event) (*NotifyResponse, error) {
	if event == nil {
		return nil, status.Error(codes.InvalidArgument, "event is nil")
	}

	fmt.Printf(
		"[tsn-service] Received notification of type %s\n",
		event.GetType().String(),
	)

	switch event.GetType() {

	case EventType_STREAM_ADDED:
		return s.handleStreamAdded(ctx, event)

	case EventType_STREAM_REMOVED:
		return s.handleStreamRemoved(ctx, event)

	case EventType_NETWORK_EVENT:
		return s.handleNetworkEvent(ctx, event)

	case EventType_EVENT_TYPE_UNSPECIFIED:
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

func (s *Server) handleStreamAdded(ctx context.Context, event *Event) (*NotifyResponse, error) {

	payload := event.GetStreamAdded()
	if payload == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"STREAM_ADDED event has no payload",
		)
	}

	// Get request from k/v store
	requestIds := payload.GetRequestIds()

	var allRequestData []*uni.Request
	for _, requestId := range requestIds {
		reqData, err := storewrapper.GetUniRequestData(requestId)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"failed to get request data for request ID %s: %v",
				requestId,
				err,
			)
		}
		allRequestData = append(allRequestData, reqData)
	}

	// specify the configuration task
	// TODO: CreateOptimizationTask looses some of the request data details.
	// Therefore, allRequestData is passed to CalculateConfiguration
	// we need to improve CreateOptimizationTask to retain
	// all necessary request data and not pass it further
	task, err := CreateOptimizationTask(allRequestData)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"failed to identify optimization task: %v",
			err,
		)
	}

	// Existing TSN configuration calculation.

	newFPM, err := handler.CalculateConfiguration(task, allRequestData)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"failed to calculate configuration: %v",
			err,
		)
	}

	//TODO: returning only the configId, assuming that the client has already
	// all stream ids and can retreive stream-specific using them.
	return &NotifyResponse{
		Accepted: true,
		ConfigId: newFPM.Configuration.ConfigId,
		Message:  "stream addition accepted and configuration calculated",
	}, nil
}

func (s *Server) handleStreamRemoved(ctx context.Context, event *Event) (*NotifyResponse, error) {

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

	return &NotifyResponse{
		Accepted: true,
		Message:  "stream removal accepted",
	}, nil
}

func (s *Server) handleNetworkEvent(ctx context.Context, event *Event) (*NotifyResponse, error) {

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

	return &NotifyResponse{
		Accepted: true,
		Message:  "network event accepted",
	}, nil
}
