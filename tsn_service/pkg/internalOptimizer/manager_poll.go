package internalOptimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	managerPollInterval     = 500 * time.Millisecond
	managerPollTimeout      = 60 * time.Second
	managerGetTimeout       = 10 * time.Second
	managerMaxResponseBytes = 32 << 20
)

var managerSafeJobID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// managerJobEndpoint constructs a URL beneath OAAS_MANAGER_JOBS_URL, which
// already identifies the Manager's /api/v1/jobs collection. A job ID must
// never be allowed to change the requested URL path.
func managerJobEndpoint(jobID string, result bool) (string, error) {
	if !managerSafeJobID.MatchString(jobID) {
		return "", fmt.Errorf("invalid Manager job ID")
	}
	base := strings.TrimSpace(os.Getenv("OAAS_MANAGER_JOBS_URL"))
	if base == "" {
		return "", fmt.Errorf("OAAS_MANAGER_JOBS_URL is not configured")
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("OAAS_MANAGER_JOBS_URL must be a plain HTTP(S) jobs collection URL")
	}
	if result {
		return url.JoinPath(base, jobID, "result")
	}
	return url.JoinPath(base, jobID)
}

// managerGetJSON retrieves a bounded JSON response without logging raw
// Manager payloads. Failures are reported explicitly rather than treated as
// completed optimization.
func managerGetJSON(ctx context.Context, client *http.Client, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Manager GET request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Manager GET request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Manager GET returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, managerMaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Manager response: %w", err)
	}
	if len(body) > managerMaxResponseBytes {
		return nil, fmt.Errorf("Manager response exceeds %d bytes", managerMaxResponseBytes)
	}
	return body, nil
}

// parseManagerJobStatus accepts the flat status form used for job receipts
// and, if present, the nested job form used in Manager result envelopes.
// In both cases the returned jobId must equal the trusted submitted job ID.
func parseManagerJobStatus(data []byte, expectedJobID string) (string, error) {
	var status struct {
		JobID  string `json:"jobId"`
		Status string `json:"status"`
		Job    *struct {
			JobID  string `json:"jobId"`
			Status string `json:"status"`
		} `json:"job"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return "", fmt.Errorf("invalid Manager status JSON: %w", err)
	}
	if status.Job != nil {
		if status.JobID != "" && status.JobID != status.Job.JobID {
			return "", fmt.Errorf("conflicting Manager job IDs in status response")
		}
		if status.Status != "" && status.Status != status.Job.Status {
			return "", fmt.Errorf("conflicting Manager statuses in status response")
		}
		status.JobID = status.Job.JobID
		status.Status = status.Job.Status
	}
	if expectedJobID == "" || status.JobID != expectedJobID {
		return "", fmt.Errorf("Manager status job ID missing or mismatched")
	}
	if strings.TrimSpace(status.Status) == "" {
		return "", fmt.Errorf("Manager status response is missing status")
	}
	return status.Status, nil
}

func fetchManagerJobStatus(ctx context.Context, client *http.Client, jobID string) (string, error) {
	endpoint, err := managerJobEndpoint(jobID, false)
	if err != nil {
		return "", err
	}
	body, err := managerGetJSON(ctx, client, endpoint)
	if err != nil {
		return "", err
	}
	return parseManagerJobStatus(body, jobID)
}

func fetchCompletedManagerJobResult(ctx context.Context, client *http.Client, jobID string) (json.RawMessage, error) {
	endpoint, err := managerJobEndpoint(jobID, true)
	if err != nil {
		return nil, err
	}
	body, err := managerGetJSON(ctx, client, endpoint)
	if err != nil {
		return nil, err
	}
	// This parser (from the preceding step) checks the trusted job ID and
	// COMPLETED status in the result envelope before extracting optimizerResult.
	return parseCompletedManagerResult(body, jobID)
}

// waitForCompletedManagerResult waits for a completed Manager job and retrieves
// its optimizerResult. It does NOT assert solution success, validate the v1.3
// contract, or apply changes to a forwarding-plane model. Those are later steps.
// Nonterminal states remain pending until a bounded timeout or cancellation.
func waitForCompletedManagerResult(ctx context.Context, jobID string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, managerPollTimeout)
	defer cancel()
	client := &http.Client{Timeout: managerGetTimeout}

	for {
		status, err := fetchManagerJobStatus(ctx, client, jobID)
		if err != nil {
			return nil, fmt.Errorf("Manager job status: %w", err)
		}
		switch status {
		case "COMPLETED":
			return fetchCompletedManagerJobResult(ctx, client, jobID)
		case "FAILED", "CANCELLED", "CANCELED", "TIMED_OUT", "EXPIRED":
			return nil, fmt.Errorf("Manager job %s ended with status %s", jobID, status)
		}

		timer := time.NewTimer(managerPollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, fmt.Errorf("Manager job %s did not complete: %w", jobID, ctx.Err())
		case <-timer.C:
		}
	}
}
