package internalOptimizer

import (
	"encoding/json"
	"strings"
	"testing"

	"OpenCNC/common/structures/pcp"
	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"

	"google.golang.org/protobuf/proto"
)

func TestRequestQueuePrioritiesAndBaseline(t *testing.T) {
	original := baselinePCPTestFPM()
	port := original.Configuration.NodeConfigs[0].PortConfigs[0]
	port.PcpMappingEnabled = proto.Bool(true)
	port.PcpConfig = &pcp.PcpPortConfig{TrafficClassTable: []*pcp.TrafficClassMappingEntry{
		{Priority: 7, EgressQueueId: 2},
		{Priority: 5, EgressQueueId: 7},
	}}
	before := proto.Clone(original)

	requestCopy, err := withDefaultPCPQueueMappings(original)
	if err != nil {
		t.Fatal(err)
	}
	plane, err := requestadapter.ExtractForwardingPlane(requestCopy, requestadapter.SchemaJSON())
	if err != nil {
		t.Fatal(err)
	}
	if err := requestadapter.ApplyCNCQueuePriorities(plane, requestCopy); err != nil {
		t.Fatal(err)
	}
	nodes := plane["configuration"].(map[string]any)["nodeConfigs"].([]any)
	ports := nodes[0].(map[string]any)["portConfigs"].([]any)
	first := ports[0].(map[string]any)
	table := first["queuePriorityTable"].([]any)
	if len(table) != 8 || table[2].(map[string]any)["schedulingPriority"] != 6 ||
		table[7].(map[string]any)["schedulingPriority"] != 1 {
		t.Fatal("queue ordering must be independent of PCP")
	}
	if _, ok := ports[1].(map[string]any)["queuePriorityTable"]; ok {
		t.Fatal("four-queue port must not receive an eight-queue policy")
	}
	request := map[string]any{"schemaVersion": "1.4", "forwardingPlane": plane,
		"mechanisms": []map[string]string{{"name": "CBS"}}}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contractBaselineForJob(data, original); err != nil {
		t.Fatalf("expected enrichment rejected: %v", err)
	}
	table[7].(map[string]any)["schedulingPriority"] = 9
	data, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contractBaselineForJob(data, original); err == nil || !strings.Contains(err.Error(), "configuration differs") {
		t.Fatalf("unexpected priority should be rejected: %v", err)
	}
	table[7].(map[string]any)["schedulingPriority"] = 1
	request["mechanisms"] = []map[string]string{{"name": "TAS"}}
	data, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contractBaselineForJob(data, original); err == nil || !strings.Contains(err.Error(), "configuration differs") {
		t.Fatalf("CBS-only policy must not be accepted for TAS: %v", err)
	}
	if !proto.Equal(original, before) {
		t.Fatal("original native model changed")
	}
}
