package stub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	storewrapper "OpenCNC_config_service/common/store-wrapper"
	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/gui_service/internal/domain"

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

type Backend struct{}

var activityLog = struct {
	mu      sync.RWMutex
	entries []domain.EventLog
}{}

const maxActivityLogEntries = 100

func NewBackend() *Backend {
	return &Backend{}
}

func (b *Backend) GetDashboard(_ context.Context) (domain.DashboardData, error) {
	return domain.DashboardData{
		PacketsForwarded: "12,458",
		PacketDrops:      "37",
		ActiveStreams:    "8",
		TopTalkers: []string{
			"Node-A",
			"Node-B",
			"CNC-Controller",
			"Sensor-01",
			"Gateway-01",
		},
		NetworkModel: domain.NetworkModel{
			Nodes: []domain.Node{
				{
					ID:    "node-001",
					Name:  "CNC Controller",
					Type:  "controller",
					State: "online",
					Ports: "4",
					Links: "3",
				},
				{
					ID:    "node-002",
					Name:  "Machine 01",
					Type:  "cnc-machine",
					State: "online",
					Ports: "2",
					Links: "2",
				},
				{
					ID:    "node-003",
					Name:  "Operator Station",
					Type:  "workstation",
					State: "online",
					Ports: "1",
					Links: "1",
				},
				{
					ID:    "node-004",
					Name:  "PLC Controller",
					Type:  "plc",
					State: "online",
					Ports: "8",
					Links: "1",
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

func (b *Backend) GetNodes(_ context.Context) ([]domain.Node, error) {
	return []domain.Node{}, nil
}

func (b *Backend) GetLinks(_ context.Context) ([]domain.Link, error) {
	return []domain.Link{}, nil
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

func (b *Backend) EditNode(_ context.Context, nodeID string) (domain.OperationResult, error) {
	return b.result("editNode", map[string]string{"nodeId": nodeID}), nil
}

func (b *Backend) DeleteNode(_ context.Context, nodeID string) (domain.OperationResult, error) {
	return b.result("deleteNode", map[string]string{"nodeId": nodeID}), nil
}

func (b *Backend) AddLink(_ context.Context, source, destination, bandwidth string) (domain.OperationResult, error) {
	return b.result("addLink", map[string]string{"source": source, "destination": destination, "bandwidth": bandwidth, "linkId": "link-stub-001"}), nil
}

func (b *Backend) UpdateLink(_ context.Context, linkID string) (domain.OperationResult, error) {
	return b.result("updateLink", map[string]string{"linkId": linkID}), nil
}

func (b *Backend) DeleteLink(_ context.Context, linkID string) (domain.OperationResult, error) {
	return b.result("deleteLink", map[string]string{"linkId": linkID}), nil
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
