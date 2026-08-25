package mstp

/*
Update the MSTP CIST port table
*/

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Update the configuration on the MSTP CIST port table
// schema tree config with the provided input, and return an pb.update of the same configuration
//
//	Input parameters (settable)
//		pathCost:
//		edgePort:
//		macEnabled:
//		restricedRole:
//		restrictedTcn:
//		protocolMigration:
//		enableBPDURx:
//		enableBPDUTx:
//		pseudoRootId:
//		IsL2Gp:
//
//	Input parameters (keys)
//		componentId: name of the component in the bridge
//		port: the port on the bridge
//
//	Input parameters (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetMstpCistPortTable(root *st.SchemaTree, pathCost int, edgePort bool, macEnabled bool, restrictedRole bool,
	restrictedTcn bool, protocolMigration bool, enableBPDURx bool, enableBPDUTx bool, pseudoRootId []byte,
	isL2Gp bool, port uint, componentId uint, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {

	invalidPseudoRootIdErr := invalidPseudoRootId(pseudoRootId)
	if invalidPseudoRootIdErr != nil {
		return nil, nil, invalidPseudoRootIdErr
	}

	adminPathCostErr := invalidAdminPathCost(pathCost)
	if adminPathCostErr != nil {
		return nil, nil, adminPathCostErr
	}

	adminPathCostUpdateTree, adminPathCostUpdatePb := setMstpCistPortAdminPathCost(root, pathCost, componentId, port, deviceIp)
	adminEdgePortUpdateTree, adminEdgePortUpdatePb := setMstpCistPortAdminEdgePort(root, edgePort, componentId, port, deviceIp) // likely default false, but yang file says otherwise
	macEnabledUpdateTree, macEnabledUpdatePb := setMstpCistPortMacEnabled(root, macEnabled, componentId, port, deviceIp)
	restrictedRoleUpdateTree, restrictedRoleUpdatePb := setMstpCistPortRestrictedRole(root, restrictedRole, componentId, port, deviceIp)
	restrictedTcnUpdateTree, restrictedTcnUpdatePb := setMstpCistPortRestrictedTcn(root, restrictedTcn, componentId, port, deviceIp)
	protocolMigrationUpdateTree, protocolMigrationUpdatePb := setMstpCistPortProtocolMigration(root, protocolMigration, componentId, port, deviceIp)
	enableBPDURxUpdateTree, enableBPDURxUpdatePb := setMstpCistPortEnableBPDURx(root, enableBPDURx, componentId, port, deviceIp)
	enableBPDUTxUpdateTree, enableBPDUTxUpdatePb := setMstpCistPortEnableBPDUTx(root, enableBPDUTx, componentId, port, deviceIp)
	pseudoRootIdUpdateTree, pseudoRootIdUpdatePb := setMstpCistPortPseudoRootId(root, pseudoRootId, componentId, port, deviceIp)
	isL2GpUpdateTree, isL2GpUpdatePb := setMstpCistPortIsL2Gp(root, isL2Gp, componentId, port, deviceIp)

	return []*st.SchemaTree{adminPathCostUpdateTree, adminEdgePortUpdateTree, macEnabledUpdateTree, restrictedRoleUpdateTree, restrictedTcnUpdateTree, protocolMigrationUpdateTree, enableBPDURxUpdateTree, enableBPDUTxUpdateTree, pseudoRootIdUpdateTree, isL2GpUpdateTree}, []*pb.Update{adminPathCostUpdatePb, adminEdgePortUpdatePb, macEnabledUpdatePb, restrictedRoleUpdatePb, restrictedTcnUpdatePb, protocolMigrationUpdatePb, enableBPDURxUpdatePb, enableBPDUTxUpdatePb, pseudoRootIdUpdatePb, isL2GpUpdatePb}, nil
}

/*
To update the configuration on the MSTP CIST port table, with default values
NOTICE: This has some default values and some settable values
Key parameters: componentId, port
Parameters to set: protocolMigration, pseudoRootId
*/
func SetDefaultMstpCistPortTable(root *st.SchemaTree, protocolMigration bool, pseudoRootId []byte, port uint, componentId uint, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	// AdminEdgePort is likely default false, but yang file says otherwise
	// TODO: set correct MacEnabled value, likely optional
	return SetMstpCistPortTable(root, 0, false, false, false, false, protocolMigration, true, true, pseudoRootId, false, port, componentId, deviceIp)

}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

/*
Check if the provided path cost is valid, need to be run before setting it in the config
*/
func invalidAdminPathCost(pathCost int) error {
	if pathCost >= 0 && pathCost <= 200000000 {
		return nil
	} else {
		return errors.New("Invalid adminPathCost. Value:" + fmt.Sprint(pathCost) + "adminPathCost should be between 0 and 200000000")
	}
}

/*
Check if the provided pseudo root id is valid, need to be run before setting it in the config
*/
func invalidPseudoRootId(pseudoRootId []byte) error {
	if len(pseudoRootId) == 8 {
		return nil
	} else {
		return errors.New("Invalid PseudoRootId. PseudoRootId is an array of " + fmt.Sprint(len(pseudoRootId)) + " bytes. PseudoRootId should be 8 bytes")
	}
}

/* --------------------------------------------------------------------------- */
/* ----------------------- MSTP CIST Port Table Values ----------------------- */
/* --------------------------------------------------------------------------- */

// Update "Admin Path Cost" parameter
// Parameter to set: pathCost (0 <= pathCost 200000000)
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortAdminPathCost(root *st.SchemaTree, pathCost int, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortAdminPathCost")
	treeLvl3.Value = fmt.Sprint(pathCost)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbIntTypeVal(pathCost))
	return treeLvl3, update
}

// Update "Admin Edge Port" parameter
// Parameter to set: edgePort
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortAdminEdgePort(root *st.SchemaTree, edgePort bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortAdminEdgePort")
	treeLvl3.Value = fmt.Sprint(edgePort)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(edgePort))
	return treeLvl3, update
}

// Update "MAC Enabled" parameter
// Parameter to set: macEnabled
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortMacEnabled(root *st.SchemaTree, macEnabled bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortMacEnabled")
	treeLvl3.Value = fmt.Sprint(macEnabled)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(macEnabled))
	return treeLvl3, update
}

// Update "Restricted Role" parameter
// Parameter to set: restrictedRole
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortRestrictedRole(root *st.SchemaTree, restrictedRole bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortRestrictedRole")
	treeLvl3.Value = fmt.Sprint(restrictedRole)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(restrictedRole))
	return treeLvl3, update
}

// Update "Restricted Tcn" parameter
// Parameter to set: restrictedTcn
// Key values: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortRestrictedTcn(root *st.SchemaTree, restrictedTcn bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortRestrictedTcn")
	treeLvl3.Value = fmt.Sprint(restrictedTcn)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(restrictedTcn))
	return treeLvl3, update
}

// Update "Protocol Migration" parameter
// Parameter to set: protocolMigration
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortProtocolMigration(root *st.SchemaTree, protocolMigration bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortProtocolMigration")
	treeLvl3.Value = fmt.Sprint(protocolMigration)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(protocolMigration))
	return treeLvl3, update
}

// Update "Enable BPDURx" parameter
// Parameter to set: bpduRx
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortEnableBPDURx(root *st.SchemaTree, bpduRx bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortEnableBPDURx")
	treeLvl3.Value = fmt.Sprint(bpduRx)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(bpduRx))
	return treeLvl3, update
}

// Update "Enable BPDUTx" parameter
// Parameter to set: bpduTx
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortEnableBPDUTx(root *st.SchemaTree, bpduTx bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortEnableBPDUTx")
	treeLvl3.Value = fmt.Sprint(bpduTx)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(bpduTx))
	return treeLvl3, update
}

// Update "Pseudo Root ID" parameter
// Parameter to set: pseudoRootId (an array of 8 bytes)
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortPseudoRootId(root *st.SchemaTree, pseudoRootId []byte, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortPseudoRootId")
	treeLvl3.Value = fmt.Sprint(pseudoRootId)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBytesTypeVal(pseudoRootId))
	return treeLvl3, update
}

// Update "Is L2Gp"
// Parameter to set: isL2Gp
// Key parameters: componentId, port, deviceIp
// The root of the config tree: root
func setMstpCistPortIsL2Gp(root *st.SchemaTree, isL2Gp bool, componentId uint, port uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpCistPortTable", "ieee8021MstpCistPortEntry", "ieee8021MstpCistPortComponentId", fmt.Sprint(componentId), "ieee8021MstpCistPortNum", fmt.Sprint(port))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistPortIsL2Gp")
	treeLvl3.Value = fmt.Sprint(isL2Gp)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbBoolTypeVal(isL2Gp))
	return treeLvl3, update
}
