package internalOptimizer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Consolidated from manager_client_test.go
// Test HTTP submission with a mock Manager.

func TestSubmitOaaSRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"schemaVersion":"1.4"}` {
				t.Errorf("unexpected request body: %s", body)
			}

			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{"jobId":"test-001"}`))
		},
	))
	defer server.Close()

	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL)

	result, err := submitOaaSRequest(
		context.Background(),
		[]byte(`{"schemaVersion":"1.4"}`),
	)

	if err != nil {
		t.Fatal(err)
	}

	if string(result) != `{"jobId":"test-001"}` {
		t.Fatalf("unexpected response: %s", result)
	}
}

func TestSubmitOaaSRequestRejectsHTTP422(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(
				`{"detail":{"code":"INVALID_OPTIMIZER_REQUEST"}}`,
			))
		},
	))
	defer server.Close()

	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL)

	result, err := submitOaaSRequest(
		context.Background(),
		[]byte(`{"schemaVersion":"1.4"}`),
	)

	if err == nil {
		t.Fatal("expected HTTP 422 error")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %s", result)
	}

	if !strings.Contains(err.Error(), "HTTP 422") ||
		!strings.Contains(err.Error(), "INVALID_OPTIMIZER_REQUEST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubmitOaaSRequestRejectsHTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail":"Internal Server Error"}`))
		},
	))
	defer server.Close()

	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL)

	result, err := submitOaaSRequest(
		context.Background(),
		[]byte(`{"schemaVersion":"1.4"}`),
	)

	if err == nil {
		t.Fatal("expected HTTP 500 error")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %s", result)
	}

	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected HTTP 500 in error, got: %v", err)
	}
}
func TestSubmitOaaSRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusAccepted)
		},
	))
	defer server.Close()

	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		50*time.Millisecond,
	)
	defer cancel()

	result, err := submitOaaSRequest(
		ctx,
		[]byte(`{"schemaVersion":"1.4"}`),
	)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result, got: %s", result)
	}
}

func TestParseManagerJobReceiptRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			"malformed JSON",
			`{"jobId":`,
			"invalid Manager response",
		},
		{
			"missing job ID",
			`{"status":"QUEUED"}`,
			"Manager response missing jobId",
		},
		{
			"missing status",
			`{"jobId":"job-001"}`,
			"Manager response missing status",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			receipt, err := parseManagerJobReceipt(
				[]byte(tc.response),
			)

			if err == nil || !strings.Contains(err.Error(), tc.expected) {
				t.Fatalf("expected %q, got %v", tc.expected, err)
			}

			if receipt != nil {
				t.Fatalf("expected nil receipt, got %+v", receipt)
			}
		})
	}
}

// Test submission and receipt parsing together
// This verifies that OpenCNC correctly handles a successful HTTP 202 acknowledgement and extracts the Manager's job ID and status.

// It does not test real Manager execution or snapshot persistence.
func TestSubmitAndParseManagerReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(
				`{"jobId":"job-001","status":"QUEUED"}`,
			))
		},
	))
	defer server.Close()

	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL)

	response, err := submitOaaSRequest(
		context.Background(),
		[]byte(`{"schemaVersion":"1.4"}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	receipt, err := parseManagerJobReceipt(response)
	if err != nil {
		t.Fatal(err)
	}

	if receipt.JobID != "job-001" || receipt.Status != "QUEUED" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
}

// OpenCNC must reject submission when OAAS_MANAGER_JOBS_URL is not configured.
func TestSubmitOaaSRequestMissingURL(t *testing.T) {
	t.Setenv("OAAS_MANAGER_JOBS_URL", "")

	result, err := submitOaaSRequest(
		context.Background(),
		[]byte(`{"schemaVersion":"1.4"}`),
	)

	if err == nil {
		t.Fatal("expected missing configuration error")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %s", result)
	}

	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// This tests connection failure using a closed local mock server. It does not require a running OaaS Manager.
func TestSubmitOaaSRequestConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	url := server.URL
	server.Close() // Simulate an unreachable Manager.

	t.Setenv("OAAS_MANAGER_JOBS_URL", url)

	result, err := submitOaaSRequest(
		context.Background(),
		[]byte(`{"schemaVersion":"1.4"}`),
	)

	if err == nil {
		t.Fatal("expected connection failure")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %s", result)
	}

	if !strings.Contains(err.Error(), "OaaS Manager request failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Consolidated from manager_poll_test.go
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

// Consolidated from manager_result_test.go
func TestParseCompletedManagerResultExtractsOnlyOptimizerPayload(t *testing.T) {
	input := []byte(`{
		"job":{"jobId":"job-001","status":"COMPLETED"},
		"optimizerResult":{"schemaVersion":"1.3","status":"success"},
		"validation":{"status":"notRun","deployable":null},
		"finalStatus":"notEvaluated"
	}`)
	payload, err := parseCompletedManagerResult(input, "job-001")
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schemaVersion"] != "1.3" || decoded["status"] != "success" {
		t.Fatalf("unexpected optimizer result: %s", payload)
	}
	if _, ok := decoded["job"]; ok {
		t.Fatal("extracted optimizer result must not include Manager job envelope")
	}
}

func TestParseCompletedManagerResultRejectsUnsafeInputs(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		jobID    string
		contains string
	}{
		{"missing trusted job ID", `{}`, "", "required"},
		{"malformed JSON", `{`, "job-001", "invalid Manager job result"},
		{"missing result job ID", `{"job":{"status":"COMPLETED"},"optimizerResult":{}}`, "job-001", "job ID"},
		{"mismatched job", `{"job":{"jobId":"other","status":"COMPLETED"},"optimizerResult":{}}`, "job-001", "job ID"},
		{"queued job", `{"job":{"jobId":"job-001","status":"QUEUED"},"optimizerResult":{}}`, "job-001", "not completed"},
		{"failed job", `{"job":{"jobId":"job-001","status":"FAILED"},"optimizerResult":{}}`, "job-001", "not completed"},
		{"missing optimizer result", `{"job":{"jobId":"job-001","status":"COMPLETED"}}`, "job-001", "result object"},
		{"null optimizer result", `{"job":{"jobId":"job-001","status":"COMPLETED"},"optimizerResult":null}`, "job-001", "result object"},
		{"wrong result shape", `{"job":{"jobId":"job-001","status":"COMPLETED"},"optimizerResult":[1]}`, "job-001", "result object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCompletedManagerResult([]byte(tt.payload), tt.jobID)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("expected error containing %q, got %v", tt.contains, err)
			}
			if result != nil {
				t.Fatalf("expected nil result on error, got %s", result)
			}
		})
	}
}
