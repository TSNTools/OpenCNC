package internalOptimizer

import (
	"errors"
	"testing"

	"OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/topology"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
)

// Test successful retrieval of a node's DeviceModel,
// including its hardware limitations.
func TestLoadNodeDeviceModels(t *testing.T) {
	originalLookup := deviceModelLookup
	t.Cleanup(func() {
		deviceModelLookup = originalLookup
	})

	var requestedName string

	deviceModelLookup = func(name string) (*devicemodelregistry.DeviceModel, error) {
		requestedName = name

		return &devicemodelregistry.DeviceModel{
			Name: name,
			Limitations: []*devicemodelregistry.DeviceLimitation{
				{
					TargetPath: "/test/gate-control-entry",
					Parameter: &devicemodelregistry.LimitationParameter{
						Name:       "length",
						Unit:       "entries",
						TargetKind: devicemodelregistry.TargetKind_YANG_NODE_KIND_LIST,
					},
					Comparison: devicemodelregistry.LimitationComparison_LIMITATION_LESS_OR_EQUAL,
					Threshold:  512,
				},
			},
		}, nil
	}

	fpm := &forwarding_plane.ForwardingPlaneModel{
		Topology: &topology.Topology{
			Nodes: []*topology.Node{
				{
					Name: "SW1",
					DeviceInfo: &topology.DeviceInfo{
						DeviceModel: "switch-model-A",
					},
				},
			},
		},
	}

	models, err := loadNodeDeviceModels(fpm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestedName != "switch-model-A" {
		t.Fatalf("wrong model requested: %s", requestedName)
	}

	model := models["SW1"]
	if model == nil || model.GetName() != "switch-model-A" {
		t.Fatal("incorrect DeviceModel returned for SW1")
	}

	// Verify the hardware limitation is preserved.
	limits := model.GetLimitations()

	if len(limits) != 1 {
		t.Fatalf("expected 1 limitation, got %d", len(limits))
	}

	limit := limits[0]

	if limit.GetThreshold() != 512 {
		t.Fatalf("expected threshold 512, got %v", limit.GetThreshold())
	}

	if limit.GetComparison() != devicemodelregistry.LimitationComparison_LIMITATION_LESS_OR_EQUAL {
		t.Fatal("incorrect limitation comparison")
	}

	if limit.GetParameter().GetName() != "length" {
		t.Fatal("incorrect limitation parameter")
	}

	if limit.GetTargetPath() != "/test/gate-control-entry" {
		t.Fatal("incorrect limitation target path")
	}
}

// Test error propagation when the lookup fails.
func TestLoadNodeDeviceModelsLookupFailure(t *testing.T) {
	originalLookup := deviceModelLookup
	t.Cleanup(func() {
		deviceModelLookup = originalLookup
	})

	expectedErr := errors.New("etcd unavailable")

	deviceModelLookup = func(name string) (*devicemodelregistry.DeviceModel, error) {
		return nil, expectedErr
	}

	fpm := &forwarding_plane.ForwardingPlaneModel{
		Topology: &topology.Topology{
			Nodes: []*topology.Node{
				{
					Name: "SW1",
					DeviceInfo: &topology.DeviceInfo{
						DeviceModel: "switch-model-A",
					},
				},
			},
		},
	}

	_, err := loadNodeDeviceModels(fpm)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected lookup error, got: %v", err)
	}
}

// Test that a nil DeviceModel is rejected.
func TestLoadNodeDeviceModelsNilResult(t *testing.T) {
	originalLookup := deviceModelLookup
	t.Cleanup(func() {
		deviceModelLookup = originalLookup
	})

	deviceModelLookup = func(name string) (*devicemodelregistry.DeviceModel, error) {
		return nil, nil
	}

	fpm := &forwarding_plane.ForwardingPlaneModel{
		Topology: &topology.Topology{
			Nodes: []*topology.Node{
				{
					Name: "SW1",
					DeviceInfo: &topology.DeviceInfo{
						DeviceModel: "switch-model-A",
					},
				},
			},
		},
	}

	_, err := loadNodeDeviceModels(fpm)

	if err == nil {
		t.Fatal("expected error for nil DeviceModel")
	}
}
