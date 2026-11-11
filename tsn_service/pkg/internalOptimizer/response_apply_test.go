package internalOptimizer

import (
	"reflect"
	"strings"
	"testing"

	"OpenCNC/common/structures/qav"
	"OpenCNC/common/structures/qbv"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	tsncontracts "github.com/Kashaf14/tsn-optimization-contracts"
	"google.golang.org/protobuf/proto"
)

func responseApplyOriginal() *forwardingplane.ForwardingPlaneModel {
	pcpEnabled := true
	return &forwardingplane.ForwardingPlaneModel{
		Metadata: &forwardingplane.ModelMetadata{ModelId: "original-network", Revision: 7},
		Topology: &topology.Topology{Nodes: []*topology.Node{{
			Name: "SW1", Ports: []*topology.Port{{Id: "P4", NumberOfQueues: 8}, {Id: "P5", NumberOfQueues: 8}},
		}}},
		Configuration: &topology_config.TopologyConfig{
			ConfigId: "preserved-config",
			NodeConfigs: []*topology_config.NodeConfig{{
				NodeId: "SW1",
				PortConfigs: []*topology_config.PortConfig{
					{
						PortId: "P4", Description: "must stay",
						PcpMappingEnabled: &pcpEnabled,
						Gcl:               &qbv.GateControlList{CycleTime: 500000},
						QueueConfigs:      []*topology_config.QueueConfig{{QueueId: 6, MaxFrameSize: 1500}},
					},
					{PortId: "P5", Description: "other-port"},
				},
			}},
			LinkConfigs: []*topology_config.LinkConfig{{LinkId: "L1"}},
		},
	}
}

func validatedFixture(original *forwardingplane.ForwardingPlaneModel, applicable bool, payload string) *validatedOptimizationResult {
	return &validatedOptimizationResult{
		Original:        original,
		Validation:      &tsncontracts.ResponseValidationReport{Applicable: applicable},
		OptimizerResult: []byte(payload),
	}
}

const cbsUpdateFixture = `{"status":"success","solution":{"changes":{"configuration":{"nodeConfigs":[{"nodeId":"SW1","portConfigs":[{"portId":"P4","queueConfigs":[{"queueId":5,"scheduling":"CREDIT_BASED_SHAPER","cbs":[{"egressQueueId":5,"adminState":"ENABLED","adminIdleSlopeBps":12000000,"sendSlopeBps":88000000,"hiCreditBytes":26,"loCreditBytes":175}]}]}]}]}}}}`

func findQueueInTest(t *testing.T, model *forwardingplane.ForwardingPlaneModel, nodeID, portID string, queueID uint32) *topology_config.QueueConfig {
	t.Helper()
	for _, n := range model.GetConfiguration().GetNodeConfigs() {
		if n.GetNodeId() != nodeID {
			continue
		}
		for _, p := range n.GetPortConfigs() {
			if p.GetPortId() != portID {
				continue
			}
			for _, q := range p.GetQueueConfigs() {
				if q.GetQueueId() == queueID {
					return q
				}
			}
		}
	}
	t.Fatalf("queue %s/%s/%d not found", nodeID, portID, queueID)
	return nil
}

func TestApplyValidatedOptimizerResponsePreservesOriginal(t *testing.T) {
	original := responseApplyOriginal()
	before := proto.Clone(original)
	merged, err := applyValidatedOptimizerResponse(validatedFixture(original, true, cbsUpdateFixture))
	if err != nil {
		t.Fatal(err)
	}
	if merged == original || proto.Equal(merged, original) {
		t.Fatal("expected an independent updated model")
	}
	if !proto.Equal(original, before) {
		t.Fatal("original forwarding plane was modified")
	}
	if !proto.Equal(merged.GetTopology(), original.GetTopology()) {
		t.Fatal("topology changed")
	}
	if !proto.Equal(merged.GetMetadata(), original.GetMetadata()) {
		t.Fatal("metadata changed")
	}
	if !reflect.DeepEqual(merged.GetStreams(), original.GetStreams()) {
		t.Fatal("streams changed")
	}
	if !proto.Equal(merged.GetConfiguration().GetLinkConfigs()[0], original.GetConfiguration().GetLinkConfigs()[0]) {
		t.Fatal("links changed")
	}
	if merged.GetConfiguration().GetConfigId() != original.GetConfiguration().GetConfigId() {
		t.Fatal("configuration ID changed")
	}
	port := merged.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[0]
	origPort := original.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[0]
	if !proto.Equal(port.GetGcl(), origPort.GetGcl()) || port.GetDescription() != "must stay" || !port.GetPcpMappingEnabled() {
		t.Fatal("existing port settings were not preserved")
	}
	if !proto.Equal(merged.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[1], original.GetConfiguration().GetNodeConfigs()[0].GetPortConfigs()[1]) {
		t.Fatal("unmentioned port was modified")
	}
	untouchedQueue := findQueueInTest(t, merged, "SW1", "P4", 6)
	if untouchedQueue.GetMaxFrameSize() != 1500 {
		t.Fatal("unrelated queue was modified")
	}
	changed := findQueueInTest(t, merged, "SW1", "P4", 5)
	if changed.GetScheduling() != topology_config.SchedulingModel_CREDIT_BASED_SHAPER {
		t.Fatal("CBS scheduling not applied")
	}
	if len(changed.GetCbs()) != 1 {
		t.Fatal("missing CBS record")
	}
	cbs := changed.GetCbs()[0]
	if cbs.GetAdminState() != qav.AdminState_ENABLED || cbs.GetEgressQueueId() != 5 ||
		cbs.GetAdminIdleSlopeBps() != 12000000 || cbs.GetSendSlopeBps() != 88000000 ||
		cbs.GetHiCreditBytes() != 26 || cbs.GetLoCreditBytes() != 175 {
		t.Fatalf("incorrect CBS values: %+v", cbs)
	}
}

func TestApplyValidatedOptimizerResponseMergesExistingCBSFields(t *testing.T) {
	original := responseApplyOriginal()
	p := original.Configuration.NodeConfigs[0].PortConfigs[0]
	p.QueueConfigs = append(p.QueueConfigs, &topology_config.QueueConfig{
		QueueId:      5,
		MaxFrameSize: 1300,
		Scheduling:   topology_config.SchedulingModel_CREDIT_BASED_SHAPER.Enum(),
		Cbs: []*qav.CbsQueueConfig{{
			EgressQueueId: 5, AdminState: qav.AdminState_ENABLED,
			AdminIdleSlopeBps: 100, SendSlopeBps: 900,
			HiCreditBytes: 7, LoCreditBytes: 8, Description: "keep this",
		}},
	})
	payload := `{"status":"success","solution":{"changes":{"configuration":{"nodeConfigs":[{"nodeId":"SW1","portConfigs":[{"portId":"P4","queueConfigs":[{"queueId":5,"cbs":[{"egressQueueId":5,"adminIdleSlopeBps":250}]}]}]}]}}}}`
	merged, err := applyValidatedOptimizerResponse(validatedFixture(original, true, payload))
	if err != nil {
		t.Fatal(err)
	}
	q := findQueueInTest(t, merged, "SW1", "P4", 5)
	if q.GetMaxFrameSize() != 1300 || q.GetScheduling() != topology_config.SchedulingModel_CREDIT_BASED_SHAPER {
		t.Fatal("unmentioned queue fields changed")
	}
	cbs := q.GetCbs()[0]
	if cbs.GetAdminIdleSlopeBps() != 250 || cbs.GetSendSlopeBps() != 900 ||
		cbs.GetHiCreditBytes() != 7 || cbs.GetLoCreditBytes() != 8 ||
		cbs.GetAdminState() != qav.AdminState_ENABLED || cbs.GetDescription() != "keep this" {
		t.Fatalf("sparse CBS update lost original settings: %+v", cbs)
	}
	if original.Configuration.NodeConfigs[0].PortConfigs[0].QueueConfigs[1].GetCbs()[0].GetAdminIdleSlopeBps() != 100 {
		t.Fatal("mutated original CBS configuration")
	}
}

func TestApplyValidatedOptimizerResponseNoOpReturnsClone(t *testing.T) {
	original := responseApplyOriginal()
	payload := `{"status":"success","solution":{"changes":{}}}`
	cloned, err := applyValidatedOptimizerResponse(validatedFixture(original, false, payload))
	if err != nil {
		t.Fatal(err)
	}
	if cloned == original || !proto.Equal(cloned, original) {
		t.Fatal("no-op should return equal independent clone")
	}
}

func TestApplyValidatedOptimizerResponseRejectsBadInputs(t *testing.T) {
	cases := []struct{ name, payload, expected string }{
		{"unknown node", strings.Replace(cbsUpdateFixture, `"nodeId":"SW1"`, `"nodeId":"unknown"`, 1), "unknown physical node"},
		{"unknown port", strings.Replace(cbsUpdateFixture, `"portId":"P4"`, `"portId":"bad"`, 1), "unknown physical port"},
		{"out of range queue", strings.ReplaceAll(cbsUpdateFixture, `"queueId":5`, `"queueId":9`), "outside the"},
		{"mismatched CBS queue", strings.Replace(cbsUpdateFixture, `"egressQueueId":5`, `"egressQueueId":6`, 1), "does not match"},
		{"new partial CBS record", strings.Replace(cbsUpdateFixture, `"adminState":"ENABLED","adminIdleSlopeBps":12000000,"sendSlopeBps":88000000,"hiCreditBytes":26,"loCreditBytes":175`, `"adminState":"ENABLED"`, 1), "complete parameters"},
		{"non-CBS scheduling", strings.Replace(cbsUpdateFixture, `"CREDIT_BASED_SHAPER"`, `"STRICT_PRIORITY"`, 1), "non-CBS"},
		{"missing scheduling on new queue", strings.Replace(cbsUpdateFixture, `"scheduling":"CREDIT_BASED_SHAPER",`, ``, 1), "explicitly select"},
		{"unsuccessful result", strings.Replace(cbsUpdateFixture, `"status":"success"`, `"status":"failed"`, 1), "unsuccessful"},
		{"no effective changes but applicable", `{"status":"success","solution":{"changes":{}}}`, "contains no changes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := responseApplyOriginal()
			before := proto.Clone(original)
			merged, err := applyValidatedOptimizerResponse(validatedFixture(original, true, tc.payload))
			if err == nil || !strings.Contains(err.Error(), tc.expected) {
				t.Fatalf("expected error containing %q, got model %v and error %v", tc.expected, merged, err)
			}
			if merged != nil {
				t.Fatal("invalid merge returned a model")
			}
			if !proto.Equal(before, original) {
				t.Fatal("failed merge modified original")
			}
		})
	}
}

func TestApplyValidatedOptimizerResponseRequiresValidation(t *testing.T) {
	if model, err := applyValidatedOptimizerResponse(nil); err == nil || model != nil {
		t.Fatal("nil validation accepted")
	}
	result := validatedFixture(responseApplyOriginal(), true, cbsUpdateFixture)
	result.Validation = nil
	if model, err := applyValidatedOptimizerResponse(result); err == nil || model != nil {
		t.Fatal("missing report accepted")
	}
}

func TestApplyValidatedOptimizerResponseRejectsDuplicateOriginalConfig(t *testing.T) {
	original := responseApplyOriginal()
	duplicate := proto.Clone(original.Configuration.NodeConfigs[0]).(*topology_config.NodeConfig)
	original.Configuration.NodeConfigs = append(original.Configuration.NodeConfigs, duplicate)
	merged, err := applyValidatedOptimizerResponse(validatedFixture(original, true, cbsUpdateFixture))
	if err == nil || merged != nil {
		t.Fatal("duplicate original configuration was accepted")
	}
}

// Standard response fields are not equivalent to native CNC mappings.
// The adapter MUST fail closed rather than silently ignoring unsupported paths.
func TestApplyValidatedOptimizerResponseRejectsUnsupportedChangePaths(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{"routing", `{"status":"success","solution":{"changes":{"streams":[{"id":"stream-1"}]}}}`},
		{"link config", `{"status":"success","solution":{"changes":{"configuration":{"linkConfigs":[{"linkId":"L1"}]}}}}`},
		{"port GCL", `{"status":"success","solution":{"changes":{"configuration":{"nodeConfigs":[{"nodeId":"SW1","portConfigs":[{"portId":"P4","gcl":{"cycleTime":500}}]}]}}}}`},
		{"queue other field", `{"status":"success","solution":{"changes":{"configuration":{"nodeConfigs":[{"nodeId":"SW1","portConfigs":[{"portId":"P4","queueConfigs":[{"queueId":5,"maxFrameSize":1400}]}]}]}}}}`},
		{"mixed supported and unsupported", `{"status":"success","solution":{"changes":{"configuration":{"nodeConfigs":[{"nodeId":"SW1","portConfigs":[{"portId":"P4","queueConfigs":[{"queueId":5,"scheduling":"CREDIT_BASED_SHAPER","cbs":[{"egressQueueId":5,"adminState":"ENABLED","adminIdleSlopeBps":12000000,"sendSlopeBps":88000000,"hiCreditBytes":26,"loCreditBytes":175}]}]}]}]},"streams":[]}}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := responseApplyOriginal()
			before := proto.Clone(original)
			got, err := applyValidatedOptimizerResponse(validatedFixture(original, true, tc.payload))
			if err == nil || got != nil {
				t.Fatalf("expected unsupported change rejection, got model %v, err %v", got, err)
			}
			if !proto.Equal(original, before) {
				t.Fatal("rejected changes mutated preserved original")
			}
		})
	}
}

func TestApplyValidatedOptimizerResponseRejectsChangesInNoOp(t *testing.T) {
	original := responseApplyOriginal()
	got, err := applyValidatedOptimizerResponse(validatedFixture(original, false, cbsUpdateFixture))
	if err == nil || got != nil {
		t.Fatalf("nonempty changes cannot be a no-op: model=%v err=%v", got, err)
	}
}
