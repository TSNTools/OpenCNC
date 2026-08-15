package monitor

import (
	"OpenCNC_config_service/monitor_service/pkg/collectors"
	"OpenCNC_config_service/monitor_service/pkg/meters"
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

type Monitor interface {
	// Identity of the monitored resource.
	Resource() *monitoring.ResourceKey

	// Start begins the monitoring loop.
	Start() error

	// Stop stops the monitoring loop.
	Stop()

	// AddCollector registers a collector used by this monitor.
	AddCollector(collector collectors.Collector) error

	// AddMeter registers a meter for this monitor's resource.
	AddMeter(meter meters.Meter) error

	// Run performs one monitoring cycle:
	//
	// 1. Collect samples
	// 2. Feed samples to meters
	// 3. Evaluate meters according to their dependencies
	// 4. Evaluate thresholds (TODO)
	// 5. Generate events/actions (TODO)
	Run() error
}
