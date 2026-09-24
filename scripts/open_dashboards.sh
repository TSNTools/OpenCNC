#!/usr/bin/env bash
# ==============================================================================
# Script: open_dashboards.sh
# Description: Port-forwards Istio observability dashboards (Kiali, Grafana, Prometheus).
#              Can be run on any machine where kubectl has access to the cluster.
# ==============================================================================

set -euo pipefail

PIDS=()

cleanup() {
    echo ""
    echo "Stopping dashboard port-forwarding..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    exit 0
}

trap cleanup SIGINT SIGTERM

echo "=========================================================="
echo " Starting Istio Observability Dashboards Port-Forwarding"
echo "=========================================================="

# 1. Kiali (Inter-pod communication, traffic graph, request/response latency)
if kubectl get svc kiali -n istio-system >/dev/null 2>&1; then
    kubectl port-forward svc/kiali -n istio-system 20001:20001 >/dev/null 2>&1 &
    PIDS+=($!)
    echo "✔ Kiali Dashboard (Inter-pod latency & service topology):"
    echo "    http://localhost:20001/kiali"
fi

# 2. Grafana (Pod CPU/Memory resource usage & Istio Workload metrics)
if kubectl get svc grafana -n istio-system >/dev/null 2>&1; then
    kubectl port-forward svc/grafana -n istio-system 3000:3000 >/dev/null 2>&1 &
    PIDS+=($!)
    echo "✔ Grafana (Pod resource usage & Istio dashboards):"
    echo "    http://localhost:3000"
fi

# 3. Prometheus (Metrics backend & raw PromQL queries)
if kubectl get svc prometheus -n istio-system >/dev/null 2>&1; then
    kubectl port-forward svc/prometheus -n istio-system 9090:9090 >/dev/null 2>&1 &
    PIDS+=($!)
    echo "✔ Prometheus (Raw metrics & queries):"
    echo "    http://localhost:9090"
fi

echo ""
echo "Press Ctrl+C to stop all port-forwarding."
wait
