package notificationServer

import (
	"fmt"
	"net"

	"tsn-service/pkg/structures/notification"
	"tsn-service/pkg/structures/notificationService"

	//	"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"google.golang.org/grpc"
)

//var log = logger.GetLogger()

// Create gRPC server
func CreateServer(protocol string, addr string) {
	lis, err := net.Listen(protocol, addr)
	if err != nil {
		fmt.Println("Failed to listen: %v", err)
		//	log.Fatalf("Failed to listen: %v", err)
	}
	fmt.Println("Listening on %v", addr)

	//log.Infof("Now listening on %v", addr)

	s := notification.Server{}

	grpcServer := grpc.NewServer()

	fmt.Println("Created grpc server!")

	//log.Info("Created grpc server!")

	notification.RegisterNotificationServer(grpcServer, &s)
	fmt.Println("Started to serve...")

	//log.Info("Starting to serve...")

	err = grpcServer.Serve(lis)
	if err != nil {
		//	log.Fatalf("Failed to serve: %v", err)
	}
}

func CreateNotificationServiceServer(protocol string, addr string) {
	lis, err := net.Listen(protocol, addr)
	if err != nil {
		//log.Fatalf("Failed to listen: %v", err)
	}

	//log.Infof("Now listening on %v", addr)

	s := notificationService.Server{}

	grpcServer := grpc.NewServer()

	//log.Info("Created grpc server!")

	notificationService.RegisterNotificationServer(grpcServer, &s)

	//log.Info("Starting to serve...")

	err = grpcServer.Serve(lis)
	if err != nil {
		//log.Fatalf("Failed to serve: %v", err)
	}
}
