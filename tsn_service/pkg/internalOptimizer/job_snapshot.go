package internalOptimizer

import (
	"encoding/base64"
	"fmt"
	"strings"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"github.com/google/uuid"
)

const jobSnapshotPrefix = "optimizations.jobs."

func jobSnapshotKey(jobID string) (string, error) {
	if strings.TrimSpace(jobID) == "" {
		return "", fmt.Errorf("job ID is required")
	}

	// Encode the ID so it is safe to use as an etcd key.
	encoded := base64.RawURLEncoding.EncodeToString([]byte(jobID))
	return jobSnapshotPrefix + encoded, nil
}

// Associate a Manager job with its preserved CNC snapshot.
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

// Retrieve the original CNC model belonging to a Manager job.
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
