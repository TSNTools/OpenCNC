package main

import (
	"fmt"
	"log"

	opencncModel "OpenCNC_config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	netconf "OpenCNC_config_service/pkg/plugins/netconf"
	qbv "OpenCNC_config_service/pkg/structures/qbv"
	"OpenCNC_config_service/pkg/structures/topology"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

func TestQbvPlugin() {
	logger := log.New(log.Writer(), "[TEST-QBV] ", log.LstdFlags)

	// Create plugin
	plugin := netconf.NewQbvNetconfPlugin(logger)

	// Fake GateControlList message
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

	// Map the message to YGOT structure
	mapped, err := plugin.Map(proto.Message(gcl))
	if err != nil {
		logger.Fatalf("Map failed: %v", err)
	}

	// Fake device target
	target := managementSessions.DeviceTarget{
		InterfaceName: "eth1",
	}

	// Build XML
	xmlStr, err := plugin.BuildXML(
		mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable),
		target,
	)
	if err != nil {
		logger.Fatalf("BuildXML failed: %v", err)
	}

	fmt.Println("===== GENERATED XML =====")
	fmt.Println(xmlStr)
	fmt.Println("=========================")
}

func TestQbvPlugin_tttech() {
	logger := log.New(log.Writer(), "[TEST-tttech-QBV] ", log.LstdFlags)

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
	plugin := netconf.NewQbvNetconfPlugin_tttech(logger)

	// Fake GateControlList message
	gcl := &qbv.GateControlList{
		ScheduleId: "sched-tttech",
		AdminState: qbv.AdminState_ENABLED,
		BaseTime:   0, // can be 0 for relative time
		CycleTime:  1, // 1ms
		Entries: []*qbv.GateControlEntry{
			{Index: 0, TimeInterval: 10_000, GateStates: []byte{127}},
			{Index: 1, TimeInterval: 10_000, GateStates: []byte{128}},
			{Index: 2, TimeInterval: 650_000, GateStates: []byte{127}},
			{Index: 3, TimeInterval: 330_000, GateStates: []byte{127}},
		},
	}

	// Map the message to YGOT structure
	mapped, err := plugin.Map(proto.Message(gcl))
	if err != nil {
		logger.Fatalf("Map failed: %v", err)
	}

	// Push the config
	err = plugin.Push(mapped, target)
	if err != nil {
		logger.Fatalf("Push failed: %v", err)
	}
}
