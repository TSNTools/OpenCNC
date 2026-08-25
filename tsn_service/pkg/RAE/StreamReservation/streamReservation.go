package streamreservation

import (
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Set the MRP External Control Table
// Input parameters (settable):
//
//	mrpExternalControl:
//	adminRequestList:
//	adminRequestListLen:
//
// Input parameters (keys):
//
//	TODO: Add keys
//
// Input parameters (other):
//
//	deviceIp: IP-address to the switch where the update is made
//	root: a reference to the root of the SchemaTree where the update is made
//
// TODO:
//
//	Unsure how to handle "Octet String" value in adminRequestList
//	Calculate the adminRequestListLen instead of having it as a parameter
//	Add functions to check if the values of the parameters are valid if possible
func SetMrpExternalControlTable(root *st.SchemaTree, mrpExternalControl bool, adminRequestList string, adminRequestListLen uint32, deviceIp string) ([]*st.SchemaTree, []*pb.Update) {
	externalCtrlValTree, externalCtrlValPb := setExternalControlValue(root, mrpExternalControl, deviceIp)
	adminRequestListTree, adminRequestListPb := setAdminRequestListValue(root, adminRequestList, deviceIp)
	adminRequestListLenTree, adminRequestListLenPb := setAdminRequestListLengthValue(root, adminRequestListLen, deviceIp)

	return []*st.SchemaTree{externalCtrlValTree, adminRequestListTree, adminRequestListLenTree}, []*pb.Update{externalCtrlValPb, adminRequestListPb, adminRequestListLenPb}
}

// Update the MRP External Control Table table with default values
func SetDefaultMrpExternalControlTable(root *st.SchemaTree, deviceIp string) ([]*st.SchemaTree, []*pb.Update) {
	return SetMrpExternalControlTable(root, false, "", 0, deviceIp)
}

// NOTE: This function is not based on any yang-file
// NOTE: I am not very sure about the path
// Parameter to set: mrpExternalControl
// Key parameter: deviceIp
func setExternalControlValue(root *st.SchemaTree, mrpExternalControl bool, deviceIp string) (*st.SchemaTree, *pb.Update) {
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(root, nil, "ieee8021TsnRemoteMgmtMsrpMrpExternalControlTable")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "ieee8021TsnRemoteMgmtMsrpMrpExternalControlEntry")
	pathTree, pathPb := path.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "ieee8021TsnRemoteMgmtMsrpMrpExternalControl")
	pathTree.Value = fmt.Sprint(mrpExternalControl)
	update := pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbBoolTypeVal(mrpExternalControl))

	return pathTree, update
}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

//TODO: Add functions to check if the values of the parameters are valid

/* --------------------------------------------------------------------------- */
/* ----------------------- MSTP CIST Table Values ---------------------------- */
/* --------------------------------------------------------------------------- */

// NOTE: This function is not based on any yang-file
// NOTE: I am not very sure about the path
// NOTE: Unsure if this is read-write
// NOTE: It is likely that the MSRP MRP External Control table have keys, the path when the keys are found
// Parameter to set: adminRequestList
// Key parameter: deviceIp
func setAdminRequestListValue(root *st.SchemaTree, adminRequestList string, deviceIp string) (*st.SchemaTree, *pb.Update) {
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(root, nil, "ieee8021TsnRemoteMgmtMsrpMrpExternalControlTable")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "ieee8021TsnRemoteMgmtMsrpMrpExternalControlEntry")
	pathTree, pathPb := path.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "ieee8021TsnRemoteMgmtMrpAdminRequestList")
	pathTree.Value = adminRequestList
	update := pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(adminRequestList))

	return pathTree, update
}

// NOTE: This function is not based on any yang-file
// NOTE: I am not very sure about the path
// NOTE: Unsure if this is read-write
// NOTE: It is likely that the MSRP MRP External Control table have keys, the path when the keys are found
// Parameter to set: adminRequestListLength
// Key parameter: deviceIp
func setAdminRequestListLengthValue(root *st.SchemaTree, adminRequestListLength uint32, deviceIp string) (*st.SchemaTree, *pb.Update) {
	pathLvl1Tree, pathLvl1Pb := path.GetParam0Keys(root, nil, "ieee8021TsnRemoteMgmtMsrpMrpExternalControlTable")
	pathLvl2Tree, pathLvl2Pb := path.GetParam0Keys(pathLvl1Tree, pathLvl1Pb, "ieee8021TsnRemoteMgmtMsrpMrpExternalControlEntry")
	pathTree, pathPb := path.GetParam0Keys(pathLvl2Tree, pathLvl2Pb, "ieee8021TsnRemoteMgmtMrpAdminRequestListLength")
	pathTree.Value = fmt.Sprint(adminRequestListLength)
	update := pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(uint(adminRequestListLength)))

	return pathTree, update
}
