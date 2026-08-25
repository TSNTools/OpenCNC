package streamfilterinst

/*
Functions to set each value in the stream filter instance table
*/

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	"tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
	set stream-handle-specification in the frame instance table at the desired port

key parameters:

	port, deviceIp

Parameters to set:

	streamId: id for the stream that the port is for
*/
func setStreamHandleSpecification(root *st.SchemaTree, port string, deviceIp string, streamId int) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-filter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "stream-handle-specification")

	pathLvl2Tree.Value = fmt.Sprint(streamId)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbIntTypeVal(streamId))
	return update
}

/*
	set priority-specification in the frame instance table at the desired port

key parameters:

	port, deviceIp

Parameters to set:

	prio
*/
func setPrioritySpecification(root *st.SchemaTree, port string, deviceIp string, prio int) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-filter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "priority-specification")

	pathLvl2Tree.Value = fmt.Sprint(prio)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbIntTypeVal(prio))
	return update
}

/*
	Link the frame instance table and gate instance table

key parameters:

	port, deviceIp

Parameters to set:

	gateID: the stream gate instance table that it is connected to
*/
func setStreamGateInstanceId(root *st.SchemaTree, port string, deviceIp string, gateID int) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-filter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "stream-gate-instance-id")

	pathLvl2Tree.Value = fmt.Sprint(gateID)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbIntTypeVal(gateID))
	return update
}

/*
	Set all the filter spec tables

key parameters:

	port, deviceIp
	flowId: for which flow meter instance table the max sdu size is for

Parameters to set:

	maxSduSize: max sdu size for an flow meter instance table
*/
func setFilterSpecTables(root *st.SchemaTree, port string, deviceIp string, maxSduSize []uint, flowId []uint) ([]*pb.Update, error) {
	var finalUpdates []*pb.Update

	if len(maxSduSize) != len(flowId) {
		return nil, errors.New("Must be the same amount of maxSduSize and FlowID")
	}

	for i := 0; i < len(maxSduSize); i++ {
		finalUpdates = append(finalUpdates, setFilterSpecTable(root, port, deviceIp, maxSduSize[i], flowId[i]))
	}
	return finalUpdates, nil
}

/*
	Set one filter specification with the desired values

key parameters:

	port, deviceIp
	flowId: for which flow meter instance table the max sdu size is for

Parameters to set:

	maxSduSize: max sdu size for an flow meter instance table
*/
func setFilterSpecTable(root *st.SchemaTree, port string, deviceIp string, maxSduSize uint, flowId uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-filter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam1Key(pathLvl1Tree, pathLvl1Pb, "", "filter-specification-table", "flow-meter-instence-identifier", fmt.Sprint(flowId)) // Unsure if "filter-specification-table" and "filter-specification-entry" or just "filter-specification-table" is used
	pathLvl2Tree.Value = fmt.Sprint(maxSduSize)

	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(maxSduSize))
	return update
}

/*
	set StreamBlockedDueToOversizeFramevalueEnabled in the frame instance table at the desired port

key parameters:

	port, deviceIp

Parameters to set:

	blockedEnabled: the boolean to be set
*/
func setStreamBlockedDueToOversizeFrameEnabled(root *st.SchemaTree, port string, deviceIp string, blockedEnabled bool) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-filter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "stream-blocked-due-to-oversize-frame-enabled")

	pathLvl2Tree.Value = fmt.Sprint(blockedEnabled)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(blockedEnabled))
	return update
}

/*
	set StreamBlockedDueToOversizeFramevalue in the frame instance table at the desired port

key parameters:

	port, deviceIp

Parameters to set:

	blocked: the value it should be set to
*/
func setStreamBlockedDueToOversizeFrame(root *st.SchemaTree, port string, deviceIp string, blocked bool) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-filter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "stream-blocked-due-to-oversize-frame")

	pathLvl2Tree.Value = fmt.Sprint(blocked)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(blocked))
	return update
}
