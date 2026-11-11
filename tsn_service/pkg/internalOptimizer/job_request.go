package internalOptimizer

import (
	"encoding/json"
	"fmt"
)

// jobRequestKey keeps the original, submitted OptimizerRequest JSON alongside
// the existing job-ID -> native ForwardingPlaneModel snapshot link.
// The request must be saved as sent; rebuilding it after completion can change
// scope, options, capabilities, or even the original network state.
func jobRequestKey(jobID string) (string, error) {
	key, err := jobSnapshotKey(jobID)
	if err != nil {
		return "", err
	}
	return key + ".request", nil
}

// saveJobRequest stores the exact v1.4 JSON bytes already validated by
// buildOaaSRequestWithDeviceModels. It never logs request contents.
func saveJobRequest(jobID string, requestJSON []byte) error {
	key, err := jobRequestKey(jobID)
	if err != nil {
		return err
	}
	var request map[string]json.RawMessage
	if err := json.Unmarshal(requestJSON, &request); err != nil || request == nil {
		return fmt.Errorf("cannot store invalid original request JSON")
	}
	if string(request["schemaVersion"]) != `"1.4"` ||
		len(request["forwardingPlane"]) == 0 ||
		string(request["forwardingPlane"]) == "null" {
		return fmt.Errorf("cannot store request without v1.4 forwarding-plane context")
	}
	if err := snapshotWrite(append([]byte(nil), requestJSON...), key); err != nil {
		return fmt.Errorf("save original request for Manager job: %w", err)
	}
	return nil
}

// loadJobRequest retrieves the exact submitted request. A missing record is a
// blocking validation error, never grounds for silently reconstructing it.
func loadJobRequest(jobID string) ([]byte, error) {
	key, err := jobRequestKey(jobID)
	if err != nil {
		return nil, err
	}
	requestJSON, err := snapshotRead(key)
	if err != nil {
		return nil, fmt.Errorf("load original request for Manager job: %w", err)
	}
	var request map[string]json.RawMessage
	if err := json.Unmarshal(requestJSON, &request); err != nil || request == nil ||
		string(request["schemaVersion"]) != `"1.4"` ||
		len(request["forwardingPlane"]) == 0 ||
		string(request["forwardingPlane"]) == "null" {
		return nil, fmt.Errorf("stored original request is missing or invalid")
	}
	return append([]byte(nil), requestJSON...), nil
}
