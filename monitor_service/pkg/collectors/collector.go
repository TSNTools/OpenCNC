package collectors

import (
	"OpenCNC_config_service/monitor_service/structures/monitoring"
	"context"
)

type Collector interface {
	// Get the list of counters that this collector is appointed to collect.
	Counters() []*monitoring.Counter

	// Protocol implemented by this collector.
	Protocol() string

	// Target resource that this collector is appointed to collect from.
	Target() *monitoring.ResourceKey

	// PollInterval returns the recommended polling interval for this collector.
	PollInterval() uint32

	// Discover which OpenCNC counters are available on the target.
	SupportedCounters() ([]*monitoring.Counter, error)

	//
	AddCounter(counter *monitoring.Counter) error
	HasCounter(counterName string) bool

	// Read a subset (or all) of the supported counters.
	Collect(
		counters []*monitoring.Counter,
		ctx context.Context,
	) ([]*monitoring.DataSample, error)
}
