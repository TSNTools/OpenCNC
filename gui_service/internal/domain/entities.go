package domain

type DashboardData struct {
	PacketsForwarded string       `json:"packetsForwarded"`
	PacketDrops      string       `json:"packetDrops"`
	ActiveStreams    string       `json:"activeStreams"`
	TopTalkers       []string     `json:"topTalkers"`
	NetworkModel     NetworkModel `json:"networkModel"`
}

type NetworkModel struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type DeviceModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Vendor  string `json:"vendor"`
	Yang    string `json:"yang"`
}

type MonitoringItem struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type MonitoringTarget struct {
	ID       string   `json:"id"`
	Node     string   `json:"node"`
	Port     string   `json:"port"`
	Label    string   `json:"label"`
	Counters []string `json:"counters"`
	Metrics  []string `json:"metrics"`
}

type MonitoringValue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type MonitoringTargetData struct {
	TargetID string            `json:"targetId"`
	Label    string            `json:"label"`
	Node     string            `json:"node"`
	Port     string            `json:"port"`
	Values   []MonitoringValue `json:"values"`
}

type Node struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	State   string   `json:"state"`
	PortIds []string `json:"ports"`
	Links   string   `json:"links"`
}

type Link struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	State       string `json:"state"`
	Bandwidth   string `json:"bandwidth"`
}

type Stream struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Source          string `json:"source"`
	Listeners       string `json:"listeners"`
	Characteristics string `json:"characteristics"`
}

type EventLog struct {
	Time          string `json:"time"`
	Type          string `json:"type"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
	Topic         string `json:"topic"`
	CorrelationID string `json:"correlationId"`
}

type OperationResult struct {
	Success bool              `json:"success"`
	Name    string            `json:"name"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

type LogsQuery struct {
	Severity string
	OrderBy  string
}

type MonitoringDataQuery struct {
	Node      string
	TargetID  string
	MetricIDs []string
}
