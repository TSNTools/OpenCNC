#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

NAMESPACE="opencnc"
CLUSTER="opencnc"
STATE_FILE="$ROOT_DIR/.opencnc-install-state"

log() {
    echo "==> $*"
}

remove_opencnc() {
    if ! command -v kubectl >/dev/null 2>&1; then
        return
    fi

    if kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log "Removing OpenCNC Kubernetes resources..."

        kubectl delete namespace "$NAMESPACE" \
            --ignore-not-found=true \
            --wait=true
    else
        log "OpenCNC namespace is already removed."
    fi
}

remove_cluster_if_requested() {
    local cluster_created="$1"

    if [[ "$cluster_created" != "true" ]]; then
        return
    fi

    if ! command -v k3d >/dev/null 2>&1; then
        echo
        echo "The OpenCNC installer created the k3d cluster '$CLUSTER',"
        echo "but k3d is no longer installed."
        echo "The cluster cannot be removed by this script."
        echo
        return
    fi

    if ! k3d cluster list 2>/dev/null |
        awk 'NR > 1 {print $1}' |
        grep -qx "$CLUSTER"; then

        log "The k3d cluster '$CLUSTER' is already removed."
        return
    fi

    echo
    echo "The OpenCNC installer created the Kubernetes cluster '$CLUSTER'."
    echo
    read -r -p "Do you want to remove this Kubernetes cluster? [y/N]: " answer

    case "$answer" in
        y|Y|yes|YES)
            log "Removing Kubernetes cluster '$CLUSTER'..."
            k3d cluster delete "$CLUSTER"
            ;;
        *)
            log "Keeping Kubernetes cluster '$CLUSTER'."
            ;;
    esac
}

remove_images() {
    if ! command -v docker >/dev/null 2>&1; then
        return
    fi

    log "Removing OpenCNC Docker images..."

    docker image rm -f \
        opencnc/config-service:latest \
        opencnc/monitor-service:latest \
        opencnc/tsn-service:latest \
        opencnc/main-service:latest \
        opencnc/gui-service:latest \
        >/dev/null 2>&1 || true
}

cleanup_state() {
    rm -f "$STATE_FILE"
}

main() {
    local cluster_created="false"

    if [[ -f "$STATE_FILE" ]]; then
        # shellcheck disable=SC1090
        source "$STATE_FILE" || true
        cluster_created="${CLUSTER_CREATED:-false}"
    fi

    echo
    echo "========================================"
    echo " Uninstalling OpenCNC"
    echo "========================================"
    echo

    remove_opencnc
    remove_images
    remove_cluster_if_requested "$cluster_created"
    cleanup_state

    echo
    echo "========================================"
    echo " OpenCNC has been uninstalled"
    echo "========================================"
    echo
}

main "$@"