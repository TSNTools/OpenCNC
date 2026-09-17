package uni_server

import (
	"context"
	"fmt"
	"net"
	"time"

	"OpenCNC/common/observability"
	observabilityv1 "OpenCNC/common/structures/logging"
	uni "OpenCNC/common/structures/uni"
	handler "OpenCNC/main_service/pkg/event-handler"

	grpc "google.golang.org/grpc"
)

type Server struct {
	UnimplementedUniServiceServer
	obs *observability.Client
}

func NewServer(obs *observability.Client) *Server {
	return &Server{
		obs: obs,
	}
}

func (s *Server) AddStream(ctx context.Context, req *uni.ConfigRequest) (*uni.ConfigResponse, error) {
	start := time.Now()

	if req == nil {
		return nil, fmt.Errorf("received nil AddStream request")
	}

	if s.obs != nil {
		_ = s.obs.Info(ctx, "[Main-service] Received AddStream request")
	}

	// Check whether the client cancelled the request.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use exactly the same event handler as the HTTP server.
	confID, err := handler.HandleAddStreamEvent(ctx, s.obs, req, time.Now())

	if err != nil {
		if s.obs != nil {
			_ = s.obs.Error(ctx, fmt.Sprintf(
				"Failed handling AddStream event: %v",
				err,
			))
		}

		return nil, fmt.Errorf(
			"failed handling AddStream event: %w",
			err,
		)
	}

	// createResponse
	response, err := createResponse(
		ctx,
		s.obs,
		confID,
		req,
	)

	if err != nil {
		if s.obs != nil {
			_ = s.obs.Error(ctx, fmt.Sprintf(
				"Failed to create UNI response: %v",
				err,
			))
		}

		return nil, fmt.Errorf(
			"failed to create UNI response: %w",
			err,
		)
	}

	if s.obs != nil {
		_ = s.obs.Metric(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"uni_request_response_duration",
			observabilityv1.MetricType_METRIC_TYPE_GAUGE,
			float64(time.Since(start).Milliseconds()),
			"ms",
			map[string]string{
				"interface": "grpc",
				"operation": "add_stream",
			},
		)
	}

	return response, nil
}

func StartGrpcServer(
	ctx context.Context,
	port uint16,
	obs *observability.Client,
) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to listen on port %d: %v",
				port,
				err,
			))

			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"uni.grpc",
				"start_failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"server",
				"",
				fmt.Sprintf("Failed to start UNI gRPC server: %v", err),
			)
		}

		return fmt.Errorf(
			"failed to listen on port %d: %w",
			port,
			err,
		)
	}

	grpcServer := grpc.NewServer()

	RegisterUniServiceServer(
		grpcServer,
		NewServer(obs),
	)

	if obs != nil {
		_ = obs.Info(ctx, "Starting UNI gRPC server")

		_ = obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"uni.grpc",
			"started",
			observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
			"server",
			"",
			"UNI gRPC server started successfully",
		)
	}

	if err := grpcServer.Serve(listener); err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"gRPC server failed: %v",
				err,
			))

			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"uni.grpc",
				"failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"server",
				"",
				fmt.Sprintf("UNI gRPC server failed: %v", err),
			)
		}

		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}
