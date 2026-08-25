package streamgateinst

/*
Functions to set each value in the stream gate instance table
*/

import (
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	rae "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
	Set the gate-enabled value in the gate parameter table

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	gateEnabled
*/
func SetGateParaTblGateEnabled(root *st.SchemaTree, port string, deviceIp string, gateEnabled bool) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "gate-enabled")

	pathLvl2Tree.Value = fmt.Sprint(gateEnabled)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(gateEnabled))
	return update
}

/*
Set the admin-gate-states value in the gate parameter table

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	adminGateState
*/
func SetGateParaTblAdminGateStates(root *st.SchemaTree, port string, deviceIp string, adminGateState uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-gate-states")

	pathLvl2Tree.Value = fmt.Sprint(adminGateState)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(adminGateState))
	return update
}

/*
Set the admin-control-list-length value in the gate parameter table

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	ctrlListLen
*/
func SetGateParaTblCtrlListLen(root *st.SchemaTree, port string, deviceIp string, ctrlListLen uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-control-list-length")

	pathLvl2Tree.Value = fmt.Sprint(ctrlListLen)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(ctrlListLen))
	return update
}

/*
Set the numerator for the admin cycle time

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	numerator
*/
func SetGateParaTblCycleTimeNum(root *st.SchemaTree, port string, deviceIp string, numerator uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-cycle-time")
	pathLvl3Tree, pathLvl3Pb := rae.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "psfp-admin-cycle-time-numerator")

	pathLvl3Tree.Value = fmt.Sprint(numerator)
	update = pbMethods.GetUpdate(deviceIp, pathLvl3Pb, pbMethods.GetPbUintTypeVal(numerator))
	return update
}

/*
Set the denominator for the admin cycle time

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	denominator
*/
func SetGateParaTblCycleTimeDen(root *st.SchemaTree, port string, deviceIp string, denominator uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-cycle-time")
	pathLvl3Tree, pathLvl3Pb := rae.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "psfp-admin-cycle-time-denominator")

	pathLvl3Tree.Value = fmt.Sprint(denominator)
	update = pbMethods.GetUpdate(deviceIp, pathLvl3Pb, pbMethods.GetPbUintTypeVal(denominator))
	return update
}

/*
Set the cycle time extension

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	extension
*/
func SetGateParaTblCycleTimeExt(root *st.SchemaTree, port string, deviceIp string, extension uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-cycle-time-extension")

	pathLvl2Tree.Value = fmt.Sprint(extension)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(extension))
	return update
}

/* Control list  */

/*
Set the operation name for the controllist with the index as the key
TODO: might not be an string

key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on
	index: Which index in the control list

Parameters to set:

	opername
*/
func SetGateParaTblCtrlListOperName(root *st.SchemaTree, port string, deviceIp string, index int, opername string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam1Key(pathLvl1Tree, pathLvl1Pb, "", "admin-control-list", "index", string(index))
	pathLvl3Tree, pathLvl3Pb := rae.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "operation-name")

	pathLvl3Tree.Value = opername
	update = pbMethods.GetUpdate(deviceIp, pathLvl3Pb, pbMethods.GetPbStringTypeVal(opername))
	return update
}

/*
Set gate state value for the setGatesStates
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on
	index: Which index in the control list

Parameters to set:

	sgsParams
*/
func SetGateParaTblCtrlListSgsGateState(root *st.SchemaTree, port string, deviceIp string, index int, sgsParams uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam1Key(pathLvl1Tree, pathLvl1Pb, "", "admin-control-list", "index", string(index))
	pathLvl3Tree, pathLvl3Pb := rae.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "sgs-params")
	pathLvl4Tree, pathLvl4Pb := rae.GetParam0Keys(pathLvl3Tree, pathLvl3Pb, "gate-states-value")

	pathLvl4Tree.Value = fmt.Sprint(sgsParams)
	update = pbMethods.GetUpdate(deviceIp, pathLvl4Pb, pbMethods.GetPbUintTypeVal(sgsParams))
	return update
}

/*
Set time interval for the setGatesStates
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on
	index: Which index in the control list

Parameters to set:

	timeInter
*/
func SetGateParaTblCtrlListSgsTimeInterval(root *st.SchemaTree, port string, deviceIp string, index int, timeInter uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam1Key(pathLvl1Tree, pathLvl1Pb, "", "admin-control-list", "index", string(index))
	pathLvl3Tree, pathLvl3Pb := rae.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "sgs-params")
	pathLvl4Tree, pathLvl4Pb := rae.GetParam0Keys(pathLvl3Tree, pathLvl3Pb, "time-interval-value")

	pathLvl4Tree.Value = fmt.Sprint(timeInter)
	update = pbMethods.GetUpdate(deviceIp, pathLvl4Pb, pbMethods.GetPbUintTypeVal(timeInter))
	return update
}

/* ---------------------------- <Base times values> ---------------------------------------- */

/*
Set seconds for base time
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	baseTimeSec
*/
func SetGateParaTblBaseTimeSec(root *st.SchemaTree, port string, deviceIp string, baseTimeSec int) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-base-time")

	pathLvl2Tree.Value = fmt.Sprint(baseTimeSec)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbIntTypeVal(baseTimeSec))
	return update
}

/*
Set fractional seconds for base time
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	baseTimeFrac
*/
func SetGateParaTblBaseTimeSecFrac(root *st.SchemaTree, port string, deviceIp string, baseTimeFrac int) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-base-time")

	pathLvl2Tree.Value = fmt.Sprint(baseTimeFrac)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbIntTypeVal(baseTimeFrac))
	return update
}

/* ---------------------------- </Base times values> ---------------------------------------- */

/*
set config changed
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	configChanged
*/
func SetGateParaTblConfigChange(root *st.SchemaTree, port string, deviceIp string, configChanged bool) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "config-change")

	pathLvl2Tree.Value = fmt.Sprint(configChanged)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(configChanged))
	return update
}

// TODO: add to the yang file
/*
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	adminIpv
*/
func setStreamGateInstanceTableAdminIpv(root *st.SchemaTree, port string, deviceIp string, adminIpv uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "admin-ipv")

	pathLvl2Tree.Value = fmt.Sprint(adminIpv)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(adminIpv))
	return update
}

// TODO: add to the yang file
// might not need to use this one (Should be enough with admin ipv)
/*
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	operIpv
*/
func setStreamGateInstanceTableOperIpv(root *st.SchemaTree, port string, deviceIp string, operIpv uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "oper-ipv")

	pathLvl2Tree.Value = fmt.Sprint(operIpv)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(operIpv))
	return update
}

// TODO: add to the yang file
/*
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	drxEnabled
*/
func setStreamGateInstanceTableDrxEnable(root *st.SchemaTree, port string, deviceIp string, drxEnabled bool) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "drx-enabled")

	pathLvl2Tree.Value = fmt.Sprint(drxEnabled)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(drxEnabled))
	return update
}

// TODO: add to the yang file
/*
key parameters:

	port: the port that the table are at
	deviceIp: the device the port is on

Parameters to set:

	drx
*/
func setStreamGateInstanceTableDrx(root *st.SchemaTree, port string, deviceIp string, drx bool) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "gate-parameters")
	pathLvl2Tree, pathLvl2Pb := rae.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "drx")

	pathLvl2Tree.Value = fmt.Sprint(drx)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(drx))
	return update
}
