package internalOptimizer

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	streamconfig "OpenCNC/common/structures/stream_config"
	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"OpenCNC/common/structures/qbv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// These tests exercise the actual shared v0.2.5 validation path with a
// preserved OpenCNC snapshot, followed by native application. They do NOT
// exercise a live OaaS Manager, CBS computation, or hardware deployment.
func boundaryRequestAndOriginal(t *testing.T) ([]byte, *forwardingplane.ForwardingPlaneModel, []map[string]any) {
	t.Helper()
	// Reuse the existing CNC fixture converter rather than inventing another
	// topology/protobuf representation. It has ES1 -> SW1 -> SW2 -> ES2.
	_, original := originalRequestAndSnapshotForResponseTest(t)
	if len(original.GetStreams()) != 2 ||
		original.GetStreams()[0].GetConfiguration() == nil ||
		len(original.GetStreams()[0].GetConfiguration().GetPath()) != 2 {
		t.Fatal("expected existing two-stream, two-hop CNC fixture")
	}

	// Capture the same route in response format, then clear it only from the
	// first stream. The second stream remains fixed and must be preserved.
	path := make([]map[string]any, 0, len(original.Streams[0].Configuration.Path))
	for _, hop := range original.Streams[0].Configuration.Path {
		path = append(path, map[string]any{
			"hopNumber":     hop.GetHopNumber(),
			"bridgeId":      hop.GetBridgeId(),
			"ingressPortId": hop.GetIngressPortId(),
			"egressPortId":  hop.GetEgressPortId(),
		})
	}
	original.Streams[0].Configuration.Path = nil

	fixture, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}
	var cncInput struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
	}
	if err := json.Unmarshal(fixture, &cncInput); err != nil {
		t.Fatal(err)
	}
	var task optimizationcontract.OptimizationTask
	if err := protojson.Unmarshal(cncInput.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	firstID := original.GetStreams()[0].GetDefinition().GetStreamId()
	key := requestadapter.StreamKey(firstID.GetMacAddress(), firstID.GetUniqueId())
	requestFPM, err := withDefaultPCPQueueMappings(original)
	if err != nil {
		t.Fatal(err)
	}

	request, err := requestadapter.BuildOptimizerRequest(
		&task, requestFPM, requestadapter.SchemaJSON(),
		requestadapter.RequestOptions{
			CNCQueuePriorityOrder: true,
			Mechanisms:            []requestadapter.MechanismSelection{{Name: "CBS"}},
			Streams: map[string]requestadapter.StreamOptions{
				key: {Routing: "optimize", Scheduling: "optimize", Placement: "fixed"},
			},
		},
	)
	if err != nil {
		t.Fatalf("build actual v1.4 request for pathless stream: %v", err)
	}
	return request, original, path
}

func boundaryResponse(t *testing.T, original *forwardingplane.ForwardingPlaneModel, path []map[string]any, withCBS bool) []byte {
	t.Helper()
	identity := original.GetStreams()[0].GetDefinition().GetStreamId()
	changes := map[string]any{
		"streams": []any{map[string]any{
			"streamId": map[string]any{
				"macAddress": identity.GetMacAddress(),
				"uniqueId":   identity.GetUniqueId(),
			},
			"configuration": map[string]any{"path": path},
		}},
	}
	if withCBS {
		changes["configuration"] = map[string]any{
			"nodeConfigs": []any{map[string]any{
				"nodeId": "SW1",
				"portConfigs": []any{map[string]any{
					"portId": "s1-out",
					"queueConfigs": []any{map[string]any{
						"queueId":    3,
						"scheduling": "CREDIT_BASED_SHAPER",
						"cbs": []any{map[string]any{
							"egressQueueId":     3,
							"adminState":        "ENABLED",
							"adminIdleSlopeBps": 12000000,
							"sendSlopeBps":      88000000,
							"hiCreditBytes":     26,
							"loCreditBytes":     175,
						}},
					}},
				}},
			}},
		}
	}
	result := map[string]any{
		"schemaVersion":    "1.3",
		"optimizer":        map[string]any{"id": "CBS", "name": "CBS", "version": "unversioned"},
		"optimizedAgainst": nil,
		"status":           "success",
		"solution": map[string]any{
			"type": "configuration", "representation": "partial", "changes": changes,
		},
		"outcome": map[string]any{
			"feasibility": "feasible", "optimality": "notProven", "configurationChanged": true,
		},
		"kpis": map[string]any{}, "warnings": []any{}, "errors": []any{},
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestBoundaryRealSharedValidationThenNativeGeneratedRoute(t *testing.T) {
	useFakeSnapshotStore(t)
	request, original, path := boundaryRequestAndOriginal(t)
	before := proto.Clone(original)
	const jobID = "job-boundary-route-only"
	saveResponseContextForTest(t, jobID, request, original)

	validated, err := validateRetrievedOptimizerPayload(jobID, boundaryResponse(t, original, path, false))
	if err != nil {
		t.Fatalf("actual shared v0.2.5 validation rejected generated route: %v", err)
	}
	if validated.Validation == nil || !validated.Validation.Applicable {
		t.Fatalf("valid route response should be semantically applicable: %+v", validated.Validation)
	}
	// Shared JSON-level validation must not pretend CNC application has been
	// verified; that is the responsibility of the native boundary below.
	foundPending := false
	for _, issue := range validated.Validation.NotVerified {
		if issue.Validator == "RESP-ROUTE-02" {
			foundPending = true
		}
	}
	if !foundPending {
		t.Fatal("expected shared report to retain native route-application NOT_VERIFIED caveat")
	}
	updated, err := applyValidatedOptimizerResponse(validated)
	if err != nil {
		t.Fatalf("native path projection after shared validation: %v", err)
	}
	if updated == nil || updated == original {
		t.Fatal("native result should be an independent updated model")
	}
	expected := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)
	// A two-hop route should be inserted while every other native field stays.
	for _, v := range path {
		expected.Streams[0].Configuration.Path = append(expected.Streams[0].Configuration.Path,
			&streamconfig.StreamHop{
				HopNumber: v["hopNumber"].(uint32), BridgeId: v["bridgeId"].(string),
				IngressPortId: v["ingressPortId"].(string), EgressPortId: v["egressPortId"].(string),
			})
	}
	if !proto.Equal(updated, expected) {
		t.Fatal("native result differs from the preserved FPM beyond its generated stream path")
	}
	if !proto.Equal(original, before) {
		t.Fatal("validation or application mutated the preserved original FPM")
	}
}

func TestBoundaryRealSharedValidationThenCombinedCBSAndRoute(t *testing.T) {
	useFakeSnapshotStore(t)
	request, original, path := boundaryRequestAndOriginal(t)
	before := proto.Clone(original)
	const jobID = "job-boundary-cbs-route"
	saveResponseContextForTest(t, jobID, request, original)
	validated, err := validateRetrievedOptimizerPayload(jobID, boundaryResponse(t, original, path, true))
	if err != nil {
		t.Fatalf("actual shared validation rejected combined response: %v", err)
	}
	if validated.Validation == nil || !validated.Validation.Applicable {
		t.Fatal("combined response should be semantically applicable")
	}
	updated, err := applyValidatedOptimizerResponse(validated)
	if err != nil {
		t.Fatalf("native CBS+route application: %v", err)
	}
	if len(updated.GetStreams()[0].GetConfiguration().GetPath()) != len(path) {
		t.Fatal("combined response lost the generated path")
	}
	if findQueueInTest(t, updated, "SW1", "s1-out", 3).GetCbs()[0].GetAdminIdleSlopeBps() != 12000000 {
		t.Fatal("combined response lost CBS queue values")
	}
	if !proto.Equal(updated.GetStreams()[1], original.GetStreams()[1]) || !proto.Equal(original, before) {
		t.Fatal("unrelated stream or original model modified")
	}
}

func TestBoundaryRealSharedValidationRejectsBrokenRouteBeforeApplication(t *testing.T) {
	useFakeSnapshotStore(t)
	request, original, path := boundaryRequestAndOriginal(t)
	before := proto.Clone(original)
	const jobID = "job-boundary-invalid-route"
	saveResponseContextForTest(t, jobID, request, original)
	path[0]["ingressPortId"] = "nonexistent-port"
	validated, err := validateRetrievedOptimizerPayload(jobID, boundaryResponse(t, original, path, false))
	if err == nil || validated != nil || !strings.Contains(err.Error(), "Route segment") {
		t.Fatalf("invalid route must fail shared validation before native apply: validated=%v, err=%v", validated, err)
	}
	if !proto.Equal(original, before) {
		t.Fatal("rejected response changed original model")
	}
}

func TestBoundaryRealSharedValidationRejectsMissingRouteNoOp(t *testing.T) {
	useFakeSnapshotStore(t)
	request, original, _ := boundaryRequestAndOriginal(t)
	const jobID = "job-boundary-missing-route"
	saveResponseContextForTest(t, jobID, request, original)
	validated, err := validateRetrievedOptimizerPayload(jobID, []byte(successfulNoChangeCBSResponse))
	if err == nil || validated != nil || !strings.Contains(err.Error(), "Missing optimize-route") {
		t.Fatalf("successful no-op must not skip missing generated route: validated=%v, err=%v", validated, err)
	}
}

func TestBoundaryRealSharedValidationThenTASApplication(t *testing.T) {
	useFakeSnapshotStore(t)

	_, original := originalRequestAndSnapshotForResponseTest(t)

	t.Setenv("OAAS_MECHANISM", "TAS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false")
	t.Setenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES", "")

	// Declare TAS support for the synthetic test topology.
	for _, node := range original.GetTopology().GetNodes() {
		for _, port := range node.GetPorts() {
			if caps := port.GetCapabilities(); caps != nil {
				caps.SupportsTas = true
				caps.SupportsTimeSync = true
			}
		}
	}

	// Reuse the existing CNC task fixture.
	data, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}

	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
	}

	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	var task optimizationcontract.OptimizationTask
	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}

	before := proto.Clone(original)

	request, err := buildOaaSRequest(&task, original)
	if err != nil {
		t.Fatalf("build TAS request: %v", err)
	}

	const jobID = "job-boundary-tas"
	saveResponseContextForTest(t, jobID, request, original)

	payload := `{
        "schemaVersion": "1.3",
        "optimizer": {
            "id": "test-tas",
            "name": "TAS",
            "version": "test"
        },
        "optimizedAgainst": null,
        "status": "success",
        "solution": {
            "type": "configuration",
            "representation": "partial",
            "changes": {
                "configuration": {
                    "nodeConfigs": [{
                        "nodeId": "SW1",
                        "portConfigs": [{
                            "portId": "s1-out",
                            "gcl": {
                                "scheduleId": "tas-test",
                                "cycleTime": 1000000,
                                "adminState": "ENABLED",
                                "entries": [
                                    {
                                        "index": 0,
                                        "startTimeNs": 0,
                                        "timeInterval": 10000,
                                        "gateStates": "CA=="
                                    },
                                    {
                                        "index": 1,
                                        "startTimeNs": 10000,
                                        "timeInterval": 990000,
                                        "gateStates": "AQ=="
                                    }
                                ]
                            }
                        }]
                    }]
                },
                "streams": [{
                    "streamId": {
                        "macAddress": "02:00:00:00:00:01",
                        "uniqueId": 1
                    },
                    "configuration": {
                        "timeAwareOffsetNs": 0
                    }
                }]
            }
        },
        "outcome": {
            "feasibility": "feasible",
            "optimality": "notProven",
            "configurationChanged": true
        },
        "kpis": {},
        "warnings": [],
        "errors": []
    }`

	// Actual shared contract validation.
	validated, err := validateRetrievedOptimizerPayload(
		jobID, []byte(payload),
	)
	if err != nil {
		t.Fatalf("shared TAS validation failed: %v", err)
	}

	if validated.Validation == nil || !validated.Validation.Applicable {
		t.Fatal("TAS response must be applicable")
	}

	// Native response application.
	updated, err := applyValidatedOptimizerResponse(validated)
	if err != nil {
		t.Fatalf("native TAS application failed: %v", err)
	}

	expected := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)

	expected.Streams[0].Configuration.TimeAwareOffsetNs = proto.Uint64(0)

	for _, node := range expected.Configuration.NodeConfigs {
		if node.GetNodeId() != "SW1" {
			continue
		}

		for _, port := range node.PortConfigs {
			if port.GetPortId() == "s1-out" {
				port.Gcl = &qbv.GateControlList{
					ScheduleId: "tas-test",
					CycleTime:  1000000,
					AdminState: qbv.AdminState_ENABLED,
					Entries: []*qbv.GateControlEntry{
						{
							Index:        0,
							TimeInterval: 10000,
							GateStates:   []byte{8},
						},
						{
							Index:        1,
							TimeInterval: 990000,
							GateStates:   []byte{1},
						},
					},
				}
			}
		}
	}

	if !proto.Equal(updated, expected) {
		t.Fatal("incorrect TAS mapping or unrelated fields modified")
	}

	if !proto.Equal(original, before) ||
		!proto.Equal(validated.Original, before) {
		t.Fatal("preserved original model was modified")
	}
}
