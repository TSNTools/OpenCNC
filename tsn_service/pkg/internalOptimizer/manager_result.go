package internalOptimizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ManagerJobResult holds only the fields needed to extract a completed job's
// optimizer response. It is a Manager envelope, not an OptimizerResponse v1.3.
// Extraction does not validate the optimizer response or authorize application.
type ManagerJobResult struct {
	Job struct {
		JobID  string `json:"jobId"`
		Status string `json:"status"`
	} `json:"job"`
	OptimizerResult json.RawMessage `json:"optimizerResult"`
}

// parseCompletedManagerResult extracts the raw optimizer payload only after
// checking the trusted Manager job identifier and completed state.
// The caller MUST subsequently apply shared structural/semantic validation
// against the original request and preserved forwarding-plane model.
func parseCompletedManagerResult(data []byte, expectedJobID string) (json.RawMessage, error) {
	if strings.TrimSpace(expectedJobID) == "" {
		return nil, fmt.Errorf("expected Manager job ID is required")
	}

	var result ManagerJobResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid Manager job result JSON: %w", err)
	}
	if strings.TrimSpace(result.Job.JobID) == "" || result.Job.JobID != expectedJobID {
		return nil, fmt.Errorf("Manager job ID is missing or does not match the submitted job")
	}
	if result.Job.Status != "COMPLETED" {
		return nil, fmt.Errorf("Manager job %s is not completed (status: %s)", expectedJobID, result.Job.Status)
	}
	payload := bytes.TrimSpace(result.OptimizerResult)
	if len(payload) == 0 || bytes.Equal(payload, []byte("null")) || payload[0] != '{' {
		return nil, fmt.Errorf("completed Manager job is missing an optimizer result object")
	}

	return append(json.RawMessage(nil), payload...), nil
}
