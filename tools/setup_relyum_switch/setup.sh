#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_DIR}"

if [[ -f "${SCRIPT_DIR}/setup_relyum" ]]; then
    "${SCRIPT_DIR}/setup_relyum"
else
    go run "${SCRIPT_DIR}/main.go"
fi
