package main

import (
	"log"

	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/pcp"
	"OpenCNC/common/structures/qbv"
	"OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	vlan "OpenCNC/common/structures/vlan"
	"OpenCNC/config_service/pkg/engine"
	netconfbackend "OpenCNC/config_service/pkg/protocolbackends/netconf"

	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/proto"
)

var logger = log.New(log.Writer(), "[TEST-tttech-TRAFFIC-CLASSES] ", log.LstdFlags)

func TestNetconfProtocol() {
	target := target
	nodecfg := nodecfg
	cred := cred

	backend, err := netconfbackend.NewNetconfBackend(
		target,
		cred,
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to create Netconf backend: %v", err)
	}

	//backend := protocolbackends.NewNetconfBackend("netconf", plugin_qbv, plugin_pcp)
	operation := &engine.Operation{
		Config:  nodecfg,
		Backend: backend,
	}
	tx := &engine.ConfigurationTransaction{
		ConfigId:   "test-transaction-1",
		Operations: []engine.Operation{*operation},
	}

	tx.Commit()
}

var cred = &credentials.ManagementCredentials{
	Authentication: &credentials.ManagementCredentials_UsernamePassword{
		UsernamePassword: &credentials.UsernamePassword{
			Username: "root",
			Password: "",
		},
	},
}

var target = &topology.Node{
	Name: "bridge-1",
	Type: topology.NodeRole_BRIDGE,
	Ports: []*topology.Port{
		{
			Id:             "sw0p3",
			Name:           "sw0p3",
			NumberOfQueues: 8,
			Capabilities: &topology.InterfaceCapabilities{
				PortSpeed:        1000,
				SupportsTimeSync: true,
				SupportsTas:      true,
				SupportsCbs:      true,
			},
		},
	},
	ManagementInfo: &topology.ManagementInfo{
		IpAddress:      "192.168.0.1",
		ManagementPort: 830,
		Protocol:       topology.ManagementProtocol_NETCONF,
	},
	Properties: &topology.NodeProperties{
		ProcessingDelayNs: ygot.Int32(800),
	},
	DeviceInfo: &topology.DeviceInfo{
		DeviceModel: "TTTech-EVB",
	},
}

var nodecfg = &topology_config.NodeConfig{
	Bridge: &topology_config.BridgeConfig{
		VlanConfig: &vlan.BridgeVlanConfig{
			VlanRegistrationEntries: []*vlan.VlanRegistrationEntry{
				{
					DatabaseId: 1,
					VlanIds:    []uint32{10, 20, 30},
					EntryType:  vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC,
					PortMaps: []*vlan.VlanPortMap{
						{
							PortId:                "sw0p3",
							RegistrarAdminControl: vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_NORMAL,
							VlanTransmitted:       vlan.VlanTransmitted_VLAN_TRANSMITTED_TAGGED,
						},
						{
							PortId:                "sw0p4",
							RegistrarAdminControl: vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_NORMAL,
							VlanTransmitted:       vlan.VlanTransmitted_VLAN_TRANSMITTED_TAGGED,
						},
					},
				},
			},
			VidToFidMappings: []*vlan.VidToFidMapping{
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
			PcpConfig: &pcp.PcpPortConfig{
				TrafficClassTable: []*pcp.TrafficClassMappingEntry{
					{Priority: 0, EgressQueueId: 0},
					{Priority: 4, EgressQueueId: 4},
					{Priority: 5, EgressQueueId: 5},
					{Priority: 7, EgressQueueId: 7},
				},
			},
			QueueConfigs: []*topology_config.QueueConfig{
				{QueueId: 0, MaxFrameSize: 512, ShaperRateBps: 1000000000},
			},
			VlanMemberships: []*vlan.VlanMembership{
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
