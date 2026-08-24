package meters

import (
	"fmt"

	"OpenCNC_config_service/common/observability"
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

type MeterFactory struct {
	Type monitoring.MetricType
	New  func(resource *monitoring.ResourceKey, obs *observability.Client, metric *monitoring.Metric) (Meter, error)
}

var registry = map[monitoring.MetricType]MeterFactory{}

func Register(factory MeterFactory) {
	if factory.New == nil {
		panic("meters: register called with nil factory")
	}

	if _, exists := registry[factory.Type]; exists {
		panic(fmt.Sprintf("meters: factory already registered for metric type %s", factory.Type.String()))
	}

	registry[factory.Type] = factory
}

func New(resource *monitoring.ResourceKey, obs *observability.Client, metric *monitoring.Metric) (Meter, error) {
	if resource == nil {
		return nil, fmt.Errorf("resource is nil")
	}

	if metric == nil {
		return nil, fmt.Errorf("metric is nil")
	}

	factory, exists := registry[metric.Type]
	if !exists {
		return nil, fmt.Errorf("no meter registered for metric type %s", metric.Type.String())
	}

	return factory.New(resource, obs, metric)
}
