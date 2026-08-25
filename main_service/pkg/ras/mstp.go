package ras

/*
Functions to send request to update configuration for MSTP (Multiple Spanning Tree Protocol) tables
*/

import (
	"context"
	"fmt"
	notification "main-service/pkg/structures/notification"

	"google.golang.org/grpc"
	// "git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

// Tell RAE to set configuration for MSTP Cist Port Table
func setConfigMstpCistPortTable(
	newPathCost int32, newEdgePort bool, newMacEnabled bool, newRestrictedRole bool,
	newRestrictedTcn bool, newProtocolMigration bool, newEnableBPDURx bool, newEnableBPDUTx bool,
	newPseudoRootId []byte, newIsL2Gp bool, newPort uint32, newComponentID uint32,
	newDeviceIP string, newKVGetter bool, newCSSetter bool) (err error) {

	var conn *grpc.ClientConn
	conn, err = grpc.Dial("tsn-service:5150", grpc.WithInsecure())
	if err != nil {
		//log.Fatalf("Failed dialing tsn-service: %v", err)
		fmt.Printf("Failed dialing tsn-service: %v", err)

		return err
	}

	defer conn.Close()

	client := notification.NewNotificationClient(conn)

	var input = notification.InMstpCistPortTableRequest{
		PathCost:          newPathCost,
		EdgePort:          newEdgePort,
		MacEnabled:        newMacEnabled,
		RestrictedRole:    newRestrictedRole,
		RestrictedTcn:     newRestrictedTcn,
		ProtocolMigration: newProtocolMigration,
		EnableBPDURx:      newEnableBPDURx,
		EnableBPDUTx:      newEnableBPDUTx,
		PseudoRootId:      newPseudoRootId,
		IsL2Gp:            newIsL2Gp,
		Port:              newPort,
		ComponentID:       newComponentID,
		DeviceIP:          newDeviceIP,
		KVGetter:          newKVGetter,
		CSSetter:          newCSSetter,
	}
	_, answer := client.UpdateConfigMstpCistPortTable(context.Background(), &input)
	fmt.Printf("Response from TSN-RAE for MSTP Cist port table: %v", answer)

	//log.Infof("Response from TSN-RAE for MSTP Cist port table: %v", answer)
	return nil
}

// Tell RAE to set configuration for MSTP Cist Table
func setConfigMstpCistTable(
	newMaxHops int32, newComponentID uint32, newDeviceIP string,
	newKVGetter bool, newCSSetter bool) (err error) {

	var conn *grpc.ClientConn
	conn, err = grpc.Dial("tsn-service:5150", grpc.WithInsecure())
	if err != nil {
		//log.Fatalf("Failed dialing tsn-service: %v", err)
		fmt.Printf("Failed dialing tsn-service: %v", err)

		return err
	}

	defer conn.Close()

	client := notification.NewNotificationClient(conn)

	var input = notification.InMstpCistTableRequest{
		MaxHops:     newMaxHops,
		ComponentID: newComponentID,
		DeviceIP:    newDeviceIP,
		KVGetter:    newKVGetter,
		CSSetter:    newCSSetter,
	}
	_, answer := client.UpdateConfigMstpCistTable(context.Background(), &input)
	fmt.Printf("Response from TSN-RAE for MSTP Cist port table: %v", answer)

	//log.Infof("Response from TSN-RAE for MSTP Cist port table: %v", answer)
	return nil
}

/*
To send a request to update the MSTP´s port table at specific device and port, with specific MSTP id
input, keys:

	componentId
	port
	mstid

input, values to set:

	prio
	pathCost
*/
func setConfigMstpPortTable(newPrio int32, newPathCost int32, keycomponentId uint32,
	keyport uint32, keymstid uint32, newKVGetter bool, newCSSetter bool) (err error) {

	var conn *grpc.ClientConn
	conn, err = grpc.Dial("tsn-service:5150", grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed dialing tsn-service: %v", err)

		return err
	}

	defer conn.Close()

	client := notification.NewNotificationClient(conn)

	var input = notification.InMstpPortTableRequest{
		Priority:    newPrio,
		PathCost:    newPathCost,
		ComponentID: keycomponentId,
		Port:        keyport,
		MstID:       keymstid,
		KVGetter:    newKVGetter,
		CSSetter:    newCSSetter,
	}
	_, answer := client.UpdateConfigMstpPortTable(context.Background(), &input)
	fmt.Printf("Response from TSN-RAE for MSTP port table: %v", answer)

	//log.Infof("Response from TSN-RAE for MSTP port table: %v", answer)
	return nil
}
