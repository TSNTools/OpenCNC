package internalOptimizer

import (
	"bytes"
	"strings"
	"testing"
)

func TestJobRequestRoundTripPreservesExactBytes(t *testing.T) {
	useFakeSnapshotStore(t)
	requestJSON := []byte(`{ "schemaVersion":"1.4", "forwardingPlane":{"topology":{"nodes":[]}} }`)
	original := append([]byte(nil), requestJSON...)
	if err := saveJobRequest("job-test-123", requestJSON); err != nil {
		t.Fatal(err)
	}
	// Mutating the caller's buffer must not change the stored request.
	requestJSON[0] = 'x'
	loaded, err := loadJobRequest("job-test-123")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(loaded, original) {
		t.Fatalf("stored request differs from submitted bytes: %s", loaded)
	}
	// Mutating the returned buffer must not modify the stored request either.
	loaded[0] = 'x'
	loadedAgain, err := loadJobRequest("job-test-123")
	if err != nil || !bytes.Equal(loadedAgain, original) {
		t.Fatalf("stored request was not preserved: %v", err)
	}
}

func TestJobRequestRejectsMissingOrInvalidContext(t *testing.T) {
	useFakeSnapshotStore(t)
	cases := []struct {
		name string
		job  string
		body string
	}{
		{"missing job", "", `{"schemaVersion":"1.4","forwardingPlane":{}}`},
		{"malformed JSON", "job-1", `{`},
		{"wrong version", "job-1", `{"schemaVersion":"1.3","forwardingPlane":{}}`},
		{"missing forwarding plane", "job-1", `{"schemaVersion":"1.4"}`},
		{"null forwarding plane", "job-1", `{"schemaVersion":"1.4","forwardingPlane":null}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := saveJobRequest(tc.job, []byte(tc.body)); err == nil {
				t.Fatal("expected request storage to reject missing or invalid context")
			}
		})
	}
	if _, err := loadJobRequest("unknown-job"); err == nil || !strings.Contains(err.Error(), "load original request") {
		t.Fatalf("unknown job must fail closed, got: %v", err)
	}
}
