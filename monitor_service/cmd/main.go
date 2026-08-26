package main

import (
	"context"
	"log"
	"net"
	"time"

	"OpenCNC/common/observability"
	"OpenCNC/monitor_service/pkg/catalog"
	"OpenCNC/monitor_service/pkg/engine"
	service "OpenCNC/monitor_service/structures/grpc_server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	obsClient, err := observability.NewFromEnv("monitor-service")
	if err != nil {
		log.Fatalf("Observability init failed: %v", err)
	}
	if obsClient != nil {
		defer func() {
			_ = obsClient.Close()
		}()

		startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = obsClient.EmitHealthStarted(startupCtx, "monitor-service-startup", "monitor-service started")
	}

	catalog, err := catalog.LoadCatalog()
	if err != nil {
		log.Fatalf("failed loading catalog: %v", err)
	}

	monitorEngine := engine.NewEngine(catalog, obsClient)
	server := service.NewMonitorServer(monitorEngine)

	listener, err := net.Listen("tcp", ":5151")
	if err != nil {
		if obsClient != nil {
			obsClient.FatalF("Failed to listen on :5151: %v", err)
		}
		log.Fatalf("failed listening on :5151: %v", err)
	}

	grpcServer := grpc.NewServer()
	service.RegisterMonitorServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	if obsClient != nil {
		obsClient.Println("gRPC monitor server started on port 5151")
	}

	if err := grpcServer.Serve(listener); err != nil {
		if obsClient != nil {
			errCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = obsClient.EmitHealthError(errCtx, "monitor-service-serve", err.Error(), "")
		}
		if obsClient != nil {
			obsClient.FatalF("gRPC monitor server failed: %v", err)
		}
		log.Fatalf("gRPC monitor server failed: %v", err)
	}
}
