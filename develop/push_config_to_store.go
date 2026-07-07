package main

import (
	store "OpenCNC_config_service/pkg/store-wrapper"
	"fmt"

	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/proto"

	qbv "OpenCNC_config_service/pkg/structures/qbv"
	"OpenCNC_config_service/pkg/structures/topology"
	topology_config "OpenCNC_config_service/pkg/structures/topology_config"
)

var bridge = &topology.Node{
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
		UserName:       "root",
		ManagementPort: 830,
		Protocol:       topology.ManagementProtocol_NETCONF,
	},
	Properties: &topology.NodeProperties{
		Bridge: &topology.BridgeProperties{
			ProcessingDelayNs: 800,
		},
	},
	NodeConfigId: proto.String("config-1"),
}

var cfg = &topology_config.NodeConfig{
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
				ScheduleId:  "gcl-1",
				BaseTime:    1672531200000000000,
				BaseTimeRef: qbv.TimestampReference_UNIX_EPOCH,
				CycleTime:   1000000000,
				AdminState:  qbv.AdminState_ENABLED,
				Description: "TTTech bridge gate schedule",
				Entries: []*qbv.GateControlEntry{
					{Index: 1, TimeInterval: 250000000, GateStates: []byte{0x01}},
					{Index: 2, TimeInterval: 250000000, GateStates: []byte{0x00}},
					{Index: 3, TimeInterval: 250000000, GateStates: []byte{0x01}},
					{Index: 4, TimeInterval: 250000000, GateStates: []byte{0x00}},
				},
			},
		},
	},
}

/*
	topo := &topology.Topology{
		Nodes: []*topology.Node{{
			Name:           "sw1",
			ManagementInfo: &topology.ManagementInfo{Protocol: topology.ManagementProtocol_NETCONF},
			Ports:          []*topology.Port{{Name: "eth0"}},
		}},
	}

	cfg := &topology_config.TopologyConfig{
		NodeConfigs: []*topology_config.NodeConfig{{
			NodeId: "sw1",
			PortConfigs: []*topology_config.PortConfig{{
				PortId: "eth0",
			}},
		}},
	}
*/

func StoreBridge_with_config() error {
	// Serialize the bridge and config to bytes
	bridgeBytes, err := proto.Marshal(bridge)
	if err != nil {
		return fmt.Errorf("failed to serialize bridge: %v", err)
	}

	configBytes, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	// Store the serialized bridge and config in the store
	err = store.SendToStore(bridgeBytes, "bridges."+bridge.Name)
	if err != nil {
		return fmt.Errorf("failed to store bridge: %v", err)
	}

	err = store.SendToStore(configBytes, "configurations."+cfg.ConfigId)
	if err != nil {
		return fmt.Errorf("failed to store config: %v", err)
	}

	return nil
}

func pullBridge_config(bridgeName string) (*topology_config.NodeConfig, error) {
	// Pull the serialized bridge and config from the store
	bridgeBytes, err := store.GetFromStore("bridges." + bridgeName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bridge: %v", err)
	}

	// Deserialize the bytes back into the original structures
	var retrievedBridge topology.Node
	err = proto.Unmarshal(bridgeBytes, &retrievedBridge)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize bridge: %v", err)
	}

	configId := *retrievedBridge.NodeConfigId

	configBytes, err := store.GetFromStore("configurations." + configId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve config: %v", err)
	}

	var retrievedConfig topology_config.NodeConfig
	err = proto.Unmarshal(configBytes, &retrievedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize config: %v", err)
	}

	return &retrievedConfig, nil
}

func StoreSchedule() error {

	gcl := &qbv.GateControlList{
		ScheduleId: "sched1",
		AdminState: qbv.AdminState_ENABLED,
		BaseTime:   1_500_000_000, // ns
		CycleTime:  10_000_000,    // ns
		// NON-STANDARD FIELD: admin-gate-states for testing
		InterfaceTimeOffsetNs: ygot.Int64(0), // optional, not used here
		Entries: []*qbv.GateControlEntry{
			{
				Index:        0,
				TimeInterval: 500_000,
				GateStates:   []byte{0x0F}, // all gates open for testing
			},
			{
				Index:        1,
				TimeInterval: 500_000,
				GateStates:   []byte{0x03}, // partial gates open
			},
		},
	}
	gclId := gcl.GetScheduleId()

	// Create a URN where the serialized request will be stored
	urn := "configurations.schedules." + gclId

	sched, err := proto.Marshal(gcl)

	// Send serialized request to it's specific path in a store
	err = store.SendToStore(sched, urn)
	if err != nil {
		//log.Errorf("Failed storing schedule: %v", err)
		return err
	}

	return nil
}

func pullScheduleFromStore(scheduleId string) (*qbv.GateControlList, error) {
	// Create a URN where the serialized request will be stored
	urn := "configurations.schedules." + scheduleId

	// Pull serialized request from store
	rawsched, err := store.GetFromStore(urn)
	if err != nil {
		return nil, err
	}

	var gcl = &qbv.GateControlList{}
	err = proto.Unmarshal(rawsched, gcl)
	if err != nil {
		return nil, err
	}

	fmt.Println(gcl)
	return gcl, nil
}

func main() {
	//StoreSchedule()
	//pullScheduleFromStore("sched1")

	StoreBridge_with_config()
	config, _ := pullBridge_config("bridge-1")
	fmt.Println(config)

}
