package streamgateinst

/*
Functions to set the tables that are required for PSFP (Per-Stream Filtering and Policy)
Which is an requirement for TSN traffic (IEEE 802.1 QCC 46.1.3.3, e)

Ref (for all methods):
	IEEE 802.1Q 8.6.5.1 - 8.6.5.3
	IEEE 802.1Q 12.31
	ieee802-dot1q-sched (gate-parameters)
*/

import (
	"errors"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"

	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

//var log = logger.GetLogger()

/*
the stream gate instance/parameters table
Ref: IEEE 802.1 Q 8.6.5.1.2 and IEEE Q 12.31.3

key parameter:

	port: port the table is at
	deviceIp: device the port is on

parameters to set:

	gateEnabled(bool):
		if traffic scheduling should be active or not
		default: true
	adminGateState (uint8):
		if a gate should be open or not
		Default: all open (aka 255) // 8.6.9.4.5
	ctrlListLen:
		psfp admin control list lenght: Num of entries in PSFPAdminControlList
		default; 0
	For each PSFPAdminControlList:
		opNameList:
			operation name
		gatesStatesOpenCtrlList:
			PSFPgateStateValue: initial gate state (open or closed)
			Default: Open
		ipvCtrlList:
			IPV (Internal Priority Value)
			Default: null
		timeIntervalCtrlList:
			time interval value, how many ticks so far
			default 0 (as an start value)
	PSFP-Admin-Cycle-Time (cycleTimeNumerator/cycleTimeDenominator)
		Rational number of seconds between gate operations (numerator/denominator)
		The time it takes for a sequence of gate operations to repeat themselves
	cycleTimeExtension: (uint)
		Permitted time that the gating cycle can extend in nanoseconds
	adminBaseTime:
		psfp-admin-base-time; PTP start time
	configChanged
		psfp-config-change; start variable for the list config state machine
		set to true by managment, false by the state machine
		default true
	adminIpv
		default null
	operIpv
		default null
	drxEnabled
		If the drx value should be take into consideration
		default false
	drx
		default false
*/
func setStreamGateInstanceTable(root *st.SchemaTree, port string, deviceIp string,
	gateEnabled bool, adminGateState uint, ctrlListLen uint,
	opNameList []string, sgsGateState []uint, sgsTimeInt []uint, ipvCtrlList []uint, timeIntervalCtrlList []uint,
	cycleTimeNumerator uint, cycleTimeDenominator uint, cycleTimeExtension uint,
	baseTimeSec int, baseTimeFrac int, configChanged bool, adminIpv uint, operIpv uint, drxEnabled bool, drx bool) (
	updates []*pb.Update, err error) {

	updates = append(updates, SetGateParaTblGateEnabled(root, port, deviceIp, gateEnabled))
	updates = append(updates, SetGateParaTblAdminGateStates(root, port, deviceIp, adminGateState))
	updates = append(updates, SetGateParaTblCtrlListLen(root, port, deviceIp, ctrlListLen))

	// Not a 100 that the path for this is correct
	if int(ctrlListLen) < len(opNameList) {
		err = errors.New("the control list lenght are less than the number of control list, which it should not be")
	} else {
		// set control lists
		for i := 0; i < len(opNameList); i++ {
			updates = append(updates, SetGateParaTblCtrlListOperName(root, port, deviceIp, i, opNameList[i]))
			updates = append(updates, SetGateParaTblCtrlListSgsGateState(root, port, deviceIp, i, sgsGateState[i]))
			updates = append(updates, SetGateParaTblCtrlListSgsTimeInterval(root, port, deviceIp, i, sgsTimeInt[i]))
		}

	}

	// psfp admin cycle time
	updates = append(updates, SetGateParaTblCycleTimeNum(root, port, deviceIp, cycleTimeNumerator))
	updates = append(updates, SetGateParaTblCycleTimeDen(root, port, deviceIp, cycleTimeDenominator))

	updates = append(updates, SetGateParaTblCycleTimeExt(root, port, deviceIp, cycleTimeExtension))

	// base time, defined in the yang file?
	updates = append(updates, SetGateParaTblBaseTimeSec(root, port, deviceIp, baseTimeSec))
	updates = append(updates, SetGateParaTblBaseTimeSecFrac(root, port, deviceIp, baseTimeFrac))

	updates = append(updates, SetGateParaTblConfigChange(root, port, deviceIp, configChanged))

	// Not in the yang file know
	updates = append(updates, setStreamGateInstanceTableAdminIpv(root, port, deviceIp, adminIpv))
	updates = append(updates, setStreamGateInstanceTableOperIpv(root, port, deviceIp, operIpv))
	updates = append(updates, setStreamGateInstanceTableDrxEnable(root, port, deviceIp, drxEnabled))
	updates = append(updates, setStreamGateInstanceTableDrx(root, port, deviceIp, drx))

	return updates, nil
}
