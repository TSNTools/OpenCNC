package pcp

/*
Help functions for pcp.go
*/

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
To map PCP, drop eligible and priority when decoding depending on the PCP type
Ref: row of table 6-3 in IEEE 802.1Q-2018 6.9.3
Input

	pcpType: the PCP type used

Output

	pcp: pcp (priority)
	dei: drop eligible
*/
func getPCPDecoding(pcpType string) (pcp []int, dei []bool) {
	switch pcpType {
	case "8P0D":
		pcp = []int{7, 6, 5, 4, 3, 2, 1, 0}
		dei = []bool{false, false, false, false, false, false, false, false}
	case "7P1D":
		pcp = []int{7, 6, 4, 4, 3, 2, 1, 0}
		dei = []bool{false, false, false, true, false, false, false, false}
	case "6P2D":
		pcp = []int{7, 6, 4, 4, 2, 2, 1, 0}
		dei = []bool{false, false, false, true, false, true, false, false}
	case "5P3D":
		pcp = []int{7, 6, 4, 4, 2, 2, 1, 0}
		dei = []bool{false, false, false, true, false, true, false, true}
	default:
		return nil, nil
	}
	return pcp, dei
}

/*
Return the list that maps the ports traffic classes to the frames priority
Is dependent on the number of traffic classes (queues) at the port
Ref: IEEE 802.1Q 8.6.6, table 8-5
input:

	nrTrafficClasses: nr of queues (must be 1-8)

output:

	prio: priority mapping list after nr of traffic classes
*/
func getDefaultTrafficClasses(nrTrafficClasses int) (prio []int, err error) {
	switch nrTrafficClasses {
	case 1:
		prio = []int{0, 0, 0, 0, 0, 0, 0, 0}
	case 2:
		prio = []int{0, 0, 0, 0, 1, 1, 1, 1}
	case 3:
		prio = []int{0, 0, 0, 0, 1, 1, 2, 2}
	case 4:
		prio = []int{0, 0, 1, 1, 2, 2, 3, 3}
	case 5:
		prio = []int{0, 0, 1, 1, 2, 2, 3, 4}
	case 6:
		prio = []int{1, 0, 2, 2, 3, 3, 4, 5}
	case 7:
		prio = []int{1, 0, 2, 3, 4, 4, 5, 6}
	case 8:
		prio = []int{1, 0, 2, 3, 4, 5, 6, 7}
	default:
		err = errors.New(fmt.Sprint(nrTrafficClasses) + "is not a valid number of traffic classes")
		return nil, err
	}
	return prio, nil
}

/*
Get priority point value for encoding table
input:

	pcpType: the pcp type that is used
	Priority: the priority of the stream
	dei: Drop eligibility

Output:

	prioPointVal: the priority point value that the pcp type, prio and dei points to in the encoding table
*/
func getPriorityPointValue(pcpType string, priority int, dei bool) (prioPointVal int, err error) {
	deiFalse8P0D := []int{0, 1, 2, 3, 4, 5, 6, 7}
	deiTrue8P0D := []int{0, 1, 2, 3, 4, 5, 6, 7}
	deiFalse7P1D := []int{0, 1, 2, 3, 5, 5, 6, 7}
	deiTrue7P1D := []int{0, 1, 2, 3, 4, 4, 6, 7}
	deiFalse6P2D := []int{0, 1, 3, 3, 5, 5, 6, 7}
	deiTrue6P2D := []int{0, 1, 2, 2, 4, 4, 6, 7}
	deiFalse5P3D := []int{1, 1, 3, 3, 5, 5, 6, 7}
	deiTrue5P3D := []int{0, 0, 2, 2, 4, 4, 6, 7}

	if priority < 0 || priority > 7 {
		prioPointVal = -1
		err = errors.New("Priority (" + fmt.Sprint(priority) + ") is out of range. Range is [0-7]")
		return prioPointVal, err
	}
	err = nil
	if dei {
		switch pcpType {
		case "8P0D":
			prioPointVal = deiTrue8P0D[priority]
		case "7P1D":
			prioPointVal = deiTrue7P1D[priority]
		case "6P2D":
			prioPointVal = deiTrue6P2D[priority]
		case "5P3D":
			prioPointVal = deiTrue5P3D[priority]
		default:
			prioPointVal = -1
			err = errors.New(pcpType + " is an invalid PCP type. Valid PCP types are: 8P0D, 7P1D, 6P2D, 5P3D")
		}
	} else if !dei {
		switch pcpType {
		case "8P0D":
			prioPointVal = deiFalse8P0D[priority]
		case "7P1D":
			prioPointVal = deiFalse7P1D[priority]
		case "6P2D":
			prioPointVal = deiFalse6P2D[priority]
		case "5P3D":
			prioPointVal = deiFalse5P3D[priority]
		default:
			prioPointVal = -1
			err = errors.New(pcpType + " is an invalid PCP type. Valid PCP types are: 8P0D, 7P1D, 6P2D, 5P3D")
		}
	} else {
		prioPointVal = -1
		err = errors.New("Bug in the code! dei is not an boolean")
	}
	return prioPointVal, err
}

// TODO: Control that PCP an PCPtype has not been mixed up
/* For the decoding table: get the priority and drop eligible, depending on the pcpType and pcp

input:
	pcpType: the pcp type used, aka "8P0D", "7P1D", "6P2D", "5P3D"
	pcp: Priority code point

output:
	prio: priority
	dei: drop eligible
*/
func getPriorityAndDropEligibleValue(pcpType string, pcp int) (prio int, dei bool, err error) {
	priority8P0D := []int{0, 1, 2, 3, 4, 5, 6, 7}
	dei8P0D := []bool{false, false, false, false, false, false, false, false}
	priority7P1D := []int{0, 1, 2, 3, 4, 4, 6, 7}
	dei7P1D := []bool{false, false, false, false, true, false, false, false}
	priority6P2D := []int{0, 1, 2, 2, 4, 4, 6, 7}
	dei6P2D := []bool{false, false, true, false, true, false, false, false}
	priority5P3D := []int{0, 0, 2, 2, 4, 4, 6, 7}
	dei5P3D := []bool{true, false, true, false, true, false, false, false}

	// if invalid pcp (can be max 8 queues)
	if pcp < 0 || pcp > 7 {
		prio = -1
		dei = false
		err = errors.New("PCP value (" + fmt.Sprint(pcp) + ") is out of range. Range is [0-7]")
	} else {
		err = nil
		switch pcpType {
		case "8P0D":
			prio = priority8P0D[pcp]
			dei = dei8P0D[pcp]
		case "7P1D":
			prio = priority7P1D[pcp]
			dei = dei7P1D[pcp]
		case "6P2D":
			prio = priority6P2D[pcp]
			dei = dei6P2D[pcp]
		case "5P3D":
			prio = priority5P3D[pcp]
			dei = dei5P3D[pcp]
		default:
			prio = -1
			dei = false
			err = errors.New(pcpType + " is an invalid PCP type. Valid PCP types are: 8P0D, 7P1D, 6P2D, 5P3D")
		}
	}
	return prio, dei, err
}

/*
	Update to set the values in pcp encoding table

key parameters:

	port, deviceIp

parameters to set:

	pcpType: PCP type, aka "8P0D", "7P1D", "6P2D", "5P3D"
	prio: priority
	dei: Drop eligible
	pcp: Priority Code Point
*/
func setPcpEncodingTableValue(root *st.SchemaTree, port string, deviceIp string,
	pcpType string, prio int, dei bool, pcp int) (update *pb.Update) {

	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)

	pathEncodingTree, pathEncodingPb :=
		path.GetParam1Key(bridgePathTree, bridgePathPb, "", "pcp-encoding-table", "pcp", pcpType)

	pathPrioMapTree, pathPrioMapPb :=
		path.GetParam2Keys(pathEncodingTree, pathEncodingPb, "", "priority-map", "priority", fmt.Sprint(prio), "dei", fmt.Sprint(dei))

	pathTree, pathPb := path.GetParam0Keys(pathPrioMapTree, pathPrioMapPb, "priority-code-point")
	pathTree.Value = fmt.Sprint(pcp)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(uint(pcp)))
	return update
}

/*
	Update to set the values in pcp decoding table, one update if drop eligible is true, one if drop eligible is false

key parameters:

	port, deviceIp

parameters to set:

	pcpType: PCP type, aka "8P0D", "7P1D", "6P2D", "5P3D"
	priority:
	dei: Drop eligible
	pcp: Priority Code Point
*/
func setPcpDecodingTableValue(root *st.SchemaTree, port string, deviceIp string,
	pcpType string, pcp int, priority int, dei bool) (updates []*pb.Update) {

	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)

	pathDecodingTree, pathDecodingPb :=
		path.GetParam1Key(bridgePathTree, bridgePathPb, "", "pcp-decoding-table", "pcp", pcpType)

	pathPrioMapTree, pathPrioMapPb :=
		path.GetParam1Key(pathDecodingTree, pathDecodingPb, "", "priority-map", "priority-code-point", fmt.Sprint(pcp))

	pathPrioTree, pathPrioPb := path.GetParam0Keys(pathPrioMapTree, pathPrioMapPb, "priority")
	PcpDecodingTablePriorityUpdate := pbMethods.GetUpdate(deviceIp, pathPrioPb, pbMethods.GetPbUintTypeVal(uint(priority)))
	pathPrioTree.Value = fmt.Sprint(priority)

	pathDeiTree, pathDeiPb := path.GetParam0Keys(pathPrioMapTree, pathPrioMapPb, "dei")
	pathDeiTree.Value = fmt.Sprint(dei)
	PcpDecodingTableDropEligibleUpdate := pbMethods.GetUpdate(deviceIp, pathDeiPb, pbMethods.GetPbBoolTypeVal(dei))

	updates = []*pb.Update{PcpDecodingTablePriorityUpdate, PcpDecodingTableDropEligibleUpdate}

	return updates
}

/*
	Build update for change traffic class priority

key parameters:

	port, deviceIp

parameters to set:

	priority: which priority should be set
	priorityIndex: index for where the priority should be put in the config
*/
func setTrafficClassPriorityAtIndex(root *st.SchemaTree, priorityIndex int, priority int, port string, deviceIp string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathTrafficClassTree, pathTrafficClassPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "traffic-class")
	pathTree, pathPb := path.GetParam0Keys(pathTrafficClassTree, pathTrafficClassPb, "priority"+fmt.Sprint(priorityIndex))
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(uint(priority)))
	pathTree.Value = fmt.Sprint(priority)
	return update
}

/*
	Build update for change priority regeneration

key parameters:

	port, deviceIp

parameters to set:

	priority: which priority should be set
	priorityIndex: index for where the priority should be put in the config
*/
func setPriorityRegenerationAtIndex(root *st.SchemaTree, priorityIndex int, priority int, port string, deviceIp string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathPrioRegTree, pathPrioRegPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "priority-regeneration")
	pathTree, pathPb := path.GetParam0Keys(pathPrioRegTree, pathPrioRegPb, "priority"+fmt.Sprint(priorityIndex))
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(uint(priority)))
	pathTree.Value = fmt.Sprint(priority)
	return update
}
