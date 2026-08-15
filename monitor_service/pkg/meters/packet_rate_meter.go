package meters

import (
	"fmt"

	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

/*
	var PacketRateMetric = &monitoring.Metric{
		Name:                 "unicast-packet-rate",
		Type:                 monitoring.MetricType_PACKET_RATE,
		EvaluationIntervalMs: 1000,
		InputIds: []string{
			"in-unicast-pkts",
			"out-unicast-pkts",
		},
		WindowSize:  2,
		Description: "Combined incoming and outgoing unicast packet rate.",
		Thresholds: []*monitoring.Threshold{
			{
				Value:      70000, // packets/sec
				Comparison: monitoring.Comparison_GREATER_OR_EQUAL,
				Event: &monitoring.MonitoringEvent{
					Type:     monitoring.EventType_PERFORMANCE_DEGRADATION,
					Severity: monitoring.Severity_WARNING,
					Actions: []monitoring.EventAction{
						monitoring.EventAction_NOTIFY,
					},
				},
			},
			{
				Value:      90000, // packets/sec
				Comparison: monitoring.Comparison_GREATER_OR_EQUAL,
				Event: &monitoring.MonitoringEvent{
					Type:     monitoring.EventType_CONGESTION,
					Severity: monitoring.Severity_CRITICAL,
					Actions: []monitoring.EventAction{
						monitoring.EventAction_ALERT,
						monitoring.EventAction_REQUEST_RECONFIGURATION,
					},
				},
			},
		},
	}
*/
func init() {
	Register(MeterFactory{
		Type: monitoring.MetricType_PACKET_RATE,
		New: func(resource *monitoring.ResourceKey, metric *monitoring.Metric) (Meter, error) {
			return NewPacketRateMeter(resource, metric)
		},
	})
}

type PacketRateMeter struct {
	metric  *monitoring.Metric
	nodeID  string
	portID  string
	windows map[string]*SamplesWindow
}

func NewPacketRateMeter(resource *monitoring.ResourceKey, PacketRateMetric *monitoring.Metric) (*PacketRateMeter, error) {

	if PacketRateMetric.Name != "unicast-packet-rate" {
		return nil, fmt.Errorf(
			"invalid metric: expected unicast-packet-rate, got %q",
			PacketRateMetric.Name,
		)
	}

	if PacketRateMetric.WindowSize < 2 {
		return nil, fmt.Errorf(
			"packet rate metric requires window size >= 2",
		)
	}

	windows := make(map[string]*SamplesWindow)

	for _, inputID := range PacketRateMetric.InputIds {
		windows[inputID] = NewSampleWindow(PacketRateMetric.WindowSize)
	}

	return &PacketRateMeter{
		metric:  PacketRateMetric,
		nodeID:  resource.GetNodeId(),
		portID:  resource.GetPortId(),
		windows: windows,
	}, nil
}

func (m *PacketRateMeter) Name() string {
	return m.metric.Name
}

func (m *PacketRateMeter) Metric() *monitoring.Metric {
	return m.metric
}

func (m *PacketRateMeter) Type() monitoring.MetricType {
	return m.metric.Type
}

func (m *PacketRateMeter) UsesInput(inputID string) bool {
	for _, id := range m.metric.InputIds {
		if id == inputID {
			return true
		}
	}

	return false
}

func (m *PacketRateMeter) AddSample(sample *monitoring.DataSample) error {
	if sample == nil {
		return fmt.Errorf("sample is nil")
	}

	if sample.Timestamp == nil {
		return fmt.Errorf("sample timestamp is nil")
	}

	if sample.Id != m.nodeID {
		return nil
	}

	// meter can be configured for a node not a specific port
	if m.portID != "" && sample.Source.GetPortId() != m.portID {
		return nil
	}

	window, exists := m.windows[sample.Id]
	if !exists {
		return nil
	}

	window.Add(sample)

	return nil
}

func (m *PacketRateMeter) Ready() bool {
	for _, inputID := range m.metric.InputIds {
		window := m.windows[inputID]

		if !window.Ready() {
			return false
		}
	}

	return true
}

func (m *PacketRateMeter) Evaluate() (*monitoring.DataSample, error) {
	if !m.Ready() {
		return nil, fmt.Errorf(
			"meter %q is not ready",
			m.Name(),
		)
	}

	firstInputID := m.metric.InputIds[0]
	referenceSamples := m.windows[firstInputID].Samples()

	startSample := referenceSamples[0]
	endSample := referenceSamples[len(referenceSamples)-1]

	startTime := startSample.Timestamp.AsTime()
	endTime := endSample.Timestamp.AsTime()

	elapsed := endTime.Sub(startTime)

	if elapsed <= 0 {
		return nil, fmt.Errorf(
			"invalid sample interval: %v",
			elapsed,
		)
	}

	var totalDelta float64

	for _, inputID := range m.metric.InputIds {
		samples := m.windows[inputID].Samples()

		startSample := samples[0]
		endSample := samples[len(samples)-1]

		delta := endSample.Value - startSample.Value

		if delta < 0 {
			return nil, fmt.Errorf(
				"Data %q decreased from %f to %f",
				inputID,
				startSample.Value,
				endSample.Value,
			)
		}

		totalDelta += delta
	}

	packetRate := totalDelta / elapsed.Seconds()

	var portID *string
	if m.portID != "" {
		portID = &m.portID
	}

	return &monitoring.DataSample{
		Id:        m.metric.Name,
		Source:    &monitoring.ResourceKey{NodeId: m.nodeID, PortId: portID},
		Timestamp: endSample.Timestamp,
		Value:     packetRate,
		Kind:      monitoring.DataType_METRIC,
	}, nil
}

func (m *PacketRateMeter) Reset() {
	for _, window := range m.windows {
		window.Reset()
	}
}
