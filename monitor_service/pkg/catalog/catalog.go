package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"OpenCNC/monitor_service/structures/monitoring"
)

var (
	defaultCountersPath = "monitor_service/pkg/catalog/available_counters.json"
	defaultMetricsPath  = "monitor_service/pkg/catalog/available_metrics.json"
)

type Catalog struct {
	Items    []monitoring.Capability `json:"items"`
	Counters []*monitoring.Counter   `json:"counters"`
	Metrics  []*monitoring.Metric    `json:"metrics"`
}

func readConfigFile(envVar, defaultRelPath string, fallbacks ...string) ([]byte, error) {
	if env := os.Getenv(envVar); env != "" {
		if b, err := os.ReadFile(env); err == nil {
			return b, nil
		}
	}
	candidates := append([]string{defaultRelPath}, fallbacks...)
	if _, filename, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(filename), filepath.Base(defaultRelPath)))
	}
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			return b, nil
		}
	}
	return os.ReadFile(defaultRelPath)
}

func LoadCatalog() (*Catalog, error) {
	countersData, err := readConfigFile(
		"COUNTERS_PATH",
		defaultCountersPath,
		"pkg/catalog/available_counters.json",
		"../pkg/catalog/available_counters.json",
		"available_counters.json",
		"/home/opencnc/OpenCNC/monitor_service/pkg/catalog/available_counters.json",
	)
	if err != nil {
		return nil, err
	}

	metricsData, err := readConfigFile(
		"METRICS_PATH",
		defaultMetricsPath,
		"pkg/catalog/available_metrics.json",
		"../pkg/catalog/available_metrics.json",
		"available_metrics.json",
		"/home/opencnc/OpenCNC/monitor_service/pkg/catalog/available_metrics.json",
	)
	if err != nil {
		return nil, err
	}

	catalog := &Catalog{}

	if err := json.Unmarshal(countersData, &catalog); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(metricsData, &catalog); err != nil {
		return nil, err
	}

	// Add counters.
	for _, counter := range catalog.Counters {
		if counter == nil {
			continue
		}

		catalog.Items = append(catalog.Items, monitoring.Capability{
			Name:        counter.Name,
			Description: counter.Description,
			Kind:        monitoring.DataType_RAW,
		})
	}

	// Add metrics.
	for _, metric := range catalog.Metrics {
		catalog.Items = append(catalog.Items, monitoring.Capability{
			Name:        metric.Name,
			Description: metric.Description,
			Kind:        monitoring.DataType_METRIC,
		})
	}

	return catalog, nil
}

func (c *Catalog) GetItem(name string) *monitoring.Capability {
	if c == nil {
		return nil
	}

	for i := range c.Items {
		if c.Items[i].Name == name {
			return &c.Items[i]
		}
	}

	return nil
}

func (c *Catalog) GetCounter(name string) *monitoring.Counter {
	if c == nil {
		return nil
	}

	for i := range c.Counters {
		if c.Counters[i].Name == name {
			return c.Counters[i]
		}
	}

	return nil
}

func (c *Catalog) GetCounterByID(name string) *monitoring.Counter {
	if c == nil {
		return nil
	}

	for i := range c.Counters {
		if c.Counters[i] != nil && c.Counters[i].Name == name {
			return c.Counters[i]
		}
	}
	return nil
}

func (c *Catalog) GetMetricByID(name string) *monitoring.Metric {
	if c == nil {
		return nil
	}

	for i := range c.Metrics {
		if c.Metrics[i].Name == name {
			return c.Metrics[i]
		}
	}
	return nil
}

func (c *Catalog) GetItemsByResources(resource *monitoring.ResourceKey) []monitoring.Capability {
	if c == nil || resource == nil {
		return nil
	}

	return c.Items
}
