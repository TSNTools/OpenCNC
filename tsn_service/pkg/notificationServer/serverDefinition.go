package notificationServer

import (
	"fmt"
	"net"

	"OpenCNC/tsn_service/pkg/structures/notification"

	"google.golang.org/grpc"
)

func CreateServer(protocol string, addr string) {
	lis, err := net.Listen(protocol, addr)
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	fmt.Printf("Listening on %v\n", addr)

	s := Server{}

	grpcServer := grpc.NewServer()

	notification.RegisterNotificationServer(grpcServer, &s)

	fmt.Println("Started to serve...")

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}
