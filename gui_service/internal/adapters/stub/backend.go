package stub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	storewrapper "OpenCNC_config_service/common/store-wrapper"
	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/gui_service/internal/domain"
	"OpenCNC_config_service/monitor_service/structures/monitoring"

	"google.golang.org/protobuf/encoding/protojson"
)

type uploadWrapper struct {
	Models []uploadModel `json:"models"`
}

func (u *uploadWrapper) UnmarshalJSON(data []byte) error {
	type rawWrapper struct {
		Models []uploadModel `json:"models"`
	}

	var wrapped rawWrapper
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Models) > 0 {
		u.Models = wrapped.Models
		return nil
	}

	var model uploadModel
	if err := json.Unmarshal(data, &model); err == nil && strings.TrimSpace(model.Name) != "" {
		u.Models = []uploadModel{model}
		return nil
	}

	return json.Unmarshal(data, &wrapped)
}

type uploadModel struct {
	Name      string       `json:"model-name"`
	YangFiles []uploadYang `json:"yang-files"`
}

type uploadYang struct {
	Name      string `json:"file-name"`
	Revision  string `json:"file-revision"`
	Structure string `json:"structure"`
}

func (u *uploadYang) UnmarshalJSON(data []byte) error {
	type rawYang struct {
		Name        string `json:"file-name"`
		Revision    string `json:"file-revision"`
		Structure   string `json:"structure"`
		Description string `json:"description"`
	}

	var parsed rawYang
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	u.Name = parsed.Name
	u.Revision = parsed.Revision
	u.Structure = parsed.Structure
	if strings.TrimSpace(u.Structure) == "" {
		u.Structure = parsed.Description
	}
	return nil
}

type Backend struct {
	state *MonitoringState
}

var activityLog = struct {
	mu      sync.RWMutex
	entries []domain.EventLog
}{}

const maxActivityLogEntries = 100

func NewBackend(state *MonitoringState) *Backend {
	return &Backend{
		state: state,
	}
}

func (b *Backend) GetDashboard(r context.Context) (domain.DashboardData, error) {
	nodes, err := b.GetNodes(r)
	if err != nil {
		return domain.DashboardData{}, err
	}
	links, err := b.GetLinks(r)
	if err != nil {
		return domain.DashboardData{}, err
	}

	return domain.DashboardData{
		PacketsForwarded: "12,458",
		PacketDrops:      "37",
		ActiveStreams:    "8",
		TopTalkers: []string{
			"",
		},
		NetworkModel: domain.NetworkModel{
			Nodes: nodes,
			Links: links,
		},
		/*
			NetworkModel: domain.NetworkModel{
					Nodes: []domain.Node{
						{
							ID:      "node-001",
							Name:    "CNC Controller",
							Type:    "controller",
							State:   "online",
							PortIds: []string{"4"},
							Links:   "3",
						},
						{
							ID:      "node-002",
							Name:    "Machine 01",
							Type:    "cnc-machine",
							State:   "online",
							PortIds: []string{"2"},
							Links:   "2",
						},
						{
							ID:      "node-003",
							Name:    "Operator Station",
							Type:    "workstation",
							State:   "online",
							PortIds: []string{"1"},
							Links:   "1",
						},
						{
							ID:      "node-004",
							Name:    "PLC Controller",
							Type:    "plc",
							State:   "online",
							PortIds: []string{"8"},
							Links:   "1",
						},
					},

					Links: []domain.Link{
						{
							ID:          "link-001",
							Source:      "node-001",
							Destination: "node-002",
							State:       "active",
							Bandwidth:   "100 Mbps",
						},
						{
							ID:          "link-002",
							Source:      "node-001",
							Destination: "node-003",
							State:       "active",
							Bandwidth:   "1 Gbps",
						},
						{
							ID:          "link-003",
							Source:      "node-001",
							Destination: "node-004",
							State:       "active",
							Bandwidth:   "1 Gbps",
						},
					},
			},
		*/
	}, nil
}

func (b *Backend) GetDeviceModels(_ context.Context) ([]domain.DeviceModel, error) {
	registry, err := storewrapper.GetDeviceModelRegistry()
	if err != nil {
		return []domain.DeviceModel{}, err
	}

	models := make([]domain.DeviceModel, 0, len(registry.DeviceModels))
	for _, model := range registry.DeviceModels {
		if model == nil || model.GetName() == "" {
			continue
		}

		yangNames := make([]string, 0, len(model.GetYangFiles()))
		for _, yf := range model.GetYangFiles() {
			if yf == nil || yf.GetName() == "" {
				continue
			}
			yangNames = append(yangNames, yf.GetName())
		}

		models = append(models, domain.DeviceModel{
			ID:      model.GetName(),
			Name:    model.GetName(),
			Version: "-",
			Vendor:  "-",
			Yang:    strings.Join(yangNames, ", "),
		})
	}

	return models, nil
}

func (b *Backend) GetMonitoringCounters(_ context.Context) ([]domain.MonitoringItem, error) {
	resp, err := requestGetCapabilities(nil)
	if err != nil {
		fmt.Println(err)
	}
	counters := make([]domain.MonitoringItem, 0)
	for _, capability := range resp.GetCapabilities() {
		if capability.GetKind() == monitoring.DataType_RAW {
			counters = append(counters, domain.MonitoringItem{ID: capability.GetName(), Label: capability.GetName(), Description: capability.GetDescription()})
		}
	}
	return counters, nil
}

func (b *Backend) GetMonitoringMetrics(_ context.Context) ([]domain.MonitoringItem, error) {
	resp, err := requestGetCapabilities(nil)
	if err != nil {
		fmt.Println(err)
	}
	metrics := make([]domain.MonitoringItem, 0)
	for _, capability := range resp.GetCapabilities() {
		if capability.GetKind() == monitoring.DataType_METRIC {
			metrics = append(metrics, domain.MonitoringItem{ID: capability.GetName(), Label: capability.GetName(), Description: capability.GetDescription()})
		}
	}
	return metrics, nil
}

func (b *Backend) GetMonitoringTargets(r context.Context) ([]domain.MonitoringTarget, error) {
	nodes, err := b.GetNodes(r)
	if err != nil {
		return nil, err
	}
	targets := make([]domain.MonitoringTarget, 0)
	for _, node := range nodes {
		// Node counters and metrics.
		target := domain.MonitoringTarget{
			ID:       node.ID,
			Node:     node.Name,
			Port:     "",
			Label:    node.Name,
			Counters: []string{},
			Metrics:  []string{},
		}
		resource := monitoring.ResourceKey{
			NodeId: node.ID,
			PortId: nil,
		}
		resp, err := requestGetCapabilities(&resource)
		if err != nil {
			fmt.Println(err)
		}
		for _, capability := range resp.GetCapabilities() {
			if capability.GetKind() == monitoring.DataType_RAW {
				target.Counters = append(target.Counters, capability.GetName())
			}
			if capability.GetKind() == monitoring.DataType_METRIC {
				target.Metrics = append(target.Metrics, capability.GetName())
			}
		}
		targets = append(targets, target)

		// Ports counters and metrics.
		for _, port := range node.PortIds {

			target := domain.MonitoringTarget{
				ID:       fmt.Sprintf("%s_%s", node.ID, port),
				Node:     node.Name,
				Port:     port,
				Label:    fmt.Sprintf("%s - %s", node.Name, port),
				Counters: []string{},
				Metrics:  []string{},
			}
			// get specific capabilities for this resource

			resource := monitoring.ResourceKey{
				NodeId: node.ID,
				PortId: &port,
			}
			resp, err := requestGetCapabilities(&resource)
			if err != nil {
				fmt.Println(err)
			}
			for _, capability := range resp.GetCapabilities() {
				if capability.GetKind() == monitoring.DataType_RAW {
					target.Counters = append(target.Counters, capability.GetName())
				}
				if capability.GetKind() == monitoring.DataType_METRIC {
					target.Metrics = append(target.Metrics, capability.GetName())
				}
			}
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func formatMetricValue(data MetricData) string {
	return fmt.Sprintf("%v", data.Value)
}

func (b *Backend) GetMonitoringData(_ context.Context, query domain.MonitoringDataQuery) ([]domain.MonitoringTargetData, error) {
	//fmt.Printf("[GUI] GetMonitoringData called with query: %+v\n", query)
	//Ask monitor-service to StartMonitoring().
	parts := strings.Split(query.TargetID, "_")
	resource := &monitoring.ResourceKey{
		NodeId: parts[0],
		PortId: &parts[1],
	}
	_, err := requestStartMonitoring(query.TargetID, resource, query.MetricIDs)
	if err != nil {
		return nil, err
	}
	// Update the shared state to let the kafka consumer collect the requested metrics.
	b.state.SetWantedMetrics(query.MetricIDs)

	// Read the latest values currently available.
	rtData := map[string]map[string]string{
		query.TargetID: {},
	}
	for _, metricID := range query.MetricIDs {
		data, ok := b.state.GetMetric(metricID)
		if !ok {
			continue
		}

		rtData[query.TargetID][metricID] = formatMetricValue(data)
	}

	// TODO replace it with Kafka consumer to get real-time data from monitor service
	rawValues := map[string]map[string]string{
		"D_Port1": {
			"rx_packets":          "1500",
			"tx_packets":          "18",
			"rx_drops":            "2",
			"tx_drops":            "1",
			"unicast-packet-rate": "41%",
			"latency_mean":        "0.8 ms",
			"packet_loss_rate":    "0.12%",
		},
		"sw1_sw0p4": {
			"rx_packets":       "700",
			"tx_packets":       "1498",
			"bandwidth_usage":  "72%",
			"latency_mean":     "1.3 ms",
			"packet_loss_rate": "0.08%",
		},
		"sw2_sw0p1": {
			"rx_packets":        "924",
			"tx_packets":        "907",
			"stream_rejections": "5",
			"bandwidth_usage":   "58%",
			"packet_loss_rate":  "0.20%",
		},
	}

	targets, err := b.GetMonitoringTargets(context.Background())
	if err != nil {
		return nil, err
	}

	counters, err := b.GetMonitoringCounters(context.Background())
	if err != nil {
		return nil, err
	}

	metrics, err := b.GetMonitoringMetrics(context.Background())
	if err != nil {
		return nil, err
	}

	labelByID := map[string]string{}
	for _, item := range counters {
		labelByID[item.ID] = item.Label
	}
	for _, item := range metrics {
		labelByID[item.ID] = item.Label
	}

	metricIDs := query.MetricIDs
	if len(metricIDs) == 0 {
		metricIDs = []string{"rx_packets", "tx_packets"}
	}

	filtered := make([]domain.MonitoringTargetData, 0)
	for _, target := range targets {
		if query.TargetID != "" && target.ID != query.TargetID {
			continue
		}
		if query.Node != "" && !strings.EqualFold(target.Node, query.Node) {
			continue
		}

		values := make([]domain.MonitoringValue, 0)
		for _, metricID := range metricIDs {
			valueByMetric, found := rawValues[target.ID]
			if !found {
				continue
			}

			value, ok := valueByMetric[metricID]
			if !ok {
				continue
			}

			values = append(values, domain.MonitoringValue{
				ID:    metricID,
				Label: labelByID[metricID],
				Value: value,
			})
		}

		filtered = append(filtered, domain.MonitoringTargetData{
			TargetID: target.ID,
			Label:    target.Label,
			Node:     target.Node,
			Port:     target.Port,
			Values:   values,
		})
	}

	return filtered, nil
}

func (b *Backend) GetNodes(_ context.Context) ([]domain.Node, error) {
	nodes, err := storewrapper.GetNodes()
	if err != nil {
		return nil, err
	}
	domainNodes := make([]domain.Node, len(nodes))

	for i, n := range nodes {

		domainNodes[i] = domain.Node{
			ID:      n.Name,
			Name:    n.Name,
			Type:    n.Type.String(),
			PortIds: []string{},
		}
		for _, port := range n.GetPorts() {
			domainNodes[i].PortIds = append(domainNodes[i].PortIds, port.GetId())
		}
	}
	return domainNodes, nil
}

func (b *Backend) GetLinks(_ context.Context) ([]domain.Link, error) {
	links, err := storewrapper.GetLinks()
	if err != nil {
		return nil, err
	}
	domainLinks := make([]domain.Link, len(links))
	for i, link := range links {
		domainLinks[i] = domain.Link{
			ID:          link.Id,
			Source:      link.SourceNode,
			Destination: link.DestinationNode,
			Bandwidth:   strconv.FormatInt(link.BandwidthMbps*1000000, 10),
		}
	}
	return domainLinks, nil
}

func (b *Backend) GetStreams(_ context.Context) ([]domain.Stream, error) {
	return []domain.Stream{}, nil
}

func (b *Backend) GetLogs(_ context.Context, query domain.LogsQuery) ([]domain.EventLog, error) {
	activityLog.mu.RLock()
	entries := append([]domain.EventLog(nil), activityLog.entries...)
	activityLog.mu.RUnlock()

	if severity := strings.TrimSpace(query.Severity); severity != "" {
		filtered := make([]domain.EventLog, 0, len(entries))
		for _, entry := range entries {
			if strings.EqualFold(entry.Severity, severity) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	if strings.EqualFold(query.OrderBy, "severity") {
		severityRank := map[string]int{"error": 0, "warning": 1, "info": 2}
		sort.SliceStable(entries, func(i, j int) bool {
			left := severityRank[strings.ToLower(entries[i].Severity)]
			right := severityRank[strings.ToLower(entries[j].Severity)]
			if left == right {
				return entries[i].Time > entries[j].Time
			}
			return left < right
		})
		return entries, nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Time > entries[j].Time
	})
	return entries, nil
}

func (b *Backend) GetRecentEvents(_ context.Context) ([]domain.EventLog, error) {
	entries, err := b.GetLogs(context.Background(), domain.LogsQuery{OrderBy: "time"})
	if err != nil {
		return nil, err
	}
	if len(entries) > 8 {
		return entries[:8], nil
	}
	return entries, nil
}

func (b *Backend) RefreshData(_ context.Context) (domain.OperationResult, error) {
	return b.result("refreshData", map[string]string{"result": "dashboard refresh requested"}), nil
}

func (b *Backend) UploadModel(_ context.Context, query string) (domain.OperationResult, error) {
	if strings.TrimSpace(query) == "" {
		return domain.OperationResult{
			Success: false,
			Name:    "uploadModel",
			Message: "no JSON payload found; choose a model file first",
			Data:    map[string]string{},
		}, nil
	}

	var wrapper uploadWrapper
	if err := json.Unmarshal([]byte(query), &wrapper); err != nil {
		return domain.OperationResult{
			Success: false,
			Name:    "uploadModel",
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Data:    map[string]string{},
		}, nil
	}

	if len(wrapper.Models) == 0 {
		return domain.OperationResult{
			Success: false,
			Name:    "uploadModel",
			Message: "no device models found in payload",
			Data:    map[string]string{},
		}, nil
	}

	stored := 0
	skipped := []string{}

	for _, modelInput := range wrapper.Models {
		if strings.TrimSpace(modelInput.Name) == "" {
			skipped = append(skipped, "<empty-name>")
			b.recordEvent("warning", "uploadModel", "device-models", "Skipped model upload with empty name", "")
			continue
		}

		model := &devicemodelregistry.DeviceModel{Name: modelInput.Name}
		for _, yang := range modelInput.YangFiles {
			if strings.TrimSpace(yang.Name) == "" {
				continue
			}
			model.YangFiles = append(model.YangFiles, &devicemodelregistry.YangFile{
				Name:        yang.Name,
				Revision:    yang.Revision,
				Description: yang.Structure,
			})
		}

		if err := storewrapper.StoreDeviceModel(model); err != nil {
			log.Printf("[GUI] failed storing model %s: %v", modelInput.Name, err)
			skipped = append(skipped, modelInput.Name)
			b.recordEvent("error", "uploadModel", "device-models", fmt.Sprintf("Failed storing model %s", modelInput.Name), modelInput.Name)
			continue
		}
		b.recordEvent("info", "uploadModel", "device-models", fmt.Sprintf("Stored model %s", modelInput.Name), modelInput.Name)
		stored++
	}

	if stored == 0 {
		return domain.OperationResult{
			Success: false,
			Name:    "uploadModel",
			Message: "no models stored",
			Data: map[string]string{
				"stored":  "0",
				"skipped": strings.Join(skipped, ","),
			},
		}, nil
	}

	return domain.OperationResult{
		Success: true,
		Name:    "uploadModel",
		Message: fmt.Sprintf("%d model(s) stored", stored),
		Data: map[string]string{
			"stored":  fmt.Sprintf("%d", stored),
			"skipped": strings.Join(skipped, ","),
		},
	}, nil
}

func (b *Backend) EditModel(_ context.Context, modelID, name, version, vendor, yang string) (domain.OperationResult, error) {
	if strings.TrimSpace(modelID) == "" {
		b.recordEvent("warning", "editModel", "device-models", "Edit requested without selecting a model", "")
		return domain.OperationResult{
			Success: false,
			Name:    "editModel",
			Message: "no model selected",
			Data:    map[string]string{},
		}, nil
	}

	current, err := storewrapper.GetDeviceModel(modelID)
	if err != nil {
		b.recordEvent("error", "editModel", "device-models", fmt.Sprintf("Model %s not found", modelID), modelID)
		return domain.OperationResult{
			Success: false,
			Name:    "editModel",
			Message: fmt.Sprintf("model %s not found: %v", modelID, err),
			Data:    map[string]string{"modelId": modelID},
		}, nil
	}

	if strings.TrimSpace(name) != "" && strings.TrimSpace(name) != current.GetName() {
		previousName := current.GetName()
		current.Name = name
		if err := storewrapper.DeleteDeviceModel(modelID); err != nil {
			b.recordEvent("error", "editModel", "device-models", fmt.Sprintf("Failed renaming model %s", modelID), modelID)
			return domain.OperationResult{
				Success: false,
				Name:    "editModel",
				Message: fmt.Sprintf("failed to rename model %s: %v", modelID, err),
				Data:    map[string]string{"modelId": modelID},
			}, nil
		}
		if err := storewrapper.StoreDeviceModel(current); err != nil {
			b.recordEvent("error", "editModel", "device-models", fmt.Sprintf("Failed saving renamed model %s", modelID), modelID)
			return domain.OperationResult{
				Success: false,
				Name:    "editModel",
				Message: fmt.Sprintf("failed to save renamed model %s: %v", modelID, err),
				Data:    map[string]string{"modelId": modelID},
			}, nil
		}
		b.recordEvent("info", "editModel", "device-models", fmt.Sprintf("Renamed model %s to %s", previousName, current.GetName()), current.GetName())
	} else {
		b.recordEvent("info", "editModel", "device-models", fmt.Sprintf("Updated model %s", current.GetName()), current.GetName())
	}

	return domain.OperationResult{
		Success: true,
		Name:    "editModel",
		Message: fmt.Sprintf("model %s updated", current.GetName()),
		Data: map[string]string{
			"modelId": current.GetName(),
			"name":    current.GetName(),
			"version": version,
			"vendor":  vendor,
			"yang":    yang,
		},
	}, nil
}

func (b *Backend) DeleteModel(_ context.Context, modelID string) (domain.OperationResult, error) {
	if strings.TrimSpace(modelID) == "" {
		b.recordEvent("warning", "deleteModel", "device-models", "Delete requested without selecting a model", "")
		return domain.OperationResult{
			Success: false,
			Name:    "deleteModel",
			Message: "no model selected",
			Data:    map[string]string{},
		}, nil
	}

	if err := storewrapper.DeleteDeviceModel(modelID); err != nil {
		b.recordEvent("error", "deleteModel", "device-models", fmt.Sprintf("Failed deleting model %s", modelID), modelID)
		return domain.OperationResult{
			Success: false,
			Name:    "deleteModel",
			Message: fmt.Sprintf("failed to delete model %s: %v", modelID, err),
			Data:    map[string]string{"modelId": modelID},
		}, nil
	}
	b.recordEvent("info", "deleteModel", "device-models", fmt.Sprintf("Deleted model %s", modelID), modelID)

	return domain.OperationResult{
		Success: true,
		Name:    "deleteModel",
		Message: fmt.Sprintf("model %s deleted", modelID),
		Data:    map[string]string{"modelId": modelID},
	}, nil
}

func (b *Backend) UploadTopology(_ context.Context, query string) (domain.OperationResult, error) {
	if strings.TrimSpace(query) == "" {
		return domain.OperationResult{
			Success: false,
			Name:    "uploadTopology",
			Message: "no JSON payload found; choose a topology file first",
			Data:    map[string]string{},
		}, nil
	}

	topologyObj := &topology.Topology{}

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	if err := unmarshaler.Unmarshal([]byte(query), topologyObj); err != nil {
		return domain.OperationResult{
			Success: false,
			Name:    "uploadTopology",
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Data:    map[string]string{},
		}, nil
	}

	if err := storewrapper.DeleteTopology(); err != nil {
		fmt.Printf("Failed to delete topology: %v\n", err)
	}

	if err := storewrapper.StoreTopology(topologyObj); err != nil {
		b.recordEvent(
			"error",
			"uploadTopology",
			"topology",
			fmt.Sprintf("Failed storing topology: %v", err),
			"",
		)

		return domain.OperationResult{
			Success: false,
			Name:    "uploadTopology",
			Message: fmt.Sprintf("failed to store topology: %v", err),
			Data:    map[string]string{},
		}, nil
	}

	b.recordEvent(
		"info",
		"uploadTopology",
		"topology",
		"Topology imported successfully",
		"",
	)

	return domain.OperationResult{
		Success: true,
		Name:    "uploadTopology",
		Message: "topology imported successfully",
		Data: map[string]string{
			"status": "stored",
		},
	}, nil
}

func (b *Backend) AddNode(_ context.Context, query string) (domain.OperationResult, error) {
	if strings.TrimSpace(query) == "" {
		return domain.OperationResult{
			Success: false,
			Name:    "addNode",
			Message: "no JSON payload found; choose a node file first",
			Data:    map[string]string{},
		}, nil
	}

	var node topology.Node
	if err := protojson.Unmarshal([]byte(query), &node); err != nil {
		return domain.OperationResult{
			Success: false,
			Name:    "addNode",
			Message: fmt.Sprintf("failed to parse node JSON: %v", err),
			Data:    map[string]string{},
		}, nil
	}

	if strings.TrimSpace(node.GetName()) == "" {
		return domain.OperationResult{
			Success: false,
			Name:    "addNode",
			Message: "node name cannot be empty",
			Data:    map[string]string{},
		}, nil
	}

	if err := storewrapper.StoreNode(&node); err != nil {
		return domain.OperationResult{
			Success: false,
			Name:    "addNode",
			Message: fmt.Sprintf("failed to store node %s: %v", node.GetName(), err),
			Data:    map[string]string{"nodeId": node.GetName()},
		}, nil
	}

	b.recordEvent("info", "addNode", "topology", fmt.Sprintf("Stored node %s", node.GetName()), node.GetName())

	return domain.OperationResult{
		Success: true,
		Name:    "addNode",
		Message: fmt.Sprintf("node %s stored", node.GetName()),
		Data: map[string]string{
			"nodeId":   node.GetName(),
			"nodeType": node.GetType().String(),
		},
	}, nil
}

func parseNodeRole(raw string) (topology.NodeRole, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	switch normalized {
	case "", "UNKNOWN":
		return topology.NodeRole_UNKNOWN, true
	case "END_STATION", "ENDNODE", "END_NODE":
		return topology.NodeRole_END_STATION, true
	case "BRIDGE":
		return topology.NodeRole_BRIDGE, true
	case "BRIDGED_END_STATION", "BRIDGED_ENDNODE", "BRIDGED_END_NODE":
		return topology.NodeRole_BRIDGED_END_STATION, true
	default:
		return topology.NodeRole_UNKNOWN, false
	}
}

func (b *Backend) EditNode(_ context.Context, nodeID, name, nodeType, state, ports, links string) (domain.OperationResult, error) {
	if strings.TrimSpace(nodeID) == "" {
		b.recordEvent("warning", "editNode", "nodes", "Edit requested without selecting a node", "")
		return domain.OperationResult{
			Success: false,
			Name:    "editNode",
			Message: "no node selected",
			Data:    map[string]string{},
		}, nil
	}

	current, _, err := storewrapper.GetNode(nodeID)
	if err != nil {
		b.recordEvent("error", "editNode", "nodes",
			fmt.Sprintf("Node %s not found", nodeID), nodeID)

		return domain.OperationResult{
			Success: false,
			Name:    "editNode",
			Message: fmt.Sprintf("node %s not found: %v", nodeID, err),
			Data:    map[string]string{"nodeId": nodeID},
		}, nil
	}

	previousName := current.GetName()
	name = strings.TrimSpace(name)
	nodeType = strings.TrimSpace(nodeType)

	if name != "" {
		current.Name = name
	}

	if nodeType != "" {
		parsedRole, ok := parseNodeRole(nodeType)
		if !ok {
			b.recordEvent("warning", "editNode", "nodes", fmt.Sprintf("Invalid node type for %s: %s", nodeID, nodeType), nodeID)
			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf("invalid node type: %s", nodeType),
				Data:    map[string]string{"nodeId": nodeID},
			}, nil
		}
		current.Type = parsedRole
	}

	if previousName != current.GetName() {
		if err := storewrapper.DeleteNode(nodeID); err != nil {
			b.recordEvent("error", "editNode", "nodes", fmt.Sprintf("Failed renaming node %s", nodeID), nodeID)
			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf("failed to rename node %s: %v", nodeID, err),
				Data:    map[string]string{"nodeId": nodeID},
			}, nil
		}

		if err := storewrapper.StoreNode(current); err != nil {
			b.recordEvent("error", "editNode", "nodes", fmt.Sprintf("Failed saving renamed node %s", current.GetName()), current.GetName())
			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf("failed to save renamed node %s: %v", current.GetName(), err),
				Data:    map[string]string{"nodeId": current.GetName()},
			}, nil
		}

		b.recordEvent("info", "editNode", "nodes", fmt.Sprintf("Renamed node %s to %s", previousName, current.GetName()), current.GetName())
	} else {
		if err := storewrapper.EditNode(current); err != nil {
			b.recordEvent("error", "editNode", "nodes", fmt.Sprintf("Failed saving updated node %s", current.GetName()), current.GetName())
			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf("failed to save updated node %s: %v", current.GetName(), err),
				Data:    map[string]string{"nodeId": current.GetName()},
			}, nil
		}

		b.recordEvent("info", "editNode", "nodes", fmt.Sprintf("Updated node %s", current.GetName()), current.GetName())
	}

	return domain.OperationResult{
		Success: true,
		Name:    "editNode",
		Message: fmt.Sprintf("node %s updated", current.GetName()),
		Data: map[string]string{
			"nodeId": current.GetName(),
			"name":   current.GetName(),
			"type":   current.GetType().String(),
			"state":  state,
			"ports":  ports,
			"links":  links,
		},
	}, nil
}

func (b *Backend) DeleteNode(_ context.Context, nodeID string) (domain.OperationResult, error) {
	if strings.TrimSpace(nodeID) == "" {
		b.recordEvent("warning", "deleteNode", "nodes",
			"Delete requested without selecting a node", "")

		return domain.OperationResult{
			Success: false,
			Name:    "deleteNode",
			Message: "no node selected",
			Data:    map[string]string{},
		}, nil
	}

	if err := storewrapper.DeleteNode(nodeID); err != nil {
		b.recordEvent("error", "deleteNode", "nodes",
			fmt.Sprintf("Failed deleting node %s", nodeID), nodeID)

		return domain.OperationResult{
			Success: false,
			Name:    "deleteNode",
			Message: fmt.Sprintf("failed to delete node %s: %v", nodeID, err),
			Data:    map[string]string{"nodeId": nodeID},
		}, nil
	}

	b.recordEvent(
		"info",
		"deleteNode",
		"nodes",
		fmt.Sprintf("Deleted node %s", nodeID),
		nodeID,
	)

	return domain.OperationResult{
		Success: true,
		Name:    "deleteNode",
		Message: fmt.Sprintf("node %s deleted", nodeID),
		Data:    map[string]string{"nodeId": nodeID},
	}, nil
}

func (b *Backend) AddLink(_ context.Context, source, destination, bandwidth string) (domain.OperationResult, error) {
	source = strings.TrimSpace(source)
	destination = strings.TrimSpace(destination)
	bandwidth = strings.TrimSpace(bandwidth)

	if source == "" {
		b.recordEvent(
			"warning",
			"addLink",
			"links",
			"Add link requested without a source node",
			"",
		)

		return domain.OperationResult{
			Success: false,
			Name:    "addLink",
			Message: "source node cannot be empty",
			Data:    map[string]string{},
		}, nil
	}

	if destination == "" {
		b.recordEvent(
			"warning",
			"addLink",
			"links",
			"Add link requested without a destination node",
			"",
		)

		return domain.OperationResult{
			Success: false,
			Name:    "addLink",
			Message: "destination node cannot be empty",
			Data:    map[string]string{},
		}, nil
	}

	if source == destination {
		return domain.OperationResult{
			Success: false,
			Name:    "addLink",
			Message: "source and destination cannot be the same node",
			Data: map[string]string{
				"source":      source,
				"destination": destination,
			},
		}, nil
	}

	bandwidthMbps, err := strconv.ParseInt(bandwidth, 10, 64)
	if err != nil || bandwidthMbps < 0 {
		return domain.OperationResult{
			Success: false,
			Name:    "addLink",
			Message: fmt.Sprintf("invalid bandwidth: %s", bandwidth),
			Data: map[string]string{
				"source":      source,
				"destination": destination,
				"bandwidth":   bandwidth,
			},
		}, nil
	}

	linkID := fmt.Sprintf("%s-%s", source, destination)

	link := &topology.Link{
		Id:                 linkID,
		SourceNode:         source,
		DestinationNode:    destination,
		BandwidthMbps:      bandwidthMbps,
		PropagationDelayNs: 0,
	}

	if err := storewrapper.StoreLink(link); err != nil {
		b.recordEvent(
			"error",
			"addLink",
			"links",
			fmt.Sprintf("Failed storing link %s: %v", linkID, err),
			linkID,
		)

		return domain.OperationResult{
			Success: false,
			Name:    "addLink",
			Message: fmt.Sprintf("failed to store link %s: %v", linkID, err),
			Data: map[string]string{
				"linkId": linkID,
			},
		}, nil
	}

	b.recordEvent(
		"info",
		"addLink",
		"links",
		fmt.Sprintf("Stored link %s → %s", source, destination),
		linkID,
	)

	return domain.OperationResult{
		Success: true,
		Name:    "addLink",
		Message: fmt.Sprintf("link %s → %s stored", source, destination),
		Data: map[string]string{
			"linkId":      linkID,
			"source":      source,
			"destination": destination,
			"bandwidth":   strconv.FormatInt(bandwidthMbps, 10),
		},
	}, nil
}

func (b *Backend) UpdateLink(_ context.Context, linkID, source, destination, bandwidth string) (domain.OperationResult, error) {

	linkID = strings.TrimSpace(linkID)
	source = strings.TrimSpace(source)
	destination = strings.TrimSpace(destination)
	bandwidth = strings.TrimSpace(bandwidth)

	if linkID == "" {
		b.recordEvent(
			"warning",
			"updateLink",
			"links",
			"Update requested without selecting a link",
			"",
		)

		return domain.OperationResult{
			Success: false,
			Name:    "updateLink",
			Message: "no link selected",
			Data:    map[string]string{},
		}, nil
	}

	current, _, err := storewrapper.GetLink(linkID)
	if err != nil {
		b.recordEvent(
			"error",
			"updateLink",
			"links",
			fmt.Sprintf("Link %s not found", linkID),
			linkID,
		)

		return domain.OperationResult{
			Success: false,
			Name:    "updateLink",
			Message: fmt.Sprintf("link %s not found: %v", linkID, err),
			Data: map[string]string{
				"linkId": linkID,
			},
		}, nil
	}

	if source != "" {
		current.SourceNode = source
	}

	if destination != "" {
		current.DestinationNode = destination
	}

	if bandwidth != "" {
		bandwidthMbps, err := strconv.ParseInt(bandwidth, 10, 64)
		if err != nil || bandwidthMbps < 0 {
			return domain.OperationResult{
				Success: false,
				Name:    "updateLink",
				Message: fmt.Sprintf("invalid bandwidth: %s", bandwidth),
				Data: map[string]string{
					"linkId": linkID,
				},
			}, nil
		}

		current.BandwidthMbps = bandwidthMbps
	}

	if err := storewrapper.EditLink(current); err != nil {
		b.recordEvent(
			"error",
			"updateLink",
			"links",
			fmt.Sprintf("Failed saving updated link %s: %v", linkID, err),
			linkID,
		)

		return domain.OperationResult{
			Success: false,
			Name:    "updateLink",
			Message: fmt.Sprintf("failed to save updated link %s: %v", linkID, err),
			Data: map[string]string{
				"linkId": linkID,
			},
		}, nil
	}

	b.recordEvent(
		"info",
		"updateLink",
		"links",
		fmt.Sprintf("Updated link %s", linkID),
		linkID,
	)

	return domain.OperationResult{
		Success: true,
		Name:    "updateLink",
		Message: fmt.Sprintf("link %s updated", linkID),
		Data: map[string]string{
			"linkId":      current.GetId(),
			"source":      current.GetSourceNode(),
			"destination": current.GetDestinationNode(),
			"bandwidth":   strconv.FormatInt(current.GetBandwidthMbps(), 10),
		},
	}, nil
}

func (b *Backend) DeleteLink(_ context.Context, linkID string) (domain.OperationResult, error) {
	linkID = strings.TrimSpace(linkID)

	if linkID == "" {
		b.recordEvent(
			"warning",
			"deleteLink",
			"links",
			"Delete requested without selecting a link",
			"",
		)

		return domain.OperationResult{
			Success: false,
			Name:    "deleteLink",
			Message: "no link selected",
			Data:    map[string]string{},
		}, nil
	}

	if err := storewrapper.DeleteLink(linkID); err != nil {
		b.recordEvent(
			"error",
			"deleteLink",
			"links",
			fmt.Sprintf("Failed deleting link %s", linkID),
			linkID,
		)

		return domain.OperationResult{
			Success: false,
			Name:    "deleteLink",
			Message: fmt.Sprintf("failed to delete link %s: %v", linkID, err),
			Data: map[string]string{
				"linkId": linkID,
			},
		}, nil
	}

	b.recordEvent(
		"info",
		"deleteLink",
		"links",
		fmt.Sprintf("Deleted link %s", linkID),
		linkID,
	)

	return domain.OperationResult{
		Success: true,
		Name:    "deleteLink",
		Message: fmt.Sprintf("link %s deleted", linkID),
		Data: map[string]string{
			"linkId": linkID,
		},
	}, nil
}

func (b *Backend) AddStream(_ context.Context, query string) (domain.OperationResult, error) {
	return b.result("addStream", map[string]string{"query": query, "streamId": "stream-stub-001"}), nil
}

func (b *Backend) RemoveStream(_ context.Context, streamID string) (domain.OperationResult, error) {
	return b.result("removeStream", map[string]string{"streamId": streamID}), nil
}

func (b *Backend) FilterLogs(_ context.Context, severity string) (domain.OperationResult, error) {
	return b.result("filterLogs", map[string]string{"severity": severity}), nil
}

func (b *Backend) OrderLogs(_ context.Context, orderBy string) (domain.OperationResult, error) {
	return b.result("orderLogs", map[string]string{"orderBy": orderBy}), nil
}

func (b *Backend) result(name string, data map[string]string) domain.OperationResult {
	log.Printf("[GUI] adapter.stub.%s called", name)
	return domain.OperationResult{
		Success: true,
		Name:    name,
		Message: fmt.Sprintf("%s executed (stub)", name),
		Data:    data,
	}
}

func (b *Backend) recordEvent(severity, eventType, topic, message, correlationID string) {
	entry := domain.EventLog{
		Time:          time.Now().Format(time.RFC3339),
		Type:          eventType,
		Severity:      severity,
		Message:       message,
		Topic:         topic,
		CorrelationID: correlationID,
	}

	activityLog.mu.Lock()
	activityLog.entries = append(activityLog.entries, entry)
	if len(activityLog.entries) > maxActivityLogEntries {
		activityLog.entries = append([]domain.EventLog(nil), activityLog.entries[len(activityLog.entries)-maxActivityLogEntries:]...)
	}
	activityLog.mu.Unlock()
}
