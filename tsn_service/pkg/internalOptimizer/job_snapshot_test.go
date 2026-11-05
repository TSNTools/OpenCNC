// verifies Manager job ID → snapshot ID mapping.
package internalOptimizer

import (
	"testing"

	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

func TestJobSnapshotMapping(t *testing.T) {
	useFakeSnapshotStore(t)

	original := &forwardingplane.ForwardingPlaneModel{
		Configuration: &topology_config.TopologyConfig{
			ConfigId: "original-config",
		},
	}

	// Step 1: Preserve original CNC model.
	snapshotID, err := saveOptimizationSnapshot(original)
	if err != nil {
		t.Fatal(err)
	}

	// Step 2: Associate it with Manager's job ID.
	jobID := "job-test-123"

	if err := saveJobSnapshotLink(jobID, snapshotID); err != nil {
		t.Fatal(err)
	}

	// Step 3: Retrieve original model using job ID.
	restored, err := loadSnapshotForJob(jobID)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(original, restored) {
		t.Fatal("restored model does not match original")
	}

	// Step 4: Unknown jobs must not retrieve a snapshot.
	if _, err := loadSnapshotForJob("unknown-job"); err == nil {
		t.Fatal("expected error for unknown job")
	}
}
