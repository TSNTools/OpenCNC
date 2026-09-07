package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"time"

	"OpenCNC/common/observability"
	eventhandler "OpenCNC/main_service/pkg/event-handler"
	"OpenCNC/main_service/pkg/nni"

	//"OpenCNC/main_service/pkg/uni"
	uni_server "OpenCNC/main_service/pkg/uni"

	"google.golang.org/grpc"
)

const (
	NNI_SERVER_PORT      uint16 = 8000
	UNI_GRPC_SERVER_PORT uint16 = 5153
	UNI_HTTP_SERVER_PORT uint16 = 8081
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	obsClient, err := observability.NewFromEnv("main-service")
	if err != nil {
		log.Fatalf("Observability init failed: %v", err)
	}
	if obsClient != nil {
		defer func() {
			_ = obsClient.Close()
		}()

		startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = obsClient.EmitHealthStarted(startupCtx, "main-service-startup", "main-service started")
	}

	// Start NNI server
	go nni.StartServer(ctx, NNI_SERVER_PORT, obsClient)

	// Start UNI grpc server
	// go uni_server.StartHttpServer(ctx, UNI_HTTP_SERVER_PORT, obsClient)
	go uni_server.StartGrpcServer(ctx, UNI_GRPC_SERVER_PORT, obsClient)

	// Not working on local network, needs to be connected to switches
	//switches := counterConfHandler.GetMonitorConfigDevices()
	//if err := startDeviceDataCollection(switches); err != nil {
	//log.Errorf("Failed data collection for switches: %v", err)
	//}

	// Don't start the UNI before the system is available, so that the user
	// sees the UNI only when the subsystems are available
	//	log.Info("Starting to check for config subsystem availability...")

	// Start UNI server
	//go uni.StartServer()

	//pollConfigSubsystemForAvailability()

	select {}
}

func StartUniGrpcServer(
	ctx context.Context,
	port string,
	obs *observability.Client,
) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if obs != nil {
			obs.Error(ctx, fmt.Sprintf(
				"Failed to listen for UNI gRPC server on port %s: %v",
				port,
				err,
			))
		}

		return fmt.Errorf(
			"failed to listen on port %s: %w",
			port,
			err,
		)
	}

	grpcServer := grpc.NewServer()

	uni_server.RegisterUniServiceServer(
		grpcServer,
		uni_server.NewServer(obs),
	)

	if obs != nil {
		obs.Info(ctx, fmt.Sprintf(
			"UNI gRPC server listening on :%s",
			port,
		))
	}

	return grpcServer.Serve(listener)
}

func pollConfigSubsystemForAvailability(ctx context.Context, obs *observability.Client) bool {
	for {
		if obs != nil {
			obs.Info(ctx, "Trying to connect to config-service...")
		}

		_, err := eventhandler.ConnectToGnmiService(ctx, obs, "config-service:5150")
		if err != nil {
			if obs != nil {
				obs.Error(ctx, fmt.Sprintf(
					"Config-service not reachable, retrying in 10s: %v",
					err,
				))
			}

			select {
			case <-ctx.Done():
				return false
			case <-time.After(10 * time.Second):
			}

			continue
		}

		if obs != nil {
			obs.Info(ctx, "Connected to config-service!")
		}

		return true
	}
}
