package flowmeterinst

/*
Set configuration for the flow meter instance table
*/

import (
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	"tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
set identifier for the flow meter instance table

key parameters:

	port, deviceIp

Parameters to set:

	flowId
*/
func setFlowMeterInstanceTableID(root *st.SchemaTree, port string, deviceIp string, flowId uint) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "flow-meter-instance-identifier")

	pathLvl2Tree.Value = fmt.Sprint(flowId)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(flowId))
	return update
}

/*
set Committed Information Rate (CIR) (bits/sec).
CIR provided performance by the SP(Service Provider), exact, or might be set a little higher
key parameters:

	port, deviceIp

Parameters to set:

	cir
*/
func setCommittedInformationRate(root *st.SchemaTree, port string, deviceIp string, cir uint) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "committed-information-rate")

	pathLvl2Tree.Value = fmt.Sprint(cir)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(cir))
	return update
}

/*
set Committed burst Rate (CBS) (octets).
CBS must be at least as large as the MFS (Maximum Frame Size)

key parameters:

	port, deviceIp

Parameters to set:

	cbs
*/
func setCommittedBurstnRate(root *st.SchemaTree, port string, deviceIp string, cbs uint) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "committed-burst-rate")

	pathLvl2Tree.Value = fmt.Sprint(cbs)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(cbs))
	return update
}

/*
set excess information rate (EIR) (bits/sec).
EIR is the additional bit-rate that the subscriber can utilize and
for which traffic can probably pass through the CEN (Carrier Ethernet Network) as long as there is no congestion.

key parameters:

	port, deviceIp

Parameters to set:

	eir
*/
func setExcessInformationRate(root *st.SchemaTree, port string, deviceIp string, eir uint) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "excess-information-rate")

	pathLvl2Tree.Value = fmt.Sprint(eir)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbUintTypeVal(eir))
	return update
}

/*
set Coupling flag (CF) (true or false)

key parameters:

	port, deviceIp

Parameters to set:

	cf
*/
func setCouplingFlag(root *st.SchemaTree, port string, deviceIp string, cf bool) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "coupling-flag")

	pathLvl2Tree.Value = fmt.Sprint(cf)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(cf))
	return update
}

/*
set color mode (CM) (true or false)

key parameters:

	port, deviceIp

Parameters to set:

	cm
*/
func setColorMode(root *st.SchemaTree, port string, deviceIp string, cm bool) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "color-mode")

	pathLvl2Tree.Value = fmt.Sprint(cm)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(cm))
	return update
}

/*
set setDropOnYellow (true or false)

key parameters:

	port, deviceIp

Parameters to set:

	dropOnYellow
*/
func setDropOnYellow(root *st.SchemaTree, port string, deviceIp string, dropOnYellow bool) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "drop-on-yellow")

	pathLvl2Tree.Value = fmt.Sprint(dropOnYellow)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(dropOnYellow))
	return update
}

/*
set mark-all-frame-red-enabled (true or false).
Tells if it matters if variable setRed matters

key parameters:

	port, deviceIp

Parameters to set:

	redEnabled
*/
func setRedEnabled(root *st.SchemaTree, port string, deviceIp string, redEnabled bool) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "mark-all-frame-red-enabled")
	pathLvl2Tree.Value = fmt.Sprint(redEnabled)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(redEnabled))
	return update
}

/*
set mark-all-frame-red
changed to true the bridge, default should be false

key parameters:

	port, deviceIp

Parameters to set:

	red
*/
func setRed(root *st.SchemaTree, port string, deviceIp string, red bool) (update *pb.Update) {
	bridgePathTree, bridgebridgePathPb := path.GetPath2Bridge(root, port)
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(bridgePathTree, bridgebridgePathPb, "flow-meter-instance-table")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "mark-all-frame-red")

	pathLvl2Tree.Value = fmt.Sprint(red)
	update = pbMethods.GetUpdate(deviceIp, pathLvl2Pb, pbMethods.GetPbBoolTypeVal(red))
	return update
}
