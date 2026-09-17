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
	"OpenCNC/tsn_service/pkg/internalOptimizer"
	"OpenCNC/tsn_service/pkg/notificationServer"

	"google.golang.org/grpc"
)

func main() {
	///////////////////////
	NotificationServerPort, err := strconv.ParseUint(
		configuration.GetEnv("TSN_SERVICE_PORT", "5152"), 10, 16,
	)
	if err != nil {
		log.Fatalf("invalid TSN_SERVICE_PORT: %v", err)
	}
	///////////////////////
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	obsClient, err := observability.NewFromEnv("tsn-service")
	if err != nil {
		log.Fatalf("Observability init failed: %v", err)
	}

	if obsClient != nil {
		defer func() {
			_ = obsClient.Close()
		}()

		startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_ = obsClient.EmitHealthStarted(
			startupCtx,
			"tsn-service-startup",
			"tsn-service started",
		)
	}

	// Create default schedule and store it in k/v store
	if err := internalOptimizer.CreateDefaultSchedule(ctx, obsClient); err != nil {
		if obsClient != nil {
			_ = obsClient.Error(ctx, fmt.Sprintf(
				"Failed creating default schedule: %v",
				err,
			))
		}
		return
	}

	if obsClient != nil {
		_ = obsClient.Info(ctx, "Let's start the server!")
	}

	// Used to get device configuration and config+state data from a device
	// go test()

	// Start notification-server
	go CreateNotificationServer(ctx, "tcp", uint16(NotificationServerPort), obsClient)

	select {}
}

func CreateNotificationServer(
	ctx context.Context,
	protocol string,
	port uint16,
	obs *observability.Client,
) {
	lis, err := net.Listen(protocol, fmt.Sprintf(":%d", port))
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to listen: %v",
				err,
			))
		}
		return
	}

	if obs != nil {
		_ = obs.Info(ctx, fmt.Sprintf(
			"Listening on %d",
			port,
		))
	}

	s := notificationServer.NewServer(obs)

	grpcServer := grpc.NewServer()

	notificationServer.RegisterNotificationServer(grpcServer, s)

	if obs != nil {
		_ = obs.Info(ctx, "Started to serve...")
	}

	if err := grpcServer.Serve(lis); err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to serve: %v",
				err,
			))
		}
	}
}

/*
func test() {
	time.Sleep(time.Second * 90)

	tree, err := store.GetDeviceConfig("192.168.0.2")
	if err != nil {
		//	log.Errorf("Failed getting device config: %v", err)
		return
	}

	store.StoreDeviceConfig("192.168.0.2", tree)
}
*/
