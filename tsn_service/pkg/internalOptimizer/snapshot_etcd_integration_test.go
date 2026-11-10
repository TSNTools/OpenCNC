// It uses real etcd and leaves two uniquely identified records for our subsequent restart test.
package internalOptimizer

import (
	"os"
	"testing"

	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"google.golang.org/protobuf/proto"
)

func TestB8RealEtcdSnapshotRoundTrip(t *testing.T) {
	if os.Getenv("B8_REAL_ETCD") != "1" {
		t.Skip("real etcd test disabled")
	}
	if os.Getenv("ETCD_HOST") != "127.0.0.1" ||
		os.Getenv("ETCD_PORT") != "2379" {
		t.Fatal("B8 requires isolated etcd at 127.0.0.1:2379")
	}

	original := &forwardingplane.ForwardingPlaneModel{
		Metadata: &forwardingplane.ModelMetadata{
			ModelId:  "b8-test",
			Revision: 7,
		},
		Topology: &topology.Topology{
			Nodes: []*topology.Node{{Name: "B8-SW1"}},
		},
		Configuration: &topology_config.TopologyConfig{
			ConfigId: "original-config",
		},
	}

	expected := proto.Clone(original)

	snapshotID, err := saveOptimizationSnapshot(original)
	if err != nil {
		t.Fatal(err)
	}

	original.Configuration.ConfigId = "modified-config"

	restored, err := loadOptimizationSnapshot(snapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(expected, restored) {
		t.Fatal("etcd snapshot differs from original")
	}

	jobID := "b8-test-" + snapshotID
	if err := saveJobSnapshotLink(jobID, snapshotID); err != nil {
		t.Fatal(err)
	}

	restoredByJob, err := loadSnapshotForJob(jobID)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(expected, restoredByJob) {
		t.Fatal("job-linked snapshot differs from original")
	}

	t.Logf("B8 snapshotID=%s jobID=%s", snapshotID, jobID)
}

func TestB8RealEtcdSnapshotSurvivesRestart(t *testing.T) {
	if os.Getenv("B8_VERIFY_RESTART") != "1" {
		t.Skip("restart verification disabled")
	}
	if os.Getenv("ETCD_HOST") != "127.0.0.1" ||
		os.Getenv("ETCD_PORT") != "2379" {
		t.Fatal("B8 requires isolated etcd at 127.0.0.1:2379")
	}

	snapshotID := os.Getenv("B8_SNAPSHOT_ID")
	jobID := os.Getenv("B8_JOB_ID")
	if snapshotID == "" || jobID == "" {
		t.Fatal("snapshot ID and job ID are required")
	}

	expected := &forwardingplane.ForwardingPlaneModel{
		Metadata: &forwardingplane.ModelMetadata{
			ModelId:  "b8-test",
			Revision: 7,
		},
		Topology: &topology.Topology{
			Nodes: []*topology.Node{{Name: "B8-SW1"}},
		},
		Configuration: &topology_config.TopologyConfig{
			ConfigId: "original-config",
		},
	}

	restored, err := loadOptimizationSnapshot(snapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(expected, restored) {
		t.Fatal("snapshot content changed after restart")
	}

	restoredByJob, err := loadSnapshotForJob(jobID)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(expected, restoredByJob) {
		t.Fatal("job-linked snapshot changed after restart")
	}

	t.Log("Original snapshot and job mapping survived etcd restart")
}
