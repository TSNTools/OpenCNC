package main

import (
	"log"

	managementSessions "OpenCNC_config_service/config_service/pkg/managementSessions"
	netconf "OpenCNC_config_service/config_service/pkg/plugins/netconf"
	topology "OpenCNC_config_service/common/structures/topology"
	topology_config "OpenCNC_config_service/common/structures/topology_config"

	"github.com/gogo/protobuf/proto"
)

func TestPriorityPlugin_tttech() {
	logger := log.New(log.Writer(), "[TEST-tttech-PRIORITY] ", log.LstdFlags)

	// device target
	target := managementSessions.DeviceTarget{
		InterfaceName: "sw0p3",
		Logger:        logger,
		Secret:        "", // password is empty
		Info: &topology.ManagementInfo{
			IpAddress:      "192.168.0.1", // IP address
			UserName:       "root",        // username
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
		TrafficClassTable: []*topology_config.TrafficClassTableEntry{
			{Pcp: 0, EgressQueueId: 0},
			{Pcp: 4, EgressQueueId: 4},
			{Pcp: 5, EgressQueueId: 5},
			{Pcp: 7, EgressQueueId: 7},
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
