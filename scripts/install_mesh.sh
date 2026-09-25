#!/usr/bin/env bash
# ==============================================================================
# Script: install_mesh.sh
# Description: Scalable, automated Istio service mesh installer with full
#              observability stack (Prometheus, Grafana, Kiali, Jaeger).
#              Compatible with kind, k3d, minikube, and cloud Kubernetes.
# ==============================================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

NAMESPACE="opencnc"
MESH_DIR="$ROOT_DIR/k8s/mesh"

log() {
    echo "==> $*"
}

fail() {
    echo "ERROR: $*" >&2
    exit 1
}

# 1. Verify Prerequisites
if ! command -v kubectl >/dev/null 2>&1; then
    fail "kubectl is required but not installed."
fi

if ! command -v istioctl >/dev/null 2>&1; then
    log "istioctl not found in PATH. Checking local installation..."
    if [[ -x "$HOME/istio-1.29.1/bin/istioctl" ]]; then
        export PATH="$HOME/istio-1.29.1/bin:$PATH"
    elif [[ -x "/usr/local/bin/istioctl" ]]; then
        export PATH="/usr/local/bin:$PATH"
    else
        log "Downloading and installing istioctl..."
        curl -sL https://istio.io/downloadIstio | sh -
        latest_istio="$(find "$PWD" -maxdepth 1 -name "istio-*" -type d | sort -V | tail -n1)"
        if [[ -n "$latest_istio" && -x "$latest_istio/bin/istioctl" ]]; then
            export PATH="$latest_istio/bin:$PATH"
        else
            fail "Failed to automatically install istioctl. Please install it manually."
        fi
    fi
fi

log "Using istioctl version: $(istioctl version --remote=false)"

# 2. Local Cluster Image Loading (if applicable)
context="$(kubectl config current-context 2>/dev/null || true)"
log "Kubernetes context: $context"

load_mesh_images() {
    local images=(
        "docker.io/istio/pilot:1.29.1"
        "docker.io/istio/proxyv2:1.29.1"
        "docker.io/prom/prometheus:v3.8.1"
        "docker.io/grafana/grafana:12.0.1"
        "quay.io/kiali/kiali:v2.21"
    )

    case "$context" in
        kind-*)
            local kind_name="${context#kind-}"
            log "Loading mesh images into kind ($kind_name)..."
            for img in "${images[@]}"; do
                if docker image inspect "$img" >/dev/null 2>&1; then
                    kind load docker-image "$img" --name "$kind_name" || true
                fi
            done
            ;;
        k3d-*)
            local k3d_name="${context#k3d-}"
            log "Loading mesh images into k3d ($k3d_name)..."
            for img in "${images[@]}"; do
                if docker image inspect "$img" >/dev/null 2>&1; then
                    k3d image import "$img" -c "$k3d_name" || true
                fi
            done
            ;;
        minikube)
            log "Loading mesh images into minikube..."
            for img in "${images[@]}"; do
                if docker image inspect "$img" >/dev/null 2>&1; then
                    minikube image load "$img" || true
                fi
            done
            ;;
        *)
            log "Standard cluster detected. Images will be pulled directly by the cluster runtime."
            ;;
    esac
}

load_mesh_images

# 3. Install Istio Control Plane (istiod)
log "Installing Istio control plane..."
istioctl install --set profile=minimal -y

log "Waiting for Istiod control plane readiness..."
kubectl rollout status deployment/istiod -n istio-system --timeout=300s

# 4. Enable Automatic Sidecar Injection on OpenCNC Namespace
log "Configuring namespace '$NAMESPACE' for automatic Istio sidecar injection..."
kubectl label namespace "$NAMESPACE" istio-injection=enabled --overwrite

# 5. Deploy Observability Addons (Prometheus, Grafana, Kiali, Jaeger)
log "Deploying observability stack (Prometheus, Grafana, Kiali)..."
if [[ -f "$MESH_DIR/prometheus.yaml" ]]; then
    kubectl apply -f "$MESH_DIR/prometheus.yaml"
fi
if [[ -f "$MESH_DIR/grafana.yaml" ]]; then
    kubectl apply -f "$MESH_DIR/grafana.yaml"
fi
if [[ -f "$MESH_DIR/kiali.yaml" ]]; then
    kubectl apply -f "$MESH_DIR/kiali.yaml"
fi

# 6. Restart OpenCNC Deployments to Inject Envoy Sidecars
log "Restarting OpenCNC services to inject Istio Envoy sidecars..."
for dep in config-service monitor-service tsn-service main-service gui-service; do
    if kubectl get deployment "$dep" -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl rollout restart deployment/"$dep" -n "$NAMESPACE"
    fi
done

log "Waiting for OpenCNC services to roll out with Envoy sidecars..."
for dep in config-service monitor-service tsn-service main-service gui-service; do
    if kubectl get deployment "$dep" -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl rollout status deployment/"$dep" -n "$NAMESPACE" --timeout=300s
    fi
done

# 7. Setup Grafana Kafka Datasource & Dashboard
log "Configuring Grafana Kafka consumer & OpenCNC dashboard..."
if [[ -x "$ROOT_DIR/scripts/setup_grafana_kafka.sh" ]]; then
    "$ROOT_DIR/scripts/setup_grafana_kafka.sh" || log "Notice: setup_grafana_kafka.sh can be run manually anytime: ./scripts/setup_grafana_kafka.sh"
fi

echo ""
echo "=========================================================="
echo " Istio Service Mesh & Observability Stack are READY!"
echo "=========================================================="
echo ""
echo "To open the dashboards, run:"
echo "    ./scripts/open_dashboards.sh"
echo ""
echo "Available Dashboards:"
echo "  • Kiali (Inter-pod traffic, service graph & latency): http://localhost:20001/kiali"
echo "  • Grafana (Pod CPU/Memory resource usage):          http://localhost:3000"
echo "  • Prometheus (Raw metrics & queries):                http://localhost:9090"
echo ""
