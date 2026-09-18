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

fail() {
    echo "ERROR: $*" >&2
    exit 1
}

require_linux() {
    [[ "$(uname -s)" == "Linux" ]] || \
        fail "This installer currently supports Linux only."
}

install_docker() {
    log "Installing Docker..."

    if ! command -v apt-get >/dev/null 2>&1; then
        fail "Docker is missing. Automatic installation supports Debian/Ubuntu."
    fi

    sudo apt-get update
    sudo apt-get install -y ca-certificates curl

    curl -fsSL https://get.docker.com | sudo sh
    sudo systemctl enable --now docker

    if ! docker info >/dev/null 2>&1; then
        sudo usermod -aG docker "$USER" || true
        fail "Docker was installed. Log out and back in, then run ./install.sh again."
    fi
}

install_kubectl() {
    log "Installing kubectl..."

    sudo apt-get update
    sudo apt-get install -y curl ca-certificates

    local version
    local arch

    version="$(curl -fsSL https://dl.k8s.io/release/stable.txt)"

    case "$(uname -m)" in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            fail "Unsupported CPU architecture: $(uname -m)"
            ;;
    esac

    curl -fsSL -o /tmp/kubectl \
        "https://dl.k8s.io/release/${version}/bin/linux/${arch}/kubectl"

    sudo install -m 0755 /tmp/kubectl /usr/local/bin/kubectl
    rm -f /tmp/kubectl
}

install_k3d() {
    log "Installing k3d..."

    curl -fsSL https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh \
        | TAG=v5.8.3 bash
}

ensure_dependencies() {
    if ! command -v curl >/dev/null 2>&1; then
        if ! command -v apt-get >/dev/null 2>&1; then
            fail "curl is missing. Automatic installation supports Debian/Ubuntu."
        fi

        sudo apt-get update
        sudo apt-get install -y curl ca-certificates
    fi

    if ! command -v docker >/dev/null 2>&1; then
        install_docker
    fi

    if ! docker info >/dev/null 2>&1; then
        if command -v systemctl >/dev/null 2>&1; then
            sudo systemctl start docker >/dev/null 2>&1 || true
        fi
    fi

    docker info >/dev/null 2>&1 || \
        fail "Docker is installed but the Docker daemon is not available."

    if ! command -v kubectl >/dev/null 2>&1; then
        install_kubectl
    fi
}

cluster_created_by_us="false"
cluster_context=""

save_state() {
    cat > "$STATE_FILE" <<EOF
CLUSTER_CREATED=$cluster_created_by_us
CLUSTER_NAME=$CLUSTER
CLUSTER_CONTEXT=$cluster_context
EOF
}

ensure_kubernetes() {
    if kubectl config current-context >/dev/null 2>&1 &&
       kubectl get nodes >/dev/null 2>&1; then

        cluster_context="$(kubectl config current-context)"

        case "$cluster_context" in
            k3d-*|kind-*|minikube|docker-desktop)
                log "Using existing Kubernetes cluster: $cluster_context"
                ;;
            *)
                fail "Unsupported Kubernetes context: $cluster_context

Supported contexts:
  k3d-*
  kind-*
  minikube
  docker-desktop"
                ;;
        esac

        return
    fi

    if ! command -v k3d >/dev/null 2>&1; then
        install_k3d
    fi

    if k3d cluster list 2>/dev/null |
        awk 'NR > 1 {print $1}' |
        grep -qx "$CLUSTER"; then

        log "Using existing k3d cluster: $CLUSTER"

        k3d cluster start "$CLUSTER" >/dev/null 2>&1 || true

        k3d kubeconfig merge "$CLUSTER" \
            --kubeconfig-switch-context >/dev/null

        cluster_context="k3d-${CLUSTER}"

        kubectl config use-context "$cluster_context" >/dev/null
    else
        log "Creating Kubernetes cluster: $CLUSTER"

        k3d cluster create "$CLUSTER" \
            --agents 1 \
            --port "8080:30080@loadbalancer" \
            --port "8081:30081@loadbalancer" \
            --wait

        cluster_created_by_us="true"
        cluster_context="k3d-${CLUSTER}"

        kubectl config use-context "$cluster_context" >/dev/null

        # Record ownership immediately.
        save_state
    fi

    kubectl wait \
        --for=condition=Ready \
        nodes \
        --all \
        --timeout=180s
}

build_images() {
    log "Building OpenCNC images..."

    docker build --target config_service \
        -t opencnc/config-service:latest .

    docker build --target monitor_service \
        -t opencnc/monitor-service:latest .

    docker build --target tsn_service \
        -t opencnc/tsn-service:latest .

    docker build --target main_service \
        -t opencnc/main-service:latest .

    docker build --target gui_service \
        -t opencnc/gui-service:latest .
}

load_images() {
    local context
    context="$(kubectl config current-context)"

    case "$context" in
        k3d-*)
            log "Loading images into k3d..."

            k3d image import \
                opencnc/config-service:latest \
                opencnc/monitor-service:latest \
                opencnc/tsn-service:latest \
                opencnc/main-service:latest \
                opencnc/gui-service:latest \
                -c "${context#k3d-}"
            ;;

        docker-desktop)
            log "Using Docker Desktop images."
            ;;

        minikube)
            log "Loading images into minikube..."

            minikube image load opencnc/config-service:latest
            minikube image load opencnc/monitor-service:latest
            minikube image load opencnc/tsn-service:latest
            minikube image load opencnc/main-service:latest
            minikube image load opencnc/gui-service:latest
            ;;

        kind-*)
            log "Loading images into kind..."

            local kind_name="${context#kind-}"

            kind load docker-image \
                opencnc/config-service:latest \
                --name "$kind_name"

            kind load docker-image \
                opencnc/monitor-service:latest \
                --name "$kind_name"

            kind load docker-image \
                opencnc/tsn-service:latest \
                --name "$kind_name"

            kind load docker-image \
                opencnc/main-service:latest \
                --name "$kind_name"

            kind load docker-image \
                opencnc/gui-service:latest \
                --name "$kind_name"
            ;;

        *)
            fail "Unsupported Kubernetes context: $context"
            ;;
    esac
}

deploy() {
    log "Creating OpenCNC namespace..."

    kubectl apply -f k8s/namespace.yaml

    log "Validating Kubernetes manifests..."

    kubectl apply \
        --dry-run=server \
        -f k8s/etcd.yaml \
        -f k8s/kafka.yaml \
        -f k8s/config-service.yaml \
        -f k8s/monitor-service.yaml \
        -f k8s/tsn-service.yaml \
        -f k8s/main-service.yaml \
        -f k8s/gui-service.yaml \
        >/dev/null

    log "Deploying OpenCNC..."

    kubectl apply -f k8s/etcd.yaml
    kubectl apply -f k8s/kafka.yaml

    kubectl apply -f k8s/config-service.yaml
    kubectl apply -f k8s/monitor-service.yaml
    kubectl apply -f k8s/tsn-service.yaml
    kubectl apply -f k8s/main-service.yaml
    kubectl apply -f k8s/gui-service.yaml
}

wait_for_services() {
    log "Waiting for services..."

    for deployment in \
        etcd \
        kafka \
        config-service \
        monitor-service \
        tsn-service \
        main-service \
        gui-service
    do
        kubectl rollout status \
            "deployment/$deployment" \
            -n "$NAMESPACE" \
            --timeout=300s
    done

    kubectl wait \
        --for=condition=complete \
        job/kafka-topics \
        -n "$NAMESPACE" \
        --timeout=300s
}

main() {
    require_linux

    log "Installing OpenCNC..."

    ensure_dependencies
    ensure_kubernetes
    build_images
    load_images
    deploy
    wait_for_services

    save_state

    echo
    echo "========================================"
    echo " OpenCNC is running"
    echo "========================================"
    echo
    echo "Kubernetes context:"
    echo "  $cluster_context"
    echo

    case "$cluster_context" in
        k3d-*)
            echo "GUI:"
            echo "  http://localhost:8080/"
            echo
            echo "UNI HTTP:"
            echo "  http://localhost:8081/"
            ;;
        *)
            echo "GUI:"
            echo "  http://<KUBERNETES-NODE-IP>:30080/"
            echo
            echo "UNI HTTP:"
            echo "  http://<KUBERNETES-NODE-IP>:30081/"
            ;;
    esac

    echo
    echo "Status:"
    echo "  kubectl get pods -n $NAMESPACE"
    echo
}

main "$@"
