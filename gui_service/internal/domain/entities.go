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

type Node struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	State string `json:"state"`
	Ports string `json:"ports"`
	Links string `json:"links"`
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
