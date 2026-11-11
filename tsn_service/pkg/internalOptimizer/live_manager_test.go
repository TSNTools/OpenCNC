// Submits a real request to the running OaaS Manager.
package internalOptimizer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Existing live-submission test is deliberately unchanged.
func TestLiveManagerSubmission(t *testing.T) {
	if os.Getenv("OAAS_LIVE_TEST") != "1" {
		t.Skip("live Manager test is opt-in")
	}

	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_MANAGER_JOBS_URL", "http://127.0.0.1:8000/api/v1/jobs")

	// Read our realistic four-node, two-stream CNC fixture.
	data, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}

	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
		ForwardingPlane  json.RawMessage `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	// Adapt the old fixture's StreamRank format to OpenCNC main.
	var plane map[string]any
	if err := json.Unmarshal(fixture.ForwardingPlane, &plane); err != nil {
		t.Fatal(err)
	}

	streams, ok := plane["streams"].([]any)
	if !ok {
		t.Fatal("fixture has no streams array")
	}
	for _, raw := range streams {
		def := raw.(map[string]any)["definition"].(map[string]any)
		if oldRank, ok := def["streamRank"].(map[string]any); ok {
			def["streamRank"] = oldRank["rank"]
		}
	}

	planeJSON, err := json.Marshal(plane)
	if err != nil {
		t.Fatal(err)
	}

	var task optimizer.OptimizationTask
	var fpm forwardingplane.ForwardingPlaneModel

	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(planeJSON, &fpm); err != nil {
		t.Fatal(err)
	}

	// Simulated etcd; the Manager itself is real.
	useFakeSnapshotStore(t)

	// Actual request conversion and submission to the live Manager.
	receipt, snapshotID, err := submitOptimizationJob(
		context.Background(), &task, &fpm, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	restored, err := loadSnapshotForJob(receipt.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(&fpm, restored) {
		t.Fatal("preserved CNC snapshot does not match original")
	}

	t.Logf("Manager jobId: %s", receipt.JobID)
	t.Logf("Manager status: %s", receipt.Status)
	t.Logf("Snapshot ID: %s", snapshotID)
}

// TestLiveManagerCBSInitialConfigurationCycle controls the missing CNC trigger
// and input fixture only. The Go adapter, shared request validation, Manager
// HTTP endpoint and Manager selection/execution are real (verify remote registration).
// It intentionally does not test etcd persistence, response-contract validation,
// forwarding-plane mutation, or a live CNC notification.
func TestLiveManagerCBSInitialConfigurationCycle(t *testing.T) {
	if os.Getenv("OAAS_LIVE_CBS_CYCLE") != "1" {
		t.Skip("opt in with OAAS_LIVE_CBS_CYCLE=1")
	}

	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false") // CBS requires fixed routes.

	jobsURL := os.Getenv("OAAS_MANAGER_JOBS_URL")
	if jobsURL == "" {
		jobsURL = "http://127.0.0.1:8000/api/v1/jobs"
	}
	parsedURL, err := url.Parse(jobsURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		t.Fatalf("invalid Manager jobs URL: %q", jobsURL)
	}
	t.Setenv("OAAS_MANAGER_JOBS_URL", jobsURL)

	// Control the unavailable initial-configuration trigger and CNC fixture.
	// Use the same CNC protobuf fixture as TestLiveManagerSubmission.
	fixtureJSON, err := os.ReadFile("testdata/realistic_cnc_request.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		OptimizationTask json.RawMessage `json:"optimizationTask"`
		ForwardingPlane  json.RawMessage `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(fixtureJSON, &fixture); err != nil {
		t.Fatal(err)
	}
	var plane map[string]any
	if err := json.Unmarshal(fixture.ForwardingPlane, &plane); err != nil {
		t.Fatal(err)
	}
	streams, ok := plane["streams"].([]any)
	if !ok || len(streams) == 0 {
		t.Fatal("fixture needs CNC streams")
	}
	for _, raw := range streams {
		stream, ok := raw.(map[string]any)
		if !ok {
			t.Fatal("invalid stream in fixture")
		}
		definition, ok := stream["definition"].(map[string]any)
		if !ok {
			t.Fatal("invalid stream definition in fixture")
		}
		if oldRank, ok := definition["streamRank"].(map[string]any); ok {
			definition["streamRank"] = oldRank["rank"]
		}
	}
	planeJSON, err := json.Marshal(plane)
	if err != nil {
		t.Fatal(err)
	}
	var task optimizer.OptimizationTask
	var fpm forwardingplane.ForwardingPlaneModel
	if err := protojson.Unmarshal(fixture.OptimizationTask, &task); err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(planeJSON, &fpm); err != nil {
		t.Fatal(err)
	}
	if task.GetIntent() != optimizer.Intent_INITIAL_CONFIGURATION {
		t.Fatalf("expected INITIAL_CONFIGURATION task; got %s", task.GetIntent())
	}

	// Preserve exact bytes (no pretty-printing) for an auditable local trace.
	traceDir, err := os.MkdirTemp("", "oaas-live-cbs-cycle-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TRACE DIRECTORY: %s", traceDir)
	save := func(filename string, contents []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(traceDir, filename), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}

	// REAL OpenCNC adapter + shared Go request validation.
	requestJSON, err := buildOaaSRequestWithDeviceModels(&task, &fpm, nil)
	if err != nil {
		t.Fatalf("Go request adapter/validation failed: %v", err)
	}
	save("01_optimizer_request_sent_to_manager.json", requestJSON)
	var request struct {
		SchemaVersion    string `json:"schemaVersion"`
		Operation        string `json:"operation"`
		OptimizationTask struct {
			Intent string `json:"intent"`
		} `json:"optimizationTask"`
	}
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		t.Fatal(err)
	}
	if request.Operation != "configuration" || request.OptimizationTask.Intent != "INITIAL_CONFIGURATION" {
		t.Fatalf("wrong Go request operation/intent: %q/%q", request.Operation, request.OptimizationTask.Intent)
	}
	t.Logf("Go built and validated request: schema=%s, operation=%s, intent=%s, bytes=%d",
		request.SchemaVersion, request.Operation, request.OptimizationTask.Intent, len(requestJSON))

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// REAL OpenCNC HTTP client directly posts EXACTLY the above bytes.
	// Deliberately call the client rather than submitOptimizationJob so that
	// its raw Manager acknowledgment is available for inspection.
	ackJSON, err := submitOaaSRequest(ctx, requestJSON)
	if err != nil {
		save("02_manager_submission_error.txt", []byte(err.Error()))
		t.Fatalf("real Manager submission failed: %v", err)
	}
	save("02_manager_ack_received_in_go.json", ackJSON)
	receipt, err := parseManagerJobReceipt(ackJSON)
	if err != nil {
		t.Fatalf("Go could not parse Manager acknowledgment: %v", err)
	}
	t.Logf("Go received Manager acknowledgement: jobId=%s, status=%s", receipt.JobID, receipt.Status)

	// Real GET requests from Go to the Manager. No fake HTTP server/proxy.
	client := &http.Client{Timeout: 15 * time.Second}
	getManager := func(endpoint string) (int, []byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return 0, nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		return resp.StatusCode, body, err
	}
	jobURL := strings.TrimRight(jobsURL, "/") + "/" + url.PathEscape(receipt.JobID)

	completed := false
	for attempt := 0; attempt < 90; attempt++ {
		code, body, err := getManager(jobURL)
		if err != nil {
			t.Fatalf("real Manager job status request failed: %v", err)
		}
		save("03_latest_manager_status_received_in_go.json", body)
		if code != http.StatusOK {
			t.Fatalf("Manager status HTTP %d: %s", code, body)
		}
		var job struct {
			JobID     string          `json:"jobId"`
			Status    string          `json:"status"`
			Optimizer string          `json:"optimizer"`
			Error     json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(body, &job); err != nil {
			t.Fatalf("Go could not parse Manager job status: %v", err)
		}
		if job.JobID != receipt.JobID {
			t.Fatalf("Manager job ID mismatch: ack=%s, status=%s", receipt.JobID, job.JobID)
		}
		switch job.Status {
		case "COMPLETED":
			if job.Optimizer != "CBS" {
				t.Fatalf("Manager completed using unexpected optimizer %q (expected CBS)", job.Optimizer)
			}
			t.Logf("Manager job completed using selected optimizer: %s", job.Optimizer)
			completed = true
		case "FAILED", "CANCELLED", "CANCELED", "TIMED_OUT":
			t.Fatalf("Manager job %s; error=%s; see JSON trace", job.Status, job.Error)
		case "QUEUED", "RUNNING", "CANCEL_REQUESTED":
			// Still running; do not confuse HTTP 202 with optimizer success.
		default:
			t.Fatalf("unexpected Manager job status: %q", job.Status)
		}
		if completed {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Manager job not complete before timeout: %v", ctx.Err())
		case <-time.After(time.Second):
		}
	}
	if !completed {
		t.Fatal("Manager did not complete after 90 status checks; review trace")
	}

	resultCode, resultJSON, err := getManager(jobURL + "/result")
	if err != nil {
		t.Fatalf("real Manager result GET failed: %v", err)
	}
	save("04_manager_result_received_in_go.json", resultJSON)
	if resultCode != http.StatusOK {
		t.Fatalf("Manager result HTTP %d: %s", resultCode, resultJSON)
	}

	// Only identify the envelope shape and whether Go received a nested result.
	// Do NOT substitute this for the planned shared response contract validator.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(resultJSON, &envelope); err != nil {
		t.Fatalf("Go could not parse Manager result JSON: %v", err)
	}
	var nested json.RawMessage
	if payload, ok := envelope["optimizerResult"]; ok {
		// One known Manager response envelope; no schema version is assumed.
		nested = payload
		var job struct {
			JobID  string `json:"jobId"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(envelope["job"], &job); err != nil {
			t.Fatalf("invalid Manager result job metadata: %v", err)
		}
		if job.JobID != receipt.JobID || job.Status != "COMPLETED" {
			t.Fatalf("Manager result refers to wrong/non-completed job: %+v", job)
		}
	} else if payload, ok := envelope["result"]; ok {
		// A fallback Manager result envelope for other response versions.
		nested = payload
		var job struct {
			JobID  string `json:"jobId"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(resultJSON, &job); err != nil {
			t.Fatal(err)
		}
		if job.JobID != receipt.JobID || job.Status != "COMPLETED" {
			t.Fatalf("Manager result refers to wrong/non-completed job: %+v", job)
		}
	} else {
		// If the Manager has been migrated to a new top-level response
		// envelope, preserve the entire returned JSON for future validation.
		// Do not reject unknown versions solely for changing the wrapper.
		nested = resultJSON
		t.Log("Manager returned an unfamiliar result envelope; inspect the saved raw JSON")
	}
	if len(nested) == 0 || string(nested) == "null" || !json.Valid(nested) {
		t.Fatal("Manager result did not contain a usable optimizer result JSON")
	}
	var optimizerResult struct {
		Status        string `json:"status"`
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(nested, &optimizerResult); err != nil {
		t.Fatalf("Go could not parse nested CBS JSON: %v", err)
	}
	t.Logf("Go received CBS result: bytes=%d, reported schema=%q, reported status=%q",
		len(nested), optimizerResult.SchemaVersion, optimizerResult.Status)
	if optimizerResult.Status == "failed" || optimizerResult.Status == "noSolutionFound" {
		t.Fatalf("CBS reports unsuccessful optimization: %s (see raw result)", optimizerResult.Status)
	}
	t.Log("PASS: Go sent its JSON to the real Manager, and retrieved the completed CBS-named job result")
	t.Log("Confirm the Manager selected the external real-cbs container via its registration/logs")
	t.Logf("TRACE DIRECTORY: %s", traceDir)
	t.Log("NOT TESTED: CNC notification, etcd snapshot, independent response validation, FPM apply")
}

// TestLiveManagerCBSFullWorkflowReturnsUpdatedFPM is a controlled, opt-in
// integration test of the actual internalOptimizer orchestration function.
// It provides the CNC task/model fixture (the real initial trigger is absent),
// and replaces ONLY OpenCNC's snapshot store with an in-memory fake.
// Manager HTTP, optimizer selection/execution, shared v0.2.0 response
// validation and OpenCNC FPM delta application are exercised by production code.
// This is NOT a production-trigger, etcd, hardware or deployment test.
func TestLiveManagerCBSFullWorkflowReturnsUpdatedFPM(t *testing.T) {
	if os.Getenv("OAAS_LIVE_CBS_FULL_FLOW") != "1" {
		t.Skip("opt in with OAAS_LIVE_CBS_FULL_FLOW=1")
	}

	jobsURL := os.Getenv("OAAS_MANAGER_JOBS_URL")
	if jobsURL == "" {
		jobsURL = "http://127.0.0.1:8000/api/v1/jobs"
	}
	parsedURL, err := url.Parse(jobsURL)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		t.Fatalf("invalid Manager jobs URL: %q", jobsURL)
	}
	t.Setenv("OAAS_MANAGER_JOBS_URL", jobsURL)
	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false")

	// Use the same checked-in CNC fixture as the existing live request test.
	// This helper adapts the fixture's older StreamRank representation.
	task := integrationFixtureTask(t)
	_, original := originalRequestAndSnapshotForResponseTest(t)
	if task.GetIntent() != optimizer.Intent_INITIAL_CONFIGURATION {
		t.Fatalf("expected initial configuration, got %s", task.GetIntent())
	}
	before := proto.Clone(original).(*forwardingplane.ForwardingPlaneModel)

	// Only the etcd-backed Go snapshot read/write functions are replaced.
	// Capture the independently issued job ID from its saved request key.
	useFakeSnapshotStore(t)
	fakeWrite := snapshotWrite
	var jobID string
	snapshotWrite = func(data []byte, key string) error {
		if strings.HasPrefix(key, jobSnapshotPrefix) && strings.HasSuffix(key, ".request") {
			encoded := strings.TrimSuffix(strings.TrimPrefix(key, jobSnapshotPrefix), ".request")
			decoded, err := base64.RawURLEncoding.DecodeString(encoded)
			if err != nil {
				return err
			}
			jobID = string(decoded)
		}
		return fakeWrite(data, key)
	}
	t.Cleanup(func() { snapshotWrite = fakeWrite })

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	updated, err := executeOptimizationJob(ctx, task, original, nil)
	if err != nil {
		t.Fatalf("live Manager/CBS -> shared validation -> Go FPM apply: %v", err)
	}
	if updated == nil || updated == original || proto.Equal(updated, original) {
		t.Fatal("expected a distinct, changed forwarding-plane model")
	}
	if !proto.Equal(original, before) {
		t.Fatal("the caller's original forwarding-plane model was modified")
	}
	if jobID == "" {
		t.Fatal("production workflow did not save a request linked to a Manager job")
	}
	requestJSON, err := loadJobRequest(jobID)
	if err != nil || len(requestJSON) == 0 {
		t.Fatalf("original submitted request not preserved for job %s: %v", jobID, err)
	}
	preserved, err := loadSnapshotForJob(jobID)
	if err != nil || !proto.Equal(preserved, before) {
		t.Fatalf("original native model snapshot missing or altered for job %s: %v", jobID, err)
	}

	// Independently inspect the completed Manager envelope to establish
	// which optimizer the live Manager selected for this particular job.
	resultURL, err := managerJobEndpoint(jobID, true)
	if err != nil {
		t.Fatal(err)
	}
	envelopeBytes, err := managerGetJSON(ctx, &http.Client{Timeout: 10 * time.Second}, resultURL)
	if err != nil {
		t.Fatalf("retrieve completed Manager envelope for audit: %v", err)
	}
	var envelope struct {
		Job struct {
			JobID  string `json:"jobId"`
			Status string `json:"status"`
		} `json:"job"`
		Selection struct {
			SelectedOptimizerID string `json:"selectedOptimizerId"`
		} `json:"selection"`
		OptimizerResult struct {
			SchemaVersion string `json:"schemaVersion"`
			Status        string `json:"status"`
		} `json:"optimizerResult"`
	}
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		t.Fatalf("decode Manager envelope: %v", err)
	}
	if envelope.Job.JobID != jobID || envelope.Job.Status != "COMPLETED" ||
		envelope.Selection.SelectedOptimizerID != "CBS" ||
		envelope.OptimizerResult.SchemaVersion != "1.3" ||
		envelope.OptimizerResult.Status != "success" {
		t.Fatalf("unexpected Manager job identity/selection/result: %+v", envelope)
	}

	// Prove that untouched FPM state is preserved: strip only CBS and
	// scheduling fields on a clone of the returned model, and drop newly
	// created queue configuration entries that contain CBS settings only.
	normalized := proto.Clone(updated).(*forwardingplane.ForwardingPlaneModel)
	originalNodes := make(map[string]*topology_config.NodeConfig)
	for _, n := range original.GetConfiguration().GetNodeConfigs() {
		originalNodes[n.GetNodeId()] = n
	}
	changedQueues := 0
	for _, n := range normalized.GetConfiguration().GetNodeConfigs() {
		oldNode := originalNodes[n.GetNodeId()]
		if oldNode == nil {
			t.Fatalf("unexpected added node configuration %s", n.GetNodeId())
		}
		originalPorts := make(map[string]*topology_config.PortConfig)
		for _, p := range oldNode.GetPortConfigs() {
			originalPorts[p.GetPortId()] = p
		}
		for _, p := range n.GetPortConfigs() {
			oldPort := originalPorts[p.GetPortId()]
			if oldPort == nil {
				t.Fatalf("unexpected added port configuration %s/%s", n.GetNodeId(), p.GetPortId())
			}
			originalQueues := make(map[uint32]*topology_config.QueueConfig)
			for _, q := range oldPort.GetQueueConfigs() {
				originalQueues[q.GetQueueId()] = q
			}
			kept := make([]*topology_config.QueueConfig, 0, len(p.GetQueueConfigs()))
			for _, q := range p.GetQueueConfigs() {
				oldQueue, existing := originalQueues[q.GetQueueId()]
				if q.GetScheduling() != topology_config.SchedulingModel_CREDIT_BASED_SHAPER && !existing {
					t.Fatalf("new queue %s/%s/%d is not CBS", n.GetNodeId(), p.GetPortId(), q.GetQueueId())
				}
				for _, cbs := range q.GetCbs() {
					if cbs.GetEgressQueueId() != q.GetQueueId() {
						t.Fatalf("CBS queue identity mismatch at %s/%s/%d", n.GetNodeId(), p.GetPortId(), q.GetQueueId())
					}
				}
				if !existing {
					if len(q.GetCbs()) == 0 || q.GetMaxFrameSize() != 0 || q.GetShaperRateBps() != 0 {
						t.Fatalf("new queue %s/%s/%d contains unsupported/empty changes", n.GetNodeId(), p.GetPortId(), q.GetQueueId())
					}
					changedQueues++
					continue
				}
				if !proto.Equal(q, oldQueue) {
					changedQueues++
				}
				// Restore only the fields that the Phase 1 CBS adapter can
				// change; everything else must exactly match the baseline.
				q.Cbs = oldQueue.Cbs
				q.Scheduling = oldQueue.Scheduling
				kept = append(kept, q)
			}
			p.QueueConfigs = kept
		}
	}
	if !proto.Equal(normalized, before) {
		t.Fatal("returned FPM changed topology, streams, or non-CBS configuration")
	}
	if changedQueues == 0 {
		t.Fatal("CBS returned no effective configuration changes")
	}
	t.Logf("PASS: real Manager job %s selected CBS and returned v1.3 success", jobID)
	t.Log("PASS: OpenCNC used its production submission, retrieval, shared Go response validation, and application workflow")
	t.Logf("PASS: %d CBS queue configurations changed; original FPM and unrelated fields preserved", changedQueues)
	t.Log("NOT TESTED: actual CNC trigger, real etcd persistence, physical deployment, independent feasibility/hardware limits")
}
