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
cd /home/jittin/config-service/test/OpenCNC/add_stream/kafka_receiver
./run.sh
```
*Note: By default, CSV files will be saved in the repository root directory (`/home/jittin/config-service/test/OpenCNC`). If Kafka is running inside Kubernetes, port 9092 will be automatically forwarded.*

### Option 2: Pull CSVs from Kubernetes Pod
If the in-cluster `kafka-receiver` pod is running in Kubernetes (`namespace: opencnc`), you can fetch the CSV files anytime using:
```bash
cd /home/jittin/config-service/test/OpenCNC/add_stream/kafka_receiver
./fetch_k8s_csv.sh
```

### Option 3: Using `go run` directly
From the OpenCNC root directory:
```bash
go run ./add_stream/kafka_receiver \
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
