# OpenCNC Kafka Latency Receiver

This tool subscribes to OpenCNC Kafka topics (such as `opencnc.metrics`, `opencnc.events`, `opencnc.logs`) and continuously captures stream latencies and performance metrics into CSV files.

---

## Output CSV Files

When running, the receiver generates two CSV files in the designated output directory:

### 1. `metrics_raw.csv`
Contains every individual record received from Kafka:
- `timestamp_utc`: Timestamp when the event occurred.
- `service`: Emitting service (`main-service`, `tsn-service`, `config-service`).
- `event_kind`: Protobuf event kind (`EVENT_KIND_METRIC`, `EVENT_KIND_DOMAIN`, `EVENT_KIND_LOG`, `EVENT_KIND_AUDIT`, `EVENT_KIND_HEALTH`, or `JSON`/`TEXT`).
- `metric_name`: Name of the metric or event name.
- `value`: Numeric value (e.g. milliseconds).
- `unit`: Unit of measurement (`ms`).
- `correlation_id`: Trace / request correlation ID.
- `event_id`: Unique event envelope identifier.
- `attributes_json`: Key-value tags or attributes associated with the metric/log.
- `details`: Human-readable summary for domain, audit, health, or log events.

### 2. `stream_latencies_summary.csv`
Aggregates and breaks down each add-stream operation:
- `timestamp_utc`: Time the request completed.
- `correlation_id`: Request correlation ID.
- `stream_id`: Stream ID (if tagged).
- `request_id`: Request ID (if tagged).
- `interface`: Interface used (`http` or `grpc`).
- `total_uni_latency_ms`: Total end-to-end UNI response latency.
- `tsn_calc_latency_ms`: Calculation duration in `tsn-service`.
- `config_app_latency_ms`: Total deployment duration in `config-service`.
- `config_prep_latency_ms`: Configuration preparation/mapping duration in `config-service`.
- `config_commit_latency_ms`: NETCONF switch push/commit duration in `config-service`.
- `main_service_overhead_ms`: Estimated `main-service` orchestration time (`TotalUNI - TSN - ConfigApp`).

---

## How to Run

### Option 1: Run on Host (Recommended for local tests)
```bash
cd /home/jittin/config-service/test/OpenCNC/common/kafka_receiver
./run.sh
```
*Note: By default, CSV files will be saved in the repository root directory (`/home/jittin/config-service/test/OpenCNC`). If Kafka is running inside Kubernetes, port 9092 will be automatically forwarded.*

### Option 2: Pull CSVs from Kubernetes Pod
If the in-cluster `kafka-receiver` pod is running in Kubernetes (`namespace: opencnc`), you can fetch the CSV files anytime using:
```bash
cd /home/jittin/config-service/test/OpenCNC/common/kafka_receiver
./fetch_k8s_csv.sh
```

### Option 3: Using `go run` directly
From the OpenCNC root directory:
```bash
go run ./common/kafka_receiver \
  -brokers="localhost:9092" \
  -topics="opencnc.metrics,opencnc.events,opencnc.logs" \
  -output-dir="." \
  -from-beginning=false
```

---

## CLI Flags & Environment Variables

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| `-brokers` | `OBS_BROKERS` | `localhost:9092` | Kafka broker list (comma-separated) |
| `-topics` | `KAFKA_TOPICS` | `opencnc.metrics,opencnc.events,opencnc.logs` | Kafka topics to consume |
| `-group` | `KAFKA_GROUP` | `""` (ephemeral) | Consumer group ID (empty generates unique ID per run) |
| `-from-beginning` | `FROM_BEGINNING` | `false` | Read from topic start if true; default false only captures current run |
| `-output-dir` | `OUTPUT_DIR` | `.` (Repo Root) | Directory to save CSV files |
| `-raw-csv` | - | `metrics_raw.csv` | File name for raw metrics CSV |
| `-summary-csv` | - | `stream_latencies_summary.csv` | File name for summary CSV |
| `-flush-seconds` | `FLUSH_INTERVAL` | `1` | Interval in seconds to flush CSV writers to disk |

## Device monitoring JSON

The receiver converts `monitor-service` metric envelopes from `opencnc.metrics`
into one JSON record per sample on `opencnc.monitoring.json`. Configure with
`-monitoring-json-topic` or `KAFKA_MONITORING_JSON_TOPIC`; the explicit flag
`-monitoring-json-topic=""` disables this output. CSV and latency outputs remain available.

Example (abbreviated):

```json
{"schema_version":"1","timestamp":"2026-09-25T08:40:32Z","service":"monitor-service","node_id":"B&R","port_id":"sw0p1","metric":"if_state_in_octets","metric_type":"METRIC_TYPE_GAUGE","value":40403429,"unit":"","attributes":{"node_id":"B&R","port_id":"sw0p1"},"series_id":"<stable hash>","event_id":"<original event id>","source_topic":"opencnc.metrics","source_partition":0,"source_offset":42}
```

All devices share the topic. `series_id` (also the Kafka record key) hashes the
service, device, port, metric, unit and complete label set. Repeated samples of
the same series share a key; devices and ports remain separate. An empty port
means a node-level metric, not a sum over its interfaces. Original timestamps,
labels and event IDs are preserved. Samples missing `node_id` or containing
non-finite values are warned about and retained in CSV, but not forwarded as
misattributed JSON. The resource monitor publishes calculated packet rates once with device/port
labels and the p/s unit; the meter itself no longer duplicates that publication.

Publishing uses synchronous bounded writes and retries with backpressure, not
one goroutine per sample. Input offsets are committed after JSON broker
acknowledgment. Delivery is at-least-once: a crash after publish but before commit
can replay a sample. Deduplicate by `event_id` (or source topic/partition/offset).
No per-device or per-series cache is retained for monitoring. Monitoring samples
are excluded from the unrelated latency correlation cache.

Use the same stable `-group` value for receiver replicas. Each input topic has its
own group suffix; replicas share that topic's partitions. Useful parallelism is
bounded by **input partitions**, and output partitions determine downstream
parallelism. The supplied single-broker development setup initially uses one
partition, so it supports many devices but not increased throughput from multiple
replicas. Provision partition counts for expected load before scaling, and use
multiple brokers/replication for production durability. Changing partition counts
can change key placement; do not assume ordering across such changes. Legacy
latency correlation remains process-local and is not a distributed aggregator.
CSV files are also per-replica; the supplied `emptyDir` is not durable storage.

### Grafana

Use the existing **Kafka** datasource, topic **opencnc.monitoring.json**, JSON
message format. Start with a table to inspect `timestamp`, `node_id`, `port_id`,
`metric`, `value` and `series_id`. For a time-series panel, use timestamp as time,
value as the numeric field, filter the metric, then partition/group by `series_id`
(or node_id + port_id + metric). Never group by metric name alone. Query all
partitions if the topic is expanded; a panel pinned to partition 0 misses data.
The existing stream-latency dashboard does not automatically query this topic.
Byte samples are cumulative counters, not bytes/sec. Historical retention and
windowing depend on Kafka and the datasource; JSON forwarding is not a
Prometheus exporter or long-term time-series database.
