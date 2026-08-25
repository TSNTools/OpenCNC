package streamIdTable

/*
To set the initially configuration of the stream identification table
Ref:
	ieee802-dot1CB.yang
	IEEE 802.1 CB 6
	IEEE 802.1 CB 9.1
	IEEE 802.1 Q 12.29
	IEEE 802.1 Q 12.31
	IEEE 802.1 Q 8.6.5-8.6.6
*/

import (
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	// logger "git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

/*
Update the configuration for the stream identification table, at the k/v-store

key parameters:
  - port; port where the table is at
  - deviceIp; IP address for the device which port one look at

parameters to set:
  - maxCurrentStreams;
    he maximum number of concurrent stream identification functions supported by the application
    Default; maximum size of uint (aka math.MaxUint64)
  - streamHandleValues; control packets whose stream_handle sub parameter is equal to the entries tsnStreamIdHandle object
*/
func UpdatePSFPStreamIdTable(maxCurrentStreams uint, streamHandleValuesList []StreamHandles, port string, deviceIp string) error {

	/*
		var log = logger.GetLogger()
		// TODO: get configuration from k/v-store


		var updates []*pb.Update

		// Set configuration for Max Concurrent Stream
		updates = append(updates, setMaxConcurrentStream(&root, maxCurrentStreams, port, deviceIp))

		// set each streamHandlesTable
		for i := 0; i < len(streamHandleValuesList); i++ {
			// Set stream handles entries
			streamHandleUpdate, err := updatePSFPStreamHandleTable(&root, maxCurrentStreams, streamHandleValuesList[i], port, deviceIp)
			if err != nil {
				log.Errorf("Failed setting a stream handles table for stream identification table: %v", err)
			} else {
				updates = append(updates, streamHandleUpdate...)
			}
		}

		//TODO: update config service


	*/

	// TODO Update k/v-store
	return nil
}

/*
Set the configuration for the PSFP stream handle table
*/
func updatePSFPStreamHandleTable(root *st.SchemaTree, maxCurrentStreams uint,
	streamHandleValues StreamHandles, port string, deviceIp string) (updates []*pb.Update, err error) {

	/* ---------------------- Configure each value ---------------------- */

	// set {in/out}Facing{In/Out}putPort lists
	updates = append(updates, setInFacingInPortListIndex(
		root, streamHandleValues.streamHandle, streamHandleValues.inFacingInputPortList, port, deviceIp)...)

	updates = append(updates, setInFacingOutPortListIndex(
		root, streamHandleValues.streamHandle, streamHandleValues.inFacingOutputPortList, port, deviceIp)...)

	updates = append(updates, setOutFacingInPortListIndex(
		root, streamHandleValues.streamHandle, streamHandleValues.outFacingInputPortList, port, deviceIp)...)

	updates = append(updates, setOutFacingOutPortListIndex(
		root, streamHandleValues.streamHandle, streamHandleValues.outFacingOutputPortList, port, deviceIp)...)

	// set tsn-stream-identification-type
	updates = append(updates, setTsnStreamIdType(root, port, deviceIp, streamHandleValues.streamHandle, streamHandleValues.tsnStreamIdType))

	// Set null-stream-identification-entry
	// updates = append(updates, setNullStreamId(root, port, deviceIp, streamHandleValues.streamHandle, streamHandleValues.nullStreamId)...)

	// set source-mac-identification-entry
	updates = append(updates, setSourceMacId(root, port, deviceIp, streamHandleValues.streamHandle, streamHandleValues.sourceMacId)...)

	// set active-destination-mac-identification-entry
	// updates = append(updates, setActiveDestMacId(root, port, deviceIp, streamHandleValues.streamHandle, streamHandleValues.activeDestMacId)...)

	return updates, nil
}
