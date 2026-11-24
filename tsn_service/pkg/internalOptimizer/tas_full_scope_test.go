package internalOptimizer

import (
	"reflect"
	"testing"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestPrepareTASFullScopeTask(t *testing.T) {
	var fpm forwardingplane.ForwardingPlaneModel

	// One existing stream and one newly added stream.
	modelJSON := `{
		"streams": [
			{
				"definition": {
					"streamId": {
						"macAddress": "02:00:00:00:00:01",
						"uniqueId": 1
					}
				}
			},
			{
				"definition": {
					"streamId": {
						"macAddress": "02:00:00:00:00:01",
						"uniqueId": 2
					}
				}
			}
		]
	}`

	if err := protojson.Unmarshal([]byte(modelJSON), &fpm); err != nil {
		t.Fatal(err)
	}

	existingID := requestadapter.StreamKey("02:00:00:00:00:01", 1)
	newID := requestadapter.StreamKey("02:00:00:00:00:01", 2)

	// Simulate CNC's ADD_STREAM task.
	task := &optimizer.OptimizationTask{
		Intent: optimizer.Intent_ADD_STREAM,
		Scope: &optimizer.Scope{
			StreamIds: []string{newID},
		},
		ChangePolicy: &optimizer.ChangePolicy{
			AllowPathChanges:        true,
			AllowScheduleChanges:    true,
			AllowQueueChanges:       true,
			PreserveExistingStreams: true,
			MaxChangedStreams:       2,
			MaxConfigurationChanges: 10,
		},
	}

	originalTask := proto.Clone(task)

	options := requestadapter.RequestOptions{}

	// Identify existing and newly added streams.
	if err := setAddStreamChanges(task, &fpm, &options); err != nil {
		t.Fatal(err)
	}

	// Prepare a TAS-specific copy of the task.
	got, err := prepareTASFullScopeTask(
		task, &fpm, options.Streams,
	)
	if err != nil {
		t.Fatal(err)
	}

	// TAS must receive the complete stream set.
	want := []string{existingID, newID}

	if !reflect.DeepEqual(got.GetScope().GetStreamIds(), want) {
		t.Fatalf(
			"unexpected TAS scope: got %v, want %v",
			got.GetScope().GetStreamIds(),
			want,
		)
	}

	// Verify existing/new stream classification.
	if options.Streams[existingID].Change != "none" {
		t.Fatal("existing stream must have change=none")
	}

	if options.Streams[newID].Change != "add" {
		t.Fatal("new stream must have change=add")
	}

	// TAS request must use fixed routing and queues.
	if got.ChangePolicy.GetAllowPathChanges() ||
		got.ChangePolicy.GetAllowQueueChanges() ||
		got.ChangePolicy.GetAllowVlanChanges() ||
		got.ChangePolicy.GetAllowStreamMigration() {
		t.Fatal("TAS request permits unsupported changes")
	}

	// Scheduling and preservation must remain authorized.
	if !got.ChangePolicy.GetAllowScheduleChanges() ||
		!got.ChangePolicy.GetPreserveExistingStreams() {
		t.Fatal("TAS scheduling or preservation permission was lost")
	}

	// Numeric limits must remain unchanged.
	if got.ChangePolicy.GetMaxChangedStreams() != 2 ||
		got.ChangePolicy.GetMaxConfigurationChanges() != 10 {
		t.Fatal("TAS request changed CNC's numeric limits")
	}

	// The original CNC task must remain unchanged.
	if !proto.Equal(task, originalTask) {
		t.Fatal("original CNC optimization task was modified")
	}

	// Reject requests without scheduling permission.
	task.ChangePolicy.AllowScheduleChanges = false

	if _, err := prepareTASFullScopeTask(
		task, &fpm, options.Streams,
	); err == nil {
		t.Fatal("expected rejection when scheduling changes are prohibited")
	}
}
