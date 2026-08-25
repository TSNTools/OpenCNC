package mstp

/*
Update the MSTP Configuration table
*/

import (
	"errors"
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/* --------------------------------------------------------------------------- */
/* ----------------------------- MSTP Config Table ------------------------------ */
/* --------------------------------------------------------------------------- */

// Update the configuration for MSTP configuration table
//
//	Input parameters (settable)
//		formatSelector: the format which the MSTP Configuration uses
//		configurationName: name of the MSTP Configuration
//		revisionLevel:
//
//	Input parameters (keys)
//		componentId: name of the component in the bridge
//
//	Input parameters (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetMstpConfigTable(root *st.SchemaTree, formatSelector int, configurationName string,
	revisionLevel uint, componentId uint, deviceIp string) ([]*pb.Update, error) {

	formatSelectorErr := invalidFormatSelector(int32(formatSelector))
	if formatSelectorErr != nil {
		return nil, formatSelectorErr
	}

	configurationNameErr := invalidConfigurationName(configurationName)
	if configurationNameErr != nil {
		return nil, configurationNameErr
	}

	revisionLevelError := invalidRevisionLevel(uint32(revisionLevel))
	if revisionLevelError != nil {
		return nil, revisionLevelError
	}
	formatSelectorUpdate := setMstpConfigIdFormatSelector(root, formatSelector, componentId, deviceIp)
	configNameUpdate := setMstpConfigIdConfigurationName(root, configurationName, componentId, deviceIp)
	revisionLevelUpdate := setMstpConfigIdRevisionLevel(root, revisionLevel, componentId, deviceIp)
	return []*pb.Update{formatSelectorUpdate, configNameUpdate, revisionLevelUpdate}, nil
}

// Update the configuration for MSTP configuration table with default values
func SetDefaultMstpConfigTable(root *st.SchemaTree, configurationName string, componentId uint, deviceIp string) ([]*pb.Update, error) {
	return SetMstpConfigTable(root, 0, configurationName, 0, componentId, deviceIp)
}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

// Check that the format selector is valid, [1, 200000000]
func invalidFormatSelector(formatSelector int32) error {
	if formatSelector >= 1 && formatSelector <= 200000000 {
		return nil
	} else {
		return errors.New("Invalid MstpConfigTableFormatSelector. Value: " + fmt.Sprint(formatSelector) + ". 1 <= formatSelector <= 200000000")
	}
}

// Check that the configuration name is valid, can not be longer than 32 char
func invalidConfigurationName(configurationName string) error {
	if len(configurationName) <= 32 {
		return nil
	} else {
		return errors.New("Invalid MstpConfigTableConfigurationName. Value: " + configurationName + " is " + fmt.Sprint(len(configurationName)) + " but should be no more than 32 characters")
	}
}

// Check that the crevision level is valid, must be between [0, 65535]
func invalidRevisionLevel(revisionLevel uint32) error {
	if revisionLevel <= 65535 {
		return nil
	} else {
		return errors.New("Invalid MstpConfigTableRevisionLevel. Value: " + fmt.Sprint(revisionLevel) + ". 0 <= revisionLevel <= 65535")
	}
}

/* --------------------------------------------------------------------------- */
/* ----------------------- MSTP Config Table Values -------------------------- */
/* --------------------------------------------------------------------------- */

// Update "Format Selector" parameter
// Parameter to set: formatSelector (1 <= formatSelector <= 200000000, int32, default: 0)
// Key parameters: componentId (uint32), deviceIp
func setMstpConfigIdFormatSelector(root *st.SchemaTree, formatSelector int, componentId uint, deviceIp string) *pb.Update {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam1Key(treeLvl1, pbLvl1, "ieee8021MstpConfigIdTable", "ieee8021MstpConfigIdEntry", "ieee8021MstpConfigIdComponentId", fmt.Sprint(componentId))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpConfigIdFormatSelector")
	treeLvl3.Value = fmt.Sprint(formatSelector)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbIntTypeVal(formatSelector))
	return update
}

// Update "Configuration name" parameter
// Parameter to set: configurationName (Max 32 characters)
// Key parameters: componentId (uint32), deviceIp
func setMstpConfigIdConfigurationName(root *st.SchemaTree, configurationName string, componentId uint, deviceIp string) *pb.Update {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam1Key(treeLvl1, pbLvl1, "ieee8021MstpConfigIdTable", "ieee8021MstpConfigIdEntry", "ieee8021MstpConfigIdComponentId", fmt.Sprint(componentId))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpConfigurationName")
	treeLvl3.Value = fmt.Sprint(configurationName)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbStringTypeVal(configurationName))
	return update
}

// Update "Revision Level" parameter
// Parameter to set: revisionLevel (0 <= revisionLevel <= 65535, uint32, default: 0)
// Key parameters: componentId (uint32), deviceIp
func setMstpConfigIdRevisionLevel(root *st.SchemaTree, revisionLevel uint, componentId uint, deviceIp string) *pb.Update {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam1Key(treeLvl1, pbLvl1, "ieee8021MstpConfigIdTable", "ieee8021MstpConfigIdEntry", "ieee8021MstpConfigIdComponentId", fmt.Sprint(componentId))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpRevisionLevel")
	treeLvl3.Value = fmt.Sprint(revisionLevel)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbUintTypeVal(revisionLevel))
	return update
}
