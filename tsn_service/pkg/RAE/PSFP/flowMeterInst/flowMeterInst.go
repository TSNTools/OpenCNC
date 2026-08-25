package flowmeterinst

/*
To set the values of the flow meter instance table

Ref:

	https://wiki.mef.net/display/CESG/Bandwidth+Profile

TODO:
	Implement its paths in the yang file (can not find it in the yang files)
*/
import (
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
	to set all values in the flow meter instance table

key parameters:

	port, deviceIp

Parameters to set:

	flowId: identifier for this flow meter instance table
	cir: Committed Information Rate (bits/sec)
	cbs: Committed Burst Size (octets)
	eir: Excess Information Rate (bits/sec)
	ebs: Excess Burst size (octets)
	cf: Coupling Flag (true or false), defalut false
	colorModeEnabled: (color aware or blind), defalut false (blind)
	dropOnYellow: (true or false), defalut false
	markAllFrameRedEnabled: (true or false), defalut false
	markAllFrameRed: (true or false), defalut false
*/
func setFlowMeterInstanceTable(root *st.SchemaTree, port string, deviceIp string,
	flowId uint, cir uint, cbs uint, eir uint, ebs uint, cf bool, cm bool, dropOnYellow bool,
	markAllFrameRedEnabled bool, markAllFrameRed bool) (updates []*pb.Update, err error) {

	updates = append(updates, setFlowMeterInstanceTableID(root, port, deviceIp, flowId))
	updates = append(updates, setCommittedInformationRate(root, port, deviceIp, cir))
	updates = append(updates, setCommittedBurstnRate(root, port, deviceIp, cbs))
	updates = append(updates, setExcessInformationRate(root, port, deviceIp, eir))
	updates = append(updates, setCouplingFlag(root, port, deviceIp, cf))
	updates = append(updates, setColorMode(root, port, deviceIp, cm))
	updates = append(updates, setDropOnYellow(root, port, deviceIp, dropOnYellow))
	updates = append(updates, setRedEnabled(root, port, deviceIp, markAllFrameRedEnabled))
	updates = append(updates, setRed(root, port, deviceIp, markAllFrameRed))

	return updates, nil
}

/*
Default flow metering instance table
*/
func setDefaultFlowMeterInstanceTable(root *st.SchemaTree, port string, deviceIp string,
	flowId uint, cir uint, cbs uint, eir uint, ebs uint) (updates []*pb.Update, err error) {

	var cf bool = false
	var cm bool = false
	var dropOnYellow bool = false
	var markAllFrameRedEnabled bool = false
	var markAllFrameRed bool = false

	updates, err = setFlowMeterInstanceTable(root, port, deviceIp,
		flowId, cir, cbs, eir, ebs, cf, cm, dropOnYellow,
		markAllFrameRedEnabled, markAllFrameRed)

	return updates, err
}
