package internalOptimizer

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestLiveManagerSubmission(t *testing.T) {
	if os.Getenv("OAAS_LIVE_TEST") != "1" {
		t.Skip("live Manager test is opt-in")
	}

	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_MANAGER_JOBS_URL", "http://127.0.0.1:8000/api/v1/jobs")

	// Read our realistic four-node, two-stream CNC fixture.
	data, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}

	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
		ForwardingPlane  json.RawMessage `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	// Adapt the old fixture's StreamRank format to OpenCNC main.
	var plane map[string]any
	if err := json.Unmarshal(fixture.ForwardingPlane, &plane); err != nil {
		t.Fatal(err)
	}

	streams, ok := plane["streams"].([]any)
	if !ok {
		t.Fatal("fixture has no streams array")
	}
	for _, raw := range streams {
		def := raw.(map[string]any)["definition"].(map[string]any)
		if oldRank, ok := def["streamRank"].(map[string]any); ok {
			def["streamRank"] = oldRank["rank"]
		}
	}

	planeJSON, err := json.Marshal(plane)
	if err != nil {
		t.Fatal(err)
	}

	var task optimizer.OptimizationTask
	var fpm forwardingplane.ForwardingPlaneModel

	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(planeJSON, &fpm); err != nil {
		t.Fatal(err)
	}

	// Simulated etcd; the Manager itself is real.
	useFakeSnapshotStore(t)

	// Actual request conversion and submission to the live Manager.
	receipt, snapshotID, err := submitOptimizationJob(
		context.Background(), &task, &fpm,
	)
	if err != nil {
		t.Fatal(err)
	}

	restored, err := loadSnapshotForJob(receipt.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(&fpm, restored) {
		t.Fatal("preserved CNC snapshot does not match original")
	}

	t.Logf("Manager jobId: %s", receipt.JobID)
	t.Logf("Manager status: %s", receipt.Status)
	t.Logf("Snapshot ID: %s", snapshotID)
}
