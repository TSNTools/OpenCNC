package monitor

import (
	"OpenCNC/monitor_service/pkg/catalog"
	"OpenCNC/monitor_service/structures/monitoring"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

type Monitor interface {
	// Identity of the monitored resource.
	Resource() *monitoring.ResourceKey

	// Start begins the monitoring loop.
	Start() error

	// Stop stops the monitoring loop.
	Stop()

	// AddCollector registers a collector used by this monitor.
	AddCollector(counter *monitoring.Counter, resource *monitoring.ResourceKey, session *netconf.Session, catalog *catalog.Catalog) error

	// AddMeter registers a meter for this monitor's resource.
	AddMeter(metric *monitoring.Metric, resource *monitoring.ResourceKey) error

	// Run performs one monitoring cycle:
	//
	// 1. Collect samples
	// 2. Feed samples to meters
	// 3. Evaluate meters according to their dependencies
	// 4. Evaluate thresholds (TODO)
	// 5. Generate events/actions (TODO)
	Run() error
}
