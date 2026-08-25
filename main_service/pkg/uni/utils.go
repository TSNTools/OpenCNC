package uni

import (
	"errors"
	"fmt"
	store "main-service/pkg/store-wrapper"
	"main-service/pkg/structures/configuration"
	"main-service/pkg/structures/notification"
	"main-service/pkg/structures/schedule"
	"math"

	"google.golang.org/protobuf/encoding/protojson"
)

// Create response for add_stream/get_config
func createResponse(confId *notification.UUID, confReq *configuration.ConfigRequest) ([]byte, error) {
	var baseResp = &configuration.ConfigResponse{
		Version:   confReq.Version,
		Responses: genResponses(confReq.Requests),
	}

	// TODO: Add check if stream is okay (if stream is going to be used, then store it in the k/v store)
	// Store streams in k/v store
	if err := storeStreams(baseResp); err != nil {
		//log.Errorf("Failed storing streams in k/v store: %v", err)
		fmt.Printf("Failed storing streams in k/v store: %v", err)
		return nil, err
	}

	// rawData, err := protojson.Marshal(baseResp)
	rawData, err := protojson.Marshal(baseResp.GetResponses()[0].GetStatusGroup())
	if err != nil {
		//log.Errorf("Failed to marshal UNI response: %v", err)
		return nil, err
	}
	fmt.Printf("Marshalling UNI reponse successfully!")
	return rawData, nil
}

func storeStreams(event *configuration.ConfigResponse) error {
	for _, stream := range event.GetResponses() {
		// //log.Infof("About to store stream with mac and unique ID: %v - %v", stream.GetTalker().GetStrId().GetMacAddress(), stream.GetTalker().GetStrId().GetUniqueId())
		if err := store.StoreStream(stream); err != nil {
			//log.Errorf("Failed storing stream %v with err: %v", stream, err)
			return err
		}
	}
	return nil
}

// Generate list of responses
func genResponses(requests []*configuration.Request) []*configuration.Response {
	responses := []*configuration.Response{}
	for _, req := range requests {
		responses = append(responses, genResponse(req))
	}

	return responses
}

// Generate response
func genResponse(request *configuration.Request) *configuration.Response {
	response := &configuration.Response{
		StatusGroup: genStatusGroup(request),
	}

	return response
}

// Generate StatusGroup (contains all information in a response)
func genStatusGroup(request *configuration.Request) *configuration.StatusGroup {
	statusGroup := &configuration.StatusGroup{
		StrId:                genStreamId(1, request), // First param (1) is currently statically representing only 1 listener for the stream
		StatusInfo:           genStatusInfo(),
		FailedInterfaces:     genFailedInterfaces(),
		StatusTalkerListener: genStatusTalkerListener(request),
		EndStationInterfaces: genEndStationInterfaces(request),
	}

	return statusGroup
}

// Generate stream ID depending on the number of listeners
func genStreamId(numOfListeners int, request *configuration.Request) *configuration.StreamId {
	// If only one listener
	if numOfListeners == 1 {
		// Set stream id be the mac address and unique id of the talker
		return &configuration.StreamId{
			MacAddress: request.GetTalker().GetStrId().GetMacAddress(),
			UniqueId:   request.GetTalker().GetStrId().GetUniqueId(),
		}
	} else if numOfListeners > 1 { // If more than one listener
		// Make stream id either a specific mac address and unique id for the stream group
		//log.Info("Generating StreamId not implemented when there are more than 1 listener, a StreamId should be generated that is not the talkers ID")
	}

	//log.Errorf("Can't generate stream ID without any listeners for the stream")
	return &configuration.StreamId{}
}

// Generate StatusInfo (contains failure codes on talker, listener, and a combined?)
// Not implemented
func genStatusInfo() *configuration.StatusInfo {
	// Failure should be reported when a stream cannot be set up using the requirements for the given end station

	return &configuration.StatusInfo{}
}

// Generate list of failed interfaces
// Not implemented
func genFailedInterfaces() []*configuration.InterfaceId {

	return []*configuration.InterfaceId{}
}

// Generate list of most information related to end stations and their configuration
func genStatusTalkerListener(request *configuration.Request) []*configuration.TalkerListenerStatus {
	statusTalkerListenerList := []*configuration.TalkerListenerStatus{}

	index := 0

	// Add configuration for the talker
	talkerStatus := configuration.TalkerListenerStatus{
		Index: uint32(index),
		AccumulatedLatency: &configuration.AccumulatedLatency{
			AccumulatedLatency: getAccumulatedLatency(),
		},
		InterfaceConfiguration: genInterfaceConfigurationList(request, nil),
	}

	index += 1

	statusTalkerListenerList = append(statusTalkerListenerList, &talkerStatus)

	// Add configuration for each listener
	for _, listener := range request.GetListenerList() {
		listenerStatus := configuration.TalkerListenerStatus{
			Index: uint32(index),
			AccumulatedLatency: &configuration.AccumulatedLatency{
				AccumulatedLatency: getAccumulatedLatency(),
			},
			InterfaceConfiguration: genInterfaceConfigurationList(request, listener),
		}

		statusTalkerListenerList = append(statusTalkerListenerList, &listenerStatus)

		index += 1
	}

	return statusTalkerListenerList
}

// Don't know how to calculate accumulated latency (max latency from talker to listener through the designated path in the network)
// In the 802.1Qcc: "CNC will read the Bridge Delay (12.32.1) and Propagation Delay (12.32.2) from each Bridge in order to compute AccumulatedLatency (for step 9)"
// Required:
//   - Knowledge of bridges and interfaces in the stream path?
func getAccumulatedLatency() uint32 {
	// TODO: Get propagation delay from the elem "peer-mean-path-delay" in the ptp module (get it through using the monitor-service?)
	// TODO: Find where the Bridge Delay can be read

	return 1
}

// When listener == nil (for the talker node), the interfaceId is statically the first interface for the talker
func genInterfaceConfigurationList(request *configuration.Request, listener *configuration.ListenerGroup) []*configuration.InterfaceConfiguration {
	interfaceConfigurationList := []*configuration.InterfaceConfiguration{}

	streamFrameIdentificationType, err := getStreamFrameIdentificationType(request.GetTalker().GetDataFrameSpecification())
	if err != nil {
		//log.Warnf("Could not find the identification type for a stream's frames: %v\n Default value: 1", err)
	}

	mac, vlanTag, ipv4, ipv6 := getStreamFrameIdentification(streamFrameIdentificationType, request.GetTalker(), listener)

	if listener == nil {
		offset, err := getTimeAwareOffset(request)
		if err != nil {
			//log.Errorf("Failed getting TimeAwareOffset: %v", err)
			// TODO: Build failed interfaces list? (maybe not yet?)
			return interfaceConfigurationList
		}

		interfaceConf := configuration.InterfaceConfiguration{
			InterfaceId:     request.GetTalker().EndStationInterfaces[0].InterfaceId,
			Type:            streamFrameIdentificationType, // Interfaces can send frames using 3 types for identifying the stream's frames (1 = MAC-address/VLAN, 2 = ipv4, 3 = ipv6)
			MacAddr:         mac,
			VlanTag:         vlanTag,
			Ipv4Tup:         ipv4,
			Ipv6Tup:         ipv6,
			TimeAwareOffset: offset,
		}

		interfaceConfigurationList = append(interfaceConfigurationList, &interfaceConf)
	} else {
		for _, listenerInterface := range listener.GetEndStationInterfaces() {
			interfaceConf := configuration.InterfaceConfiguration{
				InterfaceId: listenerInterface.InterfaceId,
				Type:        streamFrameIdentificationType, // Interfaces can send frames using 3 types for identifying the stream's frames (1 = MAC-address/VLAN, 2 = ipv4, 3 = ipv6)
				MacAddr:     mac,
				VlanTag:     vlanTag,
				Ipv4Tup:     ipv4,
				Ipv6Tup:     ipv6,
			}

			interfaceConfigurationList = append(interfaceConfigurationList, &interfaceConf)
		}
	}

	return interfaceConfigurationList
}

func getStreamFrameIdentificationType(dataframeSpecList []*configuration.DataFrameSpecification) (int32, error) {
	streamFrameIdentificationType := -1

	for _, dataFrameSpec := range dataframeSpecList {
		if len(dataFrameSpec.GetMacAddr().GetSourceMac()) > 0 {
			streamFrameIdentificationType = 1
			break
		} else if len(dataFrameSpec.GetIpv4Tup().GetSrcIpAddr()) > 0 {
			streamFrameIdentificationType = 2
			break
		} else if len(dataFrameSpec.GetIpv6Tup().GetSrcIpAddr()) > 0 {
			streamFrameIdentificationType = 3
			break
		} else {
			//log.Errorf("Could not identify the streamFrameIdentificationType inside: %v", dataFrameSpec)
			return int32(1), errors.New("no streamFrameIdentificationType found")
		}
	}

	return int32(streamFrameIdentificationType), nil
}

func getStreamFrameIdentification(idType int32, talker *configuration.TalkerGroup, listener *configuration.ListenerGroup) (*configuration.IeeeMacAddress, *configuration.IeeeVlanTag, *configuration.Ipv4Tuple, *configuration.Ipv6Tuple) {
	mac := &configuration.IeeeMacAddress{}
	vlanTag := &configuration.IeeeVlanTag{}
	ipv4 := &configuration.Ipv4Tuple{}
	ipv6 := &configuration.Ipv6Tuple{}

	switch idType {
	case 1: // VLAN tagged
		// Add source and destination mac addresses
		for _, dataFrameSpec := range talker.GetDataFrameSpecification() {
			if dataFrameSpec.GetMacAddr().GetSourceMac() != "" {
				mac.SourceMac = dataFrameSpec.GetMacAddr().GetSourceMac()
				mac.DestinationMac = dataFrameSpec.GetMacAddr().GetDestinationMac()
			}
		}
		// Add vlan tag
		vlanTag = getVlanTag(talker)
	case 2: // IPv4
		ipv4 = getIpv4(talker)
	case 3: // IPv6
		ipv6 = getIpv6(talker)
	default:
		//log.Errorf("Did not find match for stream frame identification type with value: %v", idType)
	}

	return mac, vlanTag, ipv4, ipv6
}

func getVlanTag(talker *configuration.TalkerGroup) *configuration.IeeeVlanTag {
	for _, dataFrameSpec := range talker.GetDataFrameSpecification() {
		if dataFrameSpec.GetVlanTag().GetVlanId() != 0 {
			return &configuration.IeeeVlanTag{
				PriorityCodePoint: dataFrameSpec.GetVlanTag().GetPriorityCodePoint(),
				VlanId:            dataFrameSpec.GetVlanTag().GetVlanId(),
			}
		}
	}

	// Should not reach this, if it does, something is wrong
	return &configuration.IeeeVlanTag{}
}

func getIpv4(talker *configuration.TalkerGroup) *configuration.Ipv4Tuple {
	for _, dataFrameSpec := range talker.GetDataFrameSpecification() {
		if dataFrameSpec.GetIpv4Tup().GetDestPort() != 0 {
			return dataFrameSpec.GetIpv4Tup()
		}
	}

	// Should not reach this, if it does, something is wrong
	return &configuration.Ipv4Tuple{}
}

func getIpv6(talker *configuration.TalkerGroup) *configuration.Ipv6Tuple {
	for _, dataFrameSpec := range talker.GetDataFrameSpecification() {
		if dataFrameSpec.GetIpv6Tup().GetDestPort() != 0 {
			return dataFrameSpec.GetIpv6Tup()
		}
	}

	// Should not reach this, if it does, something is wrong
	return &configuration.Ipv6Tuple{}
}

// TODO: Get time aware offset from 0 + previous streams of the same traffic class?
// Need to get streams and schedule from k/v store
func getTimeAwareOffset(request *configuration.Request) (*configuration.TimeAwareOffset, error) {
	streams, err := store.GetAllStreams()
	if err != nil {
		//log.Errorf("Failed getting streams: %v", err)
		return nil, err
	}

	streamResponses := getStreamsOnSameTrafficClass(streams, request)
	offset, err := getNextOffsetForTrafficClass(streamResponses, request)
	if err != nil {
		//log.Errorf("Failed getting next offset for traffic class: %v", err)
		// return nil, err
	}

	timeAwareOffset := &configuration.TimeAwareOffset{
		Offset: offset,
	}

	return timeAwareOffset, err
}

func getStreamsOnSameTrafficClass(streams *configuration.ConfigResponse, request *configuration.Request) []*configuration.Response {
	// TODO: Identify traffic class for the new stream request

	// TODO: Search for already added streams that are using the same traffic class

	// Currently all streams are considered to be only isochronous (to simplify the system)
	return streams.GetResponses()
}

func getNextOffsetForTrafficClass(streamResponses []*configuration.Response, request *configuration.Request) (uint32, error) {
	schedule, err := store.GetSchedule("default_schedule")
	if err != nil {
		//log.Errorf("Failed getting schedule from store: %v", err)
		return 0, err
	}

	trafficClassOffsetMin, trafficClassOffsetMax := getSchedOffsetForTrafficClass("isochronous", schedule)

	//log.Infof("Window for traffic class \"%v\" is [%v, %v]", "isochronous", trafficClassOffsetMin, trafficClassOffsetMax)

	earliestTransmitOffset := request.Talker.TrafficSpecification.TimeAware.EarliestTransmitOffset
	latestTransmitOffset := request.Talker.TrafficSpecification.TimeAware.LatestTransmitOffset
	jitter := request.Talker.TrafficSpecification.TimeAware.Jitter

	// TODO: Need link/interface speed when getting frameTransmissionTime
	linkSpeed := 100000000 // 100Mbps

	// Round the frameTransmissionTime up to nearest integer (to be absolutely certain it will fit in the window)
	// x8 to go from bytes to bits
	// x1000000 to go from seconds to microseconds
	frameTransmissionTime := uint32(math.Ceil(float64(request.GetTalker().GetTrafficSpecification().GetMaxFrameSize()) * 8 / float64(linkSpeed) * 1000000))

	//log.Infof("Requested window for stream is [%v, %v], with jitter: %v, and frameTransmissionTime: %v", earliestTransmitOffset, latestTransmitOffset, jitter, frameTransmissionTime)

	// Calculate start time (the base time if no streams are interfering)
	calculatedOffset, err := getInitialStartOffset(trafficClassOffsetMin, trafficClassOffsetMax, earliestTransmitOffset, latestTransmitOffset, jitter, frameTransmissionTime)
	if err != nil {
		//log.Errorf("Failed getting inital start offset (the stream did not fit in the requested window or in the traffic class window)")
		return 0, err
	}

	calculatedOffset += jitter

	//log.Infof("Calculated offset with jitter is %v", calculatedOffset)

	// No extra streams to take into consideration (the calculated offset which is the first possible time for both the traffic class and the new stream, is used)
	if len(streamResponses) == 0 {
		return calculatedOffset, nil
	}

	// TODO: Take extra streams into consideration (add extra offset to the calculatedOffset variable)

	return calculatedOffset, nil
}

// Get the raw offset in the schedule for the start of the traffic classes window
func getSchedOffsetForTrafficClass(trafficClass string, sched *schedule.Schedule) (uint32, uint32) {
	var accumulatedOffsetMin uint32
	var accumulatedOffsetMax uint32
	scheduleCycle := sched.GetGatingCycle()

	for _, tc := range sched.GetTrafficClasses() {
		if tc.Name == trafficClass {
			// At correct traffic class
			accumulatedOffsetMax += uint32(tc.AssignedPortion) * 10000 * uint32(scheduleCycle)
			break
		} else {
			accumulatedOffsetMin += uint32(tc.AssignedPortion) * 10000 * uint32(scheduleCycle)
			accumulatedOffsetMax += uint32(tc.AssignedPortion) * 10000 * uint32(scheduleCycle)
		}
	}

	return accumulatedOffsetMin, accumulatedOffsetMax
}

// Not using jitter + not sure about the calculations on when it is okay to send (for both schedules in bridges and talker)
func getInitialStartOffset(tcOffsetMin uint32, tcOffsetMax uint32, earliestTransmitOffset uint32, latestTransmitOffset uint32,
	jitter uint32, frameTransmissionTime uint32) (uint32, error) {

	// Check if the talker can't send while still being inside the traffic class window
	if earliestTransmitOffset+frameTransmissionTime > tcOffsetMax || latestTransmitOffset+frameTransmissionTime < tcOffsetMin {
		// If the code gets here, the traffic class window is not large enough for the request window of the talker
		return 0, errors.New("could not find a suitable slot for the stream")
	}

	// Check if the talker can send at the traffic class window opening and be done before the window is closed (and also inside the talkers window)
	if tcOffsetMin+frameTransmissionTime <= tcOffsetMax && tcOffsetMin+frameTransmissionTime <= latestTransmitOffset {
		// Check if the talker wants to send later than traffic class window and if the talker could finish before tc window is closed
		if earliestTransmitOffset > tcOffsetMin && earliestTransmitOffset+frameTransmissionTime <= tcOffsetMax {
			return earliestTransmitOffset, nil
		} else if tcOffsetMin+frameTransmissionTime <= latestTransmitOffset {
			return tcOffsetMin, nil
		}

		return 0, errors.New("error finding slot for the stream")
	}

	//log.Warnf("If the code gets here, the initial start offset of when the stream can transmit is not calculated properly")

	return 0, errors.New("could not find any slots for the stream")
}

// Generate list of all interfaces included in the stream
func genEndStationInterfaces(request *configuration.Request) []*configuration.Interface {
	interfaces := []*configuration.Interface{}

	index := 0

	interfaces = append(interfaces, &configuration.Interface{
		Index:       int32(index),
		InterfaceId: request.GetTalker().GetEndStationInterfaces()[0].GetInterfaceId(),
	})

	for _, listener := range request.ListenerList {
		for _, listenerInterface := range listener.EndStationInterfaces {
			interfaceId := listenerInterface.GetInterfaceId()

			interfaces = append(interfaces, &configuration.Interface{
				Index:       int32(index),
				InterfaceId: interfaceId,
			})

			index += 1
		}
	}

	return interfaces
}

// []*configuration.Response{
// 	{
// 		StatusGroup: &configuration.StatusGroup{
// 			StrId: genStreamId(1, confReq.GetRequests()[0]),
// 			StatusInfo: &configuration.StatusInfo{
// 				TalkerStatus:   1,   // random value for now
// 				ListenerStatus: 1,   // random value for now
// 				FailureCode:    123, // random value for now
// 			},
// 			FailedInterfaces: []*configuration.InterfaceId{
// 				{
// 					MacAddress:    "11-22-33-44-55", // random value for now
// 					InterfaceName: "I-do-not-know",  // random value for now
// 				},
// 			},
// 			StatusTalkerListener: []*configuration.TalkerListenerStatus{
// 				{
// 					AccumulatedLatency: &configuration.AccumulatedLatency{
// 						AccumulatedLatency: uint32(confReq.GetRequests()[0].GetListenerList()[0].UserToNetReq.GetMaxLatency()),
// 					},
// 					InterfaceConfiguration: []*configuration.InterfaceConfiguration{
// 						{
// 							InterfaceId: &configuration.InterfaceId{
// 								MacAddress:    "11-22-33-44-55", // random value for now
// 								InterfaceName: "test",           // random value for now
// 							},
// 							Type: 123, // random value for now
// 							MacAddr: &configuration.IeeeMacAddress{
// 								DestinationMac: "destMac", // random value for now
// 								SourceMac:      "srcMac",  // random value for now
// 							},
// 							VlanTag: &configuration.IeeeVlanTag{
// 								PriorityCodePoint: 1, // random value for now
// 								VlanId:            1, // random value for now
// 							},
// 							Ipv4Tup: &configuration.Ipv4Tuple{},
// 							Ipv6Tup: &configuration.Ipv6Tuple{},
// 							TimeAwareOffset: &configuration.TimeAwareOffset{
// 								Offset: 123, // random value for now
// 							},
// 						},
// 					},
// 				},
// 			},
// 			EndStationInterfaces: []*configuration.Interface{
// 				{
// 					Index: 0, // random value for now
// 					InterfaceId: &configuration.InterfaceId{
// 						MacAddress:    "11-22-33-44-55", // random value for now
// 						InterfaceName: "testInterface",  // random value for now
// 					},
// 				},
// 			},
// 		},
// 	},
// },
