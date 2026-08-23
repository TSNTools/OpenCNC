package main

import (
	"context"
	"fmt"
	"log"

	"OpenCNC_config_service/common/observability"
	"OpenCNC_config_service/common/structures/credentials"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/monitor_service/pkg/catalog"
	"OpenCNC_config_service/monitor_service/pkg/collectors"
	"OpenCNC_config_service/monitor_service/pkg/managementSessions"
	"OpenCNC_config_service/monitor_service/structures/monitoring"

	"github.com/openconfig/ygot/ygot"
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

var creds = &credentials.ManagementCredentials{
	Authentication: &credentials.ManagementCredentials_UsernamePassword{
		UsernamePassword: &credentials.UsernamePassword{
			Username: "root",
			Password: "root",
		},
	},
}

var obsClient observability.Client

func test_netconf_collector() {

	session, err := managementSessions.CreateSessionWithPassword(
		target.ManagementInfo.IpAddress,
		creds.GetUsernamePassword(),
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
	catalog, err := catalog.LoadCatalog()

	if err != nil {
		log.Fatalf(
			"failed loading counter catalog: %v",
			err,
		)
	}

	//
	// Create collector
	//
	resource := &monitoring.ResourceKey{
		NodeId: target.Name,
	}
	obsClient, err := observability.NewFromEnv("config-service")
	if err != nil {
		log.Fatalf("Observability init failed: %v", err)
	}
	if obsClient != nil {
		defer func() {
			_ = obsClient.Close()
		}()
	}

	collector := collectors.NewNetconfCollector(
		resource,
		session,
		catalog,
		obsClient,
		1000, // PollIntervalMs, set to 1000 ms as an example
	)

	ctx := context.Background()

	//
	// Select counters to test
	//
	requestedCounters := []*monitoring.Counter{
		//catalog.GetByID("if_state_in_errors"),
		//catalog.GetByID("if_state_out_errors"),
		catalog.GetCounterByID("if_state_in_unicast_pkts"),
		catalog.GetCounterByID("if_state_out_unicast_pkts"),
	}

	//
	// Collect
	//
	samples, err := collector.Collect(
		requestedCounters,
		ctx,
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
