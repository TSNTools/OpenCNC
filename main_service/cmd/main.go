package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"time"

	"OpenCNC/common/configuration"
	"OpenCNC/common/observability"
	eventhandler "OpenCNC/main_service/pkg/event-handler"
	"OpenCNC/main_service/pkg/nni"

	//"OpenCNC/main_service/pkg/uni"
	uni_server "OpenCNC/main_service/pkg/uni"

	"google.golang.org/grpc"
)

var UNI_HTTP_SERVER_PORT = configuration.GetEnv("MAIN_SERVICE_UNI_HTTP_PORT", "8081")

func main() {

	///////////////////////
	nniPort, err := strconv.ParseUint(
		configuration.GetEnv("MAIN_SERVICE_NNI_PORT", "8000"), 10, 16,
	)
	if err != nil {
		log.Fatalf("invalid MAIN_SERVICE_NNI_PORT: %v", err)
	}

	unigrpcPort, err := strconv.ParseUint(
		configuration.GetEnv("MAIN_SERVICE_UNI_GRPC_PORT", "5153"), 10, 16,
	)
	if err != nil {
		log.Fatalf("invalid MAIN_SERVICE_UNI_GRPC_PORT: %v", err)
	}

	//unihttpPort, err := strconv.ParseUint(
	//	configuration.GetEnv("MAIN_SERVICE_UNI_HTTP_PORT", "8081"), 10, 16,
	//)
	//if err != nil {
	//	log.Fatalf("invalid MAIN_SERVICE_UNI_HTTP_PORT: %v", err)
	//}

	///////////////////////

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
	go nni.StartServer(ctx, uint16(nniPort), obsClient)

	// Start UNI grpc server
	// go uni_server.StartHttpServer(ctx, UNI_HTTP_SERVER_PORT, obsClient)
	go uni_server.StartGrpcServer(ctx, uint16(unigrpcPort), obsClient)

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

		configServiceAddress := configuration.GetEnv("CONFIG_SERVICE_HOST", "localhost") +
			":" + configuration.GetEnv("CONFIG_SERVICE_PORT", "5150")

		_, err := eventhandler.ConnectToGnmiService(ctx, obs, configServiceAddress)
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
