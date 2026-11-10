#!/usr/bin/env bash
# Local OpenCNC/Kind development helper. Rebuilds/redeploys only tsn-service.
# The Go binary is compiled on the host, so Docker does not need credentials
# for the private tsn-optimization-contracts Go module.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../../.." && pwd)"
CLUSTER="tsn-oaas"
CONTEXT="kind-${CLUSTER}"
NAMESPACE="opencnc"
DEPLOYMENT="tsn-service"
IMAGE="opencnc/tsn-service:latest"

fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

for executable in git go docker kind kubectl; do
  command -v "$executable" >/dev/null 2>&1 || fail "Missing command: $executable"
done

[[ -f "${REPO_ROOT}/Dockerfile" && -f "${REPO_ROOT}/go.mod" && \
   -f "${REPO_ROOT}/tsn_service/configs/default-schedule.yaml" ]] \
  || fail "Place this script inside tsn_service/pkg/internalOptimizer/"

actual_context="$(kubectl config current-context)"
[[ "$actual_context" == "$CONTEXT" ]] \
  || fail "Kubernetes context is '$actual_context'; expected '$CONTEXT'. Not deploying."

docker info >/dev/null 2>&1 || fail "Docker daemon is not running."

actual_image="$(kubectl -n "$NAMESPACE" get deployment "$DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].image}')"
pull_policy="$(kubectl -n "$NAMESPACE" get deployment "$DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].imagePullPolicy}')"
[[ "$actual_image" == "$IMAGE" ]] \
  || fail "Deployment image is '$actual_image', expected '$IMAGE'. Review before proceeding."
[[ "$pull_policy" == "IfNotPresent" ]] \
  || fail "Deployment imagePullPolicy is '$pull_policy'; expected IfNotPresent for local Kind images."

# A host build must target the Linux architecture of the Kind node, not macOS.
node_architectures="$(kubectl get nodes -o jsonpath='{range .items[*]}{.status.nodeInfo.architecture}{"\n"}{end}')"
GOARCH_TARGET="$(printf '%s\n' "$node_architectures" | head -n 1)"
case "$GOARCH_TARGET" in
  arm64|amd64) ;;
  *) fail "Unexpected Kind node architecture: '$GOARCH_TARGET'" ;;
esac
while IFS= read -r arch; do
  [[ -z "$arch" || "$arch" == "$GOARCH_TARGET" ]] \
    || fail "Kind cluster has mixed architectures; cannot deploy one image safely."
done <<< "$node_architectures"

cd "$REPO_ROOT"
printf '\n==> Checking diff whitespace and conflicts\n'
git diff --check

printf '\n==> Running all internalOptimizer regression tests\n'
go test ./tsn_service/pkg/internalOptimizer/... -count=1

# The temporary context contains only the compiled binary and schedule config.
# No private module source code, git credentials, or Go cache enters Docker.
BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/opencnc-tsn-build.XXXXXXXX")"
trap 'rm -rf -- "$BUILD_DIR"' EXIT

printf '\n==> Compiling TSN Service on host for linux/%s\n' "$GOARCH_TARGET"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH_TARGET" \
  go build -o "$BUILD_DIR/tsn-service" ./tsn_service/cmd
cp tsn_service/configs/default-schedule.yaml "$BUILD_DIR/default-schedule.yaml"

# Match the tsn_service stage in the repository's Dockerfile.
cat > "$BUILD_DIR/Dockerfile" <<'DOCKERFILE'
FROM debian:bookworm-slim
COPY tsn-service /app/tsn-service
RUN mkdir -p /home/opencnc/OpenCNC/tsn_service/configs
COPY default-schedule.yaml /home/opencnc/OpenCNC/tsn_service/configs/default-schedule.yaml
EXPOSE 5152
ENTRYPOINT ["/app/tsn-service"]
DOCKERFILE

printf '\n==> Packaging only TSN Service image (no Docker Go dependency download)\n'
docker build --platform "linux/$GOARCH_TARGET" -t "$IMAGE" "$BUILD_DIR"

printf '\n==> Loading image into Kind cluster %s\n' "$CLUSTER"
kind load docker-image "$IMAGE" --name "$CLUSTER"

printf '\n==> Restarting only deployment/%s (existing environment retained)\n' "$DEPLOYMENT"
kubectl -n "$NAMESPACE" rollout restart "deployment/$DEPLOYMENT"
kubectl -n "$NAMESPACE" rollout status "deployment/$DEPLOYMENT" --timeout=180s

printf '\n==> Running TSN Service pod and OaaS settings\n'
kubectl -n "$NAMESPACE" get pods -l app=tsn-service
kubectl -n "$NAMESPACE" exec "deployment/$DEPLOYMENT" -- env | grep '^OAAS_' || true
printf '\nTSN Service redeployment completed.\n'
