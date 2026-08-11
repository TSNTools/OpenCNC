package counters

import (
	"encoding/json"
	"os"

	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

type Catalog struct {
	Counters []*monitoring.Counter `json:"counters"`
}

func LoadCatalog(path string) (*Catalog, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var catalog Catalog

	err = json.Unmarshal(data, &catalog)
	if err != nil {
		return nil, err
	}

	return &catalog, nil
}

func (c *Catalog) GetByID(id string) *monitoring.Counter {

	for _, counter := range c.Counters {
		if counter.Id == id {
			return counter
		}
	}

	return nil
}
