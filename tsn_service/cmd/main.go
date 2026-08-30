package main

import (
	"fmt"
	"log"
	"net"

	"OpenCNC/tsn_service/pkg/internalOptimizer"
	"OpenCNC/tsn_service/pkg/notificationServer"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"google.golang.org/grpc"
)

//var log = logger.GetLogger()

func main() {

	// Create default schedule and store it in k/v store
	if err := internalOptimizer.CreateDefaultSchedule(); err != nil {
		log.Fatalf("Failed creating default schedule: %v", err)
		return
	}
	fmt.Println("Lets start the server!")
	// Used to get device configuration and config+state data from a device
	// go test()

	// Start notification-server
	go CreateNotificationServer("tcp", ":5150")
	fmt.Println("Created and listening to 5150!")

	select {}
}

func CreateNotificationServer(protocol string, addr string) {
	lis, err := net.Listen(protocol, addr)
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	fmt.Printf("Listening on %v\n", addr)

	s := notificationServer.Server{}

	grpcServer := grpc.NewServer()

	notificationServer.RegisterNotificationServer(grpcServer, &s)

	fmt.Println("Started to serve...")

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}

/*
func test() {
	time.Sleep(time.Second * 90)

	tree, err := store.GetDeviceConfig("192.168.0.2")
	if err != nil {
		//	log.Errorf("Failed getting device config: %v", err)
		return
	}

	store.StoreDeviceConfig("192.168.0.2", tree)
}
*/
