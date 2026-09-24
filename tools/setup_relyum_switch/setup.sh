#!/usr/bin/env bash
# ==============================================================================
# Script: setup.sh
# Description: Sets up Relyum switch device model, credentials, and topology in etcd.
#              Auto-detects Kubernetes etcd service and forwards port 2379 if needed.
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PF_PID=""
cleanup() {
    if [[ -n "${PF_PID}" ]]; then
        kill "${PF_PID}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

ETCD_TARGET_HOST="${ETCD_HOST:-localhost}"
ETCD_TARGET_PORT="${ETCD_PORT:-2379}"

# If targeting localhost and port 2379 is not reachable, check for Kubernetes etcd
if [[ "${ETCD_TARGET_HOST}" == "localhost" || "${ETCD_TARGET_HOST}" == "127.0.0.1" ]]; then
    if ! (exec 3<>/dev/tcp/127.0.0.1/"${ETCD_TARGET_PORT}") 2>/dev/null; then
        if command -v kubectl >/dev/null 2>&1 && kubectl get svc etcd -n opencnc >/dev/null 2>&1; then
            echo "[i] Port ${ETCD_TARGET_PORT} not open locally. Auto port-forwarding svc/etcd..."
            kubectl port-forward -n opencnc svc/etcd "${ETCD_TARGET_PORT}":2379 >/dev/null 2>&1 &
            PF_PID=$!
            for _ in {1..25}; do
                if (exec 3<>/dev/tcp/127.0.0.1/"${ETCD_TARGET_PORT}") 2>/dev/null; then
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

cd "${REPO_DIR}"

# Recompile if binary is missing or main.go was updated
if [[ ! -f "${SCRIPT_DIR}/setup_relyum" || "${SCRIPT_DIR}/main.go" -nt "${SCRIPT_DIR}/setup_relyum" ]]; then
    echo "[i] Compiling setup_relyum..."
    go build -o "${SCRIPT_DIR}/setup_relyum" "${SCRIPT_DIR}/main.go"
fi

"${SCRIPT_DIR}/setup_relyum"
