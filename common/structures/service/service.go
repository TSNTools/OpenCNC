package service

import (
	"context"
	"fmt"
	"os"

	"OpenCNC_config_service/common/observability"
	storewrapper "OpenCNC_config_service/common/store-wrapper"
	observabilityv1 "OpenCNC_config_service/common/structures/logging"
	"OpenCNC_config_service/common/structures/topology_config"
	"OpenCNC_config_service/config_service/pkg/engine"
)

// ConfigServiceServer implements the generated gRPC interface.
type ConfigServiceServerImpl struct {
	UnimplementedConfigServiceServer
	obs    *observability.Client
	engine *engine.MappingEngine
}

// Constructor
func NewConfigServiceServerImpl(obs *observability.Client, engine *engine.MappingEngine) *ConfigServiceServerImpl {
	return &ConfigServiceServerImpl{obs: obs, engine: engine}
}

// ApplyConfiguration receives a config request ID, retrieves it from store,
// maps it via the plugin, and pushes it to all NETCONF-capable devices.
func (s *ConfigServiceServerImpl) ApplyConfiguration(ctx context.Context, req *ConfigurationRequest) (*ConfigurationResponse, error) {

	cfg := req.GetConfiguration()
	if cfg == nil {
		return &ConfigurationResponse{
			Success: false,
			Message: "Configuration is nil",
		}, fmt.Errorf("configuration is nil")
	}

	if err := s.deployConfiguration(ctx, cfg); err != nil {
		return &ConfigurationResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	storewrapper.StoreConfiguration(cfg)

	return &ConfigurationResponse{
		Success: true,
		Message: "Configuration applied successfully",
	}, nil
}

func (s *ConfigServiceServerImpl) ApplyConfigurationById(ctx context.Context, req *ConfigurationRequest) (*ConfigurationResponse, error) {

	configId := req.GetId()
	if configId == "" {
		return &ConfigurationResponse{
			Success: false,
			Message: "Configuration ID is empty",
		}, fmt.Errorf("configuration ID is empty")
	}

	cfg, err := storewrapper.GetConfiguration(configId)
	if err != nil {
		return &ConfigurationResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	if err := s.deployConfiguration(ctx, cfg); err != nil {
		return &ConfigurationResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}
	return &ConfigurationResponse{
		Success: true,
		Message: "Configuration applied successfully",
	}, nil
}

func (s *ConfigServiceServerImpl) deployConfiguration(ctx context.Context, cfg *topology_config.TopologyConfig) error {

	topo, err := storewrapper.GetTopology()
	if err != nil {
		return err
	}

	secret := os.Getenv("NETCONF_PASSWORD")

	return s.engine.ApplyConfiguration(
		topo,
		cfg,
		secret,
	)
}

// Optional: simple health check RPC.
func (s *ConfigServiceServerImpl) Ping(ctx context.Context, _ *ConfigurationRequest) (*ConfigurationResponse, error) {
	return &ConfigurationResponse{Success: true, Message: "pong"}, nil
}

func (s *ConfigServiceServerImpl) Rollback(ctx context.Context, req *RollbackRequest) (*ConfigurationResponse, error) {

	if s.obs != nil {
		s.obs.Println("[Config-Service] Received rollback request")

		_ = s.obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"config.rollback",
			"requested",
			observabilityv1.DomainResult_DOMAIN_RESULT_ACCEPTED,
			"configuration",
			"",
			"configuration rollback request received",
		)
	}

	if s.obs != nil {
		s.obs.Println("[Config-Service] Rolling back last configuration transaction...")
	}

	err := s.engine.Rollback()
	if err != nil {

		msg := fmt.Sprintf(
			"Configuration rollback failed: %v",
			err,
		)

		if s.obs != nil {
			s.obs.Println(msg)

			_ = s.obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"config.rollback",
				"failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"configuration",
				"",
				msg,
			)

			_ = s.obs.Audit(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"config-service",
				"rollback",
				"configuration",
				"",
				observabilityv1.AuditResult_AUDIT_RESULT_FAILED,
				err.Error(),
			)
		}

		return &ConfigurationResponse{
			Success: false,
			Message: msg,
		}, err
	}

	if s.obs != nil {
		s.obs.Println("[Config-Service] Configuration rollback completed successfully")

		_ = s.obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"config.rollback",
			"succeeded",
			observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
			"configuration",
			"",
			"configuration rolled back successfully",
		)

		_ = s.obs.Audit(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"config-service",
			"rollback",
			"configuration",
			"",
			observabilityv1.AuditResult_AUDIT_RESULT_SUCCEEDED,
			"",
		)
	}

	return &ConfigurationResponse{
		Success: true,
		Message: "Configuration rolled back successfully",
	}, nil
}

/*
// Helper: apply last known configuration at startup
func (s *ConfigServiceServerImpl) ApplyLastConfiguration() {
	_, id, err := storewrapper.GetLastConfiguration()
	if err != nil {
		if s.obs != nil {
			s.obs.Printf("No previous configuration found: %v", err)
		}
		return
	}

	if s.obs != nil {
		s.obs.Println("Loaded last known configuration from store")
	}
	req := &ConfigurationRequest{Id: &id}
	_, err = s.ApplyConfigurationById(context.Background(), req)
	if err != nil {
		if s.obs != nil {
			s.obs.Printf("Failed to apply last configuration: %v", err)
		}
	} else {
		if s.obs != nil {
			s.obs.Println("Successfully applied last configuration")
		}
	}

}
*/
