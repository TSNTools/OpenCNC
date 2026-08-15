package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"OpenCNC_config_service/common/structures/topology"
	counters "OpenCNC_config_service/monitor_service/opencnc_counters_catalog"
	"OpenCNC_config_service/monitor_service/pkg/collectors"
	enginepkg "OpenCNC_config_service/monitor_service/pkg/engine"
	"OpenCNC_config_service/monitor_service/pkg/managementSessions"
	"OpenCNC_config_service/monitor_service/pkg/meters"
	monitor "OpenCNC_config_service/monitor_service/pkg/monitors"
	"OpenCNC_config_service/monitor_service/structures/monitoring"
)

const (
	defaultCatalogPath = "/home/opencnc/OpenCNC/monitor_service/opencnc_counters_catalog/opencnc_counters.json"
)

func main() {
	target := &topology.Node{
		Name: os.Getenv("MONITOR_NODE_NAME"),
		ManagementInfo: &topology.ManagementInfo{
			IpAddress: os.Getenv("MONITOR_NETCONF_HOST"),
			UserName:  os.Getenv("MONITOR_NETCONF_USER"),
		},
	}

	if target.Name == "" {
		target.Name = "bridge-1"
	}

	if target.ManagementInfo.IpAddress == "" {
		log.Fatal("MONITOR_NETCONF_HOST is required")
	}

	if target.ManagementInfo.UserName == "" {
		target.ManagementInfo.UserName = "root"
	}

	password := os.Getenv("MONITOR_NETCONF_PASS")
	catalogPath := os.Getenv("MONITOR_COUNTERS_CATALOG")
	if catalogPath == "" {
		catalogPath = defaultCatalogPath
	}

	catalog, err := counters.LoadCatalog(catalogPath)
	if err != nil {
		log.Fatalf("failed loading counter catalog: %v", err)
	}

	setCounterPollInterval(catalog, "if_state_in_unicast_pkts", 1000)
	setCounterPollInterval(catalog, "if_state_out_unicast_pkts", 1000)

	meters.PacketRateMetric.InputIds = []string{
		"if_state_in_unicast_pkts",
		"if_state_out_unicast_pkts",
	}

	session, err := managementSessions.CreateSession(
		target.ManagementInfo.IpAddress,
		target.ManagementInfo.UserName,
		password,
	)
	if err != nil {
		log.Fatalf("NETCONF connection failed: %v", err)
	}
	defer session.Close()

	collector := collectors.NewNetconfCollector(target, session, catalog)

	packetRateMeter, err := meters.NewPacketRateMeter(target.Name, "")
	if err != nil {
		log.Fatalf("failed creating packet-rate meter: %v", err)
	}

	engine := enginepkg.NewEngine()
	resourceMonitor := monitor.NewResourceMonitor(
		&monitoring.ResourceKey{NodeId: target.Name},
		engine.HandleEvent,
	)

	if err := resourceMonitor.AddCollector(collector); err != nil {
		log.Fatalf("failed adding collector: %v", err)
	}

	if err := resourceMonitor.AddMeter(packetRateMeter); err != nil {
		log.Fatalf("failed adding packet-rate meter: %v", err)
	}

	if err := resourceMonitor.Start(); err != nil {
		log.Fatalf("failed starting monitor: %v", err)
	}
	defer resourceMonitor.Stop()

	log.Printf("monitor started for node=%s", target.Name)
	log.Printf("press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("shutting down monitor")
}

func setCounterPollInterval(catalog *counters.Catalog, counterID string, intervalMS uint32) {
	if catalog == nil {
		return
	}

	counter := catalog.GetByID(counterID)
	if counter == nil {
		log.Printf("warning: counter %q not found in catalog", counterID)
		return
	}

	counter.PollIntervalMs = intervalMS
}
