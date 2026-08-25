package mstp

/*
Update the configuration of the MSTP port table
*/

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/* ---------------------------------------------------------------------------------- */
/* -------------------------------- MSTP Port Table --------------------------------- */
/* ---------------------------------------------------------------------------------- */

// Update the provided schema tree and return a update, which can be used by the config service
//	Input parameters (settable)
//		priority:
//		pathCost:
//
// 	Input parameters (keys)
//		componentId: name of the component in the bridge
//		port: the port on the bridge
//		mstpId: identifies an instance of a MSTP
//
// 	Input parameters (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made

func SetMstpPortTable(root *st.SchemaTree, priority int, pathCost int,
	componentId uint, mstid uint, port uint, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	portPriorityErr := invalidMstpPortPriority(int32(priority))
	if portPriorityErr != nil {
		return nil, nil, portPriorityErr
	}

	pathCostErr := invalidMstpPortPathCost(int32(pathCost))
	if pathCostErr != nil {
		return nil, nil, pathCostErr
	}

	priorityUpdateTree, priorityUpdatePb := setMstpPortPriority(root, priority, componentId, mstid, port, deviceIp)
	pathCostUpdateTree, pathCostUpdatePb := setMstpPathCost(root, pathCost, componentId, mstid, port, deviceIp)
	return []*st.SchemaTree{priorityUpdateTree, pathCostUpdateTree}, []*pb.Update{priorityUpdatePb, pathCostUpdatePb}, nil
}

/* ---------------------------------------------------------------------------------- */
/* --------------------------- Check if the value is valid -------------------------- */
/* ---------------------------------------------------------------------------------- */

func invalidMstpPortPriority(portPriority int32) error {
	if portPriority >= 0 && portPriority <= 15 {
		return nil
	} else {
		return errors.New("Invalid mstpPortPriority. Value: " + fmt.Sprint(portPriority) + ". 0 <= portPriority <= 15")
	}
}

func invalidMstpPortPathCost(pathCost int32) error {
	if pathCost >= 1 && pathCost <= 200000000 {
		return nil
	} else {
		return errors.New("Invalid mstpPortPathCost. Value: " + fmt.Sprint(pathCost) + ". 1 <= pathCost <= 200000000")
	}
}

/* ---------------------------------------------------------------------------------- */
/* ------------------------------ MSTP Port Table values ---------------------------- */
/* ---------------------------------------------------------------------------------- */

// Update "Port Priority" parameter
// Parameter to set: portPriority (0 <= portPriority <= 15, int32)
// Key parameters: componentId (uint32), mstid (uint32), port (uint32), deviceIp
func setMstpPortPriority(root *st.SchemaTree, portPriority int, componentId uint, mstid uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam3Keys(treeLvl1, pbLvl1, "ieee8021MstpPortTable", "ieee8021MstpPortEntry", "ieee8021MstpPortComponentId", fmt.Sprint(componentId), "ieee8021MstpPortMstId", fmt.Sprint(mstid), "ieee8021MstpPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpPortPriority")
	treeLvl3.Value = fmt.Sprint(portPriority)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbIntTypeVal(portPriority))
	return treeLvl3, update
}

// Update "Port Path Cost" parameter
// Parameter to set: pathCost (1 <= pathCost <= 200000000, int32)
// Key parameters: componentId (uint32), mstid (uint32), port (uint32), deviceIp
func setMstpPathCost(root *st.SchemaTree, pathCost int, componentId uint, mstid uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam3Keys(treeLvl1, pbLvl1, "ieee8021MstpPortTable", "ieee8021MstpPortEntry", "ieee8021MstpPortComponentId", fmt.Sprint(componentId), "ieee8021MstpPortMstId", fmt.Sprint(mstid), "ieee8021MstpPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpPortPathCost")

	treeLvl3.Value = fmt.Sprint(pathCost)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbIntTypeVal(pathCost))
	return treeLvl3, update
}
