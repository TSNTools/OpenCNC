package streamIdTable

/*
Functions to set values at the stream identification table, that are not part of another table, or is a list (as the IO facing IO port tables)
*/

import (
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	"fmt"
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	path "tsn-service/pkg/RAE/dataStructures/composit"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

//var log = logger.GetLogger()

/*
Set maxCurrentStreams at the stream identification table¨

key parameters:

	port, deviceIp

parameters to set:

	maxCurrentStreams: max number of streams the bridge can handle
*/
func setMaxConcurrentStream(root *st.SchemaTree, maxCurrentStreams uint, port string, deviceIp string) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathTree, pathPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbUintTypeVal(maxCurrentStreams))
	pathTree.Value = fmt.Sprint(maxCurrentStreams)
	return update
}

/*
Set tsn-stream-identification-type
key parameters:

	port, deviceIp, streamHandle

parameters to set:

	tsnStreamIdType: the tsn-stream-identification-type to be set
*/
func setTsnStreamIdType(root *st.SchemaTree, port string, deviceIp string, streamHandle string, tsnStreamIdType int) (update *pb.Update) {
	bridgePathTree, bridgePathPb := path.GetPath2Bridge(root, port)
	pathStreamIdTree, pathStreamIdPb := path.GetParam0Keys(bridgePathTree, bridgePathPb, "stream-identification")
	pathStreamHandleTree, pathStreamHandlePb := path.GetParam1Key(pathStreamIdTree, pathStreamIdPb, "", "stream-handles", "stream-handle", streamHandle)
	pathTree, pathPb := path.GetParam0Keys(pathStreamHandleTree, pathStreamHandlePb, "tsn-stream-identication-type")
	pathTree.Value = fmt.Sprint(tsnStreamIdType)
	update = pbMethods.GetUpdate(deviceIp, pathPb, pbMethods.GetPbIntTypeVal(tsnStreamIdType))
	return update
}
