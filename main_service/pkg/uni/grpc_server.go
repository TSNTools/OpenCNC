package uni_server

import (
	"context"
	"fmt"
	"time"

	handler "OpenCNC/main_service/pkg/event-handler"

	uni "OpenCNC/common/structures/uni"
)

type Server struct {
	UnimplementedUniServiceServer
}

func (s *Server) AddStream(ctx context.Context, req *uni.ConfigRequest) (*uni.ConfigResponse, error) {

	if req == nil {
		return nil, fmt.Errorf("received nil AddStream request")
	}

	fmt.Printf("Received AddStream request: %+v\n", req)

	// Check whether the client cancelled the request.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use exactly the same event handler as the HTTP server.
	confID, err := handler.HandleAddStreamEvent(req, time.Now())
	if err != nil {
		fmt.Printf("Failed handling AddStream event: %v\n", err)

		return nil, fmt.Errorf(
			"failed handling AddStream event: %w",
			err,
		)
	}

	// createResponse already exists in this package and already:
	//
	//   1. Gets the configuration from the store.
	//   2. Builds the ConfigResponse.
	//   3. Generates all Response/StatusGroup information.
	//   4. Serializes it using protojson.
	//
	// We reuse it instead of duplicating any of that logic.
	response, err := createResponse(confID, req)
	if err != nil {
		fmt.Printf("Failed to create UNI response: %v\n", err)

		return nil, fmt.Errorf(
			"failed to create UNI response: %w",
			err,
		)
	}

	return response, nil
}
