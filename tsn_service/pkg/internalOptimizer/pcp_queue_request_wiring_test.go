// This test creates a request with a missing mapping and verifies that the generated v1.4 JSON contains the eight default mappings while leaving the original FPM unchanged.
package internalOptimizer

import (
	"encoding/json"
	"os"
	"testing"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Regression test for the entire request-building path, not only the default
// mapping helper. The default must appear in the serialized v1.4 request while
// the original CNC forwarding-plane model remains untouched.
func TestPCPQueueDefaultReachesV14Request(t *testing.T) {
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

	// The historical fixture represents StreamRank as {"rank": "..."}.
	// Convert to the current OpenCNC protobuf enum JSON representation.
	var rawPlane map[string]any
	if err := json.Unmarshal(fixture.ForwardingPlane, &rawPlane); err != nil {
		t.Fatal(err)
	}
	for _, rawStream := range rawPlane["streams"].([]any) {
		definition := rawStream.(map[string]any)["definition"].(map[string]any)
		if rank, ok := definition["streamRank"].(map[string]any); ok {
			definition["streamRank"] = rank["rank"]
		}
	}
	planeJSON, err := json.Marshal(rawPlane)
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

	// Simulate a missing mapping on an 8-queue egress port. Other mappings
	// remain present, so the fallback must not replace their configuration.
	found := false
	for _, node := range fpm.GetConfiguration().GetNodeConfigs() {
		if node.GetNodeId() != "SW1" {
			continue
		}
		for _, port := range node.GetPortConfigs() {
			if port.GetPortId() == "s1-out" {
				port.PcpConfig = nil
				port.PcpMappingEnabled = nil
				found = true
			}
		}
	}
	if !found {
		t.Fatal("fixture is missing SW1/s1-out")
	}
	original := proto.Clone(&fpm)

	built, err := buildOaaSRequest(&task, &fpm)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(&fpm, original) {
		t.Fatal("request construction modified the original FPM")
	}

	var request struct {
		ForwardingPlane struct {
			Configuration struct {
				NodeConfigs []struct {
					NodeID      string `json:"nodeId"`
					PortConfigs []struct {
						PortID            string `json:"portId"`
						PcpMappingEnabled *bool  `json:"pcpMappingEnabled"`
					QueuePriorityTable []struct {
						QueueID int `json:"queueId"`
						SchedulingPriority int `json:"schedulingPriority"`
					} `json:"queuePriorityTable"`
						PcpConfig         struct {
							TrafficClassTable []struct {
								Priority      uint32 `json:"priority"`
								EgressQueueID uint32 `json:"egressQueueId"`
							} `json:"trafficClassTable"`
						} `json:"pcpConfig"`
					} `json:"portConfigs"`
				} `json:"nodeConfigs"`
			} `json:"configuration"`
		} `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(built, &request); err != nil {
		t.Fatal(err)
	}
	for _, node := range request.ForwardingPlane.Configuration.NodeConfigs {
		if node.NodeID != "SW1" {
			continue
		}
		for _, port := range node.PortConfigs {
			if port.PortID != "s1-out" {
				continue
			}
			if port.PcpMappingEnabled == nil || !*port.PcpMappingEnabled {
				t.Fatal("default PCP mapping was not enabled in the v1.4 request")
			}
			if len(port.QueuePriorityTable) != 8 {
				t.Fatalf("expected eight explicit queue priorities, got %d", len(port.QueuePriorityTable))
			}
			for i, entry := range port.QueuePriorityTable {
				if entry.QueueID != i || entry.SchedulingPriority != 8-i {
					t.Fatalf("unexpected queue priority: %+v", entry)
				}
			}
			if _, err := contractBaselineForJob(built, &fpm); err != nil {
				t.Fatalf("request priority baseline rejected: %v", err)
			}
			entries := port.PcpConfig.TrafficClassTable
			if len(entries) != 8 {
				t.Fatalf("expected 8 queue mappings in v1.4 JSON, got %d", len(entries))
			}
			for i, entry := range entries {
				if entry.Priority != uint32(i) || entry.EgressQueueID != uint32(i) {
					t.Fatalf("unexpected PCP mapping at index %d: %+v", i, entry)
				}
			}
			return
		}
	}
	t.Fatal("SW1/s1-out not present in built v1.4 request")
}
