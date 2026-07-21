package plugins

import "OpenCNC_config_service/monitor_service/structures/monitoring"

type Collector interface {

	// Protocol implemented by this collector.
	Protocol() string

	// Discover which OpenCNC counters are available on the target.
	SupportedCounters(target Target) ([]*monitoring.Counter, error)

	// Read a subset (or all) of the supported counters.
	Collect(
		target Target,
		counters []*monitoring.Counter,
	) ([]*monitoring.CounterSample, error)
}
