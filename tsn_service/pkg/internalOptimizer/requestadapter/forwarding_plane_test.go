package requestadapter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/encoding/protojson"
)

const testForwardingPlane = `{
  "metadata": {"modelId":"network-001","revision":"17","description":"CNC-only"},
  "topology": {
    "nodes": [
      {"name":"ES1","type":"END_STATION","ports":[{"id":"P1","number_of_queues":8,"capabilities":{"port_speed":1000}}]},
      {"name":"SW1","type":"BRIDGE","ports":[
        {"id":"P1","number_of_queues":8,"capabilities":{"port_speed":1000,"supportsTas":true}},
        {"id":"P2","number_of_queues":8,"capabilities":{"port_speed":1000,"supportsCbs":true,"supportsTas":true}}
      ]},
      {"name":"ES2","type":"END_STATION","ports":[{"id":"P1","number_of_queues":8,"capabilities":{"port_speed":1000}}]}
    ],
    "links": [
      {"id":"link-1","source_node":"ES1","destination_node":"SW1","source_port":"P1","destination_port":"P1","propagation_delay_ns":"1000","bandwidth_mbps":"1000"},
      {"id":"link-2","source_node":"SW1","destination_node":"ES2","source_port":"P2","destination_port":"P1","propagation_delay_ns":"1000","bandwidth_mbps":"1000"}
    ]
  },
  "configuration": {"configId":"cfg-001","nodeConfigs":[{"nodeId":"SW1","portConfigs":[
    {"portId":"P2","pcpMappingEnabled":true,"gcl":{"scheduleId":"test-schedule","cycleTime":"1000000","entries":[{"index":0,"timeInterval":"100000","gateStates":"IA=="}]},
    "pcpConfig":{"trafficClassTable":[{"priority":5,"egressQueueId":5}]}}
  ]}]},
  "streams": [{"definition": {
    "streamId":{"macAddress":"02:00:00:00:00:01","uniqueId":1},
    "talkerNodeId":"ES1","listenerNodeIds":["ES2"],
    "trafficSpec":{"maxFrameSize":"500","maxIntervalFrames":"1","intervalNs":"1000000"},
    "userToNetworkRequirements":{"maxLatencyNs":"1000000"},"trafficClassId":"M1"
  },"configuration":{"pcp":5}}]
}`

func loadV13Schema(t *testing.T) []byte {
	t.Helper()
	// go test ./... runs each package test from its own directory.
	path := "optimizer-request.schema.json"
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read v1.3 schema at %s: %v", path, err)
	}
	return b
}

func parseForwardingPlane(t *testing.T, input string) *forwardingplane.ForwardingPlaneModel {
	t.Helper()
	var model forwardingplane.ForwardingPlaneModel
	if err := protojson.Unmarshal([]byte(input), &model); err != nil {
		t.Fatal(err)
	}
	return &model
}

func TestExtractForwardingPlaneV13(t *testing.T) {
	model := parseForwardingPlane(t, testForwardingPlane)
	result, err := ExtractForwardingPlane(model, loadV13Schema(t))
	if err != nil {
		t.Fatal(err)
	}

	// Check actual JSON serialization, not just a Go struct.
	output, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(output, &out); err != nil {
		t.Fatal(err)
	}
	metadata := out["metadata"].(map[string]any)
	if _, exists := metadata["revision"]; exists {
		t.Fatal("CNC model revision leaked into optimizer JSON")
	}
	if metadata["modelId"] != "network-001" {
		t.Fatal("modelId was not preserved")
	}
	configuration := out["configuration"].(map[string]any)
	if _, exists := configuration["configId"]; exists {
		t.Fatal("CNC configId leaked into optimizer JSON")
	}
	streams := out["streams"].([]any)
	definition := streams[0].(map[string]any)["definition"].(map[string]any)
	if definition["id"] != "02:00:00:00:00:01/1" {
		t.Fatalf("unexpected derived stream id: %v", definition["id"])
	}
	spec := definition["trafficSpec"].(map[string]any)
	if spec["maxFrameSize"] != float64(500) {
		t.Fatalf("uint64 was not converted to JSON number: %v", spec["maxFrameSize"])
	}
	nodes := out["topology"].(map[string]any)["nodes"].([]any)
	portCaps := nodes[1].(map[string]any)["ports"].([]any)[0].(map[string]any)["capabilities"].(map[string]any)
	if _, exists := portCaps["supportsTas"]; exists {
		t.Fatal("duplicate supportsTas capability leaked into topology")
	}
	gcl := configuration["nodeConfigs"].([]any)[0].(map[string]any)["portConfigs"].([]any)[0].(map[string]any)["gcl"].(map[string]any)
	entry := gcl["entries"].([]any)[0].(map[string]any)
	if _, exists := entry["index"]; exists {
		t.Fatal("removed v1.2 GCL index is still present")
	}
	if entry["timeInterval"] != float64(100000) {
		t.Fatal("GCL duration mapping failed")
	}

	capabilities := ExtractPortCapabilities(model)
	device := capabilities["devices"].([]any)[0].(map[string]any)
	swP2 := device["ports"].(map[string]any)["P2"].(map[string]any)
	cbs := swP2["mechanisms"].(map[string]any)["CBS"].(map[string]any)
	if cbs["supported"] != true {
		t.Fatal("CBS port hardware support was not mapped")
	}
	if _, exists := cbs["supportedQueues"]; exists {
		t.Fatal("supportedQueues must not be fabricated from number_of_queues")
	}
}

func TestForwardingPlaneRejectsUnsafeInteger(t *testing.T) {
	// The schema uses JSON integer -> JS number. We must not lose precision.
	large := strings.Replace(testForwardingPlane, `"maxFrameSize":"500"`, `"maxFrameSize":"9007199254740992"`, 1)
	model := parseForwardingPlane(t, large)
	_, err := ExtractForwardingPlane(model, loadV13Schema(t))
	if err == nil || !strings.Contains(err.Error(), "safe JSON range") {
		t.Fatalf("expected loss-of-precision error, got %v", err)
	}
}

func TestForwardingPlaneRejectsDisabledPort(t *testing.T) {
	bad := strings.Replace(testForwardingPlane,
		`"portId":"P2","pcpMappingEnabled":true`,
		`"portId":"P2","adminState":"ADMIN_STATE_DISABLED","pcpMappingEnabled":true`, 1)
	model := parseForwardingPlane(t, bad)
	_, err := ExtractForwardingPlane(model, loadV13Schema(t))
	if err == nil || !strings.Contains(err.Error(), "disabled port") {
		t.Fatalf("expected unsafe topology rejection, got %v", err)
	}
}

func TestExtractPortMechanismCapabilities(t *testing.T) {
	input := strings.Replace(
		testForwardingPlane,
		`"supportsCbs":true,"supportsTas":true`,
		`"supportsCbs":true,"supportsTas":true,"supportsFrer":true,"supportsFramePreemption":true`,
		1,
	)

	if input == testForwardingPlane {
		t.Fatal("test fixture modification failed")
	}

	model := parseForwardingPlane(t, input)
	caps := ExtractPortCapabilities(model)

	devices := caps["devices"].([]any)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	device := devices[0].(map[string]any)
	if device["nodeId"] != "SW1" {
		t.Fatalf("unexpected device: %v", device["nodeId"])
	}

	port := device["ports"].(map[string]any)["P2"].(map[string]any)
	mechanisms := port["mechanisms"].(map[string]any)

	for _, name := range []string{"CBS", "TAS", "FRER"} {
		mechanism := mechanisms[name].(map[string]any)
		if mechanism["supported"] != true {
			t.Errorf("%s support was not mapped", name)
		}
	}

	tas := mechanisms["TAS"].(map[string]any)
	if tas["framePreemptionSupported"] != true {
		t.Error("frame preemption support was not mapped")
	}
}
