package metrics

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// MetricsLogPath is the JSON lines log file read by experiment benchmarks.
const MetricsLogPath = "/tmp/cnc_experiment_metrics.jsonl"

// CNCLogEntry defines the structured JSON format expected by experiment benchmarks.
type CNCLogEntry struct {
	Event             string  `json:"event"`
	NodeID            string  `json:"node_id"`
	Plugin            string  `json:"plugin,omitempty"`
	StartTimeUnixNano int64   `json:"start_time_unix_nano"`
	EndTimeUnixNano   int64   `json:"end_time_unix_nano,omitempty"`
	DurationMs        float64 `json:"duration_ms,omitempty"`
	DurationNs        int64   `json:"duration_ns,omitempty"`
	TimestampRFC3339  string  `json:"timestamp_rfc3339"`
}

var metricsMu sync.Mutex

// RecordEvent appends a structured log entry to MetricsLogPath in a thread-safe manner.
func RecordEvent(entry CNCLogEntry) {
	metricsMu.Lock()
	defer metricsMu.Unlock()

	if entry.TimestampRFC3339 == "" {
		entry.TimestampRFC3339 = time.Now().Format(time.RFC3339Nano)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(MetricsLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.Write(data)
	_, _ = f.WriteString("\n")
}

// RecordTranslation logs the duration of a plugin's Map() and BuildFeatureXML() operations.
func RecordTranslation(nodeID, plugin string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "translation",
		NodeID:            nodeID,
		Plugin:            plugin,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordPushStart logs the timestamp when NETCONF push (edit-config) starts for a node.
func RecordPushStart(nodeID string, start time.Time) {
	RecordEvent(CNCLogEntry{
		Event:             "push_start",
		NodeID:            nodeID,
		StartTimeUnixNano: start.UnixNano(),
		TimestampRFC3339:  start.Format(time.RFC3339Nano),
	})
}

// RecordPushComplete logs the completion and duration of NETCONF push (edit-config) for a node.
func RecordPushComplete(nodeID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "push_complete",
		NodeID:            nodeID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordGRPCReceive logs when an ApplyConfiguration gRPC request arrives at the server.
func RecordGRPCReceive(configID string, t time.Time) {
	RecordEvent(CNCLogEntry{
		Event:             "grpc_receive",
		NodeID:            configID,
		StartTimeUnixNano: t.UnixNano(),
		TimestampRFC3339:  t.Format(time.RFC3339Nano),
	})
}

// RecordGRPCComplete logs when an ApplyConfiguration gRPC request completes.
func RecordGRPCComplete(configID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "grpc_complete",
		NodeID:            configID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordSessionConnect logs the time taken to establish the NETCONF SSH session for a node.
func RecordSessionConnect(nodeID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "session_connect",
		NodeID:            nodeID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordPreparation logs the overall concurrent preparation/calculation phase duration across all nodes.
func RecordPreparation(configID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "preparation",
		NodeID:            configID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordConcurrentCommit logs the overall concurrent commit/push phase duration across all nodes.
func RecordConcurrentCommit(configID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "concurrent_commit",
		NodeID:            configID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordRollbackLatency logs the rollback latency from request receipt until the rollback is ready just before switch commit.
func RecordRollbackLatency(configID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "rollback_ready",
		NodeID:            configID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

// RecordRollbackPreparation logs the duration of preparing rollback payloads across all nodes.
func RecordRollbackPreparation(configID string, start, end time.Time) {
	dur := end.Sub(start)
	RecordEvent(CNCLogEntry{
		Event:             "rollback_preparation",
		NodeID:            configID,
		StartTimeUnixNano: start.UnixNano(),
		EndTimeUnixNano:   end.UnixNano(),
		DurationMs:        float64(dur.Nanoseconds()) / 1e6,
		DurationNs:        dur.Nanoseconds(),
		TimestampRFC3339:  end.Format(time.RFC3339Nano),
	})
}

