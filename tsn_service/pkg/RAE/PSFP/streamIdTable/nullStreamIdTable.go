package streamIdTable

/*
Set the configuration entries in teh null stream identification table
*/

import (
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
Set set Null-stream-identification-table at the stream identification table
Input:

	nullStream: enum with all the input values
*/
func SetNullStreamId(port string, deviceIp string, streamHandle string, nullStream NullStreamIdEntry) (update []*pb.Update) {
	update = append(update, setNullStreamIdDestMac(port, deviceIp, streamHandle, nullStream.destMac))
	//update = append(update, setNullStreamIdTagged(port, deviceIp, streamHandle, nullStream.tagged))
	update = append(update, setNullStreamIdVlan(port, deviceIp, streamHandle, nullStream.vlan))
	return update
}

/*
Set set Null-stream-identification-table at the stream identification table
Input:

	nullStream: enum with all the input values
*/
func UpdateNullStreamId(root *st.SchemaTree, port string, deviceIp string, streamHandle string, nullStream NullStreamIdEntry) (update []*pb.Update) {
	update = append(update, updateNullStreamIdDestMac(root, port, deviceIp, streamHandle, nullStream.destMac))
	update = append(update, updateNullStreamIdTagged(root, port, deviceIp, streamHandle, nullStream.tagged))
	update = append(update, updateNullStreamIdVlan(root, port, deviceIp, streamHandle, nullStream.vlan))
	return update
}

/*
Set destination mac address in Null-stream-identification-table
*/
func setNullStreamIdDestMac(port string, deviceIp string, streamHandle string, destMac string) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "null-stream-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "destination-mac")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbStringTypeVal(destMac))
	return update
}

func updateNullStreamIdDestMac(root *st.SchemaTree, port string, deviceIp string, streamHandle string, destMac string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "null-stream-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "destination-mac")
	pathTree.Value = destMac
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(destMac))
	return update
}

/*
Set tagged in Null-stream-identification-table
*/
func updateNullStreamIdTagged(root *st.SchemaTree, port string, deviceIp string, streamHandle string, tagged string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "null-stream-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "tagged")
	pathTree.Value = tagged
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(tagged))
	return update
}

/*
Set vlan in Null-stream-identification-table
*/
func setNullStreamIdVlan(port string, deviceIp string, streamHandle string, vlan uint) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "null-stream-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "vlan")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbUintTypeVal(vlan))
	return update
}

func updateNullStreamIdVlan(root *st.SchemaTree, port string, deviceIp string, streamHandle string, vlan uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "null-stream-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "vlan")

	pathTree.Value = fmt.Sprint(vlan)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(vlan))
	return update
}
