package service

import (
	"context"
	"fmt"

	"OpenCNC/common/observability"
	storewrapper "OpenCNC/common/store-wrapper"
	observabilityv1 "OpenCNC/common/structures/logging"
	"OpenCNC/monitor_service/pkg/engine"
	monitoring "OpenCNC/monitor_service/structures/monitoring"

	"google.golang.org/protobuf/proto"
)

type MonitorServer struct {
	UnimplementedMonitorServiceServer
	engine *engine.Engine
	obs    *observability.Client
}

func NewMonitorServer(engine *engine.Engine, obs *observability.Client) *MonitorServer {
	return &MonitorServer{
		engine: engine,
		obs:    obs,
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

	var err error
	defer func() {
		if err != nil && s.obs != nil {
			_ = s.obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"monitoring",
				"start_failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"monitoring_resource",
				req.Id,
				fmt.Sprintf("monitoring start failed: %v", err),
			)
		}
	}()

	if req.Resource == nil {
		err = fmt.Errorf("monitoring resource is required")
		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
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
		err = fmt.Errorf("target port not found!")
		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if len(req.Settings) == 0 {
		err = fmt.Errorf("no monitoring items specified")
		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// get node credentials
	creds, err := storewrapper.GetCredentials(req.Resource.NodeId)
	if err != nil {
		return &MonitoringResponse{
			Success: false,
			Message: "No credentials found for node: " + err.Error(),
		}, nil
	}

	counters, metrics, err := s.parseMonitoringSettings(req.Settings)
	if err != nil {
		return &MonitoringResponse{
			Success: false,
			Message: "failed to parse monitoring settings: " + err.Error(),
		}, nil
	}

	if err := s.engine.StartMonitoring(
		req.Resource,
		node,
		creds,
		counters,
		metrics,
	); err != nil {
		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if s.obs != nil {
		_ = s.obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"monitoring",
			"started",
			observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
			"monitoring_resource",
			req.Id,
			"monitoring started",
		)
	}

	return &MonitoringResponse{
		Success: true,
		Message: "monitoring started",
	}, nil
}

func (s *MonitorServer) parseMonitoringSettings(items []*MonitoringSetting) ([]*monitoring.Counter, []*monitoring.Metric, error) {
	counters := []*monitoring.Counter{}
	metrics := []*monitoring.Metric{}

	for _, item := range items {
		capability := s.engine.Catalog.GetItem(item.CapabilityId)
		if capability == nil {
			return nil, nil, fmt.Errorf("monitoring item %q not found in catalog", item.CapabilityId)
		}
		//counters
		if capability.Kind == monitoring.DataType_RAW {
			counter := proto.Clone(
				s.engine.Catalog.GetCounter(capability.Name),
			).(*monitoring.Counter)
			if counter == nil {
				return nil, nil, fmt.Errorf("item %q is not a counter", capability.Name)
			}
			counter.PollIntervalMs = item.PollIntervalMs
			counters = append(counters, counter)
		}
		//metrics
		if capability.Kind == monitoring.DataType_METRIC {
			metric := proto.Clone(
				s.engine.Catalog.GetMetricByID(capability.Name),
			).(*monitoring.Metric)
			if metric == nil {
				return nil, nil, fmt.Errorf("item %q is not a metric", capability.Name)
			}
			metric.EvaluationIntervalMs = item.PollIntervalMs
			metrics = append(metrics, metric)
		}
	}
	return counters, metrics, nil
}

func (s *MonitorServer) StopMonitoring(ctx context.Context, req *StopMonitoringRequest) (*MonitoringResponse, error) {

	if req == nil || req.Id == "" {
		return &MonitoringResponse{
			Success: false,
			Message: "monitoring id is required",
		}, nil
	}

	err := s.engine.StopMonitoring(req.Id)
	if err != nil {
		if s.obs != nil {
			_ = s.obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"monitoring",
				"stop_failed",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"monitoring_resource",
				req.Id,
				fmt.Sprintf("monitoring stop failed: %v", err),
			)
		}

		return &MonitoringResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if s.obs != nil {
		_ = s.obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"monitoring",
			"stopped",
			observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
			"monitoring_resource",
			req.Id,
			"monitoring stopped",
		)
	}

	return &MonitoringResponse{
		Success: true,
		Message: "monitoring stopped",
	}, nil
}

// ///////////////////////
func (s *MonitorServer) TestRollback(ctx context.Context, req *TestRollbackRequest) (*TestRollbackResponse, error) {
	if req == nil || req.Id == "" {
		return &TestRollbackResponse{
			Success: false,
			Message: "rollback id is required",
		}, nil
	}

	event := &monitoring.MonitoringEvent{
		Source:   &monitoring.ResourceKey{NodeId: req.NodeName, PortId: &req.PortName},
		Type:     monitoring.EventType_TYPE_UNSPECIFIED,
		Severity: monitoring.Severity_CRITICAL,
		Actions:  []monitoring.EventAction{monitoring.EventAction_REQUEST_ROLLBACK},
	}

	if err := engine.HandleRequestRollback(event, s.obs); err != nil {
		return &TestRollbackResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &TestRollbackResponse{
		Success: true,
		Message: "rollback test succeeded",
	}, nil
}