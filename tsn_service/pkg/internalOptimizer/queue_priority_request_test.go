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

	// Use a non-identity PCP mapping.
	port := original.Configuration.NodeConfigs[0].PortConfigs[0]
	port.PcpMappingEnabled = proto.Bool(true)
	port.PcpConfig = &pcp.PcpPortConfig{
		TrafficClassTable: []*pcp.TrafficClassMappingEntry{
			{Priority: 7, EgressQueueId: 2},
			{Priority: 5, EgressQueueId: 7},
		},
	}

	before := proto.Clone(original)

	requestCopy, err := withDefaultPCPQueueMappings(original)
	if err != nil {
		t.Fatal(err)
	}

	plane, err := requestadapter.ExtractForwardingPlane(
		requestCopy, requestadapter.SchemaJSON(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := requestadapter.ApplyCNCQueuePriorities(
		plane, requestCopy,
	); err != nil {
		t.Fatal(err)
	}

	nodes := plane["configuration"].(map[string]any)["nodeConfigs"].([]any)
	ports := nodes[0].(map[string]any)["portConfigs"].([]any)

	first := ports[0].(map[string]any)
	table := first["queuePriorityTable"].([]any)

	if len(table) != 8 {
		t.Fatal("expected eight queue priorities")
	}

	for i, item := range table {
		entry := item.(map[string]any)

		if entry["queueId"] != i ||
			entry["schedulingPriority"] != 8-i {
			t.Fatalf("unexpected priority: %v", entry)
		}
	}

	// Do not apply the eight-queue policy to four-queue ports.
	if _, ok := ports[1].(map[string]any)["queuePriorityTable"]; ok {
		t.Fatal("four-queue port received an eight-queue policy")
	}

	// Both CBS and TAS must accept the same configuration.
	for _, mechanism := range []string{"CBS", "TAS"} {
		request := map[string]any{
			"schemaVersion":   "1.4",
			"forwardingPlane": plane,
			"mechanisms": []map[string]string{
				{"name": mechanism},
			},
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := contractBaselineForJob(data, original); err != nil {
			t.Fatalf("%s baseline rejected: %v", mechanism, err)
		}

		// Deliberately corrupt a scheduling priority.
		table[7].(map[string]any)["schedulingPriority"] = 9

		data, err = json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		_, err = contractBaselineForJob(data, original)
		if err == nil || !strings.Contains(err.Error(), "configuration differs") {
			t.Fatalf("%s: unexpected priority should be rejected", mechanism)
		}

		// Restore the correct priority.
		table[7].(map[string]any)["schedulingPriority"] = 1
	}

	if !proto.Equal(original, before) {
		t.Fatal("original FPM was modified")
	}
}
