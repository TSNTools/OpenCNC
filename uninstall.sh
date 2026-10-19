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
    if ! command -v k3d >/dev/null 2>&1; then
        echo
        echo "k3d is not installed."
        echo "The Kubernetes cluster cannot be removed by this script."
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
    echo "The Kubernetes cluster '$CLUSTER' exists."
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

    log "Finding OpenCNC Docker images..."

    local images
    images="$(docker images 'opencnc/*' -aq 2>/dev/null | sort -u || true)"

    if [[ -n "$images" ]]; then
        log "Removing containers using OpenCNC images..."

        local containers=""
        local image_id
        local container_ids

        while read -r image_id; do
            [[ -n "$image_id" ]] || continue

            container_ids="$(docker ps -aq --no-trunc \
                --filter "ancestor=$image_id" 2>/dev/null || true)"

            if [[ -n "$container_ids" ]]; then
                containers="${containers}${container_ids}"$'\n'
            fi
        done <<< "$images"

        containers="$(printf '%s\n' "$containers" | sed '/^$/d' | sort -u)"

        if [[ -n "$containers" ]]; then
            docker rm -f $containers >/dev/null 2>&1 || true
        fi

        log "Removing OpenCNC Docker images..."

        docker rmi -f $images >/dev/null 2>&1 || true
    else
        log "No OpenCNC Docker images found."
    fi

    log "Removing dangling Docker images..."

    docker image prune -f >/dev/null 2>&1 || true
}

cleanup_state() {
    rm -f "$STATE_FILE"
}

main() {
    echo
    echo "========================================"
    echo " Uninstalling OpenCNC"
    echo "========================================"
    echo

    remove_opencnc
    remove_images
    remove_cluster_if_requested
    cleanup_state

    echo
    echo "========================================"
    echo " OpenCNC has been uninstalled"
    echo "========================================"
    echo
}

main "$@"
