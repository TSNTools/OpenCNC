package stub

import (
	"context"
	"fmt"
	"log"

	"OpenCNC_config_service/gui_service/internal/domain"
)

type Backend struct{}

func NewBackend() *Backend {
	return &Backend{}
}

func (b *Backend) GetDashboard(_ context.Context) (domain.DashboardData, error) {
	fmt.Println("read the store and populate the structure")
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
					Links: "2",
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
			},
		},
	}, nil
}

func (b *Backend) GetDeviceModels(_ context.Context) ([]domain.DeviceModel, error) {
	return []domain.DeviceModel{}, nil
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

func (b *Backend) GetLogs(_ context.Context, _ domain.LogsQuery) ([]domain.EventLog, error) {
	return []domain.EventLog{}, nil
}

func (b *Backend) GetRecentEvents(_ context.Context) ([]domain.EventLog, error) {
	return []domain.EventLog{}, nil
}

func (b *Backend) RefreshData(_ context.Context) (domain.OperationResult, error) {
	return b.result("refreshData", map[string]string{"result": "dashboard refresh requested"}), nil
}

func (b *Backend) UploadModel(_ context.Context, query string) (domain.OperationResult, error) {
	return b.result("uploadModel", map[string]string{"query": query, "modelId": "model-stub-001"}), nil
}

func (b *Backend) AddNode(_ context.Context, query string) (domain.OperationResult, error) {
	return b.result("addNode", map[string]string{"query": query, "nodeId": "node-stub-001"}), nil
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
