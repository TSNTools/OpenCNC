package main

import (
	"fmt"
	"net"

	"time"

	// admissioncontrol "OpenCNC/main_service/pkg/admission-control"

	eventhandler "OpenCNC/main_service/pkg/event-handler"
	"OpenCNC/main_service/pkg/nni"

	//"OpenCNC/main_service/pkg/uni"
	uni_server "OpenCNC/main_service/pkg/uni"

	"google.golang.org/grpc"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
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
	//store.StoreModuleRegistry()

	// Start NNI server
	go nni.StartServer()

	// Start UNI server
	//go uni.StartServer()

	// Start UNI grpc server
	go uni_server.StartHttpServer()

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

func StartUniGrpcServer(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	grpcServer := grpc.NewServer()

	uni_server.RegisterUniServiceServer(
		grpcServer,
		&uni_server.Server{},
	)

	fmt.Printf("UNI gRPC server listening on :%s\n", port)

	return grpcServer.Serve(listener)
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
