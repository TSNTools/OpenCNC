#!/usr/bin/env bash
# ==============================================================================
# Script: run.sh
# Description: Starts the OpenCNC Kafka Latency Receiver to capture metrics and
#              events from Kafka and save them to CSV files.
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Default settings (can be overridden via environment variables)
KAFKA_BROKERS="${OBS_BROKERS:-localhost:9092}"
KAFKA_TOPICS="${KAFKA_TOPICS:-opencnc.metrics,opencnc.events,opencnc.logs}"
JSON_TOPIC="${KAFKA_JSON_TOPIC:-opencnc.latencies.json}"
OUTPUT_DIR="${OUTPUT_DIR:-${ROOT_DIR}}"
FROM_BEGINNING="${FROM_BEGINNING:-false}"
GROUP_ID="${KAFKA_GROUP:-opencnc-latency-collector-$(date +%s)}"
FLUSH_INTERVAL="${FLUSH_INTERVAL:-1}"

show_help() {
    echo "Usage: $0 [KAFKA_BROKERS]"
    echo ""
    echo "Options:"
    echo "  KAFKA_BROKERS    Kafka broker addresses (default: ${KAFKA_BROKERS})"
    echo "  -h, --help       Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  OBS_BROKERS      Kafka brokers (default: localhost:9092)"
    echo "  KAFKA_TOPICS     Kafka topics (default: opencnc.metrics,opencnc.events,opencnc.logs)"
    echo "  OUTPUT_DIR       Directory to save CSV files (default: ${ROOT_DIR})"
    echo "  FROM_BEGINNING   Read from beginning if true (default: false)"
    echo "  KAFKA_GROUP      Consumer group ID (default: unique per run)"
    echo "  FLUSH_INTERVAL   Flush interval in seconds (default: 1)"
    echo ""
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

if [[ $# -ge 1 ]]; then
    KAFKA_BROKERS="$1"
fi

PF_PID=""
cleanup() {
    if [[ -n "${PF_PID}" ]]; then
        kill "${PF_PID}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

# If targeting localhost:9092 and not reachable, check if k8s svc/kafka is available and auto port-forward
if [[ "${KAFKA_BROKERS}" == *"localhost:9092"* || "${KAFKA_BROKERS}" == *"127.0.0.1:9092"* ]]; then
    if ! (exec 3<>/dev/tcp/127.0.0.1/9092) 2>/dev/null; then
        if command -v kubectl >/dev/null 2>&1 && kubectl get svc kafka -n opencnc >/dev/null 2>&1; then
            echo "[i] Port 9092 is not open locally. Auto port-forwarding svc/kafka..."
            kubectl port-forward -n opencnc svc/kafka 9092:9092 >/dev/null 2>&1 &
            PF_PID=$!
            for _ in {1..25}; do
                if (exec 3<>/dev/tcp/127.0.0.1/9092) 2>/dev/null; then
                    exec 3>&-
                    break
                fi
                sleep 0.2
            done
        fi
    else
        exec 3>&-
    fi
fi

echo "=================================================================="
echo " Starting OpenCNC Kafka Latency Receiver"
echo "=================================================================="
echo " Kafka Brokers   : ${KAFKA_BROKERS}"
echo " Topics          : ${KAFKA_TOPICS}"
echo " From Beginning  : ${FROM_BEGINNING}"
echo " Output Dir      : ${OUTPUT_DIR}"
echo " Raw CSV File    : ${OUTPUT_DIR}/metrics_raw.csv"
echo " Summary CSV File: ${OUTPUT_DIR}/stream_latencies_summary.csv"
echo "=================================================================="

cd "${ROOT_DIR}"

if [[ -f "${SCRIPT_DIR}/kafka_receiver" ]]; then
    "${SCRIPT_DIR}/kafka_receiver" \
        -brokers="${KAFKA_BROKERS}" \
        -topics="${KAFKA_TOPICS}" \
        -json-topic="${JSON_TOPIC}" \
        -group="${GROUP_ID}" \
        -from-beginning="${FROM_BEGINNING}" \
        -output-dir="${OUTPUT_DIR}" \
        -flush-seconds="${FLUSH_INTERVAL}" \
        -raw-csv="metrics_raw.csv" \
        -summary-csv="stream_latencies_summary.csv"
else
    go run ./add_stream/kafka_receiver \
        -brokers="${KAFKA_BROKERS}" \
        -topics="${KAFKA_TOPICS}" \
        -json-topic="${JSON_TOPIC}" \
        -group="${GROUP_ID}" \
        -from-beginning="${FROM_BEGINNING}" \
        -output-dir="${OUTPUT_DIR}" \
        -flush-seconds="${FLUSH_INTERVAL}" \
        -raw-csv="metrics_raw.csv" \
        -summary-csv="stream_latencies_summary.csv"
fi
