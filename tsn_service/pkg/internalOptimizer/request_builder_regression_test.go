package internalOptimizer

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"OpenCNC/common/structures/devicemodelregistry"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestRequestJSONUnchangedWithDeviceModels(t *testing.T) {
	t.Setenv("OAAS_MECHANISM", "CBS")

	data, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}

	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
		ForwardingPlane  json.RawMessage `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	// Match the StreamRank conversion used in live_manager_test.go.
	var plane map[string]any
	if err := json.Unmarshal(fixture.ForwardingPlane, &plane); err != nil {
		t.Fatal(err)
	}
	streams, ok := plane["streams"].([]any)
	if !ok {
		t.Fatal("fixture has no streams array")
	}
	for _, raw := range streams {
		def := raw.(map[string]any)["definition"].(map[string]any)
		if rank, ok := def["streamRank"].(map[string]any); ok {
			def["streamRank"] = rank["rank"]
		}
	}

	planeJSON, err := json.Marshal(plane)
	if err != nil {
		t.Fatal(err)
	}

	var task optimizer.OptimizationTask
	var fpm forwardingplane.ForwardingPlaneModel

	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(planeJSON, &fpm); err != nil {
		t.Fatal(err)
	}

	nodes := fpm.GetTopology().GetNodes()
	if len(nodes) == 0 || nodes[0] == nil {
		t.Fatal("fixture has no valid topology node")
	}

	models := map[string]*devicemodelregistry.DeviceModel{
		nodes[0].GetName(): {
			Limitations: []*devicemodelregistry.DeviceLimitation{
				{
					TargetPath: "/interfaces/interface/gate-parameter-table/admin-control-list/gate-control-entry",
					Parameter: &devicemodelregistry.LimitationParameter{
						Name:       "length",
						Unit:       "entries",
						TargetKind: devicemodelregistry.TargetKind_YANG_NODE_KIND_LIST,
					},
					Comparison: devicemodelregistry.LimitationComparison_LIMITATION_LESS_OR_EQUAL,
					Threshold:  512,
				},
			},
		},
	}

	before, err := buildOaaSRequest(&task, &fpm)
	if err != nil {
		t.Fatal(err)
	}

	after, err := buildOaaSRequestWithDeviceModels(&task, &fpm, models)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(before, after) {
		t.Fatal("Contract v1.4 JSON unexpectedly changed")
	}

	if os.Getenv("OAAS_EXPORT_JSON") == "1" {
		cncTask, err := protojson.Marshal(&task)
		if err != nil {
			t.Fatal(err)
		}

		cncPlane, err := protojson.Marshal(&fpm)
		if err != nil {
			t.Fatal(err)
		}

		cncJSON, err := json.MarshalIndent(
			map[string]json.RawMessage{
				"optimizationTask": cncTask,
				"forwardingPlane":  cncPlane,
			}, "", "  ",
		)
		if err != nil {
			t.Fatal(err)
		}

		dir, err := os.MkdirTemp("", "oaas-json-review-")
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(dir+"/cnc_input.json", cncJSON, 0600); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(dir+"/optimizer_request_v1.4.json", after, 0600); err != nil {
			t.Fatal(err)
		}

		t.Logf("JSON files saved in: %s", dir)
	}

}
