package internalOptimizer

import (
	"encoding/json"
	"strings"
	"testing"

	"OpenCNC/common/structures/stream"
	streamconfig "OpenCNC/common/structures/stream_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

// These tests exercise native projection after an assumed successful shared
// validation report. The shared v0.2.1 semantic validator has its own tests;
// these unit tests are NOT a live Manager-to-CNC integration test.
func nativeRouteApplyOriginal() *forwardingplane.ForwardingPlaneModel {
	original := responseApplyOriginal()
	identity := &stream.StreamId{MacAddress: "02:00:00:00:00:01", UniqueId: 7}
	priority := uint32(5)
	original.Streams = []*forwardingplane.StreamModel{
		{
			Definition: &stream.Stream{
				StreamId:     proto.Clone(identity).(*stream.StreamId),
				TalkerNodeId: "ES1", ListenerNodeIds: []string{"ES4"},
				TrafficClassId: "M1", Description: "preserve-definition",
			},
			Configuration: &streamconfig.StreamConfiguration{
				StreamId:    proto.Clone(identity).(*stream.StreamId),
				Description: "keep-configuration", Pcp: &priority, Enabled: true,
			},
		},
		{
			Definition: &stream.Stream{StreamId: &stream.StreamId{
				MacAddress: "02:00:00:00:00:02", UniqueId: 8,
			}, Description: "unrelated-stream"},
			Configuration: &streamconfig.StreamConfiguration{Description: "untouched"},
		},
	}
	return original
}

const nativeGeneratedRouteFixture = `{
  "status":"success",
  "solution":{"changes":{"streams":[{
    "streamId":{"macAddress":"02:00:00:00:00:01","uniqueId":7},
    "configuration":{"path":[{
      "hopNumber":0,"bridgeId":"SW1","ingressPortId":"P1","egressPortId":"P4"
    }]}
  }]}}
}`

func nativeModifyRoutePayload(t *testing.T, alter func(map[string]any)) string {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal([]byte(nativeGeneratedRouteFixture), &result); err != nil {
		t.Fatal(err)
	}
	alter(result)
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func nativeStreamChanges(result map[string]any) []any {
	return result["solution"].(map[string]any)["changes"].(map[string]any)["streams"].([]any)
}

func TestApplyValidatedOptimizerResponseGeneratedPathOnlyPreservesAllOtherFields(t *testing.T) {
	original := nativeRouteApplyOriginal()
	before := proto.Clone(original)
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, nativeGeneratedRouteFixture))
	if err != nil {
		t.Fatal(err)
	}
	if got == original || proto.Equal(got, original) {
		t.Fatal("native result must be a distinct updated model")
	}
	if !proto.Equal(original, before) {
		t.Fatal("native projection mutated the original")
	}

	expected := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)
	expected.Streams[0].Configuration.Path = []*streamconfig.StreamHop{{
		HopNumber: 0, BridgeId: "SW1", IngressPortId: "P1", EgressPortId: "P4",
	}}
	if !proto.Equal(got, expected) {
		t.Fatal("projection modified an unrelated native field or produced the wrong path")
	}
}

// Existing configurations may omit their repeated identity. Inserting a
// generated route must not silently change other saved native fields.
func TestApplyValidatedOptimizerResponseGeneratedPathPreservesAbsentConfigurationStreamID(t *testing.T) {
	original := nativeRouteApplyOriginal()
	original.Streams[0].Configuration.StreamId = nil
	before := proto.Clone(original)
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, nativeGeneratedRouteFixture))
	if err != nil {
		t.Fatal(err)
	}
	expected := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)
	expected.Streams[0].Configuration.Path = []*streamconfig.StreamHop{{
		HopNumber: 0, BridgeId: "SW1", IngressPortId: "P1", EgressPortId: "P4",
	}}
	if !proto.Equal(got, expected) {
		t.Fatal("native route application unexpectedly populated an absent existing configuration streamId")
	}
	if !proto.Equal(original, before) {
		t.Fatal("native application modified the preserved original")
	}
}

func TestApplyValidatedOptimizerResponseCombinedCBSAndGeneratedPath(t *testing.T) {
	original := nativeRouteApplyOriginal()
	before := proto.Clone(original)
	var combined map[string]any
	var routes map[string]any
	if err := json.Unmarshal([]byte(cbsUpdateFixture), &combined); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(nativeGeneratedRouteFixture), &routes); err != nil {
		t.Fatal(err)
	}
	changes := combined["solution"].(map[string]any)["changes"].(map[string]any)
	changes["streams"] = nativeStreamChanges(routes)
	payload, err := json.Marshal(combined)
	if err != nil {
		t.Fatal(err)
	}

	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, string(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.GetStreams()[0].GetConfiguration().GetPath()) != 1 {
		t.Fatal("CBS+path response lost its route")
	}
	if findQueueInTest(t, got, "SW1", "P4", 5).GetCbs()[0].GetAdminIdleSlopeBps() != 12000000 {
		t.Fatal("CBS+path response lost its CBS queue update")
	}
	if !proto.Equal(original, before) {
		t.Fatal("CBS+path projection mutated the original")
	}
	if !proto.Equal(got.Streams[1], original.Streams[1]) {
		t.Fatal("CBS+path projection modified an unrelated stream")
	}
}

func TestApplyValidatedOptimizerResponseGeneratedPathCreatesMissingConfiguration(t *testing.T) {
	original := nativeRouteApplyOriginal()
	original.Streams[0].Configuration = nil
	before := proto.Clone(original)
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, nativeGeneratedRouteFixture))
	if err != nil {
		t.Fatal(err)
	}
	cfg := got.GetStreams()[0].GetConfiguration()
	if cfg == nil || len(cfg.GetPath()) != 1 ||
		!proto.Equal(cfg.GetStreamId(), original.GetStreams()[0].GetDefinition().GetStreamId()) {
		t.Fatal("new native stream configuration must carry original composite identity and path")
	}
	if !proto.Equal(original, before) {
		t.Fatal("created stream configuration changed original")
	}
}

func TestApplyValidatedOptimizerResponseGeneratedPathWithEmptyCBSSection(t *testing.T) {
	original := nativeRouteApplyOriginal()
	payload := nativeModifyRoutePayload(t, func(result map[string]any) {
		result["solution"].(map[string]any)["changes"].(map[string]any)["configuration"] = map[string]any{}
	})
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, payload))
	if err != nil || got == nil || len(got.Streams[0].GetConfiguration().GetPath()) != 1 {
		t.Fatalf("empty CBS section with a valid path should be supported: model=%v err=%v", got, err)
	}
}

func TestApplyValidatedOptimizerResponseRejectsInvalidGeneratedPathWithoutMutation(t *testing.T) {
	cases := []struct {
		name           string
		alter          func(map[string]any)
		adjustOriginal func(*forwardingplane.ForwardingPlaneModel)
		expect         string
	}{
		{"unknown stream", func(result map[string]any) {
			nativeStreamChanges(result)[0].(map[string]any)["streamId"].(map[string]any)["uniqueId"] = 88
		}, nil, "unknown original stream"},
		{"duplicate update", func(result map[string]any) {
			changes := result["solution"].(map[string]any)["changes"].(map[string]any)
			entry := nativeStreamChanges(result)[0]
			changes["streams"] = []any{entry, entry}
		}, nil, "duplicate route update"},
		{"missing uniqueId", func(result map[string]any) {
			delete(nativeStreamChanges(result)[0].(map[string]any)["streamId"].(map[string]any), "uniqueId")
		}, nil, "requires composite streamId"},
		{"empty path", func(result map[string]any) {
			nativeStreamChanges(result)[0].(map[string]any)["configuration"].(map[string]any)["path"] = []any{}
		}, nil, "nonempty configuration.path"},
		{"wrong hop number", func(result map[string]any) {
			hops := nativeStreamChanges(result)[0].(map[string]any)["configuration"].(map[string]any)["path"].([]any)
			hops[0].(map[string]any)["hopNumber"] = 5
		}, nil, "invalid hop"},
		{"unsupported stream field", func(result map[string]any) {
			nativeStreamChanges(result)[0].(map[string]any)["definition"] = map[string]any{"trafficClassId": "OTHER"}
		}, nil, "unknown field"},
		{"unsupported route config field", func(result map[string]any) {
			nativeStreamChanges(result)[0].(map[string]any)["configuration"].(map[string]any)["pcp"] = 2
		}, nil, "unknown field"},
		{"route already exists", nil, func(original *forwardingplane.ForwardingPlaneModel) {
			original.Streams[0].Configuration.Path = []*streamconfig.StreamHop{{BridgeId: "SW1", IngressPortId: "P1", EgressPortId: "P4"}}
		}, "overwrite an existing path"},
		{"configuration stream identity mismatch", nil, func(original *forwardingplane.ForwardingPlaneModel) {
			original.Streams[0].Configuration.StreamId.UniqueId = 99
		}, "configuration identity differs"},
		{"duplicate original identity", nil, func(original *forwardingplane.ForwardingPlaneModel) {
			original.Streams = append(original.Streams, proto.Clone(original.Streams[0]).(*forwardingplane.StreamModel))
		}, "duplicate original stream identity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := nativeRouteApplyOriginal()
			if tc.adjustOriginal != nil {
				tc.adjustOriginal(original)
			}
			payload := nativeGeneratedRouteFixture
			if tc.alter != nil {
				payload = nativeModifyRoutePayload(t, tc.alter)
			}
			before := proto.Clone(original)
			got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, payload))
			if err == nil || got != nil || !strings.Contains(err.Error(), tc.expect) {
				t.Fatalf("expected rejection containing %q; model=%v err=%v", tc.expect, got, err)
			}
			if !proto.Equal(original, before) {
				t.Fatal("rejected update mutated original model")
			}
		})
	}
}

func TestApplyValidatedOptimizerResponseAtomicOnBadCBSAfterGoodRoute(t *testing.T) {
	original := nativeRouteApplyOriginal()
	before := proto.Clone(original)
	var combined map[string]any
	var route map[string]any
	if err := json.Unmarshal([]byte(cbsUpdateFixture), &combined); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(nativeGeneratedRouteFixture), &route); err != nil {
		t.Fatal(err)
	}
	changes := combined["solution"].(map[string]any)["changes"].(map[string]any)
	changes["streams"] = nativeStreamChanges(route)
	config := changes["configuration"].(map[string]any)
	config["nodeConfigs"].([]any)[0].(map[string]any)["nodeId"] = "unknown-node"
	payload, err := json.Marshal(combined)
	if err != nil {
		t.Fatal(err)
	}
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, string(payload)))
	if err == nil || got != nil || !strings.Contains(err.Error(), "unknown physical node") {
		t.Fatalf("failed CBS delta after valid route must reject full result: model=%v err=%v", got, err)
	}
	if !proto.Equal(original, before) {
		t.Fatal("bad combined update mutated preserved original")
	}
}

// Initial configuration may have a topology without a topology-config object.
// A validated generated route only changes the stream's path, not CBS queues.
func TestApplyValidatedOptimizerResponseRouteOnlyWithoutTopologyConfiguration(t *testing.T) {
	original := nativeRouteApplyOriginal()
	original.Configuration = nil
	before := proto.Clone(original)
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, nativeGeneratedRouteFixture))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.GetConfiguration() != nil ||
		len(got.GetStreams()[0].GetConfiguration().GetPath()) != 1 {
		t.Fatal("route-only projection should work without original topology configuration")
	}
	if !proto.Equal(original, before) {
		t.Fatal("route-only projection mutated preserved original")
	}
}

func TestApplyValidatedOptimizerResponseStillRejectsCBSQueuesWithoutTopologyConfiguration(t *testing.T) {
	original := nativeRouteApplyOriginal()
	original.Configuration = nil
	before := proto.Clone(original)
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, cbsUpdateFixture))
	if err == nil || got != nil || !strings.Contains(err.Error(), "CBS queue updates require preserved topology configuration") {
		t.Fatalf("CBS update without topology configuration must fail closed: model=%v err=%v", got, err)
	}
	if !proto.Equal(original, before) {
		t.Fatal("rejected CBS update mutated preserved original")
	}
}
