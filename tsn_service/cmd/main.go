package main

import (
	"fmt"
	"time"
	"tsn-service/pkg/internalOptimizer"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	server "tsn-service/pkg/notificationServer"
	store "tsn-service/pkg/storewrapper"
)

//var log = logger.GetLogger()

func main() {

	// Create default schedule and store it in k/v store
	if err := internalOptimizer.CreateDefaultSchedule(); err != nil {
		//log.Fatalf("Failed creating default schedule: %v", err)
		return
	}
	fmt.Println("Lets start the server!")
	// Used to get device configuration and config+state data from a device
	// go test()

	// Start notification-server
	go server.CreateServer("tcp", ":5150")
	fmt.Println("Created and listening to 5150!")

	go server.CreateNotificationServiceServer("tcp", ":5151")

	fmt.Println("Created and listening to 5151!")

	select {}
}

func test() {
	time.Sleep(time.Second * 90)

	tree, err := store.GetDeviceConfig("192.168.0.2")
	if err != nil {
		//	log.Errorf("Failed getting device config: %v", err)
		return
	}

	store.StoreDeviceConfig("192.168.0.2", tree)
}
