# OpenCNC Kafka Latency Receiver

This tool subscribes to OpenCNC Kafka topics (such as `opencnc.metrics` and `opencnc.events`) and continuously saves stream latencies and performance metrics into CSV files.

---

## Output CSV Files

When running, the receiver generates two CSV files in the designated output directory:

### 1. `metrics_raw.csv`
Contains every individual metric and domain event record received from Kafka:
- `timestamp_utc`: Timestamp when the event occurred.
- `service`: Emitting service (`main-service`, `tsn-service`, `config-service`).
- `event_kind`: Protobuf event kind (`EVENT_KIND_METRIC`, `EVENT_KIND_DOMAIN`).
- `metric_name`: Name of the metric (e.g., `uni_request_response_duration`, `configuration_calculation_duration`, `configuration_application_duration`, `configuration_preparation_duration`, `configuration_commit_duration`).
- `value`: Numeric value (milliseconds).
- `unit`: Unit of measurement (`ms`).
- `correlation_id`: Trace / request correlation ID.
- `event_id`: Unique event envelope identifier.
- `attributes_json`: Key-value tags associated with the metric (e.g., `interface`, `operation`).
- `details`: Human-readable summary for domain events.

### 2. `stream_latencies_summary.csv`
Aggregates and breaks down each add-stream operation by correlation ID:
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

### Option 1: Using the helper script
```bash
cd /home/jittin/config-service/test/OpenCNC/add_stream/kafka_receiver
chmod +x run.sh
./run.sh
```

### Option 2: Using `go run` directly
From the OpenCNC root directory:
```bash
go run ./add_stream/kafka_receiver \
  -brokers="localhost:9092" \
  -topics="opencnc.metrics,opencnc.events" \
  -output-dir="./add_stream/kafka_receiver" \
  -from-beginning=false
```

### CLI Flags & Environment Variables
| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| `-brokers` | `OBS_BROKERS` | `localhost:9092` | Kafka broker list (comma-separated) |
| `-topics` | `KAFKA_TOPICS` | `opencnc.metrics,opencnc.events` | Kafka topics to consume |
| `-group` | `KAFKA_GROUP` | `""` (ephemeral) | Consumer group ID (empty generates unique ID per run) |
| `-from-beginning` | - | `false` | Read from topic start if true; default false only captures current run |
| `-output-dir` | `OUTPUT_DIR` | `.` | Directory to save CSV files |
| `-raw-csv` | - | `metrics_raw.csv` | File name for raw metrics CSV |
| `-summary-csv` | - | `stream_latencies_summary.csv` | File name for summary CSV |
| `-flush-seconds` | - | `2` | Interval in seconds to flush CSV writers to disk |
