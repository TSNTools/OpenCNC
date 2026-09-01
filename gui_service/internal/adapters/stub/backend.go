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

	storewrapper "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/credentials"
	devicemodelregistry "OpenCNC/common/structures/devicemodelregistry"
	streamproto "OpenCNC/common/structures/stream"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/uni"

	"OpenCNC/gui_service/internal/domain"
	"OpenCNC/monitor_service/structures/monitoring"

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
	Vendor    string       `json:"vendor"`
	Version   string       `json:"version"`
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
	mu    sync.RWMutex
	//streams   []domain.Stream
	//streamSeq int
}

var activityLog = struct {
	mu      sync.RWMutex
	entries []domain.EventLog
}{}

const maxActivityLogEntries = 100

func NewBackend(state *MonitoringState) *Backend {
	return &Backend{
		state: state,
		//streams:   defaultStreams(),
		//streamSeq: 5,
	}
}

func defaultStreams() []domain.Stream {
	return []domain.Stream{
		newStream("str-1", "Str 1", "Nd1", []string{"Nd2", "Nd3"}, "TRAFFIC_TYPE_ISOCHRONOUS", "RANK_EMERGENCY", 10, 1000000, 1500, 1, 5000, 1000, 0, 200, 2),
		newStream("str-2", "Str 2", "Nd2", []string{"Nd3"}, "TRAFFIC_TYPE_SYNCHRONOUS", "RANK_EMERGENCY", 12, 500000, 1522, 1, 3500, 500, 0, 120, 1),
		newStream("str-3", "Str 3", "Nd3", []string{"Nd1"}, "TRAFFIC_TYPE_MANAGEMENT", "RANK_NON_EMERGENCY", 14, 2000000, 1500, 1, 8000, 3000, 50, 250, 1),
		newStream("str-4", "Str 4", "Nd4", []string{"Nd1", "Nd2"}, "TRAFFIC_TYPE_BEST_EFFORT_HIGH", "RANK_NON_EMERGENCY", 16, 1000000, 1518, 1, 10000, 1500, 0, 500, 1),
	}
}

func newStream(id, name, source string, listeners []string, trafficType, rank string, vlanID, intervalNs, maxFrameSize, maxFramesPerInterval, maxLatencyNs, maxJitterNs, minTransmitOffsetNs, maxTransmitOffsetNs, numSeamlessTrees int) domain.Stream {
	stream := domain.Stream{
		ID:                   id,
		Name:                 name,
		TalkerNodeID:         source,
		ListenerNodeIDs:      append([]string(nil), listeners...),
		TrafficType:          trafficType,
		Rank:                 rank,
		VLANID:               vlanID,
		IntervalNs:           intervalNs,
		MaxFrameSize:         maxFrameSize,
		MaxFramesPerInterval: maxFramesPerInterval,
		MaxLatencyNs:         maxLatencyNs,
		MaxJitterNs:          maxJitterNs,
		MinTransmitOffsetNs:  minTransmitOffsetNs,
		MaxTransmitOffsetNs:  maxTransmitOffsetNs,
		NumSeamlessTrees:     numSeamlessTrees,
	}
	stream.Source = source
	stream.Listeners = strings.Join(stream.ListenerNodeIDs, ", ")
	stream.Characteristics = streamCharacteristics(stream)
	return stream
}

func streamTrafficTypeLabel(value string) string {
	cleaned := strings.ReplaceAll(strings.TrimPrefix(value, "TRAFFIC_TYPE_"), "_", " ")
	words := strings.Fields(strings.ToLower(cleaned))
	for index, word := range words {
		if len(word) == 0 {
			continue
		}
		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func streamCharacteristics(stream domain.Stream) string {
	parts := []string{}
	if stream.TrafficType != "" {
		parts = append(parts, streamTrafficTypeLabel(stream.TrafficType))
	}
	if stream.IntervalNs > 0 {
		parts = append(parts, fmt.Sprintf("Interval %d ns", stream.IntervalNs))
	}
	if stream.MaxLatencyNs > 0 {
		parts = append(parts, fmt.Sprintf("Deadline %d ns", stream.MaxLatencyNs))
	}
	if stream.VLANID > 0 {
		parts = append(parts, fmt.Sprintf("VLAN %d", stream.VLANID))
	}
	if len(parts) == 0 {
		return "periodic\nDeadline..."
	}
	return strings.Join(parts, "\n")
}

func (b *Backend) TrafficSummary(_ context.Context) (domain.DashboardData, error) {
	return domain.DashboardData{
		PacketsForwarded: "12,458",
		PacketDrops:      "37",
		ActiveStreams:    "8",
		TopTalkers: []string{
			"",
		},
	}, nil
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

	dashboardData, err := b.TrafficSummary(r)
	if err != nil {
		return domain.DashboardData{}, err
	}
	dashboardData.NetworkModel = domain.NetworkModel{
		Nodes: nodes,
		Links: links,
	}
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

	return dashboardData, nil
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
			Version: model.GetVersion(),
			Vendor:  model.GetVendor(),
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
	_, err := requestStartMonitoring(query)
	if err != nil {
		return nil, err
	}
	// Update the shared state to let the kafka consumer collect the requested metrics.
	b.state.SetWantedMetrics(query.MetricIDs)
	b.state.SetWantedMetricIntervals(query.PollIntervals)

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

	filtered := make([]domain.MonitoringTargetData, 0)
	for _, target := range targets {
		if query.TargetID != "" && target.ID != query.TargetID {
			continue
		}
		if query.Node != "" && !strings.EqualFold(target.Node, query.Node) {
			continue
		}

		values := make([]domain.MonitoringValue, 0)
		for _, metricID := range query.MetricIDs {
			valueByMetric, found := rtData[target.ID]
			if !found {
				continue
			}

			value, ok := valueByMetric[metricID]
			if !ok {
				continue
			}

			values = append(values, domain.MonitoringValue{
				ID:             metricID,
				Label:          labelByID[metricID],
				Value:          value,
				PollIntervalMs: monitoringIntervalForMetric(query.PollIntervals, metricID),
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

func monitoringIntervalForMetric(intervals map[string]uint32, metricID string) uint32 {
	if len(intervals) == 0 {
		return 1000
	}

	interval, ok := intervals[metricID]
	if !ok || interval <= 0 {
		return 1000
	}

	return interval
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

		if n.DeviceInfo != nil {
			domainNodes[i].DeviceModel = n.DeviceInfo.DeviceModel
		}

		if n.ManagementInfo != nil {
			domainNodes[i].ManagementIp = n.ManagementInfo.IpAddress
			domainNodes[i].ManagementProtocol = n.ManagementInfo.Protocol.String()
			domainNodes[i].ManagementPort = n.ManagementInfo.ManagementPort
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
	streams, err := storewrapper.GetStreams()
	if err != nil {
		return nil, err
	}

	domainStreams := make([]domain.Stream, 0, len(streams))

	for _, stream := range streams {
		if stream == nil {
			continue
		}

		var rank string
		if stream.StreamRank != nil {
			rank = stream.StreamRank.Rank.String()
		}

		var destinationMAC string
		var sourceMAC string

		if stream.FrameSpec != nil {
			destinationMAC = stream.FrameSpec.DestinationMac
			sourceMAC = stream.FrameSpec.SourceMac
		}

		var intervalNs int
		var maxFrameSize int
		var maxFramesPerInterval int

		if stream.TrafficSpec != nil {
			intervalNs = int(stream.TrafficSpec.IntervalNs)
			maxFrameSize = int(stream.TrafficSpec.MaxFrameSize)
			maxFramesPerInterval = int(stream.TrafficSpec.MaxIntervalFrames)
		}

		var maxLatencyNs int
		var maxJitterNs int
		var minTransmitOffsetNs int
		var maxTransmitOffsetNs int
		var numSeamlessTrees int

		if stream.UserToNetworkRequirements != nil {
			req := stream.UserToNetworkRequirements

			maxLatencyNs = int(req.MaxLatencyNs)
			maxJitterNs = int(req.MaxJitterNs)
			numSeamlessTrees = int(req.NumSeamlessTrees)

			if req.TransmitOffsetRange != nil {
				minTransmitOffsetNs = int(req.TransmitOffsetRange.MinNs)
				maxTransmitOffsetNs = int(req.TransmitOffsetRange.MaxNs)
			}
		}

		stream := domain.Stream{
			ID:                   stream.StreamId.AsKey(),
			Name:                 stream.Description,
			TalkerNodeID:         stream.TalkerNodeId,
			ListenerNodeIDs:      append([]string(nil), stream.ListenerNodeIds...),
			Rank:                 rank,
			DestinationMAC:       destinationMAC,
			SourceMAC:            sourceMAC,
			IntervalNs:           intervalNs,
			MaxFrameSize:         maxFrameSize,
			MaxFramesPerInterval: maxFramesPerInterval,
			MaxLatencyNs:         maxLatencyNs,
			MaxJitterNs:          maxJitterNs,
			MinTransmitOffsetNs:  minTransmitOffsetNs,
			MaxTransmitOffsetNs:  maxTransmitOffsetNs,
			NumSeamlessTrees:     numSeamlessTrees,
		}

		// Derived GUI fields.
		stream.Source = stream.TalkerNodeID
		stream.Listeners = strings.Join(stream.ListenerNodeIDs, ", ")
		stream.Characteristics = streamCharacteristics(stream)

		domainStreams = append(domainStreams, stream)
	}

	return domainStreams, nil
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

		model := &devicemodelregistry.DeviceModel{
			Name:    modelInput.Name,
			Vendor:  modelInput.Vendor,
			Version: modelInput.Version,
		}
		for _, yang := range modelInput.YangFiles {
			if strings.TrimSpace(yang.Name) == "" {
				continue
			}
			model.YangFiles = append(model.YangFiles, &devicemodelregistry.YangModule{
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

	if strings.TrimSpace(version) != "" {
		current.Version = strings.TrimSpace(version)
	}

	if strings.TrimSpace(vendor) != "" {
		current.Vendor = strings.TrimSpace(vendor)
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
		if err := storewrapper.StoreDeviceModel(current); err != nil {
			b.recordEvent("error", "editModel", "device-models", fmt.Sprintf("Failed saving model %s", modelID), modelID)
			return domain.OperationResult{
				Success: false,
				Name:    "editModel",
				Message: fmt.Sprintf("failed to save model %s: %v", modelID, err),
				Data:    map[string]string{"modelId": modelID},
			}, nil
		}
		b.recordEvent("info", "editModel", "device-models", fmt.Sprintf("Updated model %s", current.GetName()), current.GetName())
	}

	return domain.OperationResult{
		Success: true,
		Name:    "editModel",
		Message: fmt.Sprintf("model %s updated", current.GetName()),
		Data: map[string]string{
			"modelId": current.GetName(),
			"name":    current.GetName(),
			"version": current.GetVersion(),
			"vendor":  current.GetVendor(),
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

func parseManagementProtocol(raw string) (topology.ManagementProtocol, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	switch normalized {
	case "", "UNRECOGNIZED":
		return topology.ManagementProtocol_UNRECOGNIZED, true
	case "NETCONF":
		return topology.ManagementProtocol_NETCONF, true
	case "SNMP":
		return topology.ManagementProtocol_SNMP, true
	default:
		return topology.ManagementProtocol_UNRECOGNIZED, false
	}
}

func (b *Backend) EditNode(_ context.Context, node domain.Node) (domain.OperationResult, error) {

	if strings.TrimSpace(node.ID) == "" {
		b.recordEvent("warning", "editNode", "nodes", "Edit requested without selecting a node", "")
		return domain.OperationResult{
			Success: false,
			Name:    "editNode",
			Message: "no node selected",
			Data:    map[string]string{},
		}, nil
	}

	current, _, err := storewrapper.GetNode(node.ID)
	if err != nil {
		b.recordEvent("error", "editNode", "nodes",
			fmt.Sprintf("Node %s not found", node.ID), node.ID)

		return domain.OperationResult{
			Success: false,
			Name:    "editNode",
			Message: fmt.Sprintf("node %s not found: %v", node.ID, err),
			Data:    map[string]string{"nodeId": node.ID},
		}, nil
	}

	previousName := current.GetName()
	node.ID = strings.TrimSpace(node.ID)
	node.Type = strings.TrimSpace(node.Type)

	if node.ID != "" {
		current.Name = node.ID
	}

	if node.Type != "" {
		parsedRole, ok := parseNodeRole(node.Type)
		if !ok {
			b.recordEvent("warning", "editNode", "nodes", fmt.Sprintf("Invalid node type for %s: %s", node.ID, node.Type), node.ID)
			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf("invalid node type: %s", node.Type),
				Data:    map[string]string{"nodeId": node.ID},
			}, nil
		}
		current.Type = parsedRole
	}

	if current.DeviceInfo == nil {
		current.DeviceInfo = &topology.DeviceInfo{}
	}

	// Device model
	if node.DeviceModel != "" {
		current.DeviceInfo.DeviceModel = node.DeviceModel
	}

	if current.ManagementInfo == nil {
		current.ManagementInfo = &topology.ManagementInfo{}
	}

	// Management information
	if node.ManagementIp != "" {
		current.ManagementInfo.IpAddress = node.ManagementIp
	}

	if node.ManagementPort != 0 {
		current.ManagementInfo.ManagementPort = node.ManagementPort
	}

	if node.ManagementProtocol != "" {
		parsedProtocol, ok := parseManagementProtocol(node.ManagementProtocol)
		if !ok {
			b.recordEvent(
				"warning",
				"editNode",
				"nodes",
				fmt.Sprintf(
					"Invalid management protocol for %s: %s",
					node.ID,
					node.ManagementProtocol,
				),
				node.ID,
			)

			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf(
					"invalid management protocol: %s",
					node.ManagementProtocol,
				),
				Data: map[string]string{"nodeId": node.ID},
			}, nil
		}

		current.ManagementInfo.Protocol = parsedProtocol
	}

	if node.PortIds != nil {
		existingPorts := make(map[string]*topology.Port)

		for _, port := range current.Ports {
			if port != nil {
				existingPorts[port.Id] = port
			}
		}

		updatedPorts := make([]*topology.Port, 0, len(node.PortIds))

		for _, portID := range node.PortIds {
			portID = strings.TrimSpace(portID)
			if portID == "" {
				continue
			}

			if existingPort, ok := existingPorts[portID]; ok {
				updatedPorts = append(updatedPorts, existingPort)
			} else {
				updatedPorts = append(updatedPorts, &topology.Port{
					Id: portID,
				})
			}
		}

		current.Ports = updatedPorts
	}

	if previousName != current.GetName() {
		if err := storewrapper.DeleteNode(node.ID); err != nil {
			b.recordEvent("error", "editNode", "nodes", fmt.Sprintf("Failed renaming node %s", node.ID), node.ID)
			return domain.OperationResult{
				Success: false,
				Name:    "editNode",
				Message: fmt.Sprintf("failed to rename node %s: %v", node.ID, err),
				Data:    map[string]string{"nodeId": node.ID},
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

	//update credentials if username provided (password can be "")
	if node.Username != "" {
		creds := &credentials.ManagementCredentials{
			Authentication: &credentials.ManagementCredentials_UsernamePassword{
				UsernamePassword: &credentials.UsernamePassword{
					Username: node.Username,
					Password: node.Password,
				},
			},
		}
		storewrapper.StoreCredentials(current.GetName(), creds)
	}

	return domain.OperationResult{
		Success: true,
		Name:    "editNode",
		Message: fmt.Sprintf("node %s updated", current.GetName()),
		Data: map[string]string{
			"nodeId": current.GetName(),
			"name":   current.GetName(),
			"type":   current.GetType().String(),
			"ports":  strings.Join(node.PortIds, ","),
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

func (b *Backend) AddStream(_ context.Context, stream domain.Stream) (domain.OperationResult, error) {

	stream = b.prepareStream(stream)

	// resolve end stations
	talkerInterface, listenerInterfaces, err := b.resolveEndStations(stream)
	if err != nil {
		return domain.OperationResult{
			Success: false,
			Name:    "addStream",
			Message: fmt.Sprintf("failed to resolve end stations for stream %s", stream.ID),
		}, err
	}
	// used talker interface MAC as stream mac
	stream.SourceMAC = talkerInterface.InterfaceId.MacAddress

	// Build ConfigRequest.
	//TODO:every stream request needs a unique Stream ID, support multiple at once
	streamID, err := generateTSNStreamID(stream.SourceMAC)
	if err != nil {
		fmt.Println("Failed to generate Stream ID for stream: ", err)
		return domain.OperationResult{
			Success: false,
			Name:    "addStream",
			Message: fmt.Sprintf("failed to generate Stream ID for stream %s", streamID.AsKey()),
		}, err
	}

	configReq := &uni.ConfigRequest{
		Version: 1.0,
		Requests: []*uni.Request{
			{
				Id: streamID.AsKey(),

				Talker: &uni.TalkerGroup{
					StrId: streamID,

					StrRank: &uni.StreamRank{
						Rank: streamRankFromString(stream.Rank),
					},

					EndStationInterfaces: []*uni.Interface{
						talkerInterface,
					},

					DataFrameSpecification: []*uni.DataFrameSpecification{
						{
							MacAddr: &uni.IeeeMacAddress{
								DestinationMac: stream.DestinationMAC,
								SourceMac:      stream.SourceMAC,
							},

							VlanTag: &uni.IeeeVlanTag{
								VlanId: uint32(stream.VLANID),
							},
						},
					},

					TrafficSpecification: buildTrafficSpecification(stream),

					UserToNetReq: &uni.UserToNetworkRequirements{
						NumSeamlessTrees: uint32(stream.NumSeamlessTrees),
						MaxLatency:       uint32(stream.MaxLatencyNs),
					},
				},

				ListenerList: buildListenerGroups(streamID, stream, listenerInterfaces),
			},
		},
	}

	if _, err := requestAddStream(configReq); err != nil {
		fmt.Println("Failed to add stream: ", err)
		return domain.OperationResult{
			Success: false,
			Name:    "addStream",
			Message: fmt.Sprintf("failed to store stream %s", stream.ID),
		}, err
	}

	b.recordEvent(
		"info",
		"addStream",
		"streams",
		fmt.Sprintf("Created stream %s", stream.Name),
		stream.ID,
	)

	return domain.OperationResult{
		Success: true,
		Name:    "addStream",
		Message: fmt.Sprintf("stream %s created", stream.ID),
		Data: map[string]string{
			"streamId": streamID.AsKey(),
			"name":     stream.Name,
		},
	}, nil
}

func streamRankFromString(rank string) streamproto.StreamRankValue {
	switch rank {
	case "RANK_EMERGENCY":
		return streamproto.StreamRankValue_RANK_EMERGENCY
	case "RANK_NON_EMERGENCY":
		return streamproto.StreamRankValue_RANK_NON_EMERGENCY
	default:
		return streamproto.StreamRankValue_RANK_NON_EMERGENCY
	}
}

func (b *Backend) UpdateStream(_ context.Context, streamID string, stream domain.Stream) (domain.OperationResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	/*
		for index := range b.streams {
			if b.streams[index].ID != streamID {
				continue
			}

			stream = b.prepareStream(stream)
			stream.ID = streamID
			b.streams[index] = stream
			b.recordEvent("info", "updateStream", "streams", fmt.Sprintf("Updated stream %s", stream.Name), streamID)
			return domain.OperationResult{
				Success: true,
				Name:    "updateStream",
				Message: fmt.Sprintf("stream %s updated", streamID),
				Data: map[string]string{
					"streamId": streamID,
					"name":     stream.Name,
				},
			}, nil
		}
	*/
	return domain.OperationResult{
		Success: false,
		Name:    "updateStream",
		Message: fmt.Sprintf("stream %s not found", streamID),
		Data: map[string]string{
			"streamId": streamID,
		},
	}, nil
}

func (b *Backend) RemoveStream(_ context.Context, id string) (domain.OperationResult, error) {
	id = strings.TrimSpace(id)

	if id == "" {
		return domain.OperationResult{
			Success: false,
			Name:    "removeStream",
			Message: "stream ID cannot be empty",
		}, nil
	}

	if err := storewrapper.DeleteStream(id); err != nil {
		return domain.OperationResult{
			Success: false,
			Name:    "removeStream",
			Message: fmt.Sprintf("failed to delete stream %s", id),
		}, err
	}

	b.recordEvent(
		"info",
		"removeStream",
		"streams",
		fmt.Sprintf("Deleted stream %s", id),
		id,
	)

	return domain.OperationResult{
		Success: true,
		Name:    "removeStream",
		Message: fmt.Sprintf("stream %s deleted", id),
		Data: map[string]string{
			"streamId": id,
		},
	}, nil
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
		Message: fmt.Sprintf("%s executed !!", name),
		Data:    data,
	}
}

func (b *Backend) prepareStream(stream domain.Stream) domain.Stream {
	stream.Name = strings.TrimSpace(stream.Name)
	stream.TalkerNodeID = strings.TrimSpace(stream.TalkerNodeID)
	stream.Source = stream.TalkerNodeID

	cleanListeners := make([]string, 0, len(stream.ListenerNodeIDs))
	for _, listener := range stream.ListenerNodeIDs {
		trimmed := strings.TrimSpace(listener)
		if trimmed != "" {
			cleanListeners = append(cleanListeners, trimmed)
		}
	}
	stream.ListenerNodeIDs = cleanListeners
	stream.Listeners = strings.Join(cleanListeners, ", ")
	stream.Characteristics = strings.TrimSpace(stream.Characteristics)
	if stream.Characteristics == "" {
		stream.Characteristics = streamCharacteristics(stream)
	}
	if stream.Name == "" {
		stream.Name = stream.ID
	}
	return stream
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
