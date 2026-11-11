package internalOptimizer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"OpenCNC/common/structures/topology_config"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Uses the existing realistic CNC fixture and a mock Manager, not a live
// OpenCNC deployment or actual optimizer.
func integrationFixtureTask(t *testing.T) *optimizer.OptimizationTask {
	t.Helper()
	data, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	var task optimizer.OptimizationTask
	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	return &task
}

// Synthetic CBS delta is tied to SW1/s1-out/queue 3 in the checked-in fixture.
// Values exercise JSON -> protobuf mapping; they are not measured results.
const mockCBSDelta = `"configuration":{"nodeConfigs":[{"nodeId":"SW1","portConfigs":[{"portId":"s1-out","queueConfigs":[{"queueId":3,"scheduling":"CREDIT_BASED_SHAPER","cbs":[{"egressQueueId":3,"adminState":"ENABLED","adminIdleSlopeBps":10000000,"sendSlopeBps":90000000,"hiCreditBytes":20,"loCreditBytes":100}]}]}]}]}`

func TestExecuteOptimizationJobMockManagerReturnsUpdatedModel(t *testing.T) {
	useFakeSnapshotStore(t)
	_, original := originalRequestAndSnapshotForResponseTest(t)
	before := proto.Clone(original)
	task := integrationFixtureTask(t)
	result := strings.Replace(successfulNoChangeCBSResponse,
		`"changes":{}`, `"changes":{`+mockCBSDelta+`}`, 1)
	result = strings.Replace(result, `"configurationChanged":false`, `"configurationChanged":true`, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"jobId":"job-e2e-1","status":"QUEUED"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/job-e2e-1":
			_, _ = w.Write([]byte(`{"jobId":"job-e2e-1","status":"COMPLETED"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/job-e2e-1/result":
			_, _ = w.Write([]byte(`{"job":{"jobId":"job-e2e-1","status":"COMPLETED"},"optimizerResult":` + result + `}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")

	updated, err := executeOptimizationJob(context.Background(), task, original, nil)
	if err != nil {
		t.Fatalf("complete request-to-return flow: %v", err)
	}
	if updated == original || proto.Equal(updated, original) {
		t.Fatal("expected independently updated forwarding-plane model")
	}
	if !proto.Equal(original, before) {
		t.Fatal("original model was modified")
	}
	if !proto.Equal(updated.GetTopology(), original.GetTopology()) ||
		!proto.Equal(updated.GetMetadata(), original.GetMetadata()) {
		t.Fatal("unrelated topology or metadata was modified")
	}
	if len(updated.GetStreams()) != len(original.GetStreams()) {
		t.Fatal("streams were lost")
	}
	var changed *topology_config.QueueConfig
	for _, n := range updated.GetConfiguration().GetNodeConfigs() {
		if n.GetNodeId() != "SW1" {
			continue
		}
		for _, p := range n.GetPortConfigs() {
			if p.GetPortId() != "s1-out" {
				continue
			}
			for _, q := range p.GetQueueConfigs() {
				if q.GetQueueId() == 3 {
					changed = q
				}
			}
		}
	}
	if changed == nil || changed.GetScheduling() != topology_config.SchedulingModel_CREDIT_BASED_SHAPER ||
		len(changed.GetCbs()) != 1 || changed.GetCbs()[0].GetAdminIdleSlopeBps() != 10000000 {
		t.Fatalf("expected validated CBS delta in returned model, got %+v", changed)
	}
}

func TestExecuteOptimizationJobRejectsFailedManagerJob(t *testing.T) {
	useFakeSnapshotStore(t)
	_, original := originalRequestAndSnapshotForResponseTest(t)
	task := integrationFixtureTask(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"jobId":"job-failed-1","status":"QUEUED"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/job-failed-1":
			_, _ = w.Write([]byte(`{"jobId":"job-failed-1","status":"FAILED"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("OAAS_MANAGER_JOBS_URL", server.URL+"/api/v1/jobs")
	updated, err := executeOptimizationJob(context.Background(), task, original, nil)
	if err == nil || updated != nil || !strings.Contains(err.Error(), "FAILED") {
		t.Fatalf("failed Manager job must not yield a model: model=%v error=%v", updated, err)
	}
}
