package main

import (
	"log"

	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	netconf "OpenCNC_config_service/pkg/plugins/netconf"
	topology "OpenCNC_config_service/pkg/structures/topology"
	topology_config "OpenCNC_config_service/pkg/structures/topology_config"

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
		DefaultPriority:   proto.Uint32(1), // default PCP for untagged
		PcpMappingEnabled: proto.Bool(true),
		TrafficClassTable: []*topology_config.TrafficClassTableEntry{
			{Pcp: 0, EgressQueueId: 3},
			{Pcp: 5, EgressQueueId: 5},
			{Pcp: 7, EgressQueueId: 6},
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

/*
profile := &tsn.Profile{
	Id:   "industrial_profile_1",
	Name: "Industrial TSN Profile",

	Type: tsn.ProfileType_INDUSTRIAL,

	Capabilities: &tsn.ProfileCapabilities{
		SupportTas:  true,
		SupportCbs:  true,
		SupportPsfp: true,
	},

	TrafficTypes: TrafficTypes,

 Bind traffic types to queues on sw0p3
	PortMappings: PortMappings,

 VLAN + PCP classification
	VlanMappings: VlanMappings,
}
*/
/*
TrafficTypes := []*tsn.TrafficType{
		{
			Id:           "tc_critical_motion",
			Name:         "Critical Motion",
			Description:  "Time-triggered motion control traffic",
			DeliveryMode: tsn.DeliveryMode_DEADLINE,
			Shaper:       tsn.Shaper_QBV,
			Mtu:          256,
			Properties: &tsn.TrafficTypeProperties{
				IsTimeTriggered:    true,
				RequiresScheduling: true,
				CycleAligned:       true,
				IsPreemptable:      false,
			},
		},
		{
			Id:           "tc_audio_video",
			Name:         "Audio Video",
			Description:  "Streamed AV traffic",
			DeliveryMode: tsn.DeliveryMode_LATENCY,
			Shaper:       tsn.Shaper_QAV,
			Mtu:          1500,
			Properties: &tsn.TrafficTypeProperties{
				IsPeriodic:          true,
				RequiresReservation: true,
				JitterTolerant:      false,
				IsPreemptable:       true,
			},
		},
		{
			Id:           "tc_best_effort",
			Name:         "Best Effort",
			Description:  "Background traffic",
			DeliveryMode: tsn.DeliveryMode_NONE,
			Shaper:       tsn.Shaper_SP,
			Mtu:          1500,
			Properties: &tsn.TrafficTypeProperties{
				JitterTolerant:     true,
				PacketLossTolerant: true,
				DropEligible:       true,
				IsPreemptable:      true,
			},
		},
	}
*/
