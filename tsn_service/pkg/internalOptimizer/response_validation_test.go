package internalOptimizer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const successfulNoChangeCBSResponse = `{
	"schemaVersion":"1.3",
	"optimizer":{"id":"CBS","name":"CBS","version":"unversioned"},
	"optimizedAgainst":null,
	"status":"success",
	"solution":{"type":"configuration","representation":"partial","changes":{}},
	"outcome":{"feasibility":"feasible","optimality":"notProven","configurationChanged":false},
	"kpis":{},"warnings":[],"errors":[]
}`

// Build a genuine OpenCNC request from the already existing B10 fixture.
// This is a fixture test, not a live Manager or production-network test.
func originalRequestAndSnapshotForResponseTest(t *testing.T) ([]byte, *forwardingplane.ForwardingPlaneModel) {
	t.Helper()
	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "")

	fixtureBytes, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
		ForwardingPlane  json.RawMessage `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	var plane map[string]any
	if err := json.Unmarshal(fixture.ForwardingPlane, &plane); err != nil {
		t.Fatal(err)
	}
	streams, ok := plane["streams"].([]any)
	if !ok {
		t.Fatal("fixture has no streams")
	}
	for _, entry := range streams {
		def, ok := entry.(map[string]any)["definition"].(map[string]any)
		if !ok {
			t.Fatal("invalid fixture definition")
		}
		if oldRank, ok := def["streamRank"].(map[string]any); ok {
			def["streamRank"] = oldRank["rank"]
		}
	}
	planeJSON, err := json.Marshal(plane)
	if err != nil {
		t.Fatal(err)
	}

	var task optimizationcontract.OptimizationTask
	var fpm forwardingplane.ForwardingPlaneModel
	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(planeJSON, &fpm); err != nil {
		t.Fatal(err)
	}
	requestJSON, err := buildOaaSRequest(&task, &fpm)
	if err != nil {
		t.Fatalf("build fixture request: %v", err)
	}
	return requestJSON, &fpm
}

func saveResponseContextForTest(t *testing.T, jobID string, requestJSON []byte, original *forwardingplane.ForwardingPlaneModel) {
	t.Helper()
	snapshotID, err := saveOptimizationSnapshot(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveJobSnapshotLink(jobID, snapshotID); err != nil {
		t.Fatal(err)
	}
	if err := saveJobRequest(jobID, requestJSON); err != nil {
		t.Fatal(err)
	}
}

func TestResponseValidationUsesStoredRequestAndPreservedModel(t *testing.T) {
	useFakeSnapshotStore(t)
	requestJSON, fpm := originalRequestAndSnapshotForResponseTest(t)
	saveResponseContextForTest(t, "job-validated", requestJSON, fpm)

	originalCopy := proto.Clone(fpm)
	validated, err := validateRetrievedOptimizerPayload("job-validated", []byte(successfulNoChangeCBSResponse))
	if err != nil {
		t.Fatal(err)
	}
	if validated.Validation == nil || validated.Validation.Applicable {
		t.Fatalf("valid no-change result must not authorize modifications: %+v", validated.Validation)
	}
	if !proto.Equal(validated.Original, originalCopy) || !proto.Equal(fpm, originalCopy) {
		t.Fatal("validation modified the original forwarding-plane model")
	}
}

func TestResponseValidationRejectsMismatchedNativeSnapshot(t *testing.T) {
	useFakeSnapshotStore(t)
	requestJSON, fpm := originalRequestAndSnapshotForResponseTest(t)
	wrong := proto.Clone(fpm).(*forwardingplane.ForwardingPlaneModel)
	wrong.Topology.Nodes[0].Name = "OTHER-NODE"
	saveResponseContextForTest(t, "job-wrong-snapshot", requestJSON, wrong)

	validated, err := validateRetrievedOptimizerPayload("job-wrong-snapshot", []byte(successfulNoChangeCBSResponse))
	if err == nil || validated != nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("mismatched snapshot must be rejected, got %v", err)
	}
}

func TestResponseValidationRejectsInvalidSchema(t *testing.T) {
	useFakeSnapshotStore(t)
	requestJSON, fpm := originalRequestAndSnapshotForResponseTest(t)
	saveResponseContextForTest(t, "job-bad-response", requestJSON, fpm)

	invalid := strings.Replace(successfulNoChangeCBSResponse, `"schemaVersion":"1.3"`, `"schemaVersion":"2.0"`, 1)
	validated, err := validateRetrievedOptimizerPayload("job-bad-response", []byte(invalid))
	if err == nil || validated != nil {
		t.Fatalf("schema-violating optimizer result must be rejected, got %v", err)
	}
}

func TestResponseValidationRejectsCBSQueueMismatch(t *testing.T) {
	useFakeSnapshotStore(t)
	requestJSON, fpm := originalRequestAndSnapshotForResponseTest(t)
	saveResponseContextForTest(t, "job-queue-mismatch", requestJSON, fpm)

	var candidate map[string]any
	if err := json.Unmarshal([]byte(successfulNoChangeCBSResponse), &candidate); err != nil {
		t.Fatal(err)
	}
	candidate["solution"].(map[string]any)["changes"] = map[string]any{
		"configuration": map[string]any{
			"nodeConfigs": []any{map[string]any{
				"nodeId": "SW1",
				"portConfigs": []any{map[string]any{
					"portId": "s1-out",
					"queueConfigs": []any{map[string]any{
						"queueId": 3,
						"cbs": []any{map[string]any{
							"egressQueueId":     4, // deliberately wrong
							"adminIdleSlopeBps": 1000000,
						}},
					}},
				}},
			}},
		},
	}
	candidate["outcome"].(map[string]any)["configurationChanged"] = true
	responseJSON, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}

	validated, err := validateRetrievedOptimizerPayload("job-queue-mismatch", responseJSON)
	if err == nil || validated != nil || !strings.Contains(err.Error(), "egressQueueId") {
		t.Fatalf("shared semantic validation must reject mismatched CBS queue, got %v", err)
	}
}

func TestResponseValidationRejectsMissingStoredContext(t *testing.T) {
	useFakeSnapshotStore(t)
	validated, err := validateRetrievedOptimizerPayload("missing-job", []byte(successfulNoChangeCBSResponse))
	if err == nil || validated != nil || !strings.Contains(err.Error(), "original request") {
		t.Fatalf("missing request must prevent result validation, got %v", err)
	}
}

func TestManagerPollingAndResponseValidationWithMockHTTP(t *testing.T) {
	useFakeSnapshotStore(t)
	requestJSON, fpm := originalRequestAndSnapshotForResponseTest(t)
	saveResponseContextForTest(t, "job-mock-1", requestJSON, fpm)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/jobs/job-mock-1":
			_, _ = w.Write([]byte(`{"jobId":"job-mock-1","status":"COMPLETED"}`))
		case "/api/v1/jobs/job-mock-1/result":
			_, _ = w.Write([]byte(`{"job":{"jobId":"job-mock-1","status":"COMPLETED"},"optimizerResult":` + successfulNoChangeCBSResponse + `}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")

	validated, err := waitAndValidateOptimizationResult(context.Background(), "job-mock-1")
	if err != nil {
		t.Fatalf("mock retrieval + validation failed: %v", err)
	}
	if validated == nil || validated.Validation == nil || validated.Validation.Applicable {
		t.Fatalf("expected validated no-op result, got %+v", validated)
	}
}
