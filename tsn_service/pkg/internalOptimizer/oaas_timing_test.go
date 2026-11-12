package internalOptimizer

import (
	"testing"
	"time"
)

func TestOAASTimingRecorderStages(t *testing.T) {
	recorder := &oaasTimingRecorder{runID: "test-run"}
	start := time.Now().Add(-time.Millisecond)
	recorder.record("oaas_request_build_ms", start, true)
	recorder.record("oaas_response_validation_ms", start, false)
	if len(recorder.samples) != 2 {
		t.Fatalf("got %d timing samples, want 2", len(recorder.samples))
	}
	if recorder.samples[0].name != "oaas_request_build_ms" || recorder.samples[0].status != "success" {
		t.Fatalf("unexpected success sample: %+v", recorder.samples[0])
	}
	if recorder.samples[1].status != "failed" {
		t.Fatalf("unexpected failure status: %+v", recorder.samples[1])
	}
	if recorder.samples[0].duration <= 0 || recorder.samples[0].end.Before(start) {
		t.Fatalf("invalid measurement: %+v", recorder.samples[0])
	}
}

func TestNilOAASTimingRecorderIsSafe(t *testing.T) {
	var recorder *oaasTimingRecorder
	recorder.record("oaas_cnc_total_ms", time.Now(), false)
	recorder.publish()
}
