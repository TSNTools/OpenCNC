package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	obs "OpenCNC/common/structures/logging"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type messageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
	Close() error
}

// One record per sample; no per-device topics or retained per-series state.
type MonitoringJSON struct {
	SchemaVersion   string            `json:"schema_version"`
	Timestamp       string            `json:"timestamp"`
	Service         string            `json:"service"`
	NodeID          string            `json:"node_id"`
	PortID          string            `json:"port_id"`
	Metric          string            `json:"metric"`
	MetricType      string            `json:"metric_type"`
	Value           float64           `json:"value"`
	Unit            string            `json:"unit"`
	Attributes      map[string]string `json:"attributes"`
	SeriesID        string            `json:"series_id"`
	EventID         string            `json:"event_id"`
	CorrelationID   string            `json:"correlation_id,omitempty"`
	SourceTopic     string            `json:"source_topic"`
	SourcePartition int               `json:"source_partition"`
	SourceOffset    int64             `json:"source_offset"`
}

func monitoringMessage(msg kafka.Message) (*kafka.Message, error) {
	var event obs.EventEnvelope
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		if err := protojson.Unmarshal(msg.Value, &event); err != nil {
			return nil, nil
		}
	}
	metric := event.GetMetric()
	if metric == nil || normalizeServiceName(event.GetSource().GetService()) != "monitor-service" {
		return nil, nil
	}
	if math.IsNaN(metric.GetValue()) || math.IsInf(metric.GetValue(), 0) {
		return nil, fmt.Errorf("non-finite monitoring value")
	}
	attrs := metric.GetAttributes()
	if attrs["node_id"] == "" {
		return nil, fmt.Errorf("monitoring metric %q has no node_id; cannot attribute device", metric.GetName())
	}
	timestamp := msg.Time.UTC()
	if event.GetOccurredAt() != nil {
		timestamp = event.GetOccurredAt().AsTime().UTC()
	}
	identity, _ := json.Marshal([]any{event.GetSource().GetService(), attrs["node_id"], attrs["port_id"], metric.GetName(), metric.GetUnit(), attrs})
	series := fmt.Sprintf("%x", sha256.Sum256(identity))
	eventID := event.GetEventId()
	if eventID == "" {
		eventID = fmt.Sprintf("%s/%d/%d", msg.Topic, msg.Partition, msg.Offset)
	}
	record := MonitoringJSON{SchemaVersion: "1", Timestamp: timestamp.Format(time.RFC3339Nano), Service: event.GetSource().GetService(), NodeID: attrs["node_id"], PortID: attrs["port_id"], Metric: metric.GetName(), MetricType: metric.GetType().String(), Value: metric.GetValue(), Unit: metric.GetUnit(), Attributes: attrs, SeriesID: series, EventID: eventID, CorrelationID: event.GetTrace().GetCorrelationId(), SourceTopic: msg.Topic, SourcePartition: msg.Partition, SourceOffset: msg.Offset}
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return &kafka.Message{Key: []byte(series), Value: data, Time: timestamp}, nil
}

func waitRetry(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Second):
		return true
	}
}

func (r *Receiver) forwardMonitoring(ctx context.Context, msg kafka.Message) error {
	if r.monitoringWriter == nil {
		return nil
	}
	output, err := monitoringMessage(msg)
	if err != nil {
		// Retain invalid/unattributable samples in the existing raw CSV, never invent a device.
		log.Printf("[WARN] Monitoring JSON skipped %s/%d/%d: %v", msg.Topic, msg.Partition, msg.Offset, err)
		return nil
	}
	if output == nil {
		return nil
	}
	for {
		publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = r.monitoringWriter.WriteMessages(publishCtx, *output)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("[WARN] Monitoring JSON publish failed; input offset not committed: %v", err)
		if !waitRetry(ctx) {
			return ctx.Err()
		}
	}
}
