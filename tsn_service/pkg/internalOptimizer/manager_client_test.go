// Test HTTP submission with a mock Manager.
package internalOptimizer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
