package main

import (
	"context"
	"fmt"

	"time"

	// admissioncontrol "main-service/pkg/admission-control"

	eventhandler "main-service/pkg/event-handler"
	"main-service/pkg/nni"
	store "main-service/pkg/store-wrapper"
	monitor "main-service/pkg/structures/temp-monitor-conf"

	"github.com/openconfig/gnmi/proto/gnmi"

	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
)

//var log = logger.GetLogger()

func main() {

	//log.Infof("Starting main service")

	//fmt.Println("Starting main service")

	// Create TSN stores
	//store.CreateStores()

	// Temporarily add adapter to k/v store
	//store.StoreAdapter("NETCONF", "gnmi-netconf-adapter")

	//Create and save the module registry
	store.StoreModuleRegistry()

	// Start NNI server
	go nni.StartServer()

	// Not working on local network, needs to be connected to switches
	//switches := counterConfHandler.GetMonitorConfigDevices()
	//if err := startDeviceDataCollection(switches); err != nil {
	//log.Errorf("Failed data collection for switches: %v", err)
	//}

	// Don't start the UNI before the system is available, so that the user
	// sees the UNI only when the subsystems are available
	//	log.Info("Starting to check for config subsystem availability...")

	// Start UNI server
	//go uni.StartServer()

	//pollConfigSubsystemForAvailability()

	select {}
}

func pollConfigSubsystemForAvailability() bool {
	for {
		fmt.Println("Trying to connect to config-service...")
		_, err := eventhandler.ConnectToGnmiService("config-service:5150")
		if err != nil {
			fmt.Println("Config-service not reachable, retrying in 10s...")
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println("Connected to config-service!")
		return true
	}
}

func startDeviceDataCollection(switches []monitor.MonitorConfig) error {
	c, err := gclient.New(context.Background(), client.Destination{
		Addrs:       []string{"monitor-service" + ":" + "11161"},
		Timeout:     5 * time.Second,
		Credentials: nil,
		TLS:         nil,
	})
	if err != nil {
		return err
	}

	for i := range switches {
		sw := &switches[i] // ✅ take pointer to avoid copying the struct

		_, err := c.(*gclient.Client).Get(context.Background(), &gnmi.GetRequest{
			Path: []*gnmi.Path{
				{
					Elem:   []*gnmi.PathElem{},
					Target: sw.DeviceIP,
				},
			},
		})
		if err != nil {
			fmt.Printf("Failed getting response: %v\n", err)
			return err
		}
	}

	return nil
}
