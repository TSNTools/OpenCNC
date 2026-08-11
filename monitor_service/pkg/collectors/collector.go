package collectors

import (
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

type Collector interface {

	// Protocol implemented by this collector.
	Protocol() string

	// Discover which OpenCNC counters are available on the target.
	SupportedCounters(target *topology.Node) ([]*monitoring.Counter, error)

	// Read a subset (or all) of the supported counters.
	Collect(
		target topology.Node,
		counters []*monitoring.Counter,
	) ([]*monitoring.CounterSample, error)
}
