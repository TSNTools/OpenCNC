package internalOptimizer

import (
	"testing"

	"OpenCNC/common/structures/devicemodelregistry"
)

func TestExtractTASGCLLimits(t *testing.T) {
	model := &devicemodelregistry.DeviceModel{
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
	}

	got, err := extractTASGCLLimits(model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 GCL limit, got %d", len(got))
	}

	if got[0].MaxEntries != 512 {
		t.Fatalf("expected maxGclEntries=512, got %d", got[0].MaxEntries)
	}
}

func TestExtractTASGCLLimitsValidation(t *testing.T) {
	path := "/interfaces/interface/gate-parameter-table/admin-control-list/gate-control-entry"

	tests := []struct {
		name      string
		path      string
		threshold float64
		wantError bool
		wantCount int
	}{
		{"negative threshold", path, -1, true, 0},
		{"fractional threshold", path, 12.5, true, 0},
		{"unrelated limitation", "/interfaces/interface/other-list", 512, false, 0},
		{"valid threshold", path, 512, false, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := &devicemodelregistry.DeviceModel{
				Limitations: []*devicemodelregistry.DeviceLimitation{
					{
						TargetPath: tc.path,
						Parameter: &devicemodelregistry.LimitationParameter{
							Name:       "length",
							Unit:       "entries",
							TargetKind: devicemodelregistry.TargetKind_YANG_NODE_KIND_LIST,
						},
						Comparison: devicemodelregistry.LimitationComparison_LIMITATION_LESS_OR_EQUAL,
						Threshold:  tc.threshold,
					},
				},
			}

			got, err := extractTASGCLLimits(model)

			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.wantError && len(got) != tc.wantCount {
				t.Fatalf("expected %d limits, got %d", tc.wantCount, len(got))
			}
		})
	}
}

// This verifies that limitations remain associated with the correct CNC node. The YANG path is still synthetic, and no port assignment is assumed.
func TestCollectTASGCLLimitsByNode(t *testing.T) {
	models := map[string]*devicemodelregistry.DeviceModel{
		"SW1": {
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
		"SW2": {
			Limitations: []*devicemodelregistry.DeviceLimitation{},
		},
	}

	got, err := collectTASGCLLimitsByNode(models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got["SW1"]) != 1 {
		t.Fatalf("expected one limitation for SW1")
	}

	if got["SW1"][0].MaxEntries != 512 {
		t.Fatalf("expected SW1 maxGclEntries=512")
	}

	if len(got["SW2"]) != 0 {
		t.Fatalf("expected no GCL limitations for SW2")
	}
}
