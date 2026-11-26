package internalOptimizer

import (
	"encoding/json"
	"strings"
	"testing"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"OpenCNC/common/structures/topology"

	"google.golang.org/protobuf/proto"
)

// Build a valid forwarding-plane fixture for baseline validation.
// The existing PCP test fixture omits capabilities because its
// original tests do not require contract projection.
func baselinePCPTestFPM() *forwardingplane.ForwardingPlaneModel {
	original := defaultPCPTestFPM()

	for _, node := range original.GetTopology().GetNodes() {
		for _, port := range node.GetPorts() {
			port.Capabilities = &topology.InterfaceCapabilities{
				PortSpeed:   1000,
				SupportsCbs: true,
			}
		}
	}

	return original
}

// Build a saved request containing the same request-only PCP defaults
// used by the CBS request builder.
func baselinePCPRequestForTest(
	t *testing.T,
	original *forwardingplane.ForwardingPlaneModel,
	mechanism string,
) []byte {
	t.Helper()

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

	request := map[string]any{
		"schemaVersion":   "1.4",
		"forwardingPlane": plane,
	}

	if mechanism != "" {
		request["mechanisms"] = []map[string]string{
			{"name": mechanism},
		}
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// The known CBS PCP fallback must be accepted without modifying
// the preserved original forwarding-plane model.
func TestBaselineAcceptsRequestOnlyCBSDefaults(t *testing.T) {
	original := baselinePCPTestFPM()
	before := proto.Clone(original)

	request := baselinePCPRequestForTest(t, original, "CBS")

	baseline, err := contractBaselineForJob(request, original)
	if err != nil {
		t.Fatalf("expected PCP fallback to be accepted: %v", err)
	}
	if len(baseline) == 0 {
		t.Fatal("missing validated baseline")
	}
	if !proto.Equal(original, before) {
		t.Fatal("preserved original FPM was modified")
	}
}

// Even for CBS, an arbitrary PCP mapping must not be accepted.
func TestBaselineRejectsUnexpectedPCPMapping(t *testing.T) {
	original := baselinePCPTestFPM()
	requestJSON := baselinePCPRequestForTest(t, original, "CBS")

	var request map[string]any
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		t.Fatal(err)
	}

	plane := request["forwardingPlane"].(map[string]any)
	config := plane["configuration"].(map[string]any)
	nodes := config["nodeConfigs"].([]any)
	ports := nodes[0].(map[string]any)["portConfigs"].([]any)
	pcp := ports[0].(map[string]any)["pcpConfig"].(map[string]any)
	table := pcp["trafficClassTable"].([]any)

	// Deliberately corrupt one expected identity mapping.
	table[0].(map[string]any)["egressQueueId"] = float64(7)

	modified, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	_, err = contractBaselineForJob(modified, original)
	if err == nil || !strings.Contains(err.Error(), "configuration differs") {
		t.Fatalf("unexpected PCP mapping should be rejected: %v", err)
	}
}

// The CBS fallback must not be accepted for unrelated mechanisms.
// PCP fallback is independent of the optimization mechanism.
func TestBaselineAcceptsPCPFallbackForTAS(t *testing.T) {
	original := baselinePCPTestFPM()
	before := proto.Clone(original)

	request := baselinePCPRequestForTest(t, original, "TAS")

	_, err := contractBaselineForJob(request, original)
	if err != nil {
		t.Fatalf("expected PCP fallback to be accepted: %v", err)
	}

	if !proto.Equal(original, before) {
		t.Fatal("original FPM was modified")
	}
}

// The fallback must never hide an actual topology mismatch.
func TestBaselineStillRejectsTopologyMismatch(t *testing.T) {
	original := baselinePCPTestFPM()
	requestJSON := baselinePCPRequestForTest(t, original, "CBS")

	var request map[string]any
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		t.Fatal(err)
	}

	plane := request["forwardingPlane"].(map[string]any)
	topology := plane["topology"].(map[string]any)
	nodes := topology["nodes"].([]any)
	nodes[0].(map[string]any)["name"] = "WRONG-NODE"

	modified, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	_, err = contractBaselineForJob(modified, original)
	if err == nil || !strings.Contains(err.Error(), "topology differs") {
		t.Fatalf("topology mismatch should be rejected: %v", err)
	}
}
