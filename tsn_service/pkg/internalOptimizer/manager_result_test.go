package internalOptimizer

import (
	"encoding/json"
	"strings"
	"testing"
)

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
