package internalOptimizer

import (
	"context"
	"os"
	"testing"
	"time"

	"OpenCNC/common/observability"
	"google.golang.org/protobuf/proto"
)

// TestLiveManagerCBSTimingMetrics is an opt-in R6 measurement smoke test.
// It uses the isolated etcd instance and real Manager/CBS services, never
// OpenCNC's operational etcd. It does not call StartOptimization or deploy.
func TestLiveManagerCBSTimingMetrics(t *testing.T) {
	if os.Getenv("OAAS_R6_TIMING") != "1" {
		t.Skip("opt in with OAAS_R6_TIMING=1")
	}
	if os.Getenv("OAAS_R6_REAL_ETCD") != "1" {
		t.Fatal("timing smoke test requires OAAS_R6_REAL_ETCD=1")
	}
	if os.Getenv("ETCD_HOST") != "127.0.0.1" || os.Getenv("ETCD_PORT") != "12379" {
		t.Fatal("timing smoke test requires isolated etcd at 127.0.0.1:12379")
	}
	if os.Getenv("OAAS_MANAGER_JOBS_URL") == "" {
		t.Setenv("OAAS_MANAGER_JOBS_URL", "http://127.0.0.1:8000/api/v1/jobs")
	}
	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false")

	task := integrationFixtureTask(t)
	_, original := originalRequestAndSnapshotForResponseTest(t)
	originalCopy := proto.Clone(original)

	obs, err := observability.NewFromEnv("tsn-service-r6-timing-test")
	if err != nil {
		t.Fatalf("create existing observability client: %v", err)
	}
	defer func() {
		if err := obs.Close(); err != nil {
			t.Logf("observability close warning: %v", err)
		}
	}()

	recorder := newOAASTimingRecorder(obs)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	updated, err := executeOptimizationJobWithTimings(ctx, task, original, nil, recorder)
	if err != nil {
		t.Fatalf("live R6 workflow with recorder: %v", err)
	}
	if updated == nil || proto.Equal(updated, original) || !proto.Equal(original, originalCopy) {
		t.Fatal("unexpected FPM update or original FPM mutation")
	}
	if recorder.jobID == "" {
		t.Fatal("missing Manager job ID on timing samples")
	}

	want := []string{
		"oaas_request_build_ms", "oaas_snapshot_ms", "oaas_submission_ms",
		"oaas_ack_parse_ms", "oaas_job_bookkeeping_ms", "oaas_wait_ms",
		"oaas_result_retrieve_ms", "oaas_result_parse_ms",
		"oaas_response_validation_ms", "oaas_fpm_apply_ms",
	}
	seen := make(map[string]bool)
	for _, sample := range recorder.samples {
		if sample.status != "success" || sample.duration < 0 {
			t.Fatalf("invalid sample: %+v", sample)
		}
		if seen[sample.name] {
			t.Fatalf("duplicate stage %s", sample.name)
		}
		seen[sample.name] = true
		t.Logf("TIMING %s: %.3f ms", sample.name, float64(sample.duration)/float64(time.Millisecond))
	}
	for _, name := range want {
		if !seen[name] {
			t.Fatalf("missing expected timing sample %s", name)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("got %d samples, want %d", len(seen), len(want))
	}

	// Wait for the asynchronous publisher to finish before this test exits.
	// Completion means an attempt was made, NOT that Kafka stored messages:
	// OBS_FAIL_OPEN and the existing producer can suppress publication errors.
	select {
	case <-recorder.publish():
		t.Logf("Published timing attempts for job %s; Kafka delivery must be checked separately", recorder.jobID)
	case <-time.After(7 * time.Second):
		t.Fatal("timing publisher did not finish within the bounded test window")
	}
}
