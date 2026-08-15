package collectors

import (
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

type Collector interface {

	// Protocol implemented by this collector.
	Protocol() string

	// Discover which OpenCNC counters are available on the target.
	SupportedCounters() ([]*monitoring.Counter, error)

	// Read a subset (or all) of the supported counters.
	Collect(
		counters []*monitoring.Counter,
	) ([]*monitoring.DataSample, error)
}
