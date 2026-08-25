package mstp

/*
Update the MSTP CIST table
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
/* ----------------------------- MSTP CIST Table ----------------------------- */
/* --------------------------------------------------------------------------- */

// Update the configuration of the MSTP CIST table
// schema tree config with the provided input, and return an pb.update of the same configuration
//
//	Input parameters (settable)
//		maxHops:
//
//	Input parameters (keys)
//		componentId: name of the component in the bridge
//
//	Input parameters (other)
//		deviceIp: IP-address to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetMstpCistTable(root *st.SchemaTree, maxHops int, componentId uint, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	invalidMaxHopsError := inValidMaxHops(maxHops)
	if invalidMaxHopsError != nil {
		return nil, nil, invalidMaxHopsError
	}

	maxHopsUpdateTree, maxHopsUpdatePb := setMstpCistMaxHops(root, maxHops, componentId, deviceIp)
	return []*st.SchemaTree{maxHopsUpdateTree}, []*pb.Update{maxHopsUpdatePb}, nil
}

/*
Update the configuratino of the MSTP CIST table, with default values
Update the schema tree config with the provided input, and return an pb.update of the same configuration
*/
func SetDefaultMstpCistTable(root *st.SchemaTree, componentId uint, deviceIp string) ([]*st.SchemaTree, []*pb.Update, error) {
	return SetMstpCistTable(root, 20, componentId, deviceIp)
}

/* --------------------------------------------------------------------------- */
/* ----------------------- Check if the value is valid ----------------------- */
/* --------------------------------------------------------------------------- */

/*
Check if max hops is valid [6, 40], need to be run before setting it to the configuration
*/
func inValidMaxHops(maxHops int) error {
	if maxHops >= 6 && maxHops <= 40 {
		return nil
	} else {
		return errors.New("Invalid MstpCistTableMaxHops. Value:" + fmt.Sprint(maxHops) + ". 6 <= maxHops <= 40")
	}
}

/* --------------------------------------------------------------------------- */
/* ----------------------- MSTP CIST Table Values ---------------------------- */
/* --------------------------------------------------------------------------- */

// Update Max Hops parameter
// Parameter to set: maxHops (6 <= maxHops <= 40, default: 20)
// Key parameters: componentId, deviceIp
func setMstpCistMaxHops(root *st.SchemaTree, maxHops int, componentId uint, deviceIp string) (*st.SchemaTree, *pb.Update) {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	treeLvl2, pbLvl2 := path.GetParam1Key(treeLvl1, pbLvl1, "ieee8021MstpCistTable", "ieee8021MstpCistEntry", "ieee8021MstpCistComponentId", fmt.Sprint(componentId))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpCistMaxHops")
	treeLvl3.Value = fmt.Sprint(maxHops)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbIntTypeVal(maxHops))
	return treeLvl3, update
}
