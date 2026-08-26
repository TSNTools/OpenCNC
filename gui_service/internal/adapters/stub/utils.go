package stub

import (
	"context"
	"fmt"
	"strings"
	"time"

	"OpenCNC/gui_service/internal/domain"
	service "OpenCNC/monitor_service/structures/grpc_server"
	"OpenCNC/monitor_service/structures/monitoring"

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

func requestStartMonitoring(query domain.MonitoringDataQuery) (*service.MonitoringResponse, error) {

	//parse  query and build the request
	parts := strings.SplitN(query.TargetID, "_", 2)
	nodeId := parts[0]
	portId := ""
	if len(parts) > 1 {
		portId = parts[1]
	}
	resource := &monitoring.ResourceKey{
		NodeId: nodeId,
		PortId: &portId,
	}

	settings := make([]*service.MonitoringSetting, len(query.MetricIDs))
	for i, metricID := range query.MetricIDs {
		intervalSeconds := query.PollIntervals[metricID]
		if intervalSeconds == 0 {
			intervalSeconds = 1
		}
		settings[i] = &service.MonitoringSetting{
			CapabilityId:   metricID,
			PollIntervalMs: intervalSeconds * 1000,
		}
	}

	req := &service.StartMonitoringRequest{
		Id:       query.TargetID,
		Resource: resource,
		Settings: settings,
	}

	// Create a context with a timeout for the gRPC call
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
	resp, err := client.StartMonitoring(ctx, req)
	//fmt.Println("requestStartMonitoring: ", req)

	if err != nil {
		return nil, fmt.Errorf(
			"monitor service StartMonitoring RPC failed: %w",
			err,
		)
	}

	return resp, nil
}
