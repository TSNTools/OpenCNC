package storewrapper

import (
	"OpenCNC/common/structures/topology_config"
	"fmt"

	"google.golang.org/protobuf/proto"
)

func GetConfiguration(confId string) (*topology_config.TopologyConfig, error) {
	// this requires all configurations in the store to be normilized to topology_config.TopologyConfig,
	//  otherwise it will fail to unmarshal
	urn := "configurations." + confId

	rawConf, err := GetFromStore(urn)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve configuration %s: %v", confId, err)
	}

	fmt.Println("Retrieved topology configuration from k/v store")

	var config topology_config.TopologyConfig
	if err := proto.Unmarshal(rawConf, &config); err != nil {
		return nil, fmt.Errorf("failed to deserialize topology configuration: %v", err)
	}

	return &config, nil
}

func StoreConfiguration(cfg *topology_config.TopologyConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot store nil configuration")
	}

	configBytes, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	err = SendToStore(
		configBytes,
		"configurations."+cfg.GetConfigId(),
	)
	if err != nil {
		return fmt.Errorf("failed to store configuration: %w", err)
	}

	return nil
}
