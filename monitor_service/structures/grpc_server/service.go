package service

import (
	"context"

	storewrapper "OpenCNC_config_service/common/store-wrapper"
	"OpenCNC_config_service/monitor_service/pkg/engine"
	monitoring "OpenCNC_config_service/monitor_service/structures/monitoring"
)

type MonitorServer struct {
	UnimplementedMonitorServiceServer
	engine *engine.Engine
}

func NewMonitorServer(engine *engine.Engine) *MonitorServer {
	return &MonitorServer{
		engine: engine,
	}
}

func (s *MonitorServer) GetCapabilities(ctx context.Context, req *CapabilitiesRequest) (*CapabilitiesResponse, error) {

	response := &CapabilitiesResponse{}

	if req.Resource == nil {
		// No target specified → all targets
		// Add catalog items to the response
		for _, item := range s.engine.Catalog.Items {
			response.Capabilities = append(
				response.Capabilities,
				&monitoring.Capability{
					Name:        item.Name,
					Description: item.Description,
					Kind:        item.Kind,
				},
			)
		}
	} else {
		// Specific target
		capabilities := s.engine.Catalog.GetItemsByResources(req.Resource)
		for i := range capabilities {
			item := &capabilities[i]
			response.Capabilities = append(
				response.Capabilities,
				&monitoring.Capability{
					Name:        item.Name,
					Description: item.Description,
					Kind:        item.Kind,
				},
			)
		}

	}

	return response, nil
}

func (s *MonitorServer) StartMonitoring(ctx context.Context, req *StartMonitoringRequest) (*MonitoringResponse, error) {

	if req == nil || req.Id == "" {
		return &MonitoringResponse{
			Success: false,
			Message: "monitoring id is required",
		}, nil
	}

	if req.Resource == nil {
		return &MonitoringResponse{
			Success: false,
			Message: "monitoring resource is required",
		}, nil
	}

	//check target
	node, _, err := storewrapper.GetNode(req.Resource.NodeId)
	if err != nil {
		return &MonitoringResponse{
			Success: false,
			Message: "target node not found!",
		}, nil
	}

	portExist := false
	for _, port := range node.Ports {
		if port.Id == *req.Resource.PortId {
			portExist = true
			break
		}
	}

	if !portExist && *req.Resource.PortId != "" {
		return &MonitoringResponse{
			Success: false,
			Message: "target port not found!",
		}, nil
	}

	if len(req.Items) == 0 {
		return &MonitoringResponse{
			Success: false,
			Message: "no monitoring items specified",
		}, nil
	}

	if err := s.engine.StartMonitoring(req.Id, req.Resource, node, req.Items); err != nil {
		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &MonitoringResponse{
		Success: true,
		Message: "monitoring started",
	}, nil
}

func (s *MonitorServer) StopMonitoring(ctx context.Context, req *StopMonitoringRequest) (*MonitoringResponse, error) {

	if req == nil || req.Id == "" {
		return &MonitoringResponse{
			Success: false,
			Message: "monitoring id is required",
		}, nil
	}

	if err := s.engine.StopMonitoring(req.Id); err != nil {
		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &MonitoringResponse{
		Success: true,
		Message: "monitoring stopped",
	}, nil
}
