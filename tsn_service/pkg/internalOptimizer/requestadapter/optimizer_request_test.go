package requestadapter

import (
	"encoding/json"
	"strings"
	"testing"

	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func TestBuildOptimizerRequestV13(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	task := &optimizationcontract.OptimizationTask{
		TaskId:  "task-001",
		Intent:  optimizationcontract.Intent_INITIAL_CONFIGURATION,
		Trigger: optimizationcontract.Trigger_USER_REQUEST,
	}
	key := StreamKey("02:00:00:00:00:01", 1)
	data, err := BuildOptimizerRequest(task, model, loadV13Schema(t), RequestOptions{
		Mechanisms:     []MechanismSelection{{Name: "CBS"}},
		TrafficClasses: []string{"M1"},
		Streams: map[string]StreamOptions{
			key: {Routing: "optimize", Scheduling: "optimize"},
		},
		AdditionalCapabilities: map[string]any{"devices": []any{
			map[string]any{"node": "SW1", "ports": map[string]any{
				"P2": map[string]any{"mechanisms": map[string]any{
					"CBS": map[string]any{"supportedQueues": []any{5, 6}},
				}},
			}},
		}},
		Metadata: map[string]any{"source": "OpenCNC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request["schemaVersion"] != "1.4" || request["operation"] != "configuration" {
		t.Fatalf("unexpected root fields: %s", data)
	}
	optTask := request["optimizationTask"].(map[string]any)
	if optTask["taskId"] != "task-001" {
		t.Fatal("task mapping missing")
	}
	if optTask["scope"].(map[string]any)["trafficClasses"].([]any)[0] != "M1" {
		t.Fatal("traffic class extension missing")
	}
	stream := request["forwardingPlane"].(map[string]any)["streams"].([]any)[0].(map[string]any)
	if stream["definition"].(map[string]any)["id"] != key {
		t.Fatal("stream ID not mapped")
	}
	caps := request["capabilities"].(map[string]any)["devices"].([]any)
	var supportedQueues bool
	for _, item := range caps {
		device := item.(map[string]any)
		if device["node"] != "SW1" {
			continue
		}
		ports := device["ports"].(map[string]any)
		cbsp2 := ports["P2"].(map[string]any)["mechanisms"].(map[string]any)["CBS"].(map[string]any)
		supportedQueues = len(cbsp2["supportedQueues"].([]any)) == 2
	}
	if !supportedQueues {
		t.Fatal("external capability inventory merge failed")
	}
}

func TestBuildRequestRejectsMissingMechanisms(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	_, err := BuildOptimizerRequest(&optimizationcontract.OptimizationTask{
		TaskId: "task-1", Intent: optimizationcontract.Intent_INITIAL_CONFIGURATION,
	}, model, loadV13Schema(t), RequestOptions{})
	if err == nil || !strings.Contains(err.Error(), "mechanism") {
		t.Fatalf("expected missing mechanism error, got %v", err)
	}
}

func TestBuildRequestRequiresChangePolicy(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	_, err := BuildOptimizerRequest(&optimizationcontract.OptimizationTask{
		TaskId: "task-2", Intent: optimizationcontract.Intent_ADD_STREAM,
	}, model, loadV13Schema(t), RequestOptions{
		Mechanisms: []MechanismSelection{{Name: "TAS"}},
	})
	if err == nil || !strings.Contains(err.Error(), "changePolicy") {
		t.Fatalf("expected missing changePolicy error, got %v", err)
	}
}

func TestBuildRequestRequiresFRERPolicy(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	_, err := BuildOptimizerRequest(&optimizationcontract.OptimizationTask{
		TaskId: "task-3", Intent: optimizationcontract.Intent_INITIAL_CONFIGURATION,
	}, model, loadV13Schema(t), RequestOptions{
		Mechanisms: []MechanismSelection{{Name: "FRER"}},
	})
	if err == nil || !strings.Contains(err.Error(), "reliability") {
		t.Fatalf("expected missing reliability error, got %v", err)
	}
}

func TestBuildRequestRejectsUnknownStreamOption(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	_, err := BuildOptimizerRequest(&optimizationcontract.OptimizationTask{
		TaskId: "task-4", Intent: optimizationcontract.Intent_INITIAL_CONFIGURATION,
	}, model, loadV13Schema(t), RequestOptions{
		Mechanisms: []MechanismSelection{{Name: "CBS"}},
		Streams:    map[string]StreamOptions{"f1": {Routing: "fixed"}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown derived stream ID") {
		t.Fatalf("expected stream ID error, got %v", err)
	}
}

func TestBuildRequestRejectsUnmappedScopeStream(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	_, err := BuildOptimizerRequest(&optimizationcontract.OptimizationTask{
		TaskId: "task-5", Intent: optimizationcontract.Intent_INITIAL_CONFIGURATION,
		Scope: &optimizationcontract.Scope{StreamIds: []string{"f1"}},
	}, model, loadV13Schema(t), RequestOptions{
		Mechanisms: []MechanismSelection{{Name: "CBS"}},
	})
	if err == nil || !strings.Contains(err.Error(), "scope.streamIds") {
		t.Fatalf("expected unknown CNC scope alias error, got %v", err)
	}
}

func TestBuildRequestRejectsPlacementWithoutApplications(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	key := StreamKey("02:00:00:00:00:01", 1)
	_, err := BuildOptimizerRequest(&optimizationcontract.OptimizationTask{
		TaskId: "task-6", Intent: optimizationcontract.Intent_INITIAL_CONFIGURATION,
	}, model, loadV13Schema(t), RequestOptions{
		Mechanisms: []MechanismSelection{{Name: "TAS"}},
		Streams: map[string]StreamOptions{
			key: {Routing: "optimize", Scheduling: "optimize", Placement: "optimize", TalkerApplicationID: "camera1", ListenerApplicationIDs: []string{"controller1"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "applications") {
		t.Fatalf("expected missing applications error, got %v", err)
	}
}

// p2Forwarding builds a minimal already-projected v1.4 stream for policy tests.
// This deliberately exercises the Go Bridge policy without needing a running CNC.
func p2Forwarding(id string, hasPath bool) map[string]any {
	stream := map[string]any{
		"definition":    map[string]any{"id": id},
		"configuration": map[string]any{},
	}
	if hasPath {
		stream["configuration"].(map[string]any)["path"] = []any{map[string]any{"hopNumber": 0}}
	}
	return map[string]any{"streams": []any{stream}}
}

func p2Decisions(t *testing.T, forwarding map[string]any) map[string]any {
	t.Helper()
	streams := forwarding["streams"].([]any)
	return streams[0].(map[string]any)["decisionVariables"].(map[string]any)
}

func TestP2DefaultsForExistingRoute(t *testing.T) {
	forwarding := p2Forwarding("02:00:00:00:00:01/1", true)
	if err := applyStreamOptions(forwarding, nil, "configuration"); err != nil {
		t.Fatal(err)
	}
	decisions := p2Decisions(t, forwarding)
	if decisions["routing"] != "fixed" || decisions["scheduling"] != "optimize" || decisions["placement"] != "fixed" {
		t.Fatalf("incorrect P2 defaults: %v", decisions)
	}
}

func TestP2RejectsDefaultFixedWithoutRoute(t *testing.T) {
	forwarding := p2Forwarding("02:00:00:00:00:01/1", false)
	err := applyStreamOptions(forwarding, nil, "configuration")
	if err == nil || !strings.Contains(err.Error(), "default fixed routing requires an existing path") {
		t.Fatalf("missing route should reject default fixed routing, got %v", err)
	}
}

func TestP2ExplicitRoutingOverrideAndPartialDefaults(t *testing.T) {
	id := "02:00:00:00:00:01/1"
	forwarding := p2Forwarding(id, false)
	if err := applyStreamOptions(forwarding, map[string]StreamOptions{
		id: {Routing: "optimize", Scheduling: "preserve"},
	}, "configuration"); err != nil {
		t.Fatal(err)
	}
	decisions := p2Decisions(t, forwarding)
	if decisions["routing"] != "optimize" || decisions["scheduling"] != "preserve" || decisions["placement"] != "fixed" {
		t.Fatalf("explicit options should override only supplied fields: %v", decisions)
	}
}

func TestP2UsesDefaultForAllStreams(t *testing.T) {
	id1, id2 := "02:00:00:00:00:01/1", "02:00:00:00:00:01/2"
	forwarding := p2Forwarding(id1, true)
	forwarding["streams"] = append(forwarding["streams"].([]any),
		map[string]any{"definition": map[string]any{"id": id2}, "configuration": map[string]any{}})
	if err := applyStreamOptions(forwarding, map[string]StreamOptions{
		id2: {Routing: "optimize"},
	}, "configuration"); err != nil {
		t.Fatal(err)
	}
	first := p2Decisions(t, forwarding)
	second := forwarding["streams"].([]any)[1].(map[string]any)["decisionVariables"].(map[string]any)
	if first["routing"] != "fixed" || second["routing"] != "optimize" || second["scheduling"] != "optimize" || second["placement"] != "fixed" {
		t.Fatalf("per-stream defaults or overrides incorrect: %v, %v", first, second)
	}
}

func TestP2UnknownStreamOptionRejectedBeforeMissingPath(t *testing.T) {
	forwarding := p2Forwarding("02:00:00:00:00:01/1", false)
	err := applyStreamOptions(forwarding, map[string]StreamOptions{
		"unknown": {Routing: "optimize"},
	}, "configuration")
	if err == nil || !strings.Contains(err.Error(), "unknown derived stream ID") {
		t.Fatalf("expected unknown stream ID rejection, got %v", err)
	}
}
