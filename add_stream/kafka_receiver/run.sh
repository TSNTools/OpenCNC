#!/usr/bin/env bash
set -e

# Default settings (can be overridden via environment variables)
KAFKA_BROKERS="${OBS_BROKERS:-localhost:9092}"
KAFKA_TOPICS="${KAFKA_TOPICS:-opencnc.metrics,opencnc.events}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"
GROUP_ID="${KAFKA_GROUP:-opencnc-latency-collector-$(date +%s)}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=================================================================="
echo " Starting OpenCNC Kafka Latency Receiver"
echo "=================================================================="
echo " Kafka Brokers : ${KAFKA_BROKERS}"
echo " Topics        : ${KAFKA_TOPICS}"
echo " Output Dir    : ${SCRIPT_DIR}/${OUTPUT_DIR}"
echo " Root Dir      : ${ROOT_DIR}"
echo "=================================================================="

cd "${ROOT_DIR}"

go run ./add_stream/kafka_receiver \
  -brokers="${KAFKA_BROKERS}" \
  -topics="${KAFKA_TOPICS}" \
  -group="${GROUP_ID}" \
  -from-beginning=false \
  -output-dir="${SCRIPT_DIR}/${OUTPUT_DIR}" \
  -raw-csv="metrics_raw.csv" \
  -summary-csv="stream_latencies_summary.csv"
