package internalOptimizer

import (
	"fmt"

	store "OpenCNC/common/store-wrapper"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const snapshotPrefix = "optimizations.snapshots."

// Allow storage functions to be replaced during unit tests.
// In production, these use OpenCNC's existing etcd storage.
var (
	snapshotWrite = store.SendToStore
	snapshotRead  = store.GetFromStore
)

// Save the complete original ForwardingPlaneModel.
func saveOptimizationSnapshot(
	model *forwardingplane.ForwardingPlaneModel,
) (string, error) {
	if model == nil {
		return "", fmt.Errorf("cannot snapshot nil forwarding plane")
	}

	data, err := proto.Marshal(model)
	if err != nil {
		return "", fmt.Errorf("serialize snapshot: %w", err)
	}

	snapshotID := uuid.NewString()

	if err := snapshotWrite(
		data,
		snapshotPrefix+snapshotID,
	); err != nil {
		return "", fmt.Errorf("save snapshot: %w", err)
	}

	return snapshotID, nil
}

// Retrieve the complete original ForwardingPlaneModel.
func loadOptimizationSnapshot(
	snapshotID string,
) (*forwardingplane.ForwardingPlaneModel, error) {
	id, err := uuid.Parse(snapshotID)
	if err != nil || id.String() != snapshotID {
		return nil, fmt.Errorf("invalid snapshot ID %q", snapshotID)
	}

	data, err := snapshotRead(snapshotPrefix + snapshotID)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	var model forwardingplane.ForwardingPlaneModel
	if err := proto.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("deserialize snapshot: %w", err)
	}

	return &model, nil
}
