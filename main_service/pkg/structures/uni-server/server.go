package uni_server

import (
	"context"
	"fmt"
	"net"

	uni_grpc "OpenCNC/main_service/pkg/structures/uni-grpc"

	"google.golang.org/grpc"
)

type Server struct {
	uni_grpc.UnimplementedUniServiceServer
}

func (s *Server) AddStream(
	ctx context.Context,
	req *uni_grpc.AddStreamRequest,
) (*uni_grpc.AddStreamResponse, error) {

	fmt.Printf("Received AddStream request: %+v\n", req)

	// Temporary: just verify that the request reaches Main.
	fmt.Printf("Stream ID: %s\n", req.GetId())
	fmt.Printf("Stream Name: %s\n", req.GetName())
	fmt.Printf("Talker: %s\n", req.GetTalkerNodeId())
	fmt.Printf("Listeners: %v\n", req.GetListenerNodeIds())

	return &uni_grpc.AddStreamResponse{
		Success:         true,
		Message:         "AddStream received by Main",
		ConfigurationId: "test-config-id",
	}, nil
}

func StartServer(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	grpcServer := grpc.NewServer()

	uni_grpc.RegisterUniServiceServer(
		grpcServer,
		&Server{},
	)

	fmt.Printf("UNI gRPC server listening on :%s\n", port)

	return grpcServer.Serve(listener)
}
