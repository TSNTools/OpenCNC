package stub

import (
	"context"
	"fmt"
	"time"

	service "OpenCNC_config_service/monitor_service/structures/grpc_server"
	"OpenCNC_config_service/monitor_service/structures/monitoring"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultMonitorServiceAddress = "localhost:5151"

func requestGetCapabilities(resource *monitoring.ResourceKey) (*service.CapabilitiesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		defaultMonitorServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"dial monitor service %s: %w",
			defaultMonitorServiceAddress,
			err,
		)
	}
	defer conn.Close()

	client := service.NewMonitorServiceClient(conn)

	resp, err := client.GetCapabilities(
		ctx,
		&service.CapabilitiesRequest{
			Resource: resource,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"monitor service GetCapabilities RPC failed: %w",
			err,
		)
	}

	return resp, nil
}

func requestStartMonitoring(
	id string,
	resource *monitoring.ResourceKey,
	items []string,
) (*service.MonitoringResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		defaultMonitorServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"dial monitor service %s: %w",
			defaultMonitorServiceAddress,
			err,
		)
	}
	defer conn.Close()

	client := service.NewMonitorServiceClient(conn)

	resp, err := client.StartMonitoring(
		ctx,
		&service.StartMonitoringRequest{
			Id:       id,
			Resource: resource,
			Items:    items,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"monitor service StartMonitoring RPC failed: %w",
			err,
		)
	}

	return resp, nil
}
