package internalOptimizer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestManagerJobEndpoint(t *testing.T) {
	t.Setenv("OAAS_MANAGER_JOBS_URL", "http://example.invalid/api/v1/jobs/")
	for _, tt := range []struct {
		result bool
		want   string
	}{
		{false, "http://example.invalid/api/v1/jobs/job_001"},
		{true, "http://example.invalid/api/v1/jobs/job_001/result"},
	} {
		got, err := managerJobEndpoint("job_001", tt.result)
		if err != nil || got != tt.want {
			t.Fatalf("endpoint=%q, err=%v; expected %q", got, err, tt.want)
		}
	}
	for _, bad := range []string{"", "../other", "job/other", "a?b"} {
		if _, err := managerJobEndpoint(bad, false); err == nil {
			t.Fatalf("job ID %q should be rejected", bad)
		}
	}
}

func TestParseManagerJobStatus(t *testing.T) {
	for _, input := range []string{
		`{"jobId":"job-1","status":"RUNNING"}`,
		`{"job":{"jobId":"job-1","status":"RUNNING"}}`,
	} {
		got, err := parseManagerJobStatus([]byte(input), "job-1")
		if err != nil || got != "RUNNING" {
			t.Fatalf("status=%q err=%v for %s", got, err, input)
		}
	}
	for _, input := range []string{
		`{"jobId":"other","status":"RUNNING"}`,
		`{"status":"RUNNING"}`,
		`{"jobId":"job-1"}`,
		`{"jobId":"job-1","status":"RUNNING","job":{"jobId":"job-1","status":"COMPLETED"}}`,
		`{`,
	} {
		if _, err := parseManagerJobStatus([]byte(input), "job-1"); err == nil {
			t.Fatalf("expected rejection of %s", input)
		}
	}
}

func TestWaitForCompletedManagerResult(t *testing.T) {
	var calls atomic.Int32
	var resultCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/jobs/job-1":
			if r.Method != http.MethodGet {
				t.Errorf("status method %s", r.Method)
			}
			status := "QUEUED"
			if calls.Add(1) > 1 {
				status = "COMPLETED"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jobId": "job-1", "status": status})
		case "/api/v1/jobs/job-1/result":
			resultCalls.Add(1)
			_, _ = w.Write([]byte(`{"job":{"jobId":"job-1","status":"COMPLETED"},"optimizerResult":{"schemaVersion":"1.3","status":"success"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")
	result, err := waitForCompletedManagerResult(context.Background(), "job-1")
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schemaVersion"] != "1.3" || resultCalls.Load() != 1 || calls.Load() != 2 {
		t.Fatalf("unexpected result or call counts: %s status=%d result=%d", result, calls.Load(), resultCalls.Load())
	}
}

func TestWaitForFailedManagerJobDoesNotFetchResult(t *testing.T) {
	var resultCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/result") {
			resultCalls.Add(1)
		}
		_, _ = w.Write([]byte(`{"jobId":"job-1","status":"FAILED"}`))
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")
	_, err := waitForCompletedManagerResult(context.Background(), "job-1")
	if err == nil || !strings.Contains(err.Error(), "FAILED") || resultCalls.Load() != 0 {
		t.Fatalf("failed job must not yield optimizer result: err=%v resultCalls=%d", err, resultCalls.Load())
	}
}

func TestWaitForCompletedManagerResultRejectsWrongResultJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/result") {
			_, _ = w.Write([]byte(`{"job":{"jobId":"other","status":"COMPLETED"},"optimizerResult":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jobId":"job-1","status":"COMPLETED"}`))
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")
	if _, err := waitForCompletedManagerResult(context.Background(), "job-1"); err == nil {
		t.Fatal("mismatched result job ID must be rejected")
	}
}

func TestWaitForManagerResultHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jobId":"job-1","status":"RUNNING"}`))
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := waitForCompletedManagerResult(ctx, "job-1")
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("expected caller deadline to stop polling, got %v", err)
	}
}

func TestManagerHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")
	_, err := waitForCompletedManagerResult(context.Background(), "job-1")
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected status HTTP 500 error, got %v", err)
	}
}
