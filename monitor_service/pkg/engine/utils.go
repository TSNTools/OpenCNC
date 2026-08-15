package engine

import (
	"context"
	"fmt"
	"time"

	configservice "OpenCNC_config_service/common/structures/service"
	"OpenCNC_config_service/monitor_service/structures/monitoring"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultConfigServiceAddress = "localhost:5150"

func handleRequestRollback(event *monitoring.MonitoringEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		defaultConfigServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("dial config service %s: %w", defaultConfigServiceAddress, err)
	}
	defer conn.Close()

	client := configservice.NewConfigServiceClient(conn)
	resp, err := client.Rollback(ctx, &configservice.RollbackRequest{})
	if err != nil {
		return fmt.Errorf("config service rollback RPC failed: %w", err)
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("config service rollback failed: %s", resp.GetMessage())
	}

	return nil
}

func handleRequestReconfiguration(event *monitoring.MonitoringEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	return fmt.Errorf("request reconfiguration is not implemented yet")
}
