package internalOptimizer

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	streamconfig "OpenCNC/common/structures/stream_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

// Native application regression, not end-to-end proof. The shared contract
// has a separate validator; validatedFixture deliberately assumes its success.
func stagedTwoStreamOriginal() *forwardingplane.ForwardingPlaneModel {
	original := nativeRouteApplyOriginal()
	second := original.Streams[1]
	second.Definition.TalkerNodeId = "ES1"
	second.Definition.ListenerNodeIds = []string{"ES4"}
	second.Definition.TrafficClassId = "M1"
	second.Configuration = &streamconfig.StreamConfiguration{Description: "keep-second"}
	unrelated := proto.Clone(second).(*forwardingplane.StreamModel)
	unrelated.Definition.StreamId.MacAddress = "02:00:00:00:00:03"
	unrelated.Definition.StreamId.UniqueId = 9
	unrelated.Definition.Description = "unrelated-existing"
	unrelated.Configuration.Description = "untouched"
	original.Streams = append(original.Streams, unrelated)
	return original
}

func stagedTwoStreamResponse() map[string]any {
	entry := func(mac string, unique int, pcp int, enabled bool) map[string]any {
		return map[string]any{
			"definition": map[string]any{
				"id":           mac + "/" + strconv.Itoa(unique),
				"streamId":     map[string]any{"macAddress": mac, "uniqueId": unique},
				"talkerNodeId": "ES1", "listenerNodeIds": []string{"ES4"},
				"trafficClassId":            "M1",
				"trafficSpec":               map[string]any{"maxFrameSize": 128, "intervalNs": 1000000, "maxIntervalFrames": 1},
				"userToNetworkRequirements": map[string]any{"maxLatencyNs": 1000000},
			},
			"configuration": map[string]any{
				"pcp": pcp, "enabled": enabled, "queueAssignments": []any{},
				"path": []any{map[string]any{
					"hopNumber": 0, "bridgeId": "SW1", "ingressPortId": "P1", "egressPortId": "P4",
				}},
			},
		}
	}
	return map[string]any{
		"status": "success",
		"solution": map[string]any{"changes": map[string]any{
			"streamAdditions": []any{
				entry("02:00:00:00:00:01", 7, 5, true),
				entry("02:00:00:00:00:02", 8, 7, false),
			},
		}},
	}
}

func stagedAdditionChanges(result map[string]any) map[string]any {
	return result["solution"].(map[string]any)["changes"].(map[string]any)
}
func stagedAdditionItems(result map[string]any) []any {
	return stagedAdditionChanges(result)["streamAdditions"].([]any)
}
func stagedAdditionConfig(result map[string]any, index int) map[string]any {
	return stagedAdditionItems(result)[index].(map[string]any)["configuration"].(map[string]any)
}
func stagedAdditionDefinition(result map[string]any, index int) map[string]any {
	return stagedAdditionItems(result)[index].(map[string]any)["definition"].(map[string]any)
}
func marshalStagedResponse(t *testing.T, result map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func TestApplyStagedAdditionsTwoStreamsNoDuplicatesAndPreservation(t *testing.T) {
	original := stagedTwoStreamOriginal()
	before := proto.Clone(original)
	payload := marshalStagedResponse(t, stagedTwoStreamResponse())
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, payload))
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(before, original) {
		t.Fatal("application modified preserved original")
	}
	if len(got.Streams) != len(original.Streams) {
		t.Fatal("staged additions were duplicated or dropped")
	}

	expected := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)
	for _, index := range []int{0, 1} {
		expected.Streams[index].Configuration.Path = []*streamconfig.StreamHop{{
			HopNumber: 0, BridgeId: "SW1", IngressPortId: "P1", EgressPortId: "P4",
		}}
	}
	expectedPCP := uint32(7)
	expected.Streams[1].Configuration.Pcp = &expectedPCP
	if !proto.Equal(got, expected) {
		t.Fatal("updated CNC model differs from expected path/PCP-only reconciliation")
	}
	if !proto.Equal(got.Streams[2], original.Streams[2]) {
		t.Fatal("unrelated existing stream was modified")
	}
}

func TestApplyStagedAdditionsCreatesMissingNativeConfiguration(t *testing.T) {
	original := stagedTwoStreamOriginal()
	original.Streams[1].Configuration = nil
	before := proto.Clone(original)
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, marshalStagedResponse(t, stagedTwoStreamResponse())))
	if err != nil {
		t.Fatal(err)
	}
	cfg := got.GetStreams()[1].GetConfiguration()
	if cfg == nil || cfg.GetPcp() != 7 || len(cfg.GetPath()) != 1 ||
		!proto.Equal(cfg.GetStreamId(), original.GetStreams()[1].GetDefinition().GetStreamId()) {
		t.Fatal("new configuration must include the validated PCP, path and composite identity")
	}
	if !proto.Equal(original, before) {
		t.Fatal("original mutated")
	}
}

func TestApplyStagedAdditionsCombinedWithCBS(t *testing.T) {
	original := stagedTwoStreamOriginal()
	before := proto.Clone(original)
	result := stagedTwoStreamResponse()
	var cbs map[string]any
	if err := json.Unmarshal([]byte(cbsUpdateFixture), &cbs); err != nil {
		t.Fatal(err)
	}
	stagedAdditionChanges(result)["configuration"] = stagedAdditionChanges(cbs)["configuration"]
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, marshalStagedResponse(t, result)))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Streams) != len(original.Streams) || len(got.Streams[0].GetConfiguration().GetPath()) != 1 ||
		len(got.Streams[1].GetConfiguration().GetPath()) != 1 {
		t.Fatal("CBS plus additions lost paths or duplicated streams")
	}
	if got.Streams[1].GetConfiguration().GetPcp() != 7 {
		t.Fatal("request-only PCP overlay lost")
	}
	if got.Configuration == nil || findQueueInTest(t, got, "SW1", "P4", 5).GetCbs()[0].GetAdminIdleSlopeBps() != 12000000 {
		t.Fatal("CBS updates lost")
	}
	if !proto.Equal(before, original) {
		t.Fatal("combined application mutated original")
	}
}

func TestApplyStagedAdditionsRejectsInvalidWithoutMutation(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(map[string]any, *forwardingplane.ForwardingPlaneModel)
		expected string
	}{
		{"duplicate response addition", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			items := stagedAdditionItems(r)
			stagedAdditionChanges(r)["streamAdditions"] = append(items, items[0])
		}, "duplicate stream addition"},
		{"unknown original identity", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			stagedAdditionDefinition(r, 1)["streamId"].(map[string]any)["uniqueId"] = 99
		}, "was not staged"},
		{"duplicate original identity", func(_ map[string]any, fpm *forwardingplane.ForwardingPlaneModel) {
			fpm.Streams = append(fpm.Streams, proto.Clone(fpm.Streams[0]).(*forwardingplane.StreamModel))
		}, "duplicate staged original"},
		{"existing route overwritten", func(_ map[string]any, fpm *forwardingplane.ForwardingPlaneModel) {
			fpm.Streams[0].Configuration.Path = []*streamconfig.StreamHop{{HopNumber: 0, BridgeId: "SW1", IngressPortId: "P1", EgressPortId: "P4"}}
		}, "overwrite an existing path"},
		{"conflicting original PCP", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			stagedAdditionConfig(r, 0)["pcp"] = 6
		}, "conflicts with preserved PCP"},
		{"out of range PCP", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			stagedAdditionConfig(r, 1)["pcp"] = 9
		}, "invalid PCP"},
		{"unexpected configuration field", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			stagedAdditionConfig(r, 1)["vlanId"] = 100
		}, "unsupported configuration field"},
		{"missing path", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			delete(stagedAdditionConfig(r, 0), "path")
		}, "lacks generated path"},
		{"incorrect native hop", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			hops := stagedAdditionConfig(r, 0)["path"].([]any)
			hops[0].(map[string]any)["hopNumber"] = 7
		}, "invalid hop"},
		{"mismatched native endpoint", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			stagedAdditionDefinition(r, 0)["talkerNodeId"] = "other"
		}, "would alter talker endpoint"},
		{"unsupported logical placement", func(r map[string]any, _ *forwardingplane.ForwardingPlaneModel) {
			stagedAdditionDefinition(r, 0)["talkerApplicationId"] = "app1"
		}, "unsupported logical talker placement"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := stagedTwoStreamOriginal()
			result := stagedTwoStreamResponse()
			tc.mutate(result, original)
			before := proto.Clone(original)
			got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, marshalStagedResponse(t, result)))
			if err == nil || got != nil || !strings.Contains(err.Error(), tc.expected) {
				t.Fatalf("expected error containing %q, got model=%v err=%v", tc.expected, got, err)
			}
			if !proto.Equal(original, before) {
				t.Fatal("failed application mutated preserved original")
			}
		})
	}
}

func TestApplyStagedAdditionsAtomicOnBadCBSAfterRoutes(t *testing.T) {
	original := stagedTwoStreamOriginal()
	before := proto.Clone(original)
	result := stagedTwoStreamResponse()
	var cbs map[string]any
	if err := json.Unmarshal([]byte(cbsUpdateFixture), &cbs); err != nil {
		t.Fatal(err)
	}
	config := stagedAdditionChanges(cbs)["configuration"].(map[string]any)
	config["nodeConfigs"].([]any)[0].(map[string]any)["nodeId"] = "not-a-node"
	stagedAdditionChanges(result)["configuration"] = config
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, marshalStagedResponse(t, result)))
	if err == nil || got != nil || !strings.Contains(err.Error(), "unknown physical node") {
		t.Fatalf("expected atomic rejection; result=%v err=%v", got, err)
	}
	if !proto.Equal(before, original) {
		t.Fatal("bad CBS after routes modified original")
	}
}
