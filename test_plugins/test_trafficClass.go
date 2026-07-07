package main

import (
	"log"

	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	netconf "OpenCNC_config_service/pkg/plugins/netconf"
	topology "OpenCNC_config_service/pkg/structures/topology"
	traffic_type "OpenCNC_config_service/pkg/structures/trafficType"
)

func TestTCPlugin_tttech() {
	logger := log.New(log.Writer(), "[TEST-tttech-TRAFFIC-CLASSES] ", log.LstdFlags)

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
	plugin_tc := netconf.NewTrafficClassNetconfPlugin(logger)

	// Fake messages for testing - in real use, these would come from the API layer
	PortMappings := &traffic_type.PortTrafficMappings{
		PortId: "sw0p3",
		Mappings: []*traffic_type.PortTrafficMapping{
			{
				TrafficTypeId: "tc_best_effort",
				QueueId:       0,
			},
			{
				TrafficTypeId: "tc_audio_video_low",
				QueueId:       4,
			},
			{
				TrafficTypeId: "tc_audio_video_high",
				QueueId:       5,
			},
			{
				TrafficTypeId: "tc_critical_motion",
				QueueId:       7, // highest priority queue
			},
		},
	}
	// Test pluggin_tc
	//// Map
	mapped, err := plugin_tc.Map(PortMappings)
	if err != nil {
		logger.Fatalf("Map failed: %v", err)
	}

	//// Push the config
	err = plugin_tc.Push(mapped, target)
	if err != nil {
		logger.Fatalf("Push failed: %v", err)
	}
}
