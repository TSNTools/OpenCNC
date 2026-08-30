package uni_server

import (
	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/stream"
	topology_config "OpenCNC/common/structures/topology_config"
	uni "OpenCNC/common/structures/uni"

	"errors"
	"fmt"
	"strings"
)

// -----------------------------------------------------------------------------
// UNI response status values.
//
// These correspond to the values currently documented in uni.proto:
//
// Talker:
//   0 = None
//   1 = Ready
//   2 = Failed
//
// Listener:
//   0 = None
//   1 = Ready
//   2 = Partial failed
//   3 = Failed
//
// FailureCode is currently not an enum in the protobuf, so 0 means success.
// -----------------------------------------------------------------------------

const (
	talkerStatusNone   int32 = 0
	talkerStatusReady  int32 = 1
	talkerStatusFailed int32 = 2

	listenerStatusNone          int32 = 0
	listenerStatusReady         int32 = 1
	listenerStatusPartialFailed int32 = 2
	listenerStatusFailed        int32 = 3

	failureCodeNone                   uint32 = 0
	failureCodeInvalidRequest         uint32 = 1
	failureCodeInterfaceNotConfigured uint32 = 2
)

// -----------------------------------------------------------------------------
// createResponse
// -----------------------------------------------------------------------------
//
// Creates a complete UNI ConfigResponse.
//
// The configuration is loaded from the store using confId. The request is used
// to identify the streams and requested end-station interfaces. The stored
// topology configuration is then used to determine whether those interfaces
// are actually configured.
//
// The function signature intentionally remains:
//
//	createResponse(confId string, confReq *uni.ConfigRequest) ([]byte, error)
func createResponse(confId string, confReq *uni.ConfigRequest) (*uni.ConfigResponse, error) {
	if confReq == nil {
		return nil, errors.New("config request is nil")
	}

	configuration, err := store.GetConfiguration(confId)
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration %q: %w", confId, err)
	}

	if configuration == nil {
		return nil, fmt.Errorf("configuration %q is nil", confId)
	}

	response := &uni.ConfigResponse{
		Version:   confReq.GetVersion(),
		Responses: make([]*uni.Response, 0, len(confReq.GetRequests())),
	}

	for _, request := range confReq.GetRequests() {
		if request == nil {
			continue
		}

		resp := genResponse(request, configuration)
		response.Responses = append(response.Responses, resp)
	}

	return response, nil
}

// -----------------------------------------------------------------------------
// Response generation
// -----------------------------------------------------------------------------

func genResponse(
	request *uni.Request,
	configuration *topology_config.TopologyConfig,
) *uni.Response {
	if request == nil {
		return &uni.Response{}
	}

	return &uni.Response{
		RequestId:   request.GetId(),
		StatusGroup: genStatusGroup(request, configuration),
	}
}

func genStatusGroup(
	request *uni.Request,
	configuration *topology_config.TopologyConfig,
) *uni.StatusGroup {
	statusInfo, failedInterfaces := genStatusInfo(request, configuration)

	return &uni.StatusGroup{
		StrId:                genStreamID(request),
		StatusInfo:           statusInfo,
		FailedInterfaces:     failedInterfaces,
		StatusTalkerListener: genStatusTalkerListener(request, configuration),
		EndStationInterfaces: genEndStationInterfaces(request),
	}
}

// -----------------------------------------------------------------------------
// Stream ID
// -----------------------------------------------------------------------------

func genStreamID(request *uni.Request) *stream.StreamId {
	if request == nil || request.GetTalker() == nil {
		return &stream.StreamId{}
	}

	streamID := request.GetTalker().GetStrId()
	if streamID == nil {
		return &stream.StreamId{}
	}

	return &stream.StreamId{
		MacAddress: streamID.GetMacAddress(),
		UniqueId:   streamID.GetUniqueId(),
	}
}

// -----------------------------------------------------------------------------
// Status
// -----------------------------------------------------------------------------

func genStatusInfo(
	request *uni.Request,
	configuration *topology_config.TopologyConfig,
) (*uni.StatusInfo, []*uni.InterfaceId) {
	if request == nil || request.GetTalker() == nil {
		return &uni.StatusInfo{
			TalkerStatus:   talkerStatusFailed,
			ListenerStatus: listenerStatusFailed,
			FailureCode:    failureCodeInvalidRequest,
		}, nil
	}

	failedInterfaces := findFailedInterfaces(request, configuration)

	talkerFailed := false
	listenerFailed := false

	talkerFailed = hasFailedTalkerInterface(request, failedInterfaces)
	listenerFailed = hasFailedListenerInterface(request, failedInterfaces)

	var talkerStatus int32
	if talkerFailed {
		talkerStatus = talkerStatusFailed
	} else {
		talkerStatus = talkerStatusReady
	}

	var listenerStatus int32

	listenerCount := countListenerInterfaces(request)
	failedListenerCount := countFailedListenerInterfaces(request, failedInterfaces)

	switch {
	case listenerCount == 0:
		listenerStatus = listenerStatusNone

	case failedListenerCount == 0:
		listenerStatus = listenerStatusReady

	case failedListenerCount < listenerCount:
		listenerStatus = listenerStatusPartialFailed

	default:
		listenerStatus = listenerStatusFailed
	}

	if !talkerFailed && !listenerFailed {
		return &uni.StatusInfo{
			TalkerStatus:   talkerStatus,
			ListenerStatus: listenerStatus,
			FailureCode:    failureCodeNone,
		}, failedInterfaces
	}

	return &uni.StatusInfo{
		TalkerStatus:   talkerStatus,
		ListenerStatus: listenerStatus,
		FailureCode:    failureCodeInterfaceNotConfigured,
	}, failedInterfaces
}

// -----------------------------------------------------------------------------
// Failed interface detection
// -----------------------------------------------------------------------------

func findFailedInterfaces(
	request *uni.Request,
	configuration *topology_config.TopologyConfig,
) []*uni.InterfaceId {
	if request == nil || configuration == nil {
		return nil
	}

	var failed []*uni.InterfaceId

	if request.GetTalker() != nil {
		for _, intf := range request.GetTalker().GetEndStationInterfaces() {
			if intf == nil || intf.GetInterfaceId() == nil {
				continue
			}

			if !interfaceExistsInConfiguration(
				intf.GetInterfaceId(),
				configuration,
			) {
				failed = append(failed, cloneInterfaceID(intf.GetInterfaceId()))
			}
		}
	}

	for _, listener := range request.GetListenerList() {
		if listener == nil {
			continue
		}

		for _, intf := range listener.GetEndStationInterfaces() {
			if intf == nil || intf.GetInterfaceId() == nil {
				continue
			}

			if !interfaceExistsInConfiguration(
				intf.GetInterfaceId(),
				configuration,
			) {
				failed = append(failed, cloneInterfaceID(intf.GetInterfaceId()))
			}
		}
	}

	return failed
}

// interfaceExistsInConfiguration checks the requested UNI interface against
// the stored topology.
//
// TopologyConfig contains NodeConfig -> PortConfig. The UNI InterfaceId has:
//
//	mac-address
//	interface-name
//
// while PortConfig currently has:
//
//	port-id
//
// Therefore interface-name is matched against port-id.
//
// If an InterfaceId contains both a MAC address and interface name, the
// interface name is still the primary identifier because PortConfig currently
// does not contain a MAC-address field.
//
// Empty interface names cannot be matched safely and are therefore considered
// not configured.
func interfaceExistsInConfiguration(
	interfaceID *uni.InterfaceId,
	configuration *topology_config.TopologyConfig,
) bool {
	if interfaceID == nil || configuration == nil {
		return false
	}

	interfaceName := strings.TrimSpace(interfaceID.GetInterfaceName())
	if interfaceName == "" {
		return false
	}

	for _, node := range configuration.GetNodeConfigs() {
		if node == nil {
			continue
		}

		for _, port := range node.GetPortConfigs() {
			if port == nil {
				continue
			}

			portID := strings.TrimSpace(port.GetPortId())

			if portID == "" {
				continue
			}

			if strings.EqualFold(portID, interfaceName) {
				return true
			}
		}
	}

	return false
}

func hasFailedTalkerInterface(
	request *uni.Request,
	failed []*uni.InterfaceId,
) bool {
	if request == nil || request.GetTalker() == nil {
		return true
	}

	for _, intf := range request.GetTalker().GetEndStationInterfaces() {
		if intf == nil || intf.GetInterfaceId() == nil {
			continue
		}

		if containsInterfaceID(failed, intf.GetInterfaceId()) {
			return true
		}
	}

	return false
}

func hasFailedListenerInterface(
	request *uni.Request,
	failed []*uni.InterfaceId,
) bool {
	if request == nil {
		return true
	}

	for _, listener := range request.GetListenerList() {
		if listener == nil {
			continue
		}

		for _, intf := range listener.GetEndStationInterfaces() {
			if intf == nil || intf.GetInterfaceId() == nil {
				continue
			}

			if containsInterfaceID(failed, intf.GetInterfaceId()) {
				return true
			}
		}
	}

	return false
}

func containsInterfaceID(
	interfaces []*uni.InterfaceId,
	target *uni.InterfaceId,
) bool {
	if target == nil {
		return false
	}

	for _, intf := range interfaces {
		if intf == nil {
			continue
		}

		if sameInterfaceID(intf, target) {
			return true
		}
	}

	return false
}

func sameInterfaceID(
	a *uni.InterfaceId,
	b *uni.InterfaceId,
) bool {
	if a == nil || b == nil {
		return false
	}

	if a.GetInterfaceName() != "" &&
		b.GetInterfaceName() != "" &&
		strings.EqualFold(
			a.GetInterfaceName(),
			b.GetInterfaceName(),
		) {
		return true
	}

	if a.GetMacAddress() != "" &&
		b.GetMacAddress() != "" &&
		strings.EqualFold(
			a.GetMacAddress(),
			b.GetMacAddress(),
		) {
		return true
	}

	return false
}

func cloneInterfaceID(
	interfaceID *uni.InterfaceId,
) *uni.InterfaceId {
	if interfaceID == nil {
		return nil
	}

	return &uni.InterfaceId{
		MacAddress:    interfaceID.GetMacAddress(),
		InterfaceName: interfaceID.GetInterfaceName(),
	}
}

// -----------------------------------------------------------------------------
// Talker/listener status
// -----------------------------------------------------------------------------

func genStatusTalkerListener(
	request *uni.Request,
	configuration *topology_config.TopologyConfig,
) []*uni.TalkerListenerStatus {
	if request == nil || request.GetTalker() == nil {
		return nil
	}

	statuses := make([]*uni.TalkerListenerStatus, 0)

	// Index 0 is always the talker.
	talker := &uni.TalkerListenerStatus{
		Index: uint32(0),
		AccumulatedLatency: &uni.AccumulatedLatency{
			AccumulatedLatency: getAccumulatedLatency(request),
		},
		InterfaceConfiguration: genInterfaceConfigurationList(
			request,
			nil,
			configuration,
		),
	}

	statuses = append(statuses, talker)

	// Listener indexes start at 1.
	for index, listener := range request.GetListenerList() {
		if listener == nil {
			continue
		}

		listenerStatus := &uni.TalkerListenerStatus{
			Index: uint32(index + 1),
			AccumulatedLatency: &uni.AccumulatedLatency{
				AccumulatedLatency: getAccumulatedLatency(request),
			},
			InterfaceConfiguration: genInterfaceConfigurationList(
				request,
				listener,
				configuration,
			),
		}

		statuses = append(statuses, listenerStatus)
	}

	return statuses
}

// The UNI response requires accumulated latency, but TopologyConfig currently
// does not expose a bridge-delay/propagation-delay model. Do not invent a
// latency value.
//
// Until the topology model contains the required delay information, zero means
// "not calculated".
func getAccumulatedLatency(_ *uni.Request) uint32 {
	return 0
}

// -----------------------------------------------------------------------------
// Interface configuration
// -----------------------------------------------------------------------------

func genInterfaceConfigurationList(
	request *uni.Request,
	listener *uni.ListenerGroup,
	configuration *topology_config.TopologyConfig,
) []*uni.InterfaceConfiguration {
	if request == nil || request.GetTalker() == nil {
		return nil
	}

	talker := request.GetTalker()

	identificationType, _ := getStreamFrameIdentificationType(
		talker.GetDataFrameSpecification(),
	)

	mac, vlanTag, ipv4, ipv6 := getStreamFrameIdentification(
		identificationType,
		talker,
	)

	var interfaces []*uni.Interface

	if listener == nil {
		// Talker.
		interfaces = talker.GetEndStationInterfaces()
	} else {
		// Listener.
		interfaces = listener.GetEndStationInterfaces()
	}

	result := make([]*uni.InterfaceConfiguration, 0, len(interfaces))

	for _, intf := range interfaces {
		if intf == nil || intf.GetInterfaceId() == nil {
			continue
		}

		// Only return interface configuration for interfaces that exist in
		// the stored topology.
		if configuration != nil &&
			!interfaceExistsInConfiguration(
				intf.GetInterfaceId(),
				configuration,
			) {
			continue
		}

		interfaceConfiguration := &uni.InterfaceConfiguration{
			InterfaceId: cloneInterfaceID(intf.GetInterfaceId()),
			Type:        identificationType,
			MacAddr:     cloneMacAddress(mac),
			VlanTag:     cloneVlanTag(vlanTag),
			Ipv4Tup:     cloneIPv4Tuple(ipv4),
			Ipv6Tup:     cloneIPv6Tuple(ipv6),
		}

		// TimeAwareOffset is only meaningful for a talker and only when the
		// request contains TimeAware information.
		if listener == nil &&
			talker.GetTrafficSpecification() != nil &&
			talker.GetTrafficSpecification().GetTimeAware() != nil {
			interfaceConfiguration.TimeAwareOffset = &uni.TimeAwareOffset{
				Offset: 0,
			}
		}

		result = append(result, interfaceConfiguration)
	}

	return result
}

// -----------------------------------------------------------------------------
// Stream frame identification
// -----------------------------------------------------------------------------

func getStreamFrameIdentificationType(
	dataframeSpecList []*uni.DataFrameSpecification,
) (int32, error) {
	for _, dataFrameSpec := range dataframeSpecList {
		if dataFrameSpec == nil {
			continue
		}

		if dataFrameSpec.GetMacAddr() != nil &&
			(dataFrameSpec.GetMacAddr().GetSourceMac() != "" ||
				dataFrameSpec.GetMacAddr().GetDestinationMac() != "") {
			return 1, nil
		}

		if dataFrameSpec.GetIpv4Tup() != nil &&
			(dataFrameSpec.GetIpv4Tup().GetSrcIpAddr() != "") {
			return 2, nil
		}

		if dataFrameSpec.GetIpv6Tup() != nil &&
			dataFrameSpec.GetIpv6Tup().GetSrcIpAddr() != "" {
			return 3, nil
		}
	}

	return 1, errors.New("could not determine stream frame identification type")
}

func getStreamFrameIdentification(
	idType int32,
	talker *uni.TalkerGroup,
) (
	*uni.IeeeMacAddress,
	*uni.IeeeVlanTag,
	*uni.Ipv4Tuple,
	*uni.Ipv6Tuple,
) {
	mac := &uni.IeeeMacAddress{}
	vlanTag := &uni.IeeeVlanTag{}
	ipv4 := &uni.Ipv4Tuple{}
	ipv6 := &uni.Ipv6Tuple{}

	if talker == nil {
		return mac, vlanTag, ipv4, ipv6
	}

	switch idType {
	case 1:
		mac = getMacAddress(talker)
		vlanTag = getVlanTag(talker)

	case 2:
		ipv4 = getIPv4(talker)

	case 3:
		ipv6 = getIPv6(talker)
	}

	return mac, vlanTag, ipv4, ipv6
}

func getMacAddress(
	talker *uni.TalkerGroup,
) *uni.IeeeMacAddress {
	for _, spec := range talker.GetDataFrameSpecification() {
		if spec == nil || spec.GetMacAddr() == nil {
			continue
		}

		mac := spec.GetMacAddr()

		if mac.GetSourceMac() != "" ||
			mac.GetDestinationMac() != "" {
			return cloneMacAddress(mac)
		}
	}

	return &uni.IeeeMacAddress{}
}

func getVlanTag(
	talker *uni.TalkerGroup,
) *uni.IeeeVlanTag {
	for _, spec := range talker.GetDataFrameSpecification() {
		if spec == nil || spec.GetVlanTag() == nil {
			continue
		}

		vlan := spec.GetVlanTag()

		if vlan.GetVlanId() != 0 ||
			vlan.GetPriorityCodePoint() != 0 {
			return cloneVlanTag(vlan)
		}
	}

	return &uni.IeeeVlanTag{}
}

func getIPv4(
	talker *uni.TalkerGroup,
) *uni.Ipv4Tuple {
	for _, spec := range talker.GetDataFrameSpecification() {
		if spec == nil || spec.GetIpv4Tup() == nil {
			continue
		}

		ip := spec.GetIpv4Tup()

		if ip.GetSrcIpAddr() != "" ||
			ip.GetDestIpAddr() != "" ||
			ip.GetSrcPort() != 0 ||
			ip.GetDestPort() != 0 {
			return cloneIPv4Tuple(ip)
		}
	}

	return &uni.Ipv4Tuple{}
}

func getIPv6(
	talker *uni.TalkerGroup,
) *uni.Ipv6Tuple {
	for _, spec := range talker.GetDataFrameSpecification() {
		if spec == nil || spec.GetIpv6Tup() == nil {
			continue
		}

		ip := spec.GetIpv6Tup()

		if ip.GetSrcIpAddr() != "" ||
			ip.GetDestIpAddr() != "" ||
			ip.GetSrcPort() != 0 ||
			ip.GetDestPort() != 0 {
			return cloneIPv6Tuple(ip)
		}
	}

	return &uni.Ipv6Tuple{}
}

// -----------------------------------------------------------------------------
// End-station interfaces
// -----------------------------------------------------------------------------

func genEndStationInterfaces(
	request *uni.Request,
) []*uni.Interface {
	if request == nil || request.GetTalker() == nil {
		return nil
	}

	result := make([]*uni.Interface, 0)

	var index int32

	for _, intf := range request.GetTalker().GetEndStationInterfaces() {
		if intf == nil {
			continue
		}

		result = append(result, &uni.Interface{
			Index:       index,
			InterfaceId: cloneInterfaceID(intf.GetInterfaceId()),
		})

		index++
	}

	for _, listener := range request.GetListenerList() {
		if listener == nil {
			continue
		}

		for _, intf := range listener.GetEndStationInterfaces() {
			if intf == nil {
				continue
			}

			result = append(result, &uni.Interface{
				Index:       index,
				InterfaceId: cloneInterfaceID(intf.GetInterfaceId()),
			})

			index++
		}
	}

	return result
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func countListenerInterfaces(
	request *uni.Request,
) int {
	if request == nil {
		return 0
	}

	count := 0

	for _, listener := range request.GetListenerList() {
		if listener == nil {
			continue
		}

		count += len(listener.GetEndStationInterfaces())
	}

	return count
}

func countFailedListenerInterfaces(
	request *uni.Request,
	failed []*uni.InterfaceId,
) int {
	if request == nil {
		return 0
	}

	count := 0

	for _, listener := range request.GetListenerList() {
		if listener == nil {
			continue
		}

		for _, intf := range listener.GetEndStationInterfaces() {
			if intf == nil || intf.GetInterfaceId() == nil {
				continue
			}

			if containsInterfaceID(
				failed,
				intf.GetInterfaceId(),
			) {
				count++
			}
		}
	}

	return count
}

func cloneMacAddress(
	in *uni.IeeeMacAddress,
) *uni.IeeeMacAddress {
	if in == nil {
		return &uni.IeeeMacAddress{}
	}

	return &uni.IeeeMacAddress{
		DestinationMac: in.GetDestinationMac(),
		SourceMac:      in.GetSourceMac(),
	}
}

func cloneVlanTag(
	in *uni.IeeeVlanTag,
) *uni.IeeeVlanTag {
	if in == nil {
		return &uni.IeeeVlanTag{}
	}

	return &uni.IeeeVlanTag{
		PriorityCodePoint: in.GetPriorityCodePoint(),
		VlanId:            in.GetVlanId(),
	}
}

func cloneIPv4Tuple(
	in *uni.Ipv4Tuple,
) *uni.Ipv4Tuple {
	if in == nil {
		return &uni.Ipv4Tuple{}
	}

	return &uni.Ipv4Tuple{
		SrcIpAddr:  in.GetSrcIpAddr(),
		DestIpAddr: in.GetDestIpAddr(),
		Dscp:       in.GetDscp(),
		Protocol:   in.GetProtocol(),
		SrcPort:    in.GetSrcPort(),
		DestPort:   in.GetDestPort(),
	}
}

func cloneIPv6Tuple(
	in *uni.Ipv6Tuple,
) *uni.Ipv6Tuple {
	if in == nil {
		return &uni.Ipv6Tuple{}
	}

	return &uni.Ipv6Tuple{
		SrcIpAddr:  in.GetSrcIpAddr(),
		DestIpAddr: in.GetDestIpAddr(),
		Dscp:       in.GetDscp(),
		Protocol:   in.GetProtocol(),
		SrcPort:    in.GetSrcPort(),
		DestPort:   in.GetDestPort(),
	}
}
