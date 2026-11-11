#!/usr/bin/env bash
# Non-destructive restart helper for the *existing* local Kind B10 pilot.
# Install at tsn_service/pkg/internalOptimizer/start-b10-demo.sh.
# It does not create a new cluster, rebuild images, submit streams or modify CNC source.
set -euo pipefail

CLUSTER="tsn-oaas"
CONTEXT="kind-${CLUSTER}"
NAMESPACE="opencnc"
REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)"
STATE_DIR="${HOME}/.cache/tsn-b10-demo"
mkdir -p "$STATE_DIR"

fail() { printf '\nERROR: %s\n' "$*" >&2; exit 1; }
note() { printf '\n==> %s\n' "$*"; }

[[ -f "$REPO_ROOT/go.mod" && -d "$REPO_ROOT/k8s" ]] \
  || fail "Place script in OpenCNC/tsn_service/pkg/internalOptimizer/"
for cmd in docker kubectl curl lsof; do
  command -v "$cmd" >/dev/null 2>&1 || fail "Missing $cmd in PATH"
done

docker info >/dev/null 2>&1 || fail "Docker is not running. Open Docker Desktop first."

note "Checking existing Kind cluster ${CLUSTER}"
NODE="${CLUSTER}-control-plane"
if ! docker ps -a --format '{{.Names}}' | grep -Fxq "$NODE"; then
  fail "Kind node '$NODE' is missing. This script does not recreate an empty cluster or its topology."
fi
if ! docker ps --format '{{.Names}}' | grep -Fxq "$NODE"; then
  note "Starting stopped Kind node ${NODE}"
  docker start "$NODE" >/dev/null
fi

# Always address the intended cluster, regardless of the user's default context.
for i in $(seq 1 45); do
  if kubectl --context "$CONTEXT" get nodes --no-headers 2>/dev/null | grep -q ' Ready '; then
    break
  fi
  if [[ "$i" -eq 45 ]]; then fail "Kind API/node not Ready; inspect docker logs $NODE"; fi
  sleep 2
done

note "Waiting for existing OpenCNC services"
for deployment in etcd kafka config-service main-service tsn-service gui-service monitor-service; do
  kubectl --context "$CONTEXT" -n "$NAMESPACE" \
    rollout status "deployment/$deployment" --timeout=120s \
    || fail "Deployment $deployment not ready; inspect kubectl logs"
done

# Open local tunnels without touching the deployments. A tunnel already occupying
# the port is not replaced automatically; we don't want to kill another process.
start_forward() {
  local label="$1" service="$2" local_port="$3" remote_port="$4"
  local logfile="$STATE_DIR/${label}-port-forward.log"
  local pidfile="$STATE_DIR/${label}-port-forward.pid"
  if lsof -nP -iTCP:"$local_port" -sTCP:LISTEN -t >/dev/null 2>&1; then
    printf 'Port %s already has a listener; not replacing it. Verify URL manually.\n' "$local_port"
    return
  fi
  nohup kubectl --context "$CONTEXT" -n "$NAMESPACE" \
    port-forward --address 127.0.0.1 "svc/$service" \
    "${local_port}:${remote_port}" >"$logfile" 2>&1 </dev/null &
  local pid=$!
  printf '%s\n' "$pid" > "$pidfile"
  for i in $(seq 1 15); do
    if ! kill -0 "$pid" 2>/dev/null; then
      fail "$label port-forward exited. Log: $logfile"
    fi
    if grep -q "Forwarding from 127.0.0.1:${local_port}" "$logfile" 2>/dev/null; then
      printf '%s forward active on localhost:%s (PID %s)\n' "$label" "$local_port" "$pid"
      return
    fi
    sleep 1
  done
  fail "$label port-forward did not become ready. Check $logfile"
}

note "Opening GUI and UNI API access from the Mac"
start_forward gui gui-service 8080 8080
start_forward main main-service 18081 8081

if curl -fsS --max-time 4 http://127.0.0.1:8080/ >/dev/null; then
  printf 'GUI reachable: http://127.0.0.1:8080\n'
else
  printf 'WARNING: GUI health not confirmed; inspect %s/gui-port-forward.log\n' "$STATE_DIR" >&2
fi

# Optional: the thesis OaaS Manager runs outside Kind on the Mac.
# Provide OAAS_MANAGER_DIR (absolute path containing main.py) to start it.
note "Checking external OaaS Manager on localhost:8000"
manager_healthy() {
  curl -fsS --max-time 3 http://127.0.0.1:8000/health 2>/dev/null \
    | grep -q '"service":"optimization-service-manager"'
}
if manager_healthy; then
  printf 'OaaS Manager is reachable at localhost:8000\n'
elif [[ -n "${OAAS_MANAGER_DIR:-}" ]]; then
  manager_dir="$OAAS_MANAGER_DIR"
  [[ -f "$manager_dir/main.py" ]] \
    || fail "OAAS_MANAGER_DIR must contain main.py: $manager_dir"
  if [[ -x "$manager_dir/.venv/bin/python" ]]; then
    python_bin="$manager_dir/.venv/bin/python"
  elif [[ -x "$manager_dir/venv/bin/python" ]]; then
    python_bin="$manager_dir/venv/bin/python"
  else
    python_bin="$(command -v python3 || true)"
    [[ -n "$python_bin" ]] || fail "Python 3 not found for OaaS Manager"
  fi
  note "Starting local OaaS Manager from ${manager_dir}"
  (
    cd "$manager_dir"
    nohup "$python_bin" -m uvicorn main:app --host 0.0.0.0 --port 8000 \
      >"$STATE_DIR/manager.log" 2>&1 </dev/null &
    echo "$!" >"$STATE_DIR/manager.pid"
  )
  manager_ready=false
  for i in $(seq 1 20); do
    if manager_healthy; then
      manager_ready=true
      break
    fi
    sleep 1
  done
  [[ "$manager_ready" == true ]] \
    || fail "Manager did not become healthy; inspect $STATE_DIR/manager.log"
  printf 'OaaS Manager healthy at localhost:8000\n'
else
  printf 'Manager is not reachable. To start it too, rerun with:\n'
  printf '  OAAS_MANAGER_DIR=/absolute/path/to/manager-directory %q\n' "$0"
fi

note "Checking TSN Service OaaS settings"
kubectl --context "$CONTEXT" -n "$NAMESPACE" \
  exec deployment/tsn-service -- env | grep '^OAAS_' || true

printf '\nB10 environment startup checks complete.\n'
printf 'GUI: http://localhost:8080\nUNI: http://127.0.0.1:18081/add_stream (POST only; do not submit for a demo unless needed)\n'
printf 'Manager: http://127.0.0.1:8000/health (separate process)\n'
printf 'Logs: %s\n' "$STATE_DIR"
printf 'NOTE: CBS remote optimizer registration/execution is a separate integration stage.\n'
