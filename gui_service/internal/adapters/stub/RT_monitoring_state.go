package stub

import "sync"

type MonitoringState struct {
	mu sync.RWMutex

	// Metrics currently requested by the GUI.
	wantedMetrics map[string]struct{}

	// Latest metric data received from Kafka.
	metrics map[string]MetricData
}

type MetricData struct {
	Name      string
	Value     float64
	Timestamp int64
}

func NewMonitoringState() *MonitoringState {
	return &MonitoringState{
		wantedMetrics: make(map[string]struct{}),
		metrics:       make(map[string]MetricData),
	}
}

// SetWantedMetrics replaces the currently requested metrics.
func (s *MonitoringState) SetWantedMetrics(metrics []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.wantedMetrics = make(map[string]struct{}, len(metrics))

	for _, metric := range metrics {
		if metric == "" {
			continue
		}

		s.wantedMetrics[metric] = struct{}{}
	}
}

// ClearWantedMetrics removes all requested metrics.
func (s *MonitoringState) ClearWantedMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.wantedMetrics = make(map[string]struct{})
}

// WantsMetric reports whether the GUI currently wants this metric.
func (s *MonitoringState) WantsMetric(metric string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.wantedMetrics[metric]
	return ok
}

// IsMonitoring reports whether at least one metric is currently requested.
func (s *MonitoringState) IsMonitoring() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.wantedMetrics) > 0
}

// SetMetric stores the latest value of a metric.
func (s *MonitoringState) SetMetric(data MetricData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics[data.Name] = data
}

// GetMetric returns the latest value of a metric.
func (s *MonitoringState) GetMetric(name string) (MetricData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.metrics[name]
	return data, ok
}
