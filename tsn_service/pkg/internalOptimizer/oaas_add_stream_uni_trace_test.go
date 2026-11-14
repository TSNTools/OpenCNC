package internalOptimizer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/uni"
	"OpenCNC/tsn_service/pkg/notificationServer"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	"github.com/gogo/protobuf/jsonpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// This diagnostic reads the user's actual UNI JSON, not the M1/PCP3 R6 fixture.
// Nothing in this file contacts Main Service, Config Service, or physical devices.
// The default test is read-only. Main Service's UNCHANGED-parser rejection is
// preserved as evidence; any subsequent format adaptation is an explicitly
// labeled, in-memory TEST ASSUMPTION. The converter stage is opt-in and writes
// ONLY to a separately verified isolated local etcd instance.
const addStreamJSON = "/tmp/oaas_single_add_stream.json"

// readOriginalUNI preserves the original raw bytes and first
// attempts the EXACT parser used by Main Service. The example's nested
// stream.StreamId and TalkerGroup.StrRank use representations not accepted by
// the Main Service parser, so a separate in-memory, test-only copy can be
// adapted for further diagnosis. This does NOT establish
// that Main Service accepts the original request.
func readOriginalUNI(t *testing.T) *uni.ConfigRequest {
	t.Helper()
	file := os.Getenv("OAAS_ADD_STREAM_UNI_JSON")
	if file == "" {
		file = addStreamJSON
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read original UNI file %s: %v", file, err)
	}
	var preview struct {
		Requests []struct {
			Talker struct {
				StreamID   json.RawMessage `json:"stream-id"`
				FrameSpecs []struct {
					VLANTag struct {
						PCP *uint32 `json:"priority-code-point"`
					} `json:"ieee802-vlan-tag"`
				} `json:"data-frame-specification"`
			} `json:"talker"`
		} `json:"requests"`
	}
	if json.Unmarshal(raw, &preview) == nil && len(preview.Requests) > 0 {
		t.Logf("ORIGINAL RAW UNI stream-id (unchanged): %s", preview.Requests[0].Talker.StreamID)
		for _, frame := range preview.Requests[0].Talker.FrameSpecs {
			if frame.VLANTag.PCP != nil {
				t.Logf("ORIGINAL RAW UNI VLAN PCP=%d", *frame.VLANTag.PCP)
			}
		}
	}
	var cfg uni.ConfigRequest
	if err := jsonpb.Unmarshal(bytes.NewReader(raw), &cfg); err != nil {
		t.Logf("REAL MAIN SERVICE PARSER: REJECTS the unchanged source JSON: %v", err)
		if !strings.Contains(err.Error(), `unknown field "mac-address" in stream.StreamId`) &&
			!strings.Contains(err.Error(), `unknown field "unique-id" in stream.StreamId`) &&
			!strings.Contains(err.Error(), `cannot unmarshal object into Go value of type stream.StreamRank`) {
			t.Fatalf("unexpected original-parser rejection; refusing to mask it with test-only formatting: %v", err)
		}
		adapted, err := adaptUNIJSONForDiagnostic(raw, t)
		if err != nil {
			t.Fatalf("cannot construct a strictly scoped test-only protobuf-format copy: %v", err)
		}
		if err := jsonpb.Unmarshal(bytes.NewReader(adapted), &cfg); err != nil {
			t.Fatalf("REAL MAIN SERVICE PARSER still rejects the test-only copy; no further normalization attempted: %v", err)
		}
		t.Log("TEST-ONLY parser-format adapted copy accepted. IMPORTANT: this is NOT acceptance of the original UNI file by Main Service")
	} else {
		t.Log("REAL MAIN SERVICE PARSER: accepts unchanged source JSON")
	}
	if len(cfg.GetRequests()) != 1 || cfg.GetRequests()[0] == nil || cfg.GetRequests()[0].GetTalker() == nil ||
		len(cfg.GetRequests()[0].GetListenerList()) != 1 {
		t.Fatalf("expected precisely one talker/listener request; got %d requests", len(cfg.GetRequests()))
	}
	return &cfg
}

// adaptUNIJSONForDiagnostic makes two *format-only* changes in a volatile JSON
// copy: nested StreamId field names/unique-ID representation, and TalkerGroup
// stream-rank enum representation. It does NOT set PCP, trafficClassId, or
// queue assignments. The 00-01 -> 1 interpretation is a TEST ASSUMPTION, not
// an authoritative UNI/CNC mapping rule.
func adaptUNIJSONForDiagnostic(raw []byte, t *testing.T) ([]byte, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	requests, ok := root["requests"].([]any)
	if !ok || len(requests) != 1 {
		return nil, fmt.Errorf("expected one requests entry")
	}
	convertID := func(parent map[string]any, path string) error {
		id, ok := parent["stream-id"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: no stream-id object", path)
		}
		mac, ok := id["mac-address"].(string)
		if !ok || mac == "" {
			return fmt.Errorf("%s: unsupported original stream-id.mac-address", path)
		}
		uniqueText, ok := id["unique-id"].(string)
		if !ok {
			return fmt.Errorf("%s: unsupported original stream-id.unique-id type", path)
		}
		segments := strings.Split(uniqueText, "-")
		if len(segments) != 2 || len(segments[0]) != 2 || len(segments[1]) != 2 {
			return fmt.Errorf("%s: unique-id %q is not two hex bytes; no guess allowed", path, uniqueText)
		}
		value, err := strconv.ParseUint(segments[0]+segments[1], 16, 16)
		if err != nil {
			return fmt.Errorf("%s: parse two-byte unique-id %q: %w", path, uniqueText, err)
		}
		delete(id, "mac-address")
		delete(id, "unique-id")
		id["macAddress"] = mac
		id["uniqueId"] = uint32(value)
		t.Logf("TEST-ONLY stream-id formatting %s: {mac-address:%q, unique-id:%q} -> {macAddress:%q, uniqueId:%d}; hex-byte interpretation is an ASSUMPTION", path, mac, uniqueText, mac, value)
		return nil
	}
	for i, rawRequest := range requests {
		req, ok := rawRequest.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("requests[%d] is not an object", i)
		}
		talker, ok := req["talker"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("requests[%d].talker is not an object", i)
		}
		if err := convertID(talker, "talker"); err != nil {
			return nil, err
		}
		// Original sample represents stream rank as {"rank":1}, but the
		// actual OpenCNC TalkerGroup.StrRank field is a protobuf enum.
		// This is a representation-only adjustment of the copied JSON.
		rankObject, ok := talker["stream-rank"].(map[string]any)
		if !ok || len(rankObject) != 1 {
			return nil, fmt.Errorf("talker.stream-rank: expected original one-field rank object; refusing to guess")
		}
		rankNumber, ok := rankObject["rank"].(float64)
		if !ok || (rankNumber != 0 && rankNumber != 1) {
			return nil, fmt.Errorf("talker.stream-rank.rank: unsupported value %v; expected OpenCNC RANK_EMERGENCY=0 or RANK_NON_EMERGENCY=1", rankObject["rank"])
		}
		talker["stream-rank"] = int(rankNumber)
		t.Logf("TEST-ONLY stream-rank formatting: {rank:%v} -> %d (OpenCNC enum numeric representation; no PCP/class/queue change)", rankNumber, int(rankNumber))
		listeners, ok := req["listener-list"].([]any)
		if !ok || len(listeners) != 1 {
			return nil, fmt.Errorf("requests[%d].listener-list: expected one listener", i)
		}
		for j, l := range listeners {
			listener, ok := l.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("listener[%d] not an object", j)
			}
			if err := convertID(listener, fmt.Sprintf("listener[%d]", j)); err != nil {
				return nil, err
			}
		}
	}
	return json.Marshal(root)
}

func sourcePCP(t *testing.T, r *uni.Request) uint32 {
	t.Helper()
	var found bool
	var value uint32
	for _, spec := range r.GetTalker().GetDataFrameSpecification() {
		if spec == nil || spec.GetVlanTag() == nil {
			continue
		}
		if found {
			t.Fatal("ambiguous multiple VLAN-tag specifications in UNI request")
		}
		found = true
		value = spec.GetVlanTag().GetPriorityCodePoint()
	}
	if !found {
		t.Fatal("original UNI request has no VLAN priority-code-point")
	}
	return value
}

func nodeForInterfaceMAC(t *testing.T, model *forwardingplane.ForwardingPlaneModel, mac string) string {
	t.Helper()
	if model.GetTopology() == nil {
		t.Fatal("captured FPM has no topology")
	}
	var result string
	for _, node := range model.GetTopology().GetNodes() {
		for _, port := range node.GetPorts() {
			if strings.EqualFold(strings.ReplaceAll(port.GetMacAddress(), ":", "-"), strings.ReplaceAll(mac, ":", "-")) {
				if result != "" {
					t.Fatalf("ambiguous interface MAC %q in captured topology", mac)
				}
				result = node.GetName()
			}
		}
	}
	if result == "" {
		t.Fatalf("no topology node contains interface MAC %q", mac)
	}
	return result
}

func readCapturedFPM(t *testing.T) *forwardingplane.ForwardingPlaneModel {
	t.Helper()
	path := os.Getenv("OAAS_ADD_STREAM_CNC_FPM_JSON")
	if path == "" {
		t.Fatal("set OAAS_ADD_STREAM_CNC_FPM_JSON to a read-only copy of the CNC forwarding_plane.json capture")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CNC capture %s: %v", path, err)
	}
	var model forwardingplane.ForwardingPlaneModel
	if err := protojson.Unmarshal(raw, &model); err != nil {
		t.Fatalf("decode native CNC FPM: %v", err)
	}
	if model.GetTopology() == nil {
		t.Fatal("captured FPM has no topology")
	}
	return &model
}

func endstationInterface(t *testing.T, r *uni.Request, talker bool) *uni.InterfaceId {
	t.Helper()
	var list []*uni.Interface
	if talker {
		list = r.GetTalker().GetEndStationInterfaces()
	} else {
		list = r.GetListenerList()[0].GetEndStationInterfaces()
	}
	if len(list) != 1 || list[0] == nil || list[0].GetInterfaceId() == nil {
		t.Fatal("expected one end-station interface with interface-id")
	}
	return list[0].GetInterfaceId()
}

// TestOaaSAddStreamUNITrace is SAFE and READ ONLY.
// It shows what the original UNI supplies, and (when the CNC capture is supplied)
// whether a real switch-port PCP -> priority -> queue mapping is available.
func TestOaaSAddStreamUNITrace(t *testing.T) {
	if os.Getenv("OAAS_ADD_STREAM_TRACE") != "1" {
		t.Skip("opt in with OAAS_ADD_STREAM_TRACE=1")
	}
	cfg := readOriginalUNI(t)
	req := cfg.GetRequests()[0]
	talker := endstationInterface(t, req, true)
	listener := endstationInterface(t, req, false)
	pcp := sourcePCP(t, req)
	ts := req.GetTalker().GetTrafficSpecification()
	if ts == nil || ts.GetInterval() == nil {
		t.Fatal("missing original UNI traffic interval")
	}
	t.Logf("SOURCE UNI: stream=%v, talkerMAC=%q talkerInterfaceName=%q, listenerMAC=%q listenerInterfaceName=%q",
		req.GetTalker().GetStrId(), talker.GetMacAddress(), talker.GetInterfaceName(), listener.GetMacAddress(), listener.GetInterfaceName())
	t.Logf("SOURCE UNI: VLAN PCP=%d, interval=%d/%d seconds, maxFrameSize=%d bytes, maxFramesPerInterval=%d",
		pcp, ts.GetInterval().GetNumerator(), ts.GetInterval().GetDenominator(), ts.GetMaxFrameSize(), ts.GetMaxFramesPerInterval())
	t.Log("SOURCE UNI: does not contain a CBS 'M1' class declaration. PCP is a VLAN marking, not an M1 classification.")
	if talker.GetInterfaceName() == "" || listener.GetInterfaceName() == "" {
		t.Log("PRE-CONVERSION BLOCKER: empty interface-name; actual CNC createStreamFromRequest() rejects this before stream storage")
	}

	if os.Getenv("OAAS_ADD_STREAM_CNC_FPM_JSON") == "" {
		t.Log("QUEUE MAPPING UNRESOLVED: no CNC FPM capture supplied. Set OAAS_ADD_STREAM_CNC_FPM_JSON to inspect actual bridge-port PCP tables (read-only).")
		return
	}
	model := readCapturedFPM(t)
	talkerNode := nodeForInterfaceMAC(t, model, talker.GetMacAddress())
	listenerNode := nodeForInterfaceMAC(t, model, listener.GetMacAddress())
	t.Logf("CAPTURED TOPOLOGY: %s -> %s (mapped from actual UNI interface MACs)", talkerNode, listenerNode)

	// For the one-bridge topology observed in the captured CNC test, report
	// PCP decoding / priority-to-queue mapping on the bridge's outgoing port.
	// This inspects configured data; it does NOT install default mappings.
	for _, link := range model.GetTopology().GetLinks() {
		if link.GetDestinationNode() != listenerNode || link.GetSourceNode() == talkerNode {
			continue
		}
		bridge := link.GetSourceNode()
		portID := link.GetSourcePort()
		for _, node := range model.GetConfiguration().GetNodeConfigs() {
			if node.GetNodeId() != bridge {
				continue
			}
			for _, port := range node.GetPortConfigs() {
				if port.GetPortId() != portID {
					continue
				}
				if port.GetPcpConfig() == nil {
					t.Logf("QUEUE MAPPING UNRESOLVED: %s/%s has no configured PCP tables", bridge, portID)
					return
				}
				if port.PcpMappingEnabled == nil || !port.GetPcpMappingEnabled() {
					t.Logf("PCP MAPPING NOT CONFIRMED ENABLED on %s/%s (enabled=%v, explicit=%v)", bridge, portID, port.GetPcpMappingEnabled(), port.PcpMappingEnabled != nil)
				}
				mapped := false
				for _, table := range port.GetPcpConfig().GetDecodingTables() {
					for _, decoding := range table.GetEntries() {
						if decoding.GetPriorityCodePoint() != pcp {
							continue
						}
						priority := decoding.GetPriority()
						for _, entry := range port.GetPcpConfig().GetTrafficClassTable() {
							if entry.GetPriority() == priority {
								t.Logf("CONFIGURED PCP LOOKUP: %s/%s PCP=%d -> priority=%d -> egressQueueID=%d (mapping type=%s; operational enablement must be verified)",
									bridge, portID, pcp, priority, entry.GetEgressQueueId(), table.GetPcpType().String())
								mapped = true
							}
						}
					}
				}
				if !mapped {
					t.Logf("QUEUE MAPPING UNRESOLVED: %s/%s does not define a complete PCP=%d lookup", bridge, portID, pcp)
				}
				return
			}
		}
		t.Logf("QUEUE MAPPING UNRESOLVED: no port configuration for %s/%s", bridge, portID)
		return
	}
	t.Log("QUEUE MAPPING UNRESOLVED: captured topology does not identify a bridge-to-listener outgoing link")
}

// TestOaaSAddStreamProductionConversionIsolated invokes the REAL
// notificationServer.CreateOptimizationTask() function. It writes the converted
// Stream to an isolated etcd, reads it back, and removes only its own key.
// It does NOT call StartOptimization, Manager, Main Service, Config Service, or any hardware.
// Requires explicit opt-in; the user must independently verify :12379 is a
// separate test-only etcd, NOT a port-forward into the real OpenCNC cluster.
func TestOaaSAddStreamProductionConversionIsolated(t *testing.T) {
	if os.Getenv("OAAS_ADD_STREAM_ISOLATED_CONVERSION") != "1" {
		t.Skip("opt in with OAAS_ADD_STREAM_ISOLATED_CONVERSION=1")
	}
	if os.Getenv("OAAS_ADD_STREAM_TRACE") != "1" ||
		os.Getenv("OAAS_CONFIRM_SEPARATE_TEST_ETCD") != "1" ||
		os.Getenv("ETCD_HOST") != "127.0.0.1" || os.Getenv("ETCD_PORT") != "12379" {
		t.Fatal("requires TRACE=1, CONFIRM_SEPARATE_TEST_ETCD=1, ETCD_HOST=127.0.0.1 and ETCD_PORT=12379")
	}
	source := readOriginalUNI(t)
	captured := readCapturedFPM(t)
	originalReq := source.GetRequests()[0]
	originalPCP := sourcePCP(t, originalReq)
	req := proto.Clone(originalReq).(*uni.Request)
	talker := endstationInterface(t, req, true)
	listener := endstationInterface(t, req, false)
	talkerNode := nodeForInterfaceMAC(t, captured, talker.GetMacAddress())
	listenerNode := nodeForInterfaceMAC(t, captured, listener.GetMacAddress())
	// Only repair the two empty interface names IN MEMORY for the converter.
	// Values are obtained from captured topology; no traffic class, PCP, or
	// queue assignment is supplied, guessed, or hardcoded.
	if talker.GetInterfaceName() == "" {
		talker.InterfaceName = talkerNode
	}
	if listener.GetInterfaceName() == "" {
		listener.InterfaceName = listenerNode
	}
	if talker.GetInterfaceName() != talkerNode || listener.GetInterfaceName() != listenerNode {
		t.Fatalf("UNI interface names disagree with captured topology: %q / %q", talker.GetInterfaceName(), listener.GetInterfaceName())
	}
	t.Logf("ISOLATED TEST-ONLY NAME RESOLUTION: talker=%s, listener=%s (copied UNI request, source file untouched)", talkerNode, listenerNode)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{"http://127.0.0.1:12379"}, DialTimeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	var testStreamKey string
	var expectedRevision int64
	t.Cleanup(func() {
		if testStreamKey != "" && expectedRevision > 0 {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := client.Txn(cleanupCtx).
				If(clientv3.Compare(clientv3.ModRevision(testStreamKey), "=", expectedRevision)).
				Then(clientv3.OpDelete(testStreamKey)).Commit()
			cancel()
			if err != nil || !resp.Succeeded {
				t.Errorf("isolated stream cleanup failed or key was modified: %v", err)
			}
		}
		if err := client.Close(); err != nil {
			t.Errorf("close isolated etcd: %v", err)
		}
	})
	// Fail closed if ANY preexisting CNC network data or stream/request exists.
	for _, prefix := range []string{"streams/", "configurations/streams/", "configurations/requests/", "endnodes/", "bridges/", "links/", "configurations/topology/"} {
		resp, err := client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithLimit(1))
		if err != nil {
			t.Fatalf("isolated store precheck: %v", err)
		}
		if len(resp.Kvs) != 0 {
			t.Fatalf("REFUSING to use etcd with existing CNC data at %q", prefix)
		}
	}

	// Actual converter stores a stream definition as an observable side effect.
	id := req.GetTalker().GetStrId()
	if id == nil {
		t.Fatal("missing original UNI stream ID")
	}
	key := "streams/" + id.AsKey()
	task, err := notificationServer.CreateOptimizationTask([]*uni.Request{req})
	if err != nil {
		t.Fatalf("real CNC UNI conversion: %v", err)
	}
	storedResponse, err := client.Get(ctx, key)
	if err != nil || len(storedResponse.Kvs) != 1 {
		t.Fatalf("find isolated converter-written stream key: %v", err)
	}
	testStreamKey = key
	expectedRevision = storedResponse.Kvs[0].ModRevision
	if task.GetIntent().String() != "ADD_STREAM" || task.GetTrigger().String() != "STREAM_ADDITION" {
		t.Fatalf("unexpected CNC task intent/trigger: %s/%s", task.GetIntent(), task.GetTrigger())
	}
	native, err := store.GetStreamByID(id.AsKey())
	if err != nil {
		t.Fatalf("read stream written by CNC converter: %v", err)
	}
	t.Logf("ACTUAL CONVERTER: intent=%s trigger=%s scope=%v", task.GetIntent(), task.GetTrigger(), task.GetScope().GetStreamIds())
	t.Logf("ACTUAL CONVERTER: UNI PCP=%d; CNC Stream.trafficClassId=%q", originalPCP, native.GetTrafficClassId())
	t.Logf("ACTUAL CONVERTER: frame destination=%q intervalNs=%d maxFrameSize=%d", native.GetFrameSpec().GetDestinationMac(), native.GetTrafficSpec().GetIntervalNs(), native.GetTrafficSpec().GetMaxFrameSize())
	config, err := store.GetStreamConfiguration(id.AsKey())
	if err != nil {
		t.Logf("ACTUAL CONVERTER: no StreamConfiguration exists; no per-stream PCP or queue assignments: %v", err)
	} else {
		t.Logf("ACTUAL CONVERTER: StreamConfiguration.pcp=%v queueAssignments=%v", config.Pcp, config.GetQueueAssignments())
	}
	t.Log("DIAGNOSTIC ONLY: this test does not claim CBS-compatible traffic class, queue assignment, Manager execution, or E2E success")
}
