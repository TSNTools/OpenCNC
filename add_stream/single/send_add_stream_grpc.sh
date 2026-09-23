#!/usr/bin/env bash
# ==============================================================================
# Script: send_add_stream_grpc.sh
# Description: Sends an Add Stream request to OpenCNC UNI gRPC endpoint containing
#              two TSN stream requests:
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
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

DEFAULT_ENDPOINT="localhost:5153"
ENDPOINT="${1:-${CNC_GRPC_ENDPOINT:-$DEFAULT_ENDPOINT}}"
TIMEOUT="${TIMEOUT:-60s}"

show_help() {
    echo "Usage: $0 [GRPC_HOST:PORT]"
    echo ""
    echo "Options:"
    echo "  GRPC_HOST:PORT   OpenCNC UNI gRPC endpoint (default: ${DEFAULT_ENDPOINT})"
    echo "  -h, --help       Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  CNC_GRPC_ENDPOINT  Override the default gRPC endpoint (e.g. localhost:5153)"
    echo "  TIMEOUT            Request timeout (default: 60s)"
    echo ""
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

# Ensure binary is built or run via go
cd "${REPO_DIR}"

if [[ -f "${SCRIPT_DIR}/send_grpc" ]]; then
    "${SCRIPT_DIR}/send_grpc" -target "${ENDPOINT}" -timeout "${TIMEOUT}"
else
    go run "${SCRIPT_DIR}/main.go" -target "${ENDPOINT}" -timeout "${TIMEOUT}"
fi
