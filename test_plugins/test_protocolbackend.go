package main

import (
	"log"

	netconf "OpenCNC_config_service/pkg/plugins/netconf"
	"OpenCNC_config_service/pkg/protocolbackends"
	"OpenCNC_config_service/pkg/structures/qbv"
	"OpenCNC_config_service/pkg/structures/topology"
	topology_config "OpenCNC_config_service/pkg/structures/topology_config"

	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/proto"
)

var logger = log.New(log.Writer(), "[TEST-tttech-TRAFFIC-CLASSES] ", log.LstdFlags)

func TestNetconfProtocol() {

	plugin_qbv := netconf.NewQbvNetconfPlugin_tttech(logger)
	plugin_pcp := netconf.NewPcpMappingNetconfPlugin(logger)
	//plugin_tc := netconf.NewTrafficClassNetconfPlugin(logger)

	backend := protocolbackends.NewNetconfBackend("netconf", plugin_qbv, plugin_pcp)

	backend.MapAndPush(nodecfg, target)
}

var target = topology.ManagementInfo{
	IpAddress:      "192.168.0.1", // IP address
	UserName:       "root",        // username
	ManagementPort: 830,           // default NETCONF port
	Protocol:       topology.ManagementProtocol_NETCONF,
}

var nodecfg = &topology_config.NodeConfig{
	ConfigId: "config-1",
	Bridge: &topology_config.BridgeConfig{
		VlanConfig: &topology_config.BridgeVlanConfig{
			VlanRegistrationEntries: []*topology_config.VlanRegistrationEntry{
				{
					DatabaseId: 1,
					VlanIds:    []uint32{10, 20, 30},
					EntryType:  "static",
					PortMaps: []*topology_config.VlanPortMap{
						{
							PortRef:               4,
							RegistrarAdminControl: "normal",
							VlanTransmitted:       "tagged",
						},
						{
							PortRef:               5,
							RegistrarAdminControl: "normal",
							VlanTransmitted:       "tagged",
						},
					},
				},
			},
			VidToFidMappings: []*topology_config.VidToFidMapping{
				{Vid: 10, Fid: 1},
				{Vid: 20, Fid: 2},
				{Vid: 30, Fid: 2},
			},
		},
	},
	PortConfigs: []*topology_config.PortConfig{
		{
			PortId:            "sw0p3",
			PcpMappingEnabled: proto.Bool(true),
			DefaultPriority:   proto.Uint32(0),
			DefaultVlanId:     proto.Uint32(1),
			TrafficClassTable: []*topology_config.TrafficClassTableEntry{
				{Pcp: 7, EgressQueueId: 0},
				{Pcp: 5, EgressQueueId: 1},
				{Pcp: 0, EgressQueueId: 7},
			},
			QueueConfigs: []*topology_config.QueueConfig{
				{QueueId: 0, MaxFrameSize: 512, ShaperRateBps: 1000000000},
			},
			VlanMemberships: []*topology_config.VlanMembership{
				{VlanId: 1, Tagged: false},
				{VlanId: 10, Tagged: true},
				{VlanId: 20, Tagged: true},
			},
			Gcl: &qbv.GateControlList{
				ScheduleId:  "sched1",
				AdminState:  qbv.AdminState_ENABLED,
				BaseTime:    1_500_000_000, // ns
				CycleTime:   10_000_000,    // ns
				Description: "TTTech bridge gate schedule",
				// NON-STANDARD FIELD: admin-gate-states for testing
				InterfaceTimeOffsetNs: ygot.Int64(0), // optional, not used here
				Entries: []*qbv.GateControlEntry{
					{Index: 0, TimeInterval: 500_000, GateStates: []byte{0x0F}}, // all gates open for testing
					{Index: 1, TimeInterval: 500_000, GateStates: []byte{0x03}}, // partial gates open
					{Index: 0, TimeInterval: 500_000, GateStates: []byte{0x0F}}, // all gates open for testing
					{Index: 1, TimeInterval: 500_000, GateStates: []byte{0x03}}, // partial gates open
				},
			},
		},
	},
}
