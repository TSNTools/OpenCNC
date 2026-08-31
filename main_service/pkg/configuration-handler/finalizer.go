package configurationhandler

import "OpenCNC/common/structures/topology_config"

func FinalizeConfiguration(config *topology_config.TopologyConfig) error {
	// Implementation for finalizing the configuration goes here.

	//insert meta data such as:
	// - basetime for applying it

	return nil
}
