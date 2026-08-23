package catalog

import (
	"encoding/json"
	"os"

	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

const (
	defaultCountersPath = "/home/opencnc/OpenCNC/monitor_service/pkg/catalog/available_counters.json"
	defaultMetricsPath  = "/home/opencnc/OpenCNC/monitor_service/pkg/catalog/available_metrics.json"
)

type Catalog struct {
	Items    []monitoring.Capability `json:"items"`
	Counters []*monitoring.Counter   `json:"counters"`
	Metrics  []*monitoring.Metric    `json:"metrics"`
}

func LoadCatalog() (*Catalog, error) {
	countersData, err := os.ReadFile(defaultCountersPath)
	if err != nil {
		return nil, err
	}

	metricsData, err := os.ReadFile(defaultMetricsPath)
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
