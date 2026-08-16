package domain

import "context"

type ReadPort interface {
	GetDashboard(ctx context.Context) (DashboardData, error)
	GetDeviceModels(ctx context.Context) ([]DeviceModel, error)
	GetMonitoringCounters(ctx context.Context) ([]MonitoringItem, error)
	GetMonitoringMetrics(ctx context.Context) ([]MonitoringItem, error)
	GetMonitoringTargets(ctx context.Context) ([]MonitoringTarget, error)
	GetMonitoringData(ctx context.Context, query MonitoringDataQuery) ([]MonitoringTargetData, error)
	GetNodes(ctx context.Context) ([]Node, error)
	GetLinks(ctx context.Context) ([]Link, error)
	GetStreams(ctx context.Context) ([]Stream, error)
	GetLogs(ctx context.Context, query LogsQuery) ([]EventLog, error)
	GetRecentEvents(ctx context.Context) ([]EventLog, error)
}

type OperationPort interface {
	RefreshData(ctx context.Context) (OperationResult, error)
	UploadModel(ctx context.Context, query string) (OperationResult, error)
	EditModel(ctx context.Context, modelID, name, version, vendor, yang string) (OperationResult, error)
	DeleteModel(ctx context.Context, modelID string) (OperationResult, error)
	AddNode(ctx context.Context, query string) (OperationResult, error)
	EditNode(ctx context.Context, nodeID, name, nodeType, state, ports, links string) (OperationResult, error)
	DeleteNode(ctx context.Context, nodeID string) (OperationResult, error)
	AddLink(ctx context.Context, source, destination, bandwidth string) (OperationResult, error)
	UpdateLink(ctx context.Context, linkID string) (OperationResult, error)
	DeleteLink(ctx context.Context, linkID string) (OperationResult, error)
	AddStream(ctx context.Context, query string) (OperationResult, error)
	RemoveStream(ctx context.Context, streamID string) (OperationResult, error)
	FilterLogs(ctx context.Context, severity string) (OperationResult, error)
	OrderLogs(ctx context.Context, orderBy string) (OperationResult, error)
}
