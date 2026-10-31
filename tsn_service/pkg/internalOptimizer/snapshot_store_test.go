package internalOptimizer

import (
	"fmt"
	"testing"

	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

// Simulate etcd using an in-memory map.
func useFakeSnapshotStore(t *testing.T) {
	t.Helper()

	storage := make(map[string][]byte)

	originalWrite := snapshotWrite
	originalRead := snapshotRead

	t.Cleanup(func() {
		snapshotWrite = originalWrite
		snapshotRead = originalRead
	})

	snapshotWrite = func(data []byte, key string) error {
		storage[key] = append([]byte(nil), data...)
		return nil
	}

	snapshotRead = func(key string) ([]byte, error) {
		data, ok := storage[key]
		if !ok {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return append([]byte(nil), data...), nil
	}
}

func TestOptimizationSnapshotRoundTrip(t *testing.T) {
	useFakeSnapshotStore(t)

	// Sample complete CNC model.
	model := &forwardingplane.ForwardingPlaneModel{
		Metadata: &forwardingplane.ModelMetadata{
			ModelId:  "test-network",
			Revision: 7,
		},
		Topology: &topology.Topology{
			Nodes: []*topology.Node{
				{Name: "SW1"},
			},
		},
		Configuration: &topology_config.TopologyConfig{
			ConfigId: "original-config",
		},
	}

	expected := proto.Clone(model)

	// Save snapshot.
	snapshotID, err := saveOptimizationSnapshot(model)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the original model after saving.
	model.Configuration.ConfigId = "modified-config"

	// Retrieve snapshot.
	restored, err := loadOptimizationSnapshot(snapshotID)
	if err != nil {
		t.Fatal(err)
	}

	// Verify snapshot is unchanged.
	if !proto.Equal(expected, restored) {
		t.Fatal("snapshot does not match original model")
	}
}
