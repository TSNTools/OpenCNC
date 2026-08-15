package meters

import (
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

// Meter can't own the goroutines that collect samples,
// because complex metrics might need other metrics to be evaluated first.
// that could create circular dependencies.

type Meter interface {
	// Name returns the configured metric name.
	Name() string

	// Metric returns the configured metric.
	Metric() *monitoring.Metric

	// Type returns the metric type implemented by this meter.
	Type() monitoring.MetricType

	// UsesCounter reports whether the meter uses the specified counter.
	UsesInput(inputID string) bool

	// AddSample provides a normalized counter sample to the meter.
	AddSample(sample *monitoring.DataSample) error

	// Ready reports whether enough samples are available to evaluate
	// the metric according to its configured window.
	Ready() bool

	// Evaluate calculates the metric from the currently available samples.
	// The returned DataSample contains only the calculated metric value.
	Evaluate() (*monitoring.DataSample, error)

	// Reset clears the meter's internal history/state.
	Reset()
}
