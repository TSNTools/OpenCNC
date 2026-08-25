package vlan

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Configure the VLAN parameters
//
//	Input values (settable):
//		pvid: the primary VLAN identifier for the bridge port
//		acceptableFrameTypes: determines if only VLAN-tagged frames, only untagged and priority tagged or all frames should be accepted
//		enableIngressFiltering: if enabled, frames which are not a member of a VID are discarded when recieved (8.6.2)
//		enableRestrictedVlan: if enabled, MVRP messages can only create or modify Dynamic VLAN Registration Entries with the same VLAN Identifier (VID) as a Static VLAN Registration Entry with the parameter registrarAdminControl set to normal (11.2.3.2.3)
//
//	Input values (keys):
//		port: The interface where the updates are made
//
//	Input values (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetBridgeVlanConfiguration(root *st.SchemaTree, pvid uint32,
	acceptableFrameTypes string, enableIngressFiltering bool,
	enableRestrictedVlan bool, port string, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {

	frameTypeError := validFrameType(acceptableFrameTypes)
	if frameTypeError != nil {
		return nil, nil, frameTypeError
	}

	pvidError := checkValidPvid(pvid)
	if pvidError != nil {
		return nil, nil, pvidError
	}

	pvidUpdateTree, pvidUpdatePb := setPvidUpdate(root, pvid, port, deviceIp)
	acceptableFrameTypesUpdateTree, acceptableFrameTypesUpdatePb := setAcceptableFrameTypes(root, acceptableFrameTypes, port, deviceIp)
	enableIngressFilteringUpdateTree, enableIngressFilteringUpdatePb := setEnableIngressFiltering(root, enableIngressFiltering, port, deviceIp)
	enableRestrictedVlanRegistrationUpdateTree, enableRestrictedVlanRegistrationUpdatePb := setEnableRestrictedVlanRegistration(root, enableRestrictedVlan, port, deviceIp)

	return []*st.SchemaTree{pvidUpdateTree, acceptableFrameTypesUpdateTree, enableIngressFilteringUpdateTree, enableRestrictedVlanRegistrationUpdateTree}, []*pb.Update{pvidUpdatePb, acceptableFrameTypesUpdatePb, enableIngressFilteringUpdatePb, enableRestrictedVlanRegistrationUpdatePb}, nil
}

// Configure the VLAN parameters, use default where possible
func SetDefaultBridgeVlanConfiguration(root *st.SchemaTree, port string, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	return SetBridgeVlanConfiguration(root, 1, "admit-all-frames", false, false, port, deviceIp)
}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

// Check that the pvid is valid, not 0 or 4095
func checkValidPvid(pvid uint32) error {
	if pvid == 0 || pvid == 4095 {
		return errors.New("Invalid pvid. Value: " + fmt.Sprint(pvid) + ". pvid cannot have the value of 0 or 4095")
	} else {
		return nil
	}

}

// Check that the provided frame type is valid
func validFrameType(acceptableFrame string) error {
	if acceptableFrame == "admit-only-VLAN-tagged-frames" ||
		acceptableFrame == "admit-only-untagged-and-priority-tagged" ||
		acceptableFrame == "admit-all-frames" {
		return nil
	}
	return errors.New("invalid frame type. Value: " +
		acceptableFrame +
		"is not one of the three acceptable ones, which are:" +
		"\n admit-only-VLAN-tagged-frames" +
		"\n admit-only-untagged-and-priority-tagged" +
		"\n admit-all-frames")
}

/* --------------------------------------------------------------------------- */
/* ------------------- VLAN Bridge Configuration Values ---------------------- */
/* --------------------------------------------------------------------------- */

// Update PVID parameter
// Parameter to set: pvid (uint32, cannot have the values of 0 or 4095, default: 1)
// Key paramters: port, deviceIp
func setPvidUpdate(root *st.SchemaTree, pvid uint32, port string, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetPath2Bridge(root, fmt.Sprint(port))
	treeLvl2, pbLvl2 := path.GetParam0Keys(treeLvl1, pbLvl1, "pvid")
	treeLvl2.Value = fmt.Sprint(pvid)
	update := pbMethods.GetUpdate(deviceIp, pbLvl2, pbMethods.GetPbUintTypeVal(uint(pvid)))
	return treeLvl2, update
}

// Update Acceptable Frame Types parameter (see IEEE 802.1Q 6.9)
// Parameter to set: acceptable frame (enumeration: "admit-all-frames" (default), "admit-only-VLAN-tagged-frames", "admit-only-untagged-and-priority-tagged")
// Key Values: port, deviceIp
func setAcceptableFrameTypes(root *st.SchemaTree, acceptableFrame string, port string, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetPath2Bridge(root, fmt.Sprint(port))
	treeLvl2, pbLvl2 := path.GetParam0Keys(treeLvl1, pbLvl1, "acceptable-frame")
	treeLvl2.Value = acceptableFrame
	update := pbMethods.GetUpdate(deviceIp, pbLvl2, pbMethods.GetPbStringTypeVal(acceptableFrame))
	return treeLvl2, update
}

// Update Enable Ingress Filtering parameter
// Parameter to set: enableIngressFiltering (boolean, default: false)
// Key Values: port, deviceIp
func setEnableIngressFiltering(root *st.SchemaTree, enableIngressFiltering bool, port string, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetPath2Bridge(root, fmt.Sprint(port))
	treeLvl2, pbLvl2 := path.GetParam0Keys(treeLvl1, pbLvl1, "enable-ingress-filtering")
	treeLvl2.Value = fmt.Sprint(enableIngressFiltering)
	update := pbMethods.GetUpdate(deviceIp, pbLvl2, pbMethods.GetPbBoolTypeVal(enableIngressFiltering))
	return treeLvl2, update
}

// Update Enable Restricted Vlan Registration parameter
// Parameter to set: enableRestrictedVlanRegistration (boolean, default: false)
// Key Values: port, deviceIp
func setEnableRestrictedVlanRegistration(root *st.SchemaTree, enableRestrictedVlanRegistration bool, port string, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetPath2Bridge(root, fmt.Sprint(port))
	treeLvl2, pbLvl2 := path.GetParam0Keys(treeLvl1, pbLvl1, "enable-restricted-vlan-registration")
	treeLvl2.Value = fmt.Sprint(enableRestrictedVlanRegistration)
	update := pbMethods.GetUpdate(deviceIp, pbLvl2, pbMethods.GetPbBoolTypeVal(enableRestrictedVlanRegistration))
	return treeLvl2, update
}
