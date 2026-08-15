package main

import (
	"fmt"
	"log"

	"OpenCNC_config_service/common/structures/topology"
	counters "OpenCNC_config_service/monitor_service/opencnc_counters_catalog"
	"OpenCNC_config_service/monitor_service/pkg/collectors"
	"OpenCNC_config_service/monitor_service/pkg/managementSessions"
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

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
		UserName:       "root",
		ManagementPort: 830,
		Protocol:       topology.ManagementProtocol_NETCONF,
	},
	Properties: &topology.NodeProperties{
		Bridge: &topology.BridgeProperties{
			ProcessingDelayNs: 800,
		},
	},
	DeviceInfo: &topology.DeviceInfo{
		DeviceModel: "TTTech-EVB",
	},
}

func test_netconf_collector() {

	session, err := managementSessions.CreateSession(
		target.ManagementInfo.IpAddress,
		target.ManagementInfo.UserName,
		"",
	)

	if err != nil {
		log.Fatalf(
			"NETCONF connection failed: %v",
			err,
		)
	}

	defer session.Close()

	//
	// Load counter catalog
	//
	catalog, err := counters.LoadCatalog(
		"/home/opencnc/OpenCNC/monitor_service/pkg/counters/opencnc_counters.json",
	)

	if err != nil {
		log.Fatalf(
			"failed loading counter catalog: %v",
			err,
		)
	}

	//
	// Create collector
	//
	collector := collectors.NewNetconfCollector(
		target,
		session,
		catalog,
	)

	//
	// Select counters to test
	//
	requestedCounters := []*monitoring.Counter{
		//catalog.GetByID("if_state_in_errors"),
		//catalog.GetByID("if_state_out_errors"),
		catalog.GetByID("if_state_in_unicast_pkts"),
		catalog.GetByID("if_state_out_unicast_pkts"),
	}

	//
	// Collect
	//
	samples, err := collector.Collect(
		requestedCounters,
	)

	if err != nil {

		log.Fatalf(
			"collection failed: %v",
			err,
		)
	}

	//
	// Print results
	//
	fmt.Println("Collected counters:")
	fmt.Println("===================")

	for _, sample := range samples {

		fmt.Printf(
			"counter=%s node=%s value=%f timestamp=%s\n",
			sample.Id,
			sample.Source.NodeId,
			sample.Value,
			sample.Timestamp.AsTime(),
		)

	}

}
