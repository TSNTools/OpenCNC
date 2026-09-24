#!/usr/bin/env bash
# ==============================================================================
# Script: fetch_k8s_csv.sh
# Description: Copies the generated CSV files directly from the running in-cluster
#              kafka-receiver pod to the local workspace.
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEST_DIR="${1:-${ROOT_DIR}}"
NAMESPACE="opencnc"

if ! command -v kubectl >/dev/null 2>&1; then
    echo "[✗] kubectl is not installed or not in PATH."
    exit 1
fi

POD_NAME="$(kubectl get pods -n "${NAMESPACE}" -l app=kafka-receiver -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"

if [[ -z "${POD_NAME}" ]]; then
    echo "[✗] No running kafka-receiver pod found in namespace '${NAMESPACE}'."
    exit 1
fi

echo "=================================================================="
echo " Fetching CSVs from Kubernetes Pod"
echo "=================================================================="
echo " Pod Name       : ${POD_NAME}"
echo " Remote Path    : /data/"
echo " Destination Dir: ${DEST_DIR}"
echo "=================================================================="

mkdir -p "${DEST_DIR}"

if kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -- test -f /data/stream_latencies_summary.csv 2>/dev/null; then
    kubectl cp "${NAMESPACE}/${POD_NAME}:/data/stream_latencies_summary.csv" "${DEST_DIR}/stream_latencies_summary.csv"
    echo "[✓] Fetched stream_latencies_summary.csv"
else
    echo "[!] /data/stream_latencies_summary.csv not found in pod yet."
fi

if kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -- test -f /data/metrics_raw.csv 2>/dev/null; then
    kubectl cp "${NAMESPACE}/${POD_NAME}:/data/metrics_raw.csv" "${DEST_DIR}/metrics_raw.csv"
    echo "[✓] Fetched metrics_raw.csv"
else
    echo "[!] /data/metrics_raw.csv not found in pod yet."
fi

echo "=================================================================="
echo " Done! Files saved to ${DEST_DIR}"
echo "=================================================================="
