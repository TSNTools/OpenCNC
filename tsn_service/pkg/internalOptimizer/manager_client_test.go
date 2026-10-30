package internalOptimizer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
