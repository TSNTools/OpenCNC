package gnmi

import (
	"context"
	"log"

	gnmi "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
)

// GNMIService implements gnmi.gNMI
type GNMIService struct {
	gnmi.UnimplementedGNMIServer
	logger *log.Logger
}

func NewGNMIService(logger *log.Logger) *GNMIService {
	return &GNMIService{logger: logger}
}

func (s *GNMIService) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	s.logger.Println("[gNMI] GetRequest:", req)
	return &gnmi.GetResponse{}, nil
}

func (s *GNMIService) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	s.logger.Println("[gNMI] SetRequest:", req)
	return &gnmi.SetResponse{}, nil
}

// Updated Subscribe signature for gNMI v1.x
func (s *GNMIService) Subscribe(stream grpc.BidiStreamingServer[gnmi.SubscribeRequest, gnmi.SubscribeResponse]) error {
	s.logger.Println("[gNMI] Subscribe called")
	// Example: read requests in a loop (can ignore for minimal stub)
	for {
		req, err := stream.Recv()
		if err != nil {
			s.logger.Println("[gNMI] Subscribe recv error:", err)
			return err
		}
		s.logger.Println("[gNMI] Received SubscribeRequest:", req)
	}
}
