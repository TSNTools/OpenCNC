package uni_server

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	handler "OpenCNC/main_service/pkg/event-handler"

	uni "OpenCNC/common/structures/uni"

	grpc "google.golang.org/grpc"
)

type Server struct {
	UnimplementedUniServiceServer
}

func (s *Server) AddStream(ctx context.Context, req *uni.ConfigRequest) (*uni.ConfigResponse, error) {

	if req == nil {
		return nil, fmt.Errorf("received nil AddStream request")
	}

	fmt.Printf("[Main-service] Received AddStream request")

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

	// createResponse
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

func StartGrpcServer(port uint16) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()

	RegisterUniServiceServer(grpcServer, &Server{})

	log.Println("Starting UNI gRPC server")

	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}
