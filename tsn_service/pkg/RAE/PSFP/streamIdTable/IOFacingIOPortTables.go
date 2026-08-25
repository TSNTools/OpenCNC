package streamIdTable

/*
Functions to set the configuration for the {in/out}Facing{In/Out}putPort list in the stream identification table
*/

import (
	"fmt"
	"tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	"tsn-service/pkg/RAE/dataStructures/composit"

	"tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
update config for InFacingInPort

key parameters:

	port, deviceIp, currStreamHandle

parameters to set:

	IOFacingIOPortList: list of values to be set
	root: The schema tree of the config that should be updated
*/
func setInFacingInPortListIndex(root *SchemaTreeMethods.SchemaTree,
	currStreamHandle string, IOFacingIOPortList []string, port string, deviceIP string) []*pb.Update {

	return setIoFacingIoPortList(root, currStreamHandle, "in-facing-input-port-list", IOFacingIOPortList, port, deviceIP)
}

/*
update config for InFacingOutPort

key parameters:

	port, deviceIp, currStreamHandle

parameters to set:

	IOFacingIOPortList: list of values to be set
	root: The schema tree of the config that should be updated
*/
func setInFacingOutPortListIndex(root *SchemaTreeMethods.SchemaTree,
	currStreamHandle string, IOFacingIOPortList []string, port string, deviceIP string) []*pb.Update {

	return setIoFacingIoPortList(root, currStreamHandle, "in-facing-output-port-list", IOFacingIOPortList, port, deviceIP)
}

/*
update config for outFacinginPort

key parameters:

	port, deviceIp, currStreamHandle

parameters to set:

	IOFacingIOPortList: list of values to be set
	root: The schema tree of the config that should be updated
*/
func setOutFacingInPortListIndex(root *SchemaTreeMethods.SchemaTree,
	currStreamHandle string, IOFacingIOPortList []string, port string, deviceIP string) []*pb.Update {

	return setIoFacingIoPortList(root, currStreamHandle, "out-facing-input-port-list", IOFacingIOPortList, port, deviceIP)
}

/*
update config for outFacingOutPort

key parameters:

	port, deviceIp, currStreamHandle

parameters to set:

	IOFacingIOPortList: list of values to be set
	root: The schema tree of the config that should be updated
*/
func setOutFacingOutPortListIndex(root *SchemaTreeMethods.SchemaTree,
	currStreamHandle string, IOFacingIOPortList []string, port string, deviceIP string) []*pb.Update {

	return setIoFacingIoPortList(root, currStreamHandle, "out-facing-output-port-list", IOFacingIOPortList, port, deviceIP)
}

/*
Set whole {in/out}Facing{In/Out}putPort list, one value at a time

key parameters:

	port, deviceIp, currStreamHandle

parameters to set:

	IOFacingIOPortList: list of values to be set
	root: The schema tree of the config that should be updated
*/
func setIoFacingIoPortList(root *SchemaTreeMethods.SchemaTree,
	currStreamHandle string, whichIoIo string, IOFacingIOPortList []string, port string, deviceIP string) (updates []*pb.Update) {

	for i := 0; i < len(IOFacingIOPortList); i++ {
		updates = append(updates, setIoFacingIoPortListIndex(
			root, currStreamHandle, i, whichIoIo, IOFacingIOPortList, port, deviceIP))
	}
	return updates
}

/*
Set {in/out}Facing{In/Out}putPort value at specific index

key parameters:

	port, deviceIp, currStreamHandle
	index: index that the FacingPort is for
	whichIoIo: which combination of in- and output should be used

parameters to set:

	IOFacingIOPortList: the value to be set
	root: The schema tree of the config that should be updated
*/
func setIoFacingIoPortListIndex(root *SchemaTreeMethods.SchemaTree, streamHandle string,
	index int, whichIoIo string, IOFacingIOPortList []string, port string, deviceIp string) (update *pb.Update) {

	// interfaces -> interface -> bridgePort -> stream-identification -> stream-handle -> IoFacingIoPor
	bridgePathTree, bridgePathPt := composit.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := composit.GetParam0Keys(bridgePathTree, bridgePathPt, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := composit.GetParam1Key(
		pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathTree, pathPb := composit.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, whichIoIo+fmt.Sprint(index))
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(IOFacingIOPortList[index]))
	pathTree.Value = IOFacingIOPortList[index]
	return update
}
