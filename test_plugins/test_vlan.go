package main

import (
	"log"

	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	netconf "OpenCNC_config_service/pkg/plugins/netconf"
	topology "OpenCNC_config_service/pkg/structures/topology"
	traffic_type "OpenCNC_config_service/pkg/structures/trafficType"
)

func TestVlanPlugin_tttech() {
	logger := log.New(log.Writer(), "[TEST-tttech-VLAN] ", log.LstdFlags)

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
	plugin_vlan := netconf.NewVlanNetconfPlugin(logger)

	// Fake messages for testing - in real use, these would come from the API layer
	VlanMappings := &traffic_type.VlanTrafficMappings{
		PortId: "sw0p3",
		Mappings: []*traffic_type.VlanTrafficMapping{
			{
				TrafficTypeId: "tc_critical_motion",
				VlanId:        100,
				Pcp:           7,
			},
			{
				TrafficTypeId: "tc_audio_video",
				VlanId:        200,
				Pcp:           5,
			},
			{
				TrafficTypeId: "tc_best_effort",
				VlanId:        1,
				Pcp:           0,
			},
		},
	}

	// Test pluggin_vlan
	//// Map
	mapped, err := plugin_vlan.Map(VlanMappings)
	if err != nil {
		logger.Fatalf("Map failed: %v", err)
	}

	//// Push the config
	err = plugin_vlan.Push(mapped, target)
	if err != nil {
		logger.Fatalf("Push failed: %v", err)
	}
}
