package service

import (
	"context"
	"fmt"
	"os"

	"OpenCNC_config_service/common/observability"
	storewrapper "OpenCNC_config_service/common/store-wrapper"
	observabilityv1 "OpenCNC_config_service/common/structures/logging"
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
func (s *ConfigServiceServerImpl) ApplyConfiguration(
	ctx context.Context,
	req *ConfigurationRequest,
) (*ConfigurationResponse, error) {

	if s.obs != nil {
		s.obs.Printf("[Config-Service] Received ApplyConfiguration request for ID: %s", req.Id)
		_ = s.obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"config.apply",
			"requested",
			observabilityv1.DomainResult_DOMAIN_RESULT_ACCEPTED,
			"configuration",
			req.Id,
			"configuration apply request received",
		)
	}

	secret := os.Getenv("NETCONF_PASSWORD")

	topoConf, err := storewrapper.GetConfiguration(req.Id)
	if err != nil {
		msg := fmt.Sprintf("Failed to get configuration from store: %v", err)
		if s.obs != nil {
			s.obs.Println(msg)
			_ = s.obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"config.apply",
				"store_fetch_failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"configuration",
				req.Id,
				msg,
			)
		}

		return &ConfigurationResponse{
			Success: false,
			Message: msg,
		}, err
	}

	topo, err := storewrapper.GetTopology()
	if err != nil {
		msg := fmt.Sprintf("Failed to get topology from store: %v", err)
		if s.obs != nil {
			s.obs.Println(msg)
			_ = s.obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"config.apply",
				"topology_fetch_failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"configuration",
				req.Id,
				msg,
			)
		}

		return &ConfigurationResponse{
			Success: false,
			Message: msg,
		}, err
	}

	if s.obs != nil {
		s.obs.Println("[Config-Service] Applying configuration through engine...")
	}

	err = s.engine.ApplyConfiguration(
		topo,
		topoConf,
		secret,
	)
	if err != nil {
		msg := fmt.Sprintf("Configuration apply failed: %v", err)
		if s.obs != nil {
			s.obs.Println(msg)
			_ = s.obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"config.apply",
				"failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"configuration",
				req.Id,
				msg,
			)
			_ = s.obs.Audit(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"config-service",
				"apply",
				"configuration",
				req.Id,
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
		s.obs.Println("[Config-Service] Configuration applied successfully!")
		_ = s.obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"config.apply",
			"succeeded",
			observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
			"configuration",
			req.Id,
			"configuration applied successfully",
		)
		_ = s.obs.Audit(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"config-service",
			"apply",
			"configuration",
			req.Id,
			observabilityv1.AuditResult_AUDIT_RESULT_SUCCEEDED,
			"",
		)
	}

	return &ConfigurationResponse{
		Success: true,
		Message: "Configuration applied successfully",
	}, nil
}

// Optional: simple health check RPC.
func (s *ConfigServiceServerImpl) Ping(ctx context.Context, _ *ConfigurationRequest) (*ConfigurationResponse, error) {
	return &ConfigurationResponse{Success: true, Message: "pong"}, nil
}

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
	resp, err := s.ApplyConfiguration(context.Background(), &ConfigurationRequest{Id: id})
	if err != nil {
		if s.obs != nil {
			s.obs.Printf("Failed to apply last configuration: %v", err)
		}
	} else {
		if s.obs != nil {
			s.obs.Println("Successfully applied last configuration:", resp.Message)
		}
	}

}
