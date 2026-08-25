package vlan

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Configure a Static VLAN Registration Entry
//
//	Input values (settable)
//		vlanTransmitted: Decides if the VLAN frames with the VID for the Static VLAN Registration Entry should be VLAN-tagged or untagged when transmitted
//			Allowed values: "tagged", "untagged"
//		registrarAdminControl: Controls forwarding behaviour by setting the MVRP and MIRP values for the VID (8.8.2)
//			Allowed values: "fixed-new-ignored", "fixed-new-propagated", "forbidden", "normal"
//
//	Input values (keys):
//		port: the port on the bridge
//		componentName: name of the component in the bridge
//		bridgeName: name of the bridge
//		databaseId: the ID of the Filtering Database for the Static VLAN Registration Entry
//		vids: the VLAN Identifiers for which the Filtering Database apply
//
//	Input values (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetStaticVlanRegistrationEntry(root *st.SchemaTree, vlanTransmitted string, registrarAdminContol string, vids string, database_id uint32, componentName string, bridgeName string, port string, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	vlanTransmittedErr := invalidVlanTransmittedValue(vlanTransmitted)
	if vlanTransmittedErr != nil {
		return nil, nil, vlanTransmittedErr
	}

	registrarAdminContolErr := invalidRegistrarAdminContolValue(registrarAdminContol)
	if registrarAdminContolErr != nil {
		return nil, nil, registrarAdminContolErr
	}

	vlanTransmittedUpdateTree, vlanTransmittedUpdatePb := setStaticVlanRegVlanTransmitted(root, vlanTransmitted, vids, database_id, componentName, bridgeName, port, deviceIp)
	adminCtrlPathUpdateTree, adminCtrlPathUpdatePb := setStaticVlanRegVlanRegAdminCtrlPath(root, registrarAdminContol, vids, database_id, componentName, bridgeName, port, deviceIp)

	return []*st.SchemaTree{vlanTransmittedUpdateTree, adminCtrlPathUpdateTree}, []*pb.Update{vlanTransmittedUpdatePb, adminCtrlPathUpdatePb}, nil
}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

func invalidVlanTransmittedValue(vlanTransmitted string) error {
	if vlanTransmitted == "tagged" || vlanTransmitted == "untagged" {
		return nil
	} else {
		return errors.New("Invalid VLAN Transmitted value: " + vlanTransmitted + ". VLAN Transmitted should one of the following values: \"tagged\", \"untagged\"")
	}
}

func invalidRegistrarAdminContolValue(registrarAdminContol string) error {
	if registrarAdminContol == "fixed-new-ignored" || registrarAdminContol == "fixed-new-propagated" || registrarAdminContol == "forbidden" || registrarAdminContol == "normal" {
		return nil
	} else {
		return errors.New("Invalid Registrar Admin Control value: " + registrarAdminContol + ". VLAN Transmitted should one of the following values: \"fixed-new-ignored\", \"fixed-new-propagated\", \"forbidden\", \"normal\"")
	}
}

/* --------------------------------------------------------------------------- */
/* -------------------- Static VLAN Configuration Values --------------------- */
/* --------------------------------------------------------------------------- */

// Set VLAN Transmitted Value
// Value to set: vlanTransmitted
// Key values: vids, databaseId, componentName, bridgeName, port, deviceIp
// Unsure about the datatypes of some of the fields (vids)
func setStaticVlanRegVlanTransmitted(root *st.SchemaTree, vlanTransmitted string, vids string, databaseId uint32, componentName string, bridgeName string, port string, deviceIp string) (*st.SchemaTree, *pb.Update) {

	treeElemLvl1, pbElemLvl1 := path.GetParamNamespace(root, nil, "ieee802-dot1q-bridge", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	treeElemLvl2, pbElemLvl2 := path.GetParam1Key(treeElemLvl1, pbElemLvl1, "bridges", "bridge", "name", bridgeName)
	treeElemLvl3, pbElemLvl3 := path.GetParam1Key(treeElemLvl2, pbElemLvl2, "", "component", "name", componentName)

	treeElemLvl4, pbElemLvl4 := path.GetParam0Keys(treeElemLvl3, pbElemLvl3, "filtering-database")
	treeElemLvl5, pbElemLvl5 := path.GetParam2Keys(treeElemLvl4, pbElemLvl4, "", "vlan-registration-entry", "database-id", fmt.Sprint(databaseId), "vids", vids) //TODO: Error in the "vids" variable, i think this should be a list

	treeElemLvl6, pbElemLvl6 := path.GetParam1Key(treeElemLvl5, pbElemLvl5, "", "port-map", "port-ref", port)
	treeElemLvl7, pbElemLvl7 := path.GetParam0Keys(treeElemLvl6, pbElemLvl6, "static-vlan-registration-entries")

	treeVlanTransmitPath, pbVlanTransmitPath := path.GetParam0Keys(treeElemLvl7, pbElemLvl7, "vlan-transmitted")
	treeVlanTransmitPath.Value = vlanTransmitted
	vlanTransmittedUpdate := pbMethods.GetUpdate(deviceIp, pbVlanTransmitPath, pbMethods.GetPbStringTypeVal(vlanTransmitted))

	return treeVlanTransmitPath, vlanTransmittedUpdate
}

// Set Registrar Admin Control value
// Value to set: registrarAdminControl
// Key values: vids, databaseId, componentName, bridgeName, port, deviceIp
// Unsure about the datatypes of some of the fields (vids)
func setStaticVlanRegVlanRegAdminCtrlPath(root *st.SchemaTree, registrarAdminContol string, vids string, database_id uint32, componentName string, bridgeName string, port string, deviceIp string) (*st.SchemaTree, *pb.Update) {

	treeElemLvl1, pbElemLvl1 := path.GetParamNamespace(root, nil, "ieee802-dot1q-bridge", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	treeElemLvl2, pbElemLvl2 := path.GetParam1Key(treeElemLvl1, pbElemLvl1, "bridges", "bridge", "name", bridgeName)
	treeElemLvl3, pbElemLvl3 := path.GetParam1Key(treeElemLvl2, pbElemLvl2, "", "component", "name", componentName)

	treeElemLvl4, pbElemLvl4 := path.GetParam0Keys(treeElemLvl3, pbElemLvl3, "filtering-database")
	treeElemLvl5, pbElemLvl5 := path.GetParam2Keys(treeElemLvl4, pbElemLvl4, "", "vlan-registration-entry", "database-id", fmt.Sprint(database_id), "vids", vids) //TODO: Error in the "vids" variable, i think this should be a list

	treeElemLvl6, pbElemLvl6 := path.GetParam1Key(treeElemLvl5, pbElemLvl5, "", "port-map", "port-ref", port)
	treeElemLvl7, pbElemLvl7 := path.GetParam0Keys(treeElemLvl6, pbElemLvl6, "static-vlan-registration-entries")

	treeRegAdminCtrlPath, pbRegAdminCtrlPath := path.GetParam0Keys(treeElemLvl7, pbElemLvl7, "registrar-admin-control")
	treeRegAdminCtrlPath.Value = registrarAdminContol
	registrarAdminControlUpdate := pbMethods.GetUpdate(deviceIp, pbRegAdminCtrlPath, pbMethods.GetPbStringTypeVal(registrarAdminContol))

	return treeRegAdminCtrlPath, registrarAdminControlUpdate
}
