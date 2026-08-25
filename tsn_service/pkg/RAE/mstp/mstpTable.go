package mstp

/*
Update the configuration on the MSTP Table
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
/* ----------------------------------- MSTP Table ----------------------------------- */
/* ---------------------------------------------------------------------------------- */

// Sets the values of the MSTP Table
//
//	Input parameters (settable):
//		bridgePriority:
//
//	Input parameters (keys)
//		componentId: name of the component in the bridge
//		mstpId: identifies an instance of a MSTP
//
//	Input parameters (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetMstpTable(root *st.SchemaTree, bridgePriority int32, mstpId uint, componentId uint, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	bridgePriorityError := invalidMstpTableBridgePriority(bridgePriority)
	if bridgePriorityError != nil {
		return nil, nil, bridgePriorityError
	}
	bridgePriorityUpdateTree, bridgePriorityUpdatePb := setMstpBridgePriority(root, bridgePriority, componentId, mstpId, deviceIp)
	return []*st.SchemaTree{bridgePriorityUpdateTree}, []*pb.Update{bridgePriorityUpdatePb}, nil
}

/* ---------------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ------------------------------ */
/* ---------------------------------------------------------------------------------- */

// Check that the bridge priority is valid [0, 15]
func invalidMstpTableBridgePriority(bridgePriority int32) error {
	if bridgePriority >= 0 && bridgePriority <= 15 {
		return nil
	} else {
		return errors.New("Invalid MstpTableBridgePriority. Value:" + fmt.Sprint(bridgePriority) + ". 0 <= bridgePriority <= 15")
	}
}

/* ---------------------------------------------------------------------------------- */
/* ----------------------- MSTP Table Values ---------------------------------------- */
/* ---------------------------------------------------------------------------------- */

// Update "Bridge Priority" parameter
// Parameter to set: bridgePriority (0 <= bridgePriortiy <= 15)
// Key parameters: componentId, port, deviceIp
func setMstpBridgePriority(root *st.SchemaTree, bridgePriority int32, msti uint, componentId uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")

	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpTable", "ieee8021MstpEntry", "ieee8021MstpComponentId", fmt.Sprint(componentId), "ieee8021MstpId", fmt.Sprint(msti))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpBridgePriority")
	treeLvl3.Value = fmt.Sprint(bridgePriority)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbIntTypeVal(int(bridgePriority)))
	return treeLvl3, update
}
