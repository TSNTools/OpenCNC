package notification

/*
Start updating the requested configuration
*/

import (
	"fmt"
	"tsn-service/pkg/RAE/mstp"
	handler "tsn-service/pkg/notificationHandler"

	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	"context"

	"google.golang.org/protobuf/types/known/emptypb"
)

//var log = logger.GetLogger()

type Server struct {
	UnimplementedNotificationServer
}

/* ------------------ <MSTP> -------------------------- */
// TODO: Should update the configuration instead of logging

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) UpdateConfigMstpCistPortTable(ctx context.Context, in *InMstpCistPortTableRequest) (empty *emptypb.Empty, err error) {
	//log.Infof("RAE: Starting to update configuration for MSTP Cist port table: %v", in)
	return &emptypb.Empty{}, nil
}

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) UpdateConfigMstpCistTable(ctx context.Context, in *InMstpCistTableRequest) (empty *emptypb.Empty, err error) {
	//log.Infof("RAE: Starting to update configuration for MSTP Cist table: %v", in)
	return &emptypb.Empty{}, nil
}

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) UpdateConfigMstpConfigTable(ctx context.Context, in *InMstpConfigTableRequest) (empty *emptypb.Empty, err error) {
	//log.Infof("RAE: Starting to update configuration for MSTP config table: %v", in)
	return &emptypb.Empty{}, nil
}

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) UpdateConfigMstpFidToMstiV2Table(ctx context.Context, in *InMstpFidToMstiV2TableRequest) (empty *emptypb.Empty, err error) {
	//log.Infof("RAE: Starting to update configuration for MSTP FidToMstiV2 table: %v", in)
	return &emptypb.Empty{}, nil
}

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) UpdateConfigMstpTable(ctx context.Context, in *InMstpTableRequest) (empty *emptypb.Empty, err error) {
	//log.Infof("RAE: Starting to update configuration for MSTP table: %v", in)
	return &emptypb.Empty{}, nil
}

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) UpdateConfigMstpPortTable(ctx context.Context, in *InMstpPortTableRequest) (empty *emptypb.Empty, err error) {

	//log.Infof("RAE: Starting to update configuration for MSTP port table: %v", in)

	var prio int = int(in.Priority)
	var path int = int(in.PathCost)
	var comp uint = uint(in.ComponentID)
	var port uint = uint(in.Port)
	var mstpid uint = uint(in.MstID)
	var deviceIP string = in.DeviceIP

	err = mstp.UpdateMstpPortTable(prio, path, comp, port, mstpid, deviceIP)

	if err != nil {
		//log.Errorf("Failed to update the config: %s", err)
	} else {
		//log.Info("Config has updated successfully")
	}

	return &emptypb.Empty{}, nil
}

/* ------------------ </MSTP> -------------------------- */

// Function provided by gRPC server (entrypoint for calculating new configurations)
func (s *Server) CalcConfig(ctx context.Context, in *IdList) (*UUID, error) {
	fmt.Println("Hello from TSN service!")
	var idStringSlice []string

	ids := in.GetValues()
	fmt.Println("Retrieved values from Main")

	for _, id := range ids {
		idStringSlice = append(idStringSlice, id.GetValue())
	}

	//log.Infof("Received notification to calculate configuration for: %s", ids)
	fmt.Printf("Received notification to calculate configuration for: %s\n", ids)

	configId, err := handler.CalculateConfiguration(idStringSlice)
	if err != nil {
		//log.Errorf("Failed calculating configuration: %v", err)
		fmt.Printf("Failed calculating configuration: %v\n", err)

		return nil, err
	}

	var transportConfId = &UUID{
		Value: configId,
	}

	return transportConfId, nil
}
