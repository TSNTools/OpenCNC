package main

import (
	"log"

	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/pcp"
	topology "OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	managementSessions "OpenCNC/config_service/pkg/managementSessions"
	netconf "OpenCNC/config_service/pkg/plugins/netconf"

	"github.com/gogo/protobuf/proto"
)

func TestPriorityPlugin_tttech() {
	logger := log.New(log.Writer(), "[TEST-tttech-PRIORITY] ", log.LstdFlags)

	// device target
	target := managementSessions.DeviceTarget{
		InterfaceName: "sw0p3",
		Logger:        logger,
		Credentials: &credentials.ManagementCredentials{
			Authentication: &credentials.ManagementCredentials_UsernamePassword{
				UsernamePassword: &credentials.UsernamePassword{
					Username: "root",
					Password: "root",
				},
			},
		},
		Info: &topology.ManagementInfo{
			IpAddress:      "192.168.0.1", // IP address
			ManagementPort: 830,           // default NETCONF port
			Protocol:       topology.ManagementProtocol_NETCONF,
		},
	}

	// Create plugin
	plugin_pcp := netconf.NewPcpMappingNetconfPlugin(logger)

	// Fake messages for testing - in real use, these would come from the API layer
	portCfg := &topology_config.PortConfig{
		PortId:            "sw0p3",
		DefaultPriority:   proto.Uint32(0),
		PcpMappingEnabled: proto.Bool(true),
		PcpConfig: &pcp.PcpPortConfig{
			TrafficClassTable: []*pcp.TrafficClassMappingEntry{
				{Priority: 0, EgressQueueId: 0},
				{Priority: 4, EgressQueueId: 4},
				{Priority: 5, EgressQueueId: 5},
				{Priority: 7, EgressQueueId: 7},
			},
		},
		QueueConfigs: []*topology_config.QueueConfig{
			{QueueId: 0},
			{QueueId: 4},
			{QueueId: 5},
			{QueueId: 7},
		},
	}

	// Test pluggin_tc
	//// Map
	mapped, err := plugin_pcp.Map(portCfg)
	if err != nil {
		logger.Fatalf("Map failed: %v", err)
	}

	//// Push the config
	err = plugin_pcp.Push(mapped, target)
	if err != nil {
		logger.Fatalf("Push failed: %v", err)
	}
}
