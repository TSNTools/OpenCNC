package main

import (
	obs "OpenCNC/common/structures/logging"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"testing"
	"time"
)

func sample(t *testing.T, node, port string, asJSON bool) kafka.Message {
	t.Helper()
	e := &obs.EventEnvelope{EventId: "event-123", OccurredAt: timestamppb.New(time.Unix(100, 0)), Source: &obs.Source{Service: "monitor-service"}, Payload: &obs.EventEnvelope_Metric{Metric: &obs.MetricEvent{Name: "in-octets", Value: 123, Attributes: map[string]string{"node_id": node, "port_id": port}}}}
	var b []byte
	var err error
	if asJSON {
		b, err = protojson.Marshal(e)
	} else {
		b, err = proto.Marshal(e)
	}
	if err != nil {
		t.Fatal(err)
	}
	return kafka.Message{Topic: "opencnc.metrics", Partition: 2, Offset: 42, Value: b}
}
func TestMonitoringSeriesIsolation(t *testing.T) {
	keys := map[string]bool{}
	for _, p := range [][2]string{{"A", "p1"}, {"B", "p1"}, {"A", "p2"}} {
		m, err := monitoringMessage(sample(t, p[0], p[1], false))
		if err != nil {
			t.Fatal(err)
		}
		if keys[string(m.Key)] {
			t.Fatal("devices/ports share series")
		}
		keys[string(m.Key)] = true
		var got MonitoringJSON
		if err = json.Unmarshal(m.Value, &got); err != nil {
			t.Fatal(err)
		}
		if got.NodeID != p[0] || got.PortID != p[1] || got.Value != 123 || got.SourceOffset != 42 || got.EventID != "event-123" {
			t.Fatalf("bad record: %+v", got)
		}
		again, err := monitoringMessage(sample(t, p[0], p[1], true))
		if err != nil || string(again.Key) != string(m.Key) {
			t.Fatal("key differs across encoding/replay")
		}
	}
}
func TestMissingDeviceAndInvalidPayload(t *testing.T) {
	if _, err := monitoringMessage(sample(t, "", "p1", false)); err == nil {
		t.Fatal("missing node accepted")
	}
	if m, err := monitoringMessage(kafka.Message{Value: []byte("not protobuf")}); m != nil || err != nil {
		t.Fatal("non-metric not ignored")
	}
}

type failingWriter struct {
	calls    int
	messages []kafka.Message
	fail     bool
}

func (w *failingWriter) WriteMessages(ctx context.Context, m ...kafka.Message) error {
	w.calls++
	if w.fail || w.calls == 1 {
		return errors.New("broker unavailable")
	}
	w.messages = append(w.messages, m...)
	return nil
}
func (w *failingWriter) Close() error { return nil }
func TestPublishRetriesAndCancellation(t *testing.T) {
	w := &failingWriter{}
	r := &Receiver{monitoringWriter: w}
	if err := r.forwardMonitoring(context.Background(), sample(t, "A", "p1", false)); err != nil {
		t.Fatal(err)
	}
	if w.calls != 2 || len(w.messages) != 1 {
		t.Fatal("publish did not retry")
	}
	w.fail = true
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.forwardMonitoring(ctx, sample(t, "A", "p1", false)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestCountersDoNotAccumulateLatencySummaries(t *testing.T) {
	r := &Receiver{rawWriter: csv.NewWriter(io.Discard), summaries: map[string]*LatencySummary{}}
	for i := 0; i < 100; i++ {
		r.processMessage(sample(t, "A", "p1", false))
	}
	if len(r.summaries) != 0 {
		t.Fatal("monitoring samples retained in latency summaries")
	}
}
func TestRejectOutputAsInput(t *testing.T) {
	if _, err := NewReceiver(Config{Topics: []string{"loop"}, MonitoringJSONTopic: "loop"}); err == nil {
		t.Fatal("feedback loop accepted")
	}
}
