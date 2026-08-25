package mstp

/*
Update the MSTP FID to MSTI V2 table
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
/* ------------------------------ FID To MSTI V2 Table ------------------------------ */
/* ---------------------------------------------------------------------------------- */

// Update the FID to MSTI V2 table
//
//	Input parameters (settable)
//		fid: filtering identifier for the FID to MSTI allocation entry
//	Input parameters (keys)
//		componentId: name of the component in the bridge
//	Input parameters (other)
//		deviceIp: IP-addredss to the switch where the update is made
//		root: a reference to the root of the SchemaTree where the update is made
func SetFidToMstiV2Table(root *st.SchemaTree, fid uint, componentId string, deviceIp string) ([]*pb.Update, error) {
	fidError := validFid(fid)
	if fidError != nil {
		return nil, fidError
	}

	fidToMstiUpdate := setFidValueForFidToMstiEntry(root, fid, componentId, deviceIp)
	return []*pb.Update{fidToMstiUpdate}, nil
}

/* ---------------------------------------------------------------------------------- */
/* ----------------------------- Check if the value is valid ------------------------ */
/* ---------------------------------------------------------------------------------- */

// Check that the fid is valid [0, 409]
func validFid(fid uint) error {
	if fid <= 409 {
		return nil
	} else {
		return errors.New("Invalid FidToMstiV2TableFid. Value: " + fmt.Sprint(fid) + " 0 <= fid <= 209")
	}
}

/* ---------------------------------------------------------------------------------- */
/* ----------------------------- FID To MSTI V2 Entries ----------------------------- */
/* ---------------------------------------------------------------------------------- */

// Update the FID value of a FID to MSTI V2 Table Entry
// Paramter to set: fid (uint32, 0 <= fid <= 409)
// Key parameters: fid, componentId, deviceIpey
func setFidValueForFidToMstiEntry(root *st.SchemaTree, fid uint, componentId string, deviceIp string) *pb.Update {
	treeLvl1, pbLvl1 := path.GetParamNamespace(root, nil, "ieee8021-mstp", "urn:ietf:pras:xml:ns:yang:smiv2:ieee8021mstp")
	treeLvl2, pbLvl2 := path.GetParam2Keys(treeLvl1, pbLvl1, "ieee8021MstpFidToMstiV2Table", "ieee8021MstpFidToMstiV2Entry", "ieee8021MstpFidToMstiV2ComponentId", componentId, "ieee8021MstpFidToMstV2Fid", fmt.Sprint(fid))
	treeLvl3, pbLvl3 := path.GetParam0Keys(treeLvl2, pbLvl2, "ieee8021MstpFidToMstV2Fid")
	treeLvl3.Value = fmt.Sprint(fid)
	update := pbMethods.GetUpdate(deviceIp, pbLvl3, pbMethods.GetPbUintTypeVal(fid))
	return update
}
