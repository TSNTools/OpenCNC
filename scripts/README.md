# OpenCNC Scripts & Observability Guide

This directory contains automation scripts for cluster management, port-forwarding, service mesh installation, and real-time Kafka visualization in Grafana.

---

## Quick Reference

| Script | Description |
| :--- | :--- |
| [`setup_grafana_kafka.sh`](file:///home/jittin/config-service/test/OpenCNC/scripts/setup_grafana_kafka.sh) | Installs Kafka plugin, configures Kafka consumer data source, and loads the OpenCNC dashboard |
| [`open_dashboards.sh`](file:///home/jittin/config-service/test/OpenCNC/scripts/open_dashboards.sh) | Port-forwards Grafana, Kiali, and Prometheus to `localhost` |
| [`install_mesh.sh`](file:///home/jittin/config-service/test/OpenCNC/scripts/install_mesh.sh) | Deploys Istio control plane, sidecars, and observability stack |
| [`forward_opencnc.sh`](file:///home/jittin/config-service/test/OpenCNC/scripts/forward_opencnc.sh) | Forwards OpenCNC microservice ports (GUI, UNI HTTP, UNI gRPC) |

---

## 1. Setup Kafka Consumer & Dashboard in Grafana

If you recreated the cluster, deleted kind, or lost Grafana configs, run:

```bash
cd /home/jittin/config-service/test/OpenCNC
./scripts/setup_grafana_kafka.sh
```

### What this script does automatically:
1. **Detects the Grafana pod** running in namespace `istio-system`.
2. **Installs the Kafka Datasource Plugin** (`hamedkarbasi93-kafka-datasource`) into `/var/lib/grafana/plugins/` and reloads the container cleanly.
3. **Provisions the Kafka Data Source** pointing to `kafka.opencnc.svc.cluster.local:9092` with fixed UID `cfz4vv44j5v5sf` and validates connectivity.
4. **Imports the Dashboard JSON** ([`opencnc_logs_dashboard.json`](file:///home/jittin/config-service/test/OpenCNC/scripts/dashboards/opencnc_logs_dashboard.json)) into Grafana via REST API.

---

## 2. Load a Custom or Modified Dashboard JSON

You can pass any custom dashboard JSON file directly to the script:

```bash
# Load your own dashboard JSON file:
./scripts/setup_grafana_kafka.sh path/to/your_dashboard.json
```

The script automatically binds the Kafka datasource UID (`cfz4vv44j5v5sf`) to all Kafka query panels in your JSON file and overwrites/updates the dashboard in Grafana.

---

## 3. View the Visualizations in Grafana

### Step 1: Start Dashboard Port-Forwarding

```bash
./scripts/open_dashboards.sh
```

### Step 2: Open Grafana

- **Direct OpenCNC Logs & Latency Dashboard:**  
  👉 **[http://localhost:3000/d/8ec5168c-3dd8-4dce-b5bb-b89826b63f1f/opencnc-logs-and-latency-monitor](http://localhost:3000/d/8ec5168c-3dd8-4dce-b5bb-b89826b63f1f/opencnc-logs-and-latency-monitor)**
- **Grafana Home:** [http://localhost:3000](http://localhost:3000)  
  *(Credentials: Username: `admin`, Password: `admin`)*

---

## 4. How to Trigger Workloads and Visualize Live Data

When you execute any OpenCNC workflow (e.g. creating a stream or uploading topology), the microservices publish structured logs and events to Kafka. Grafana visualizes them in real-time.

### Example A: Trigger a Single Stream Request

In another terminal, run the sample gRPC stream addition:

```bash
cd /home/jittin/config-service/test/OpenCNC/add_stream/single
./send_add_stream_grpc.sh
```

### Example B: Trigger Iterative / Continuous Streams

To test streaming visualization with multiple requests:

```bash
cd /home/jittin/config-service/test/OpenCNC/add_stream/itterative
./send_add_stream_grpc.sh
```

### What Happens Behind the Scenes:
```mermaid
sequenceDiagram
    autonumber
    actor User as Test Script (e.g. send_add_stream_grpc.sh)
    participant Main as main-service
    participant TSN as tsn-service
    participant Config as config-service
    participant Kafka as Kafka Broker (kafka.opencnc:9092)
    participant Receiver as kafka-receiver Pod
    participant Grafana as Grafana Dashboard

    User->>Main: AddStreamRequest (UNI gRPC)
    Main->>Kafka: Publish log & UNI start (opencnc.logs)
    Main->>TSN: Calculate Schedule
    TSN->>Kafka: Publish TSN calc duration (opencnc.metrics)
    Main->>Config: Apply Configuration
    Config->>Kafka: Publish Prep & NETCONF Commit duration (opencnc.metrics)
    Main->>User: AddStreamResponse (Success)
    Main->>Kafka: Publish uni_request_response_duration (opencnc.metrics)

    Kafka->>Receiver: Consume opencnc.metrics & opencnc.events
    Receiver->>Kafka: Correlate latencies & publish to opencnc.latencies.json

    Kafka-->>Grafana: Live Stream: opencnc.logs (Plaintext Logs Panel)
    Kafka-->>Grafana: Live Stream: opencnc.latencies.json (Timeseries & Table Panels)
```

---

## 5. Dashboard Panels Explained

The included dashboard ([`scripts/dashboards/opencnc_logs_dashboard.json`](file:///home/jittin/config-service/test/OpenCNC/scripts/dashboards/opencnc_logs_dashboard.json)) contains:

1. **OpenCNC System Logs (Kafka Consumer: `opencnc.logs`)**:
   - **Type**: `logs`
   - **Data**: Displays live, searchable log messages produced by `main-service`, `tsn-service`, `config-service`, and `monitor-service`.
   - **Features**: Auto-scroll, text wrapping, and severity filtering.

2. **Stream Latency Breakdown (Kafka JSON: `opencnc.latencies.json`)**:
   - **Type**: `timeseries` stacked bars
   - **Data**: Visualizes hop-by-hop latency breakdowns for each stream request:
     - `total_uni_latency_ms`: Total end-to-end response time
     - `tsn_calc_latency_ms`: Scheduling calculation duration in `tsn-service`
     - `config_app_latency_ms`: Total deployment time in `config-service`
     - `config_prep_latency_ms`: Internal mapping and YANG preparation
     - `config_commit_latency_ms`: Physical NETCONF commit time on the switch
     - `main_service_overhead_ms`: Orchestration and gRPC/NNI overhead

3. **Recent Stream Latencies Summary Table**:
   - **Type**: `table`
   - **Data**: Tabular list of completed stream operations with timestamp, correlation ID, interface (`grpc`/`http`), and durations.

4. **OpenCNC Pods CPU Usage**:
   - **Type**: `timeseries` (Prometheus)
   - **Data**: Live CPU consumption (in mCPU) per microservice pod.

5. **OpenCNC Pods Memory Usage**:
   - **Type**: `timeseries` (Prometheus)
   - **Data**: Working set memory (in MB) per microservice pod.

---

## 6. Kafka Topics Reference

| Topic | Format | Emitted By | Visualized In Grafana |
| :--- | :--- | :--- | :--- |
| `opencnc.logs` | Plaintext / JSON | All microservices | Panel: "OpenCNC System Logs" |
| `opencnc.latencies.json` | JSON (`LatencySummaryJSON`) | `kafka-receiver` pod | Panels: "Latency Breakdown" & "Summary Table" |
| `opencnc.metrics` | Protobuf (`EventEnvelope`) | All microservices | Consumed by `kafka-receiver` |
| `opencnc.events` | Protobuf (`EventEnvelope`) | All microservices | Consumed by `kafka-receiver` |
| `opencnc.audit` | Protobuf (`EventEnvelope`) | All microservices | In-cluster auditing |
| `opencnc.health` | Protobuf (`EventEnvelope`) | All microservices | Subsystem health checks |

---

## 7. Troubleshooting

### Check Kafka Datasource Health
Run the following curl command (requires `./scripts/open_dashboards.sh` or port-forward active):
```bash
curl -s http://admin:admin@localhost:3000/api/datasources/uid/cfz4vv44j5v5sf/health
# Expected output: {"message":"Data source is working","status":"OK"}
```

### Inspect Kafka Topics Inside the Cluster
```bash
KAFKA_POD=$(kubectl get pods -n opencnc -l app=kafka -o jsonpath='{.items[0].metadata.name}')

# List topics:
kubectl exec -n opencnc "$KAFKA_POD" -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# Follow live messages on opencnc.logs:
kubectl exec -n opencnc "$KAFKA_POD" -- /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic opencnc.logs --from-beginning

# Follow live JSON summaries on opencnc.latencies.json:
kubectl exec -n opencnc "$KAFKA_POD" -- /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic opencnc.latencies.json --from-beginning
```

### Reapply Everything from Scratch
If you wipe or recreate the cluster:
```bash
./install.sh --with-mesh
./scripts/setup_grafana_kafka.sh
./scripts/open_dashboards.sh
```

---

## 8. Visualizing Pod CPU & Memory Usage (Query Commands)

In Kubernetes and Grafana, pod resource metrics (CPU and Memory) are collected by Prometheus via `cAdvisor` and queried using **PromQL** (Prometheus Query Language, the standard metric query language equivalent to SQL).

### A. CPU Usage Queries

#### 1. CPU Usage per Pod in Millicores (`mCPU`) *(Recommended)*
Returns the CPU consumption for each application pod in the `opencnc` namespace in millicores (1 core = 1000m):
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="opencnc", container!="", container!="POD", container!="istio-proxy"}[1m])) by (pod) * 1000
```

#### 2. CPU Usage per Pod in Cores
Returns the CPU consumption in fractional cores (e.g. 0.05 cores):
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="opencnc", container!="", container!="POD", container!="istio-proxy"}[1m])) by (pod)
```

#### 3. CPU Usage across ALL Namespaces
Returns CPU for every pod in the entire cluster:
```promql
sum(rate(container_cpu_usage_seconds_total{container!="", container!="POD", container!="istio-proxy"}[1m])) by (namespace, pod) * 1000
```

#### 4. CPU Usage for a Single Specific Pod (e.g. `main-service`)
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="opencnc", pod=~"main-service.*", container!="", container!="POD"}[1m])) * 1000
```

---

### B. Memory Usage Queries

#### 1. Memory Working Set per Pod (in Megabytes - MB) *(Recommended)*
Returns actual physical memory consumed (excluding cached filesystem blocks that can be reclaimed):
```promql
sum(container_memory_working_set_bytes{namespace="opencnc", container!="", container!="POD", container!="istio-proxy"}) by (pod) / (1024 * 1024)
```

#### 2. Memory Working Set per Pod (in Raw Bytes)
Use this if setting the Grafana panel unit to **Data (IEC) > bytes (B)**:
```promql
sum(container_memory_working_set_bytes{namespace="opencnc", container!="", container!="POD", container!="istio-proxy"}) by (pod)
```

---

### C. How to Add a New CPU / Memory Panel in Grafana

1. Open Grafana at [http://localhost:3000](http://localhost:3000).
2. Go to your dashboard:
   - Either navigate to the OpenCNC dashboard ([http://localhost:3000/d/8ec5168c-3dd8-4dce-b5bb-b89826b63f1f/opencnc-logs-and-latency-monitor](http://localhost:3000/d/8ec5168c-3dd8-4dce-b5bb-b89826b63f1f/opencnc-logs-and-latency-monitor)) or click **Dashboards > New > New Dashboard**.
3. Click **Add > Visualization** (or **Add Panel**).
4. In the **Data source** dropdown, select **Prometheus** (default: `http://prometheus:9090`).
5. In the **Metrics query** field (code editor mode), paste the query:
   ```promql
   sum(rate(container_cpu_usage_seconds_total{namespace="opencnc", container!="", container!="POD", container!="istio-proxy"}[1m])) by (pod) * 1000
   ```
6. In the panel options (right sidebar):
   - **Title**: `OpenCNC Microservices CPU Usage`
   - **Legend > Format**: Set to `{{pod}}` (this cleans up the series labels to only show the pod name).
   - **Standard options > Unit**:
     - For `* 1000` (mCPU): Choose **Short** or custom unit `mCPU`.
     - For raw cores: Choose **Misc > Hertz (Hz)** or leave as **Short**.
     - For Memory: Choose **Data (IEC) > Megabytes (MiB)** or **Data (IEC) > Bytes (B)**.
7. Click **Apply** in the top right corner.

