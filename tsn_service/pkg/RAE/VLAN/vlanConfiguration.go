package vlan

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Set VLAN Configuration
//
//	Input values (settable):
//		vlanName: Name of the VLAN configuration
//
//	Input values (keys):
//		bridgeName: name of the bridge
//		componentName: name of the component in the bridge
//		vid: VLAN-identifier of the VLAN, which name is configured
//
//	Input values (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetVlanConfiguration(root *st.SchemaTree, vlanName string, vid uint32, componentName string, bridgeName string, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	vlanNameError := inValidVlanName(vlanName)
	if vlanNameError != nil {
		return nil, nil, vlanNameError
	}
	vlanNameUpdateTree, vlanNameUpdatePb := setVlanConfigurationUpdate(root, vlanName, vid, componentName, bridgeName, deviceIp)
	return []*st.SchemaTree{vlanNameUpdateTree}, []*pb.Update{vlanNameUpdatePb}, nil
}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

// check if the VLAN name is valid, can not be longer than 32 char
func inValidVlanName(vlanName string) error {
	if len(vlanName) <= 32 {
		return nil
	} else {
		return errors.New("Invalid VLAN name length. VLAN name is now " + fmt.Sprint(len(vlanName)) + "characters. VLAN name is not allowed to be more than 32 characters")
	}

}

/* --------------------------------------------------------------------------- */
/* ----------------------- VLAN Configuration Values ------------------------- */
/* --------------------------------------------------------------------------- */

// Specifies one VLAN Configuration
// Value to set: vlanName (max 32 characters)
// Key values: vid, bridgeName, deviceIp
func setVlanConfigurationUpdate(root *st.SchemaTree, vlanName string, vid uint32, componentName string, bridgeName string, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee802-dot1q-bridge", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	treeLvl2, pbLvl2 := path.GetParam1Key(treeLvl1, pbLvl1, "bridges", "bridge", "name", bridgeName)
	treeLvl3, pbLvl3 := path.GetParam1Key(treeLvl2, pbLvl2, "", "component", "name", componentName)
	treeLvl4, pbLvl4 := path.GetParam0Keys(treeLvl3, pbLvl3, "bridge-vlan")
	treeLvl5, pbLvl5 := path.GetParam1Key(treeLvl4, pbLvl4, "", "vlan", "vid", fmt.Sprint(vid))
	treePath, pbPath := path.GetParam0Keys(treeLvl5, pbLvl5, "name")
	treePath.Value = vlanName
	update := pbMethods.GetUpdate(deviceIp, pbPath, pbMethods.GetPbStringTypeVal(vlanName))

	return treePath, update
}
