package internalOptimizer

import (
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

// Consolidated from job_snapshot.go
// links a Manager jobId to the saved snapshotr

const jobSnapshotPrefix = "optimizations.jobs."

func jobSnapshotKey(jobID string) (string, error) {
	if strings.TrimSpace(jobID) == "" {
		return "", fmt.Errorf("job ID is required")
	}

	// Encode the ID so it is safe to use as an etcd key.
	encoded := base64.RawURLEncoding.EncodeToString([]byte(jobID))
	return jobSnapshotPrefix + encoded, nil
}

// link a manager's job id with its preserved CNC snapshot.
func saveJobSnapshotLink(jobID, snapshotID string) error {
	key, err := jobSnapshotKey(jobID)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(snapshotID)
	if err != nil || id.String() != snapshotID {
		return fmt.Errorf("invalid snapshot ID %q", snapshotID)
	}

	return snapshotWrite([]byte(snapshotID), key)
}

// fetch original cnc snapshot for the manager's job id
func loadSnapshotForJob(
	jobID string,
) (*forwardingplane.ForwardingPlaneModel, error) {
	key, err := jobSnapshotKey(jobID)
	if err != nil {
		return nil, err
	}

	data, err := snapshotRead(key)
	if err != nil {
		return nil, fmt.Errorf("find snapshot for job %q: %w", jobID, err)
	}

	return loadOptimizationSnapshot(string(data))
}

// Consolidated from job_request.go
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
