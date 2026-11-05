package internalOptimizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// This helper only submits a job. It does not treat 202 Accepted as optimization success.
// will handle job completion separately.
func submitOaaSRequest(ctx context.Context, requestJSON []byte) ([]byte, error) {
	url := os.Getenv("OAAS_MANAGER_JOBS_URL")
	if url == "" {
		return nil, fmt.Errorf("OAAS_MANAGER_JOBS_URL is not configured")
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, url, bytes.NewReader(requestJSON),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OaaS Manager request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("OaaS Manager returned HTTP %d: %s",
			resp.StatusCode, body)
	}

	return body, nil
}

type ManagerJobReceipt struct {
	Status string `json:"status"`
	JobID  string `json:"jobId"`
}

func parseManagerJobReceipt(data []byte) (*ManagerJobReceipt, error) {
	var receipt ManagerJobReceipt

	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, fmt.Errorf("invalid Manager response: %w", err)
	}

	if strings.TrimSpace(receipt.JobID) == "" {
		return nil, fmt.Errorf("Manager response missing jobId")
	}

	if strings.TrimSpace(receipt.Status) == "" {
		return nil, fmt.Errorf("Manager response missing status")
	}

	return &receipt, nil
}
