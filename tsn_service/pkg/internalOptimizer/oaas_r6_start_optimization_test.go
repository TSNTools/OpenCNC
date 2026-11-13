package internalOptimizer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"OpenCNC/common/observability"
	"OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/stream"
	"OpenCNC/common/structures/stream_config"
	"OpenCNC/common/structures/topology"
	"OpenCNC/common/structures/topology_config"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
	optimizer "OpenCNC/tsn_service/pkg/structures/optimization_contract"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
)

// TestR6StartOptimizationRealEtcd extends the existing R6 controlled test.
// It seeds the SAME isolated etcd with a valid CNC fixture and invokes the
// production StartOptimization entry point (including its data-gathering path).
// It does NOT exercise GUI STREAM_ADDED, a deployed CNC notification, or hardware.
func TestR6StartOptimizationRealEtcd(t *testing.T) {
	if os.Getenv("OAAS_R6_START_OPT") != "1" {
		t.Skip("opt in with OAAS_R6_START_OPT=1")
	}
	if os.Getenv("OAAS_R6_REAL_ETCD") != "1" ||
		os.Getenv("ETCD_HOST") != "127.0.0.1" ||
		os.Getenv("ETCD_PORT") != "12379" {
		t.Fatal("requires OAAS_R6_REAL_ETCD=1 and isolated etcd at 127.0.0.1:12379")
	}

	t.Setenv("OAAS_FULL_FLOW", "true")
	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_B10_OPTIMIZE_ROUTING", "false") // Valid R6 fixture has fixed paths.
	t.Setenv("OBS_ENABLED", "true")
	t.Setenv("OBS_KAFKA_ENABLED", "false")
	t.Setenv("OBS_CMD_MIRROR", "true")
	if os.Getenv("OAAS_MANAGER_JOBS_URL") == "" {
		t.Setenv("OAAS_MANAGER_JOBS_URL", "http://127.0.0.1:8000/api/v1/jobs")
	}

	// Reuse R6's existing task and native FPM fixture. This is a controlled
	// configuration test, NOT a conversion of a real ADD_STREAM task.
	task := integrationFixtureTask(t)
	_, fixture := originalRequestAndSnapshotForResponseTest(t)
	if task.GetIntent() != optimizer.Intent_INITIAL_CONFIGURATION {
		t.Fatalf("R6 fixture must be INITIAL_CONFIGURATION; got %s", task.GetIntent())
	}

	// The fixture does not contain DeviceModel references; these placeholder
	// models have no claimed device-specific hardware limitations.
	seedR6CNCDataInIsolatedEtcd(t, fixture)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// This is the baseline as actually reconstructed from the CNC store, not
	// the fixture passed directly into executeOptimizationJob by old R6.
	original := createForwardingPlaneModel(ctx, nil)
	if original == nil || original.GetTopology() == nil || original.GetConfiguration() == nil ||
		len(original.GetTopology().GetNodes()) != 4 ||
		len(original.GetTopology().GetLinks()) != 3 || len(original.GetStreams()) != 2 {
		t.Fatal("isolated CNC store did not reconstruct the expected R6 topology/configuration/streams")
	}
	before := proto.Clone(original)

	obs, err := observability.NewFromEnv("tsn-service-r6-start-optimization")
	if err != nil {
		t.Fatalf("create observability client: %v", err)
	}
	defer func() { _ = obs.Close() }()

	// Capture the Manager job ID without modifying the production submission.
	originalWrite := snapshotWrite
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
		return originalWrite(data, key)
	}
	t.Cleanup(func() { snapshotWrite = originalWrite })

	// Critical difference from old R6: call the actual entry point.
	updated, err := StartOptimization(ctx, obs, task, nil)
	if err != nil {
		t.Fatalf("StartOptimization -> Manager -> CBS -> FPM: %v", err)
	}
	if updated == nil || proto.Equal(updated, original) {
		t.Fatal("expected a changed FPM returned by StartOptimization")
	}
	if jobID == "" {
		t.Fatal("missing Manager job ID")
	}
	preserved, err := loadSnapshotForJob(jobID)
	if err != nil || !proto.Equal(preserved, before) {
		t.Fatalf("original FPM snapshot missing or modified: %v", err)
	}

	// StartOptimization returns a new model; it must not deploy or overwrite
	// the current CNC configuration in the store.
	stillStored := createForwardingPlaneModel(ctx, nil)
	if stillStored == nil || !proto.Equal(stillStored, before) {
		t.Fatal("StartOptimization unexpectedly modified the CNC store's baseline FPM")
	}

	requestJSON, err := loadJobRequest(jobID)
	if err != nil {
		t.Fatalf("read original Manager request: %v", err)
	}
	var request struct {
		Operation        string `json:"operation"`
		OptimizationTask struct {
			Intent string `json:"intent"`
		} `json:"optimizationTask"`
	}
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		t.Fatal(err)
	}
	if request.Operation != "configuration" || request.OptimizationTask.Intent != "INITIAL_CONFIGURATION" {
		t.Fatalf("unexpected request operation/intent: %s/%s", request.Operation, request.OptimizationTask.Intent)
	}

	// Independently verify the actual Manager result belongs to this job
	// and CBS was the selected optimizer, as in the existing R6 test.
	resultURL, err := managerJobEndpoint(jobID, true)
	if err != nil {
		t.Fatal(err)
	}
	envelopeJSON, err := managerGetJSON(ctx, &http.Client{Timeout: 10 * time.Second}, resultURL)
	if err != nil {
		t.Fatal(err)
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
	if err := json.Unmarshal(envelopeJSON, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Job.JobID != jobID || envelope.Job.Status != "COMPLETED" ||
		envelope.Selection.SelectedOptimizerID != "CBS" ||
		envelope.OptimizerResult.SchemaVersion != "1.3" ||
		envelope.OptimizerResult.Status != "success" {
		t.Fatalf("unexpected Manager response for %s: %+v", jobID, envelope)
	}

	t.Logf("R6 StartOptimization selected CBS; completed job %s; returned updated FPM", jobID)
	t.Log("Check the console for all 12 oaas_*_ms metrics (asynchronous stdout mirroring; not a Kafka delivery check)")
	t.Log("NOT TESTED: real GUI ADD_STREAM/CNC notification, CBS reconfiguration, independent hardware feasibility or deployment")
}

// seedR6CNCDataInIsolatedEtcd writes only exact fixture records into the
// isolated R6 etcd, and removes only those records at test cleanup.
// It refuses any preexisting CNC-domain data to avoid mixing test scenarios.
func seedR6CNCDataInIsolatedEtcd(t *testing.T, fixture *forwardingplane.ForwardingPlaneModel) {
	t.Helper()
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://127.0.0.1:12379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("connect isolated R6 etcd: %v", err)
	}
	var seededKeys []string
	t.Cleanup(func() {
		if len(seededKeys) != 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			ops := make([]clientv3.Op, 0, len(seededKeys))
			for _, key := range seededKeys {
				ops = append(ops, clientv3.OpDelete(key))
			}
			_, err := client.Txn(ctx).Then(ops...).Commit()
			cancel()
			if err != nil {
				t.Errorf("isolated etcd fixture cleanup failed: %v", err)
			}
		}
		if err := client.Close(); err != nil {
			t.Errorf("close isolated etcd: %v", err)
		}
	})

	// GetTopology/GetStreams use prefixes, so the isolated store must not
	// contain another CNC network fixture. R6's snapshot/job keys can remain.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, prefix := range []string{
		"endnodes", "bridges", "links", "streams", "configurations/streams",
		"configurations/topology/default_schedule", "device-models",
	} {
		resp, err := client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithLimit(1))
		if err != nil {
			t.Fatalf("check isolated etcd prefix %s: %v", prefix, err)
		}
		if len(resp.Kvs) != 0 {
			t.Fatalf("isolated etcd contains existing CNC data at prefix %q; refusing to overwrite it", prefix)
		}
	}

	records := make(map[string][]byte)
	add := func(key string, value proto.Message) {
		t.Helper()
		if _, exists := records[key]; exists {
			t.Fatalf("duplicate fixture key: %s", key)
		}
		b, err := proto.Marshal(value)
		if err != nil {
			t.Fatalf("marshal fixture key %s: %v", key, err)
		}
		records[key] = b
	}

	if fixture.GetTopology() == nil || fixture.GetConfiguration() == nil {
		t.Fatal("R6 fixture requires topology and current configuration")
	}
	for _, source := range fixture.GetTopology().GetNodes() {
		if source == nil || source.GetName() == "" {
			t.Fatal("R6 fixture contains nil/unnamed node")
		}
		node := proto.Clone(source).(*topology.Node)
		modelName := "oaas-r6-fixture-" + node.GetName()
		if node.DeviceInfo == nil {
			node.DeviceInfo = &topology.DeviceInfo{}
		}
		node.DeviceInfo.DeviceModel = modelName
		switch node.GetType() {
		case topology.NodeRole_BRIDGE:
			add("bridges/"+node.GetName(), node)
		case topology.NodeRole_END_STATION:
			add("endnodes/"+node.GetName(), node)
		default:
			t.Fatalf("unsupported fixture node role %s", node.GetType())
		}
		add("device-models/"+modelName, &devicemodelregistry.DeviceModel{Name: modelName})
	}
	for _, link := range fixture.GetTopology().GetLinks() {
		if link == nil || link.GetId() == "" {
			t.Fatal("R6 fixture link missing ID")
		}
		add("links/"+link.GetId(), link)
	}

	cfg := proto.Clone(fixture.GetConfiguration()).(*topology_config.TopologyConfig)
	cfg.ConfigId = defaultSchedID // StartOptimization reads this exact key.
	add("configurations/topology/"+defaultSchedID, cfg)

	for _, sm := range fixture.GetStreams() {
		if sm == nil || sm.GetDefinition() == nil || sm.GetDefinition().GetStreamId() == nil || sm.GetConfiguration() == nil {
			t.Fatal("R6 fixture stream lacks definition, stream ID or configuration")
		}
		id := sm.GetDefinition().GetStreamId()
		key := id.AsKey()
		add("streams/"+key, sm.GetDefinition())
		streamConfig := proto.Clone(sm.GetConfiguration()).(*stream_config.StreamConfiguration)
		streamConfig.StreamId = proto.Clone(id).(*stream.StreamId)
		// The existing StoreStreamConfiguration writer uses StreamId.String(),
		// whereas GetStreamConfiguration reads the AsKey() key. Seed the exact
		// key required by the production reader; do not modify external code.
		add("configurations/streams/"+key, streamConfig)
	}

	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	compares := make([]clientv3.Cmp, 0, len(keys))
	writes := make([]clientv3.Op, 0, len(keys))
	for _, key := range keys {
		compares = append(compares, clientv3.Compare(clientv3.CreateRevision(key), "=", 0))
		writes = append(writes, clientv3.OpPut(key, string(records[key])))
	}
	resp, err := client.Txn(ctx).If(compares...).Then(writes...).Commit()
	if err != nil {
		t.Fatalf("seed isolated etcd: %v", err)
	}
	if !resp.Succeeded {
		t.Fatal("fixture seeding lost a race with another etcd writer; no records overwritten")
	}
	seededKeys = keys
	t.Logf("Seeded %d isolated CNC records for R6 StartOptimization; cleanup is registered", len(keys))
}
