package app

import (
	"context"
	"log"

	"OpenCNC_config_service/gui_service/internal/domain"
)

type Service struct {
	reader     domain.ReadPort
	operations domain.OperationPort
}

func NewService(reader domain.ReadPort, operations domain.OperationPort) *Service {
	return &Service{reader: reader, operations: operations}
}

func (s *Service) GetDashboard(ctx context.Context) (domain.DashboardData, error) {
	return s.reader.GetDashboard(ctx)
}

func (s *Service) GetDeviceModels(ctx context.Context) ([]domain.DeviceModel, error) {
	return s.reader.GetDeviceModels(ctx)
}

func (s *Service) GetNodes(ctx context.Context) ([]domain.Node, error) {
	return s.reader.GetNodes(ctx)
}

func (s *Service) GetLinks(ctx context.Context) ([]domain.Link, error) {
	return s.reader.GetLinks(ctx)
}

func (s *Service) GetStreams(ctx context.Context) ([]domain.Stream, error) {
	return s.reader.GetStreams(ctx)
}

func (s *Service) GetLogs(ctx context.Context, query domain.LogsQuery) ([]domain.EventLog, error) {
	return s.reader.GetLogs(ctx, query)
}

func (s *Service) GetRecentEvents(ctx context.Context) ([]domain.EventLog, error) {
	return s.reader.GetRecentEvents(ctx)
}

func (s *Service) RefreshData(ctx context.Context) (domain.OperationResult, error) {
	log.Printf("[GUI] app.RefreshData called")
	return s.operations.RefreshData(ctx)
}

func (s *Service) UploadModel(ctx context.Context, query string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.UploadModel called")
	return s.operations.UploadModel(ctx, query)
}

func (s *Service) EditModel(ctx context.Context, modelID, name, version, vendor, yang string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.EditModel called")
	return s.operations.EditModel(ctx, modelID, name, version, vendor, yang)
}

func (s *Service) DeleteModel(ctx context.Context, modelID string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.DeleteModel called")
	return s.operations.DeleteModel(ctx, modelID)
}

func (s *Service) AddNode(ctx context.Context, query string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.AddNode called")
	return s.operations.AddNode(ctx, query)
}

func (s *Service) EditNode(ctx context.Context, nodeID string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.EditNode called")
	return s.operations.EditNode(ctx, nodeID)
}

func (s *Service) DeleteNode(ctx context.Context, nodeID string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.DeleteNode called")
	return s.operations.DeleteNode(ctx, nodeID)
}

func (s *Service) AddLink(ctx context.Context, source, destination, bandwidth string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.AddLink called")
	return s.operations.AddLink(ctx, source, destination, bandwidth)
}

func (s *Service) UpdateLink(ctx context.Context, linkID string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.UpdateLink called")
	return s.operations.UpdateLink(ctx, linkID)
}

func (s *Service) DeleteLink(ctx context.Context, linkID string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.DeleteLink called")
	return s.operations.DeleteLink(ctx, linkID)
}

func (s *Service) AddStream(ctx context.Context, query string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.AddStream called")
	return s.operations.AddStream(ctx, query)
}

func (s *Service) RemoveStream(ctx context.Context, streamID string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.RemoveStream called")
	return s.operations.RemoveStream(ctx, streamID)
}

func (s *Service) FilterLogs(ctx context.Context, severity string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.FilterLogs called")
	return s.operations.FilterLogs(ctx, severity)
}

func (s *Service) OrderLogs(ctx context.Context, orderBy string) (domain.OperationResult, error) {
	log.Printf("[GUI] app.OrderLogs called")
	return s.operations.OrderLogs(ctx, orderBy)
}
