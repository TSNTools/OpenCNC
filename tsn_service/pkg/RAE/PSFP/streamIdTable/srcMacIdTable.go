package streamIdTable

import (
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	rae "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/* ------------------------------------------------------------------------------------------------------------- */
/* ------------------------------------ Source MAC identification table ---------------------------------------- */
/* ------------------------------------------------------------------------------------------------------------- */

/*
Set maxCurrentStreams at the stream identification table
*/
func setSourceMacId(root *st.SchemaTree, port string, deviceIp string, streamHandle string, srcMac SourceMacIdEntry) (update []*pb.Update) {
	update = append(update, setSourceMacIdSourceMac(root, port, deviceIp, streamHandle, srcMac.sourceMac))
	update = append(update, setSourceMacIdTagged(root, port, deviceIp, streamHandle, srcMac.tagged))
	update = append(update, setSourceMacIdVlan(root, port, deviceIp, streamHandle, srcMac.vlan))
	return update
}

/*
Set set destination mac address in Null-stream-identification-table

key parameter:

	port, deviceIp, streamHandle

parameters to set:

	root, srcMac
*/
func setSourceMacIdSourceMac(root *st.SchemaTree, port string, deviceIp string, streamHandle string, srcMac string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")

	pathStreamHandleTree, pathStreamHandlePb :=
		rae.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb :=
		rae.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "source-mac-identification-entry")

	pathTree, pathPb := rae.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "source-mac")

	pathTree.Value = srcMac
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(srcMac))
	return update
}

/*
Set set tagged in Null-stream-identification-table

key parameter:

	port, deviceIp, streamHandle

parameters to set:

	root, tagged
*/
func setSourceMacIdTagged(root *st.SchemaTree, port string, deviceIp string, streamHandle string, tagged string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")

	pathStreamHandleTree, pathStreamHandlePb :=
		rae.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb :=
		rae.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "source-mac-identification-entry")

	pathTree, pathPb := rae.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "tagged")

	pathTree.Value = tagged
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(tagged))
	return update
}

/*
Set set vlan in Null-stream-identification-table

key parameter:

	port, deviceIp, streamHandle

parameters to set:

	root, vlan
*/
func setSourceMacIdVlan(root *st.SchemaTree, port string, deviceIp string, streamHandle string, vlan uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := rae.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := rae.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")

	pathStreamHandleTree, pathStreamHandlePb :=
		rae.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb :=
		rae.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "source-mac-identification-entry")

	pathTree, pathPb := rae.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "vlan")

	pathTree.Value = fmt.Sprint(vlan)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(vlan))
	return update
}
