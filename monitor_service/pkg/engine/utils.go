package engine

import (
	"context"
	"fmt"
	"time"

	"OpenCNC/common/configuration"
	"OpenCNC/common/observability"
	observabilityv1 "OpenCNC/common/structures/logging"
	configservice "OpenCNC/config_service/grpc_server"
	"OpenCNC/monitor_service/structures/monitoring"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var defaultConfigServiceAddress = configuration.GetEnv("CONFIG_SERVICE_HOST", "localhost") +
	":" + configuration.GetEnv("CONFIG_SERVICE_PORT", "5150")

func HandleRequestRollback(event *monitoring.MonitoringEvent, obs *observability.Client) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	if obs != nil {
		_ = obs.Event(
			context.Background(),
			observabilityv1.Severity_SEVERITY_INFO,
			"monitoring",
			"rollback_requested",
			observabilityv1.DomainResult_DOMAIN_RESULT_ACCEPTED,
			"configuration",
			"",
			"monitoring event requested configuration rollback",
		)
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
