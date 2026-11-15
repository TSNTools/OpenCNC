// tests matching streams, preserving the original FPM, rejecting conflicting PCPs, and handling invalid or ambiguous input.
package internalOptimizer

import (
	"strings"
	"testing"

	"OpenCNC/common/structures/stream"
	"OpenCNC/common/structures/stream_config"
	"OpenCNC/common/structures/uni"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

func pcpTestPlane() *forwardingplane.ForwardingPlaneModel {
	return &forwardingplane.ForwardingPlaneModel{Streams: []*forwardingplane.StreamModel{
		{Definition: &stream.Stream{StreamId: &stream.StreamId{MacAddress: "00-00-00-00-00-01", UniqueId: 1}}, Configuration: &stream_config.StreamConfiguration{}},
		{Definition: &stream.Stream{StreamId: &stream.StreamId{MacAddress: "00-00-00-00-00-02", UniqueId: 2}}, Configuration: &stream_config.StreamConfiguration{}},
	}}
}

func pcpTestRequest(mac string, id uint32, pcp uint32) *uni.Request {
	return &uni.Request{Talker: &uni.TalkerGroup{
		StrId:                  &stream.StreamId{MacAddress: mac, UniqueId: id},
		DataFrameSpecification: []*uni.DataFrameSpecification{{VlanTag: &uni.IeeeVlanTag{PriorityCodePoint: pcp}}},
	}}
}

func TestWithUNIPCPKeepsOriginalAndMatchesByStreamID(t *testing.T) {
	original := pcpTestPlane()
	before := proto.Clone(original)
	requests := []*uni.Request{
		pcpTestRequest("00-00-00-00-00-02", 2, 7),
		pcpTestRequest("00-00-00-00-00-01", 1, 6),
	}
	updated, err := withUNIPCP(original, requests)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(original, before) {
		t.Fatal("original forwarding plane was modified")
	}
	if updated == original {
		t.Fatal("expected separate request-only forwarding plane")
	}
	if updated.Streams[0].Configuration.Pcp == nil || *updated.Streams[0].Configuration.Pcp != 6 ||
		updated.Streams[1].Configuration.Pcp == nil || *updated.Streams[1].Configuration.Pcp != 7 {
		t.Fatal("UNI PCP was not mapped by stream ID")
	}
}

func TestWithUNIPCPRejectsConflictingStoredPCP(t *testing.T) {
	original := pcpTestPlane()
	existing := uint32(5)
	original.Streams[0].Configuration.Pcp = &existing
	_, err := withUNIPCP(original, []*uni.Request{pcpTestRequest("00-00-00-00-00-01", 1, 7)})
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestWithUNIPCPRejectsAmbiguousZero(t *testing.T) {
	_, err := withUNIPCP(pcpTestPlane(), []*uni.Request{pcpTestRequest("00-00-00-00-00-01", 1, 0)})
	if err == nil || !strings.Contains(err.Error(), "cannot distinguish") {
		t.Fatalf("expected proto3 presence error, got %v", err)
	}
}

func TestWithUNIPCPRejectsUnknownStream(t *testing.T) {
	_, err := withUNIPCP(pcpTestPlane(), []*uni.Request{pcpTestRequest("unknown", 9, 7)})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected unknown stream error, got %v", err)
	}
}
