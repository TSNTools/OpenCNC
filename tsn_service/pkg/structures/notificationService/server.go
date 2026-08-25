package notificationService

import (
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"golang.org/x/net/context"
)

//var log = logger.GetLogger()

type Server struct {
	UnimplementedNotificationServer
}

// Function provided by gRPC server (entrypoint for calculating new configurations)

func (s *Server) ConfigNotification(ctx context.Context, event *Event) (*Received, error) {

	//log.Infof("Input for RaeNotification: %v", event)

	return &Received{}, nil
}
