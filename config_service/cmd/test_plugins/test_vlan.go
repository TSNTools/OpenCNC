package main

import (
	"log"

	managementSessions "OpenCNC_config_service/config_service/pkg/managementSessions"
	netconf "OpenCNC_config_service/config_service/pkg/plugins/netconf"
	topology "OpenCNC_config_service/common/structures/topology"
	topology_config "OpenCNC_config_service/common/structures/topology_config"
	vlan "OpenCNC_config_service/common/structures/vlan"

	"google.golang.org/protobuf/proto"
)

func TestVlanPlugin_tttech() {
	logger := log.New(log.Writer(), "[TEST-tttech-VLAN] ", log.LstdFlags)

	target := managementSessions.DeviceTarget{
		InterfaceName: "sw0p3",
		Logger:        logger,
		Secret:        "",
		Info: &topology.ManagementInfo{
			IpAddress:      "192.168.0.1",
			UserName:       "root",
			ManagementPort: 830,
			Protocol:       topology.ManagementProtocol_NETCONF,
		},
	}

	pluginVlan := netconf.NewVlanNetconfPlugin(logger)

	mapped, err := pluginVlan.Map(VlanConfig)
	if err != nil {
		logger.Fatalf("Map failed: %v", err)
	}

	if err := pluginVlan.Push(mapped, target); err != nil {
		logger.Fatalf("Push failed: %v", err)
	}
}

var PortConfig = &topology_config.PortConfig{
	PortId:        "sw0p3",
	DefaultVlanId: proto.Uint32(30),
	VlanMemberships: []*vlan.VlanMembership{
		{VlanId: 10, Tagged: true},
		{VlanId: 20, Tagged: true},
		{VlanId: 30, Tagged: true},
	},
	VlanAdvanced: &vlan.PortVlanAdvancedConfig{
		AcceptableFrame:                   vlan.AcceptableFrameType_ACCEPTABLE_FRAME_TYPE_ADMIT_ALL_FRAMES.Enum(),
		IngressFilteringEnabled:           proto.Bool(true),
		RestrictedVlanRegistrationEnabled: proto.Bool(false),
		VidTranslationTableEnabled:        proto.Bool(true),
		EgressVidTranslationTableEnabled:  proto.Bool(true),
		VidTranslations: []*vlan.VidTranslation{
			{LocalVid: 100, RelayVid: 10},
		},
		EgressVidTranslations: []*vlan.VidTranslation{
			{LocalVid: 10, RelayVid: 100},
		},
		ProtocolGroupVidSets: []*vlan.ProtocolGroupVidSet{
			{GroupId: 100, VlanIds: []uint32{10}},
			{GroupId: 200, VlanIds: []uint32{30}},
		},
	},
}

var VlanConfig = &vlan.BridgeVlanConfig{
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
}

var Bridge = &topology_config.BridgeConfig{VlanConfig: VlanConfig}
var nodeConfig = &topology_config.NodeConfig{Bridge: Bridge, PortConfigs: []*topology_config.PortConfig{PortConfig}}
