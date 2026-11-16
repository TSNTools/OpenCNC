package notificationServer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	storewrapper "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/stream"
	"OpenCNC/common/structures/uni"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"github.com/google/uuid"
)

func CreateOptimizationTask(requests []*uni.Request) (*optimizer.OptimizationTask, error) {
	streamIDs := make([]string, 0, len(requests))

	for _, req := range requests {
		stream, err := createStreamFromRequest(req)
		if err != nil {
			return nil, fmt.Errorf("create stream from request %q: %w", req.GetId(), err)
		}

		if err := storewrapper.StoreStream(stream, stream.StreamId.AsKey()); err != nil {
			return nil, fmt.Errorf("failed to store stream from request %q: %w", req.GetId(), err)
		}

		streamIDs = append(streamIDs, stream.StreamId.AsKey())
	}

	return &optimizer.OptimizationTask{
		TaskId:  uuid.NewString(),
		Intent:  optimizer.Intent_ADD_STREAM,
		Trigger: optimizer.Trigger_STREAM_ADDITION,

		Scope: &optimizer.Scope{
			StreamIds: streamIDs,
		},

		Objectives: []*optimizer.Objective{
			{
				Metric:   optimizer.Metric_MINIMIZE_CONFIGURATION_CHANGES,
				Priority: 1,
			},
			{
				Metric:   optimizer.Metric_MINIMIZE_LATENCY,
				Priority: 2,
			},
		},

		ChangePolicy: &optimizer.ChangePolicy{
			AllowPathChanges:        true,
			AllowScheduleChanges:    true,
			AllowQueueChanges:       true,
			AllowStreamMigration:    false,
			PreserveExistingStreams: true,
			MaxChangedStreams:       uint32(len(requests)),
		},
	}, nil
}

func endpointNodeID(interfaces []*uni.Interface) (string, error) {
	if len(interfaces) == 0 || interfaces[0] == nil ||
		interfaces[0].InterfaceId == nil {
		return "", errors.New("missing endpoint interface")
	}

	macAddress := strings.TrimSpace(interfaces[0].InterfaceId.MacAddress)
	if macAddress == "" {
		return "", errors.New("missing interface MAC address")
	}

	// Find the topology node containing the port with this MAC address.
	topo, err := storewrapper.GetTopology()
	if err != nil {
		return "", fmt.Errorf("failed to read topology: %w", err)
	}
	if topo == nil {
		return "", errors.New("topology is nil")
	}

	var matchedNode string

	for _, node := range topo.GetNodes() {
		if node == nil {
			continue
		}

		for _, port := range node.GetPorts() {
			if port == nil || port.GetMacAddress() == "" {
				continue
			}

			if strings.EqualFold(port.GetMacAddress(), macAddress) {
				if node.GetName() == "" {
					return "", fmt.Errorf(
						"port with MAC %q belongs to an unnamed node",
						macAddress,
					)
				}

				if matchedNode != "" && matchedNode != node.GetName() {
					return "", fmt.Errorf(
						"MAC address %q belongs to multiple nodes",
						macAddress,
					)
				}

				matchedNode = node.GetName()
			}
		}
	}

	if matchedNode == "" {
		return "", fmt.Errorf(
			"no topology node found for MAC address %q",
			macAddress,
		)
	}

	return matchedNode, nil
}

func createFrameSpecification(talker *uni.TalkerGroup, listeners []*uni.ListenerGroup) *stream.DataFrameSpecification {

	//TODO: check the source and destination MAC addresses against listeners for consistency
	if len(talker.DataFrameSpecification) == 0 {
		return nil
	}

	spec := talker.DataFrameSpecification[0]
	if spec == nil {
		return nil
	}

	result := &stream.DataFrameSpecification{}

	if spec.MacAddr != nil {
		result.DestinationMac = spec.MacAddr.DestinationMac
		result.SourceMac = spec.MacAddr.SourceMac
	}

	// Ethertype is not explicitly represented in the UNI
	// DataFrameSpecification. We leave it unset.
	//
	// IPv4/IPv6 tuple information is currently not represented
	// in stream.DataFrameSpecification either.

	return result
}

func createTrafficSpecification(ts *uni.TrafficSpecification) (*stream.TrafficSpecification, error) {

	if ts == nil {
		return nil, nil
	}

	result := &stream.TrafficSpecification{
		MaxFrameSize:      uint64(ts.MaxFrameSize),
		MaxIntervalFrames: uint64(ts.MaxFramesPerInterval),
	}

	if ts.Interval != nil {
		if ts.Interval.Denominator == 0 {
			return nil, errors.New("interval denominator is zero")
		}

		// Assumes Numerator/Denominator represents seconds.
		// Adjust here if your UNI implementation uses another unit.
		result.IntervalNs =
			uint64(ts.Interval.Numerator) * uint64(time.Second) /
				uint64(ts.Interval.Denominator)
	}

	return result, nil
}

func createUserToNetworkRequirements(req *uni.UserToNetworkRequirements) *stream.UserToNetworkRequirements {

	if req == nil {
		return nil
	}

	return &stream.UserToNetworkRequirements{
		MaxLatencyNs:     uint64(req.MaxLatency),
		NumSeamlessTrees: req.NumSeamlessTrees,
	}
}

func createStreamFromRequest(req *uni.Request) (*stream.Stream, error) {
	//the uni request and stream object are not quite aligned;
	// additional mapping is implemented below
	//TODO: Align the protos and make this lighter or not necessary

	if req == nil {
		return nil, errors.New("nil request")
	}
	if req.Talker == nil {
		return nil, fmt.Errorf("request %q has no talker", req.GetId())
	}
	if req.Talker.StrId == nil {
		return nil, fmt.Errorf("request %q has no stream ID", req.GetId())
	}

	talker := req.Talker
	listeners := req.ListenerList

	// ------------------------------------------------------------
	// Stream ID
	// ------------------------------------------------------------
	streamID := talker.StrId

	// ------------------------------------------------------------
	// Talker node
	// ------------------------------------------------------------
	talkerNodeID, err := endpointNodeID(talker.EndStationInterfaces)
	if err != nil {
		return nil, fmt.Errorf("request %q: invalid talker interface: %w",
			req.GetId(), err)
	}

	// ------------------------------------------------------------
	// Listener nodes
	// ------------------------------------------------------------
	listenerNodeIDs := make([]string, 0, len(listeners))

	for _, listener := range listeners {
		if listener == nil {
			continue
		}

		nodeID, err := endpointNodeID(listener.EndStationInterfaces)
		if err != nil {
			return nil, fmt.Errorf(
				"request %q: invalid listener interface: %w",
				req.GetId(), err,
			)
		}

		listenerNodeIDs = append(listenerNodeIDs, nodeID)
	}

	if len(listenerNodeIDs) == 0 {
		return nil, fmt.Errorf("request %q has no listeners", req.GetId())
	}

	// ------------------------------------------------------------
	// Data-frame specification
	// ------------------------------------------------------------
	frameSpec := createFrameSpecification(talker, listeners)

	// ------------------------------------------------------------
	// Traffic specification
	// ------------------------------------------------------------
	trafficSpec, err := createTrafficSpecification(talker.TrafficSpecification)
	if err != nil {
		return nil, fmt.Errorf(
			"request %q: invalid traffic specification: %w",
			req.GetId(), err,
		)
	}

	// ------------------------------------------------------------
	// User-to-network requirements
	// ------------------------------------------------------------
	requirements := createUserToNetworkRequirements(
		talker.UserToNetReq,
	)

	// TEMPORARY CBS PILOT (Hamza, 15 November 2026):
	// Traffic classes are normally assigned through policy or optimization.
	// Since OpenCNC's policy integration is not yet available, use "M1"
	// for the current controlled CBS integration test.
	// This is not derived from PCP and is not a production classification rule.
	// Remove this hardcoded assignment when policy integration is available.
	return &stream.Stream{
		StreamId:                  streamID,
		StreamRank:                talker.StrRank,
		TalkerNodeId:              talkerNodeID,
		ListenerNodeIds:           listenerNodeIDs,
		FrameSpec:                 frameSpec,
		TrafficSpec:               trafficSpec,
		UserToNetworkRequirements: requirements,
		Description:               req.GetId(),
		TrafficClassId:            "Critical",
	}, nil
}
