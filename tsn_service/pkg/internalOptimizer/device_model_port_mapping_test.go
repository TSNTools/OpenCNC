package internalOptimizer

import (
	"reflect"
	"testing"

	"OpenCNC/common/structures/devicemodelregistry"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func b1MappingTestFPM(t *testing.T) *forwarding_plane.ForwardingPlaneModel {
	t.Helper()
	const input = `{"topology":{"nodes":[
		{"name":"SW1","type":"BRIDGE","ports":[
			{"id":"P1","capabilities":{"supportsTas":true}},
			{"id":"P2","capabilities":{"supportsTas":true}},
			{"id":"P3","capabilities":{"supportsCbs":true}},
			{"id":"P4"}]},
		{"name":"SW2","type":"BRIDGE","ports":[{"id":"P1","capabilities":{"supportsTas":true}}]},
		{"name":"SW3","type":"BRIDGE","ports":[{"id":"P1","capabilities":{"supportsTas":true}}]},
		{"name":"ES1","type":"END_STATION","ports":[{"id":"P1","capabilities":{"supportsTas":true}}]}
	]}}`
	var fpm forwarding_plane.ForwardingPlaneModel
	if err := protojson.Unmarshal([]byte(input), &fpm); err != nil {
		t.Fatal(err)
	}
	return &fpm
}

func b1SyntheticGCLModel(value float64) *devicemodelregistry.DeviceModel {
	return &devicemodelregistry.DeviceModel{Limitations: []*devicemodelregistry.DeviceLimitation{{
		TargetPath: bridgeAdminGCLPath,
		Parameter: &devicemodelregistry.LimitationParameter{
			Name: "length", Unit: "entries",
			TargetKind: devicemodelregistry.TargetKind_YANG_NODE_KIND_LIST,
		},
		Comparison: devicemodelregistry.LimitationComparison_LIMITATION_LESS_OR_EQUAL,
		Threshold:  value,
	}}}
}

func TestB1MapTASGCLLimitsToPorts(t *testing.T) {
	fpm := b1MappingTestFPM(t)
	original := proto.Clone(fpm)
	models := map[string]*devicemodelregistry.DeviceModel{
		"SW1": b1SyntheticGCLModel(64),
		"SW2": b1SyntheticGCLModel(128),
		"SW3": {},
		"ES1": b1SyntheticGCLModel(256),
	}
	limits, err := collectTASGCLLimitsByNode(models)
	if err != nil {
		t.Fatal(err)
	}
	got, err := mapTASGCLLimitsToPorts(fpm, limits)
	if err != nil {
		t.Fatal(err)
	}
	portLimit := func(n uint32) map[string]any {
		return map[string]any{"mechanisms": map[string]any{"TAS": map[string]any{"maxGclEntries": n}}}
	}
	want := map[string]any{"devices": []any{
		map[string]any{"nodeId": "SW1", "ports": map[string]any{"P1": portLimit(64), "P2": portLimit(64)}},
		map[string]any{"nodeId": "SW2", "ports": map[string]any{"P1": portLimit(128)}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mapping mismatch: got %#v, want %#v", got, want)
	}
	if !proto.Equal(fpm, original) {
		t.Fatal("original FPM was modified")
	}
}

func TestB1MapTASGCLLimitsEmpty(t *testing.T) {
	got, err := mapTASGCLLimitsToPorts(b1MappingTestFPM(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]any{"devices": []any{}}) {
		t.Fatalf("fabricated limits: %#v", got)
	}
}

func TestB1MapTASGCLLimitsRejectsAmbiguity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits map[string][]tasGCLLimit
	}{
		{"conflicting thresholds", map[string][]tasGCLLimit{"SW1": {{TargetPath: bridgeAdminGCLPath, MaxEntries: 64}, {TargetPath: bridgeAdminGCLPath, MaxEntries: 128}}}},
		{"unconfirmed XPath", map[string][]tasGCLLimit{"SW1": {{TargetPath: "/test/admin-control-list/gate-control-entry", MaxEntries: 64}}}},
		{"unknown node", map[string][]tasGCLLimit{"UNKNOWN": {{TargetPath: bridgeAdminGCLPath, MaxEntries: 64}}}},
		{"zero threshold", map[string][]tasGCLLimit{"SW1": {{TargetPath: bridgeAdminGCLPath, MaxEntries: 0}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := mapTASGCLLimitsToPorts(b1MappingTestFPM(t), tc.limits); err == nil {
				t.Fatal("expected mapping error")
			}
		})
	}
}
