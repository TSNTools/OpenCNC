package streamIdTable

import (
	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/* ------------------------------------------------------------------------------------------------------------- */
/* --------------------------- Active destination MAC identification table ------------------------------------- */
/* ------------------------------------------------------------------------------------------------------------- */

/*
Set maxCurrentStreams at the stream identification table
*/
// func setActiveDestMacId(port string, deviceIp string, streamHandle string, activeId activeDestMacIdEntry) (updates []*pb.Update) {
// 	updates = append(updates, setActiveDestMacIdUpperDestMac(port, deviceIp, streamHandle, activeId.upperDestMac))
// 	updates = append(updates, setActiveDestMacIdUpperTagged(port, deviceIp, streamHandle, activeId.upperTagged))
// 	updates = append(updates, setActiveDestMacIdUpperVlan(port, deviceIp, streamHandle, activeId.upperVlan))
// 	updates = append(updates, setActiveDestMacIdUpperPrio(port, deviceIp, streamHandle, activeId.upperPriority))
// 	updates = append(updates, setActiveDestMacIdLowerDestMac(port, deviceIp, streamHandle, activeId.lowerDestMac))
// 	updates = append(updates, setActiveDestMacIdLowerTagged(port, deviceIp, streamHandle, activeId.lowerTagged))
// 	updates = append(updates, setActiveDestMacIdLowerVlan(port, deviceIp, streamHandle, activeId.lowerVlan))
// 	return updates
// }

func setActiveDestMacIdUpperDestMac(port string, deviceIp string, streamHandle string, destMac string) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "upper-destination-mac")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbStringTypeVal(destMac))
	return update
}

func updateActiveDestMacIdUpperDestMac(root *st.SchemaTree, port string, deviceIp string, streamHandle string, destMac string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "upper-destination-mac")

	pathTree.Value = destMac
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(destMac))
	return update
}

func setActiveDestMacIdUpperTagged(port string, deviceIp string, streamHandle string, Tagg string) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "upper-tagged")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbStringTypeVal(Tagg))
	return update
}

func updateActiveDestMacIdUpperTagged(root *st.SchemaTree, port string, deviceIp string, streamHandle string, Tagg string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "upper-tagged")

	pathTree.Value = Tagg
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(Tagg))
	return update
}

func setActiveDestMacIdUpperVlan(port string, deviceIp string, streamHandle string, Vlan uint) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "upper-vlan")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbUintTypeVal(Vlan))
	return update
}

func updateActiveDestMacIdUpperVlan(root *st.SchemaTree, port string, deviceIp string, streamHandle string, Vlan uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "upper-vlan")

	pathTree.Value = fmt.Sprint(Vlan)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(Vlan))
	return update
}

func setActiveDestMacIdUpperPrio(port string, deviceIp string, streamHandle string, prio uint) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "upper-priority")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbUintTypeVal(prio))
	return update
}

func updateActiveDestMacIdUpperPrio(root *st.SchemaTree, port string, deviceIp string, streamHandle string, prio uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "upper-priority")

	pathTree.Value = fmt.Sprint(prio)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(prio))
	return update
}

func setActiveDestMacIdLowerDestMac(port string, deviceIp string, streamHandle string, destMac string) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "lower-destionation-mac")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbStringTypeVal(destMac))
	return update
}

func updateActiveDestMacIdLowerDestMac(root *st.SchemaTree, port string, deviceIp string, streamHandle string, destMac string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "lower-destionation-mac")

	pathTree.Value = destMac
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(destMac))
	return update
}

func setActiveDestMacIdLowerTagged(port string, deviceIp string, streamHandle string, Tagg string) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "lower-tagged")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbStringTypeVal(Tagg))
	return update
}

func updateActiveDestMacIdLowerTagged(root *st.SchemaTree, port string, deviceIp string, streamHandle string, Tagg string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "lower-tagged")

	pathTree.Value = Tagg
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbStringTypeVal(Tagg))
	return update
}

func setActiveDestMacIdLowerVlan(port string, deviceIp string, streamHandle string, Vlan uint) (update *pb.Update) {
	bridgePath := pbMethods.GetPath2bridge(port)
	pathStreamId := pbMethods.GetPath1lvlDown0Keys(bridgePath, "stream-identification")
	pathStreamHandle := pbMethods.GetPath1lvlDown1Key(pathStreamId, "stream-handles", "stream-handle", streamHandle)
	pathNullStreamId := pbMethods.GetPath1lvlDown0Keys(pathStreamHandle, "active-destination-mac-identification-entry")
	path := pbMethods.GetPath1lvlDown0Keys(pathNullStreamId, "lower-vlan")

	update = pbMethods.GetUpdate(deviceIp, path, pbMethods.GetPbUintTypeVal(Vlan))
	return update
}

func updateActiveDestMacIdLowerVlan(root *st.SchemaTree, port string, deviceIp string, streamHandle string, Vlan uint) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathNullStreamIdTree, pathNullStreamIdPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "active-destination-mac-identification-entry")
	pathTree, pathPb := path.GetParam0Keys(pathNullStreamIdTree, pathNullStreamIdPb, "lower-vlan")

	pathTree.Value = fmt.Sprint(Vlan)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(Vlan))
	return update
}
