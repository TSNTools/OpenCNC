package notificationServer

import (
	"errors"
	"fmt"
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
	if len(interfaces) == 0 {
		return "", errors.New("no endpoint interface")
	}

	iface := interfaces[0]
	if iface == nil || iface.InterfaceId == nil {
		return "", errors.New("missing interface ID")
	}

	if iface.InterfaceId.InterfaceName == "" {
		return "", errors.New("missing interface name: " + iface.InterfaceId.String())
	}

	return iface.InterfaceId.InterfaceName, nil
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
	// Stream rank
	// ------------------------------------------------------------
	var rank *stream.StreamRank

	if talker.StrRank != nil {
		rank = &stream.StreamRank{
			Rank:     talker.StrRank.Rank,
			Priority: 0, // UNI model currently has no explicit priority
		}
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

	return &stream.Stream{
		StreamId:                  streamID,
		StreamRank:                rank,
		TalkerNodeId:              talkerNodeID,
		ListenerNodeIds:           listenerNodeIDs,
		FrameSpec:                 frameSpec,
		TrafficSpec:               trafficSpec,
		UserToNetworkRequirements: requirements,
		Description:               req.GetId(),
	}, nil
}
