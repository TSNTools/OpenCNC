#!/usr/bin/env bash
# ==============================================================================
# Script: add_stream.sh
# Description: Sends Add Stream requests to OpenCNC UNI gRPC endpoint repeated
#              iteratively (default 50 times):
#              - Stream 1: Talker on tsn1 (enp1s0) -> Listener on tsn2 (ens6)
#              - Stream 2: Talker on tsn2 (ens6)   -> Listener on tsn1 (enp1s0)
#              Parameters for both streams:
#              - VLAN ID: 100
#              - PCP: 7 (Socket Priority 7 / Gate 7)
#              - Bridge Configuration: Required (VLAN Tag Capable = true)
#              - Cycle Time: 1 second
#              - Frames per cycle: 32
#              - Frame length: 128 bytes
#              Target Endpoint: localhost:5153 (uni_server.UniService/AddStream)
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEFAULT_ENDPOINT="localhost:5153"
ENDPOINT="${1:-${CNC_GRPC_ENDPOINT:-$DEFAULT_ENDPOINT}}"
TIMEOUT="${TIMEOUT:-60s}"
ITERATIONS="${ITERATIONS:-50}"
DELAY="${DELAY:-0s}"

show_help() {
    echo "Usage: $0 [GRPC_HOST:PORT]"
    echo ""
    echo "Options:"
    echo "  GRPC_HOST:PORT   OpenCNC UNI gRPC endpoint (default: ${DEFAULT_ENDPOINT})"
    echo "  -h, --help       Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  CNC_GRPC_ENDPOINT  Override the default gRPC endpoint (e.g. localhost:5153)"
    echo "  TIMEOUT            Request timeout per iteration (default: 60s)"
    echo "  ITERATIONS         Number of iterations (default: 50)"
    echo "  DELAY              Delay between iterations (default: 0s)"
    echo ""
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

PF_PID=""
cleanup() {
    if [[ -n "${PF_PID}" ]]; then
        kill "${PF_PID}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

# If targeting local endpoint and port 5153 is not currently reachable,
# check if Kubernetes main-service is available and auto port-forward.
if [[ "${ENDPOINT}" == "localhost:5153" || "${ENDPOINT}" == "127.0.0.1:5153" ]]; then
    if ! (exec 3<>/dev/tcp/127.0.0.1/5153) 2>/dev/null; then
        if command -v kubectl >/dev/null 2>&1 && kubectl get svc main-service -n opencnc >/dev/null 2>&1; then
            echo "[i] Port 5153 is not open locally. Auto port-forwarding svc/main-service..."
            kubectl port-forward -n opencnc svc/main-service 5153:5153 >/dev/null 2>&1 &
            PF_PID=$!
            for _ in {1..25}; do
                if (exec 3<>/dev/tcp/127.0.0.1/5153) 2>/dev/null; then
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

if [[ -f "${SCRIPT_DIR}/send_grpc" ]]; then
    "${SCRIPT_DIR}/send_grpc" -target "${ENDPOINT}" -timeout "${TIMEOUT}" -iterations "${ITERATIONS}" -delay "${DELAY}"
else
    go run "${SCRIPT_DIR}/main.go" -target "${ENDPOINT}" -timeout "${TIMEOUT}" -iterations "${ITERATIONS}" -delay "${DELAY}"
fi
