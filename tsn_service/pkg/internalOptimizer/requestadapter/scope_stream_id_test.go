package requestadapter

import (
	"encoding/json"
	"reflect"
	"testing"

	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

func TestB10MapsScopeStreamIDToContractDefinition(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	id := model.GetStreams()[0].GetDefinition().GetStreamId()
	cncKey := id.AsKey()
	contractKey := StreamKey(id.GetMacAddress(), id.GetUniqueId())
	if cncKey == contractKey {
		t.Fatalf("test needs distinct CNC/contract keys; got %q", cncKey)
	}

	task := &optimizationcontract.OptimizationTask{
		TaskId:  "b10-scope-test",
		Intent:  optimizationcontract.Intent_INITIAL_CONFIGURATION,
		Trigger: optimizationcontract.Trigger_USER_REQUEST,
		Scope:   &optimizationcontract.Scope{StreamIds: []string{cncKey}},
	}
	data, err := BuildOptimizerRequest(task, model, loadV13Schema(t), RequestOptions{
		Mechanisms: []MechanismSelection{{Name: "CBS"}},
		Streams: map[string]StreamOptions{
			contractKey: {Routing: "optimize"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		OptimizationTask struct {
			Scope struct {
				StreamIDs []string `json:"streamIds"`
			} `json:"scope"`
		} `json:"optimizationTask"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.OptimizationTask.Scope.StreamIDs, []string{contractKey}) {
		t.Fatalf("scope IDs do not match contract stream definitions: %q", result.OptimizationTask.Scope.StreamIDs)
	}
	if !reflect.DeepEqual(task.GetScope().GetStreamIds(), []string{cncKey}) {
		t.Fatal("original OpenCNC task was modified")
	}
}

func TestB10ScopeMappingHyphenMACAndUnknownID(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	id := model.GetStreams()[0].GetDefinition().GetStreamId()
	id.MacAddress = "00-00-00-00-00-01"
	id.UniqueId = 101

	original := id.AsKey() // 00-00-00-00-00-01-101
	taskJSON := map[string]any{
		"scope": map[string]any{"streamIds": []string{original, "unknown-stream"}},
	}
	normalizeTaskScopeStreamIDs(taskJSON, model)
	got := taskJSON["scope"].(map[string]any)["streamIds"].([]string)
	want := []string{"00-00-00-00-00-01/101", "unknown-stream"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got scope %q, want %q", got, want)
	}
}
