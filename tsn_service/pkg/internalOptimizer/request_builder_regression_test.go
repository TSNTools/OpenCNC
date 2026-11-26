package internalOptimizer

import (
	"OpenCNC/common/structures/devicemodelregistry"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestRequestJSONUnchangedWithDeviceModels(t *testing.T) {
	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false")

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
	t.Run("B1GCLCapabilities", func(t *testing.T) {
		// Synthetic limitation; this is not a measured hardware value.
		testFPM := proto.Clone(&fpm).(*forwardingplane.ForwardingPlaneModel)
		var nodeID, portID string
		for _, node := range testFPM.GetTopology().GetNodes() {
			if node.GetType().String() != "BRIDGE" {
				continue
			}
			for _, port := range node.GetPorts() {
				if port.GetCapabilities() == nil {
					continue
				}
				port.GetCapabilities().SupportsTas = true
				nodeID, portID = node.GetName(), port.GetId()
				break
			}
			if portID != "" {
				break
			}
		}
		if portID == "" {
			t.Fatal("fixture has no bridge port with capabilities")
		}
		original := proto.Clone(testFPM)
		baseline, err := buildOaaSRequest(&task, testFPM)
		if err != nil {
			t.Fatal(err)
		}
		mapped, err := buildOaaSRequestWithDeviceModels(&task, testFPM,
			map[string]*devicemodelregistry.DeviceModel{nodeID: b1SyntheticGCLModel(64)})
		if err != nil {
			t.Fatal(err)
		}
		var want, got map[string]any
		if err := json.Unmarshal(baseline, &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(mapped, &got); err != nil {
			t.Fatal(err)
		}
		// Add the expected numeric limit to all TAS-capable ports of this node.
		// Comparing the entire request also checks that existing flags and
		// unrelated fields remain unchanged.
		found := false
		for _, raw := range want["capabilities"].(map[string]any)["devices"].([]any) {
			device := raw.(map[string]any)
			if device["nodeId"] != nodeID {
				continue
			}
			for id, rawPort := range device["ports"].(map[string]any) {
				tas := rawPort.(map[string]any)["mechanisms"].(map[string]any)["TAS"].(map[string]any)
				if tas["supported"] == true {
					tas["maxGclEntries"] = float64(64)
					if id == portID {
						found = true
					}
				}
			}
		}
		if !found || !reflect.DeepEqual(want, got) {
			t.Fatal("request capability enrichment differs from expected mapping")
		}
		if !proto.Equal(original, testFPM) {
			t.Fatal("original FPM modified")
		}
		// Absent limitations must not fabricate a numeric hardware limit.
		empty, err := buildOaaSRequestWithDeviceModels(&task, testFPM,
			map[string]*devicemodelregistry.DeviceModel{nodeID: {}})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(baseline, empty) {
			t.Fatal("request unexpectedly changed with empty limitations")
		}
	})
	t.Run("B6ProductionDefaultDecisions", func(t *testing.T) {
		var request struct {
			ForwardingPlane struct {
				Streams []struct {
					DecisionVariables struct {
						Routing    string `json:"routing"`
						Scheduling string `json:"scheduling"`
						Placement  string `json:"placement"`
					} `json:"decisionVariables"`
				} `json:"streams"`
			} `json:"forwardingPlane"`
		}

		if err := json.Unmarshal(before, &request); err != nil {
			t.Fatal(err)
		}

		streams := request.ForwardingPlane.Streams
		if len(streams) == 0 {
			t.Fatal("expected at least one stream")
		}

		for i, stream := range streams {
			d := stream.DecisionVariables
			if d.Routing != "fixed" ||
				d.Scheduling != "optimize" ||
				d.Placement != "fixed" {
				t.Fatalf("stream %d: unexpected decisions: %+v", i, d)
			}
		}
	})

	// B10: The opt-in test flag must set routing=optimize for all fixture streams.
	// Run this BEFORE the deliberate duplicate-node mutation below.
	t.Run("B10OptimizeRoutingFlag", func(t *testing.T) {
		t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "true")

		result, err := buildOaaSRequest(&task, &fpm)
		if err != nil {
			t.Fatal(err)
		}

		var request struct {
			ForwardingPlane struct {
				Streams []struct {
					DecisionVariables struct {
						Routing    string `json:"routing"`
						Scheduling string `json:"scheduling"`
						Placement  string `json:"placement"`
					} `json:"decisionVariables"`
				} `json:"streams"`
			} `json:"forwardingPlane"`
		}

		if err := json.Unmarshal(result, &request); err != nil {
			t.Fatal(err)
		}

		if len(request.ForwardingPlane.Streams) == 0 {
			t.Fatal("expected streams")
		}

		for i, stream := range request.ForwardingPlane.Streams {
			d := stream.DecisionVariables
			if d.Routing != "optimize" ||
				d.Scheduling != "optimize" ||
				d.Placement != "fixed" {
				t.Fatalf("stream %d: unexpected B10 decisions: %+v", i, d)
			}
		}
	})

	t.Run("GenericTASRequest", func(t *testing.T) {
		t.Setenv("OAAS_MECHANISM", "TAS")
		t.Setenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES", "")
		t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false")

		for _, intent := range []optimizer.Intent{
			optimizer.Intent_INITIAL_CONFIGURATION,
			optimizer.Intent_ADD_STREAM,
		} {
			t.Run(intent.String(), func(t *testing.T) {
				testTask := proto.Clone(&task).(*optimizer.OptimizationTask)
				testTask.Intent = intent

				if intent == optimizer.Intent_ADD_STREAM {
					testTask.Scope.StreamIds = []string{
						"02:00:00:00:00:01/2",
					}

					testTask.ChangePolicy = &optimizer.ChangePolicy{
						AllowPathChanges:        true,
						AllowScheduleChanges:    true,
						AllowQueueChanges:       true,
						PreserveExistingStreams: true,
						MaxChangedStreams:       2,
						MaxConfigurationChanges: 10,
					}
				}

				originalTask := proto.Clone(testTask)
				originalFPM := proto.Clone(&fpm)

				result, err := buildOaaSRequest(testTask, &fpm)
				if err != nil {
					t.Fatal(err)
				}

				var request struct {
					OptimizationTask struct {
						Intent string `json:"intent"`
						Scope  struct {
							StreamIDs []string `json:"streamIds"`
						} `json:"scope"`
						ChangePolicy map[string]any `json:"changePolicy"`
					} `json:"optimizationTask"`
				}

				if err := json.Unmarshal(result, &request); err != nil {
					t.Fatal(err)
				}

				got := request.OptimizationTask

				if got.Intent != intent.String() {
					t.Fatal("optimization intent changed")
				}

				if !reflect.DeepEqual(
					got.Scope.StreamIDs,
					testTask.GetScope().GetStreamIds(),
				) {
					t.Fatal("optimization scope changed")
				}

				if intent == optimizer.Intent_ADD_STREAM {
					expectedPolicy := map[string]any{
						"allowPathChanges":        true,
						"allowScheduleChanges":    true,
						"allowQueueChanges":       true,
						"allowVlanChanges":        false,
						"allowStreamMigration":    false,
						"preserveExistingStreams": true,
						"maxChangedStreams":       float64(2),
						"maxConfigurationChanges": float64(10),
					}

					if !reflect.DeepEqual(got.ChangePolicy, expectedPolicy) {
						t.Fatalf("change policy modified: %+v", got.ChangePolicy)
					}
				}

				if !proto.Equal(testTask, originalTask) {
					t.Fatal("original task modified")
				}

				if !proto.Equal(&fpm, originalFPM) {
					t.Fatal("original FPM modified")
				}
			})
		}
	})
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

	// A duplicated CNC topology node must be rejected by shared validation.
	fpm.Topology.Nodes = append(fpm.Topology.Nodes, fpm.Topology.Nodes[0])

	_, err = buildOaaSRequest(&task, &fpm)
	if err == nil || (!strings.Contains(err.Error(), "Duplicate node ID") &&
		!strings.Contains(err.Error(), "capabilities contains missing or duplicate node")) {
		t.Fatalf("expected duplicate node rejection, got: %v", err)
	}
}
