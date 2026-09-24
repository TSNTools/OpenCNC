#!/usr/bin/env bash
# ==============================================================================
# Script: forward_opencnc.sh
# Description: Port-forwards OpenCNC microservice endpoints to localhost.
#              - UNI gRPC:     localhost:5153 -> svc/main-service:5153
#              - UNI HTTP:     localhost:8081 -> svc/main-service:8081
#              - NNI HTTP:     localhost:8000 -> svc/main-service:8000
#              - GUI Web:      localhost:8080 -> svc/gui-service:8080
#              - etcd store:   localhost:2379 -> svc/etcd:2379
# ==============================================================================

set -euo pipefail

NAMESPACE="${OPENCNC_NAMESPACE:-opencnc}"
PIDS=()

cleanup() {
    echo ""
    echo "Stopping OpenCNC port-forwarding..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    exit 0
}

trap cleanup SIGINT SIGTERM

if ! command -v kubectl >/dev/null 2>&1; then
    echo "ERROR: kubectl is required but not installed." >&2
    exit 1
fi

echo "=========================================================="
echo " Starting OpenCNC Port-Forwarding (namespace: $NAMESPACE)"
echo "=========================================================="

port_is_open() {
    local port="$1"
    (exec 3<>/dev/tcp/127.0.0.1/"$port") 2>/dev/null && exec 3>&- && return 0
    return 1
}

# 1. UNI gRPC (5153)
if port_is_open 5153; then
    echo "✔ Port 5153 is already listening locally."
else
    if kubectl get svc main-service -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl port-forward svc/main-service -n "$NAMESPACE" 5153:5153 >/dev/null 2>&1 &
        PIDS+=($!)
        echo "✔ Forwarded UNI gRPC: localhost:5153 -> svc/main-service:5153"
    fi
fi

# 2. UNI HTTP (8081)
if port_is_open 8081; then
    echo "✔ Port 8081 is already listening locally."
else
    if kubectl get svc main-service -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl port-forward svc/main-service -n "$NAMESPACE" 8081:8081 >/dev/null 2>&1 &
        PIDS+=($!)
        echo "✔ Forwarded UNI HTTP: http://localhost:8081 -> svc/main-service:8081"
    fi
fi

# 3. NNI HTTP (8000)
if port_is_open 8000; then
    echo "✔ Port 8000 is already listening locally."
else
    if kubectl get svc main-service -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl port-forward svc/main-service -n "$NAMESPACE" 8000:8000 >/dev/null 2>&1 &
        PIDS+=($!)
        echo "✔ Forwarded NNI HTTP: http://localhost:8000 -> svc/main-service:8000"
    fi
fi

# 4. GUI Web (8080)
if port_is_open 8080; then
    echo "✔ Port 8080 is already listening locally."
else
    if kubectl get svc gui-service -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl port-forward svc/gui-service -n "$NAMESPACE" 8080:8080 >/dev/null 2>&1 &
        PIDS+=($!)
        echo "✔ Forwarded GUI Web:  http://localhost:8080 -> svc/gui-service:8080"
    fi
fi

# 5. etcd (2379)
if port_is_open 2379; then
    echo "✔ Port 2379 is already listening locally."
else
    if kubectl get svc etcd -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl port-forward svc/etcd -n "$NAMESPACE" 2379:2379 >/dev/null 2>&1 &
        PIDS+=($!)
        echo "✔ Forwarded etcd store: localhost:2379 -> svc/etcd:2379"
    fi
fi

echo ""
echo "All available OpenCNC endpoints are forwarded to localhost."
echo "Press Ctrl+C to stop all port-forwarding."
wait
