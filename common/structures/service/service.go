package service

import (
	"context"
	"fmt"
	"log"
	"os"

	"OpenCNC_config_service/config_service/pkg/engine"
	storewrapper "OpenCNC_config_service/common/store-wrapper"
)

// ConfigServiceServer implements the generated gRPC interface.
type ConfigServiceServerImpl struct {
	UnimplementedConfigServiceServer
	logger *log.Logger
	engine *engine.MappingEngine
}

// Constructor
func NewConfigServiceServerImpl(logger *log.Logger, engine *engine.MappingEngine) *ConfigServiceServerImpl {
	return &ConfigServiceServerImpl{logger: logger, engine: engine}
}

// ApplyConfiguration receives a config request ID, retrieves it from store,
// maps it via the plugin, and pushes it to all NETCONF-capable devices.
func (s *ConfigServiceServerImpl) ApplyConfiguration(
	ctx context.Context,
	req *ConfigurationRequest,
) (*ConfigurationResponse, error) {

	s.logger.Printf("[Config-Service] Received ApplyConfiguration request for ID: %s", req.Id)

	secret := os.Getenv("NETCONF_PASSWORD")

	topoConf, err := storewrapper.GetConfiguration(req.Id)
	if err != nil {
		msg := fmt.Sprintf("Failed to get configuration from store: %v", err)
		s.logger.Println(msg)

		return &ConfigurationResponse{
			Success: false,
			Message: msg,
		}, err
	}

	topo, err := storewrapper.GetTopology()
	if err != nil {
		msg := fmt.Sprintf("Failed to get topology from store: %v", err)
		s.logger.Println(msg)

		return &ConfigurationResponse{
			Success: false,
			Message: msg,
		}, err
	}

	s.logger.Println("[Config-Service] Applying configuration through engine...")

	err = s.engine.ApplyConfiguration(
		topo,
		topoConf,
		secret,
	)
	if err != nil {
		msg := fmt.Sprintf("Configuration apply failed: %v", err)
		s.logger.Println(msg)

		return &ConfigurationResponse{
			Success: false,
			Message: msg,
		}, err
	}

	s.logger.Println("[Config-Service] Configuration applied successfully!")

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
		s.logger.Printf("No previous configuration found: %v", err)
		return
	}

	s.logger.Println("Loaded last known configuration from store")
	resp, err := s.ApplyConfiguration(context.Background(), &ConfigurationRequest{Id: id})
	if err != nil {
		s.logger.Printf("Failed to apply last configuration: %v", err)
	} else {
		s.logger.Println("Successfully applied last configuration:", resp.Message)
	}

}
