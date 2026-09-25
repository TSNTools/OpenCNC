#!/usr/bin/env bash
# ==============================================================================
# Script: setup_grafana_kafka.sh
# Description: Installs and configures the Kafka Datasource plugin in Grafana,
#              sets up the Kafka data source pointing to in-cluster Kafka,
#              and imports the OpenCNC Logs & Latency dashboard JSON.
# ==============================================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DASHBOARD_FILE="${1:-"$ROOT_DIR/scripts/dashboards/opencnc_logs_dashboard.json"}"
PLUGIN_NAME="hamedkarbasi93-kafka-datasource"
DATASOURCE_UID="cfz4vv44j5v5sf"
if [[ -z "${KAFKA_BROKER:-}" ]]; then
    if kubectl get svc -n opencnc kafka >/dev/null 2>&1; then
        KAFKA_BROKER="kafka.opencnc.svc.cluster.local:9092"
    else
        KAFKA_BROKER="172.19.0.1:9092"
    fi
fi
GRAFANA_USER="admin"
GRAFANA_PASS="admin"
GRAFANA_PORT=3000

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

if ! command -v curl >/dev/null 2>&1; then
    fail "curl is required but not installed."
fi

if ! command -v jq >/dev/null 2>&1; then
    fail "jq is required but not installed."
fi

# 2. Locate Grafana Pod
log "Locating Grafana pod..."
GRAFANA_NS="istio-system"
GRAFANA_POD="$(kubectl get pods -n "$GRAFANA_NS" -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"

if [[ -z "$GRAFANA_POD" ]]; then
    GRAFANA_POD="$(kubectl get pods -A -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
    if [[ -n "$GRAFANA_POD" ]]; then
        GRAFANA_NS="$(kubectl get pods -A -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.namespace}' 2>/dev/null || true)"
    fi
fi

if [[ -z "$GRAFANA_POD" ]]; then
    fail "Grafana pod not found. Please ensure Grafana is deployed (e.g., ./scripts/install_mesh.sh)."
fi

log "Found Grafana pod: '$GRAFANA_POD' in namespace '$GRAFANA_NS'"

# Wait for Grafana pod to be running
log "Waiting for Grafana pod to be in Running state..."
kubectl wait --for=condition=Ready pod/"$GRAFANA_POD" -n "$GRAFANA_NS" --timeout=60s >/dev/null 2>&1 || true

# 3. Locate or Prepare Kafka Datasource Plugin on Host
log "Checking Kafka plugin availability..."
LOCAL_PLUGIN_DIR=""
CANDIDATES=(
    "$HOME/kafka-plugin-dir/$PLUGIN_NAME"
    "$ROOT_DIR/scripts/plugins/$PLUGIN_NAME"
    "/tmp/$PLUGIN_NAME"
)

for c in "${CANDIDATES[@]}"; do
    if [[ -d "$c" && -f "$c/plugin.json" ]]; then
        LOCAL_PLUGIN_DIR="$c"
        break
    fi
done

if [[ -z "$LOCAL_PLUGIN_DIR" ]]; then
    log "Plugin not found locally. Preparing plugin..."
    DOWNLOAD_DIR="/tmp/kafka_plugin_download"
    mkdir -p "$DOWNLOAD_DIR"
    ZIP_FILE="$DOWNLOAD_DIR/kafka-datasource.zip"

    if [[ -f "$HOME/kafka-plugin.zip" ]]; then
        log "Using cached zip from $HOME/kafka-plugin.zip..."
        cp "$HOME/kafka-plugin.zip" "$ZIP_FILE"
    else
        log "Downloading $PLUGIN_NAME from GitHub..."
        curl -fsSL -o "$ZIP_FILE" "https://github.com/hoptical/grafana-kafka-datasource/releases/download/v1.9.0/hamedkarbasi93-kafka-datasource-1.9.0.zip" || \
        curl -fsSL -o "$ZIP_FILE" "https://github.com/hamedkarbasi93/kafka-datasource/releases/latest/download/hamedkarbasi93-kafka-datasource.zip" || \
        fail "Failed to download $PLUGIN_NAME."
    fi

    unzip -qo "$ZIP_FILE" -d "$DOWNLOAD_DIR"
    LOCAL_PLUGIN_DIR="$DOWNLOAD_DIR/$PLUGIN_NAME"
    [[ -d "$LOCAL_PLUGIN_DIR" ]] || fail "Extracted directory '$LOCAL_PLUGIN_DIR' not found."
fi

log "Using local plugin directory: $LOCAL_PLUGIN_DIR"

# 4. Check if Plugin is Installed in Pod, Copy if Needed
PLUGIN_INSTALLED_IN_POD=false
if kubectl exec -n "$GRAFANA_NS" "$GRAFANA_POD" -- test -f "/var/lib/grafana/plugins/$PLUGIN_NAME/plugin.json" 2>/dev/null; then
    PLUGIN_INSTALLED_IN_POD=true
    log "Kafka datasource plugin is already installed in pod."
fi

if [[ "$PLUGIN_INSTALLED_IN_POD" != "true" ]]; then
    log "Installing Kafka datasource plugin into Grafana pod..."
    kubectl exec -n "$GRAFANA_NS" "$GRAFANA_POD" -- mkdir -p "/var/lib/grafana/plugins"
    kubectl cp "$LOCAL_PLUGIN_DIR" "$GRAFANA_NS/$GRAFANA_POD:/var/lib/grafana/plugins/$PLUGIN_NAME"

    log "Restarting Grafana container to load the plugin..."
    kubectl exec -n "$GRAFANA_NS" "$GRAFANA_POD" -- kill 1
    sleep 4
fi

# Wait for Grafana HTTP endpoint to be responsive
log "Waiting for Grafana HTTP API to become ready..."
for i in {1..30}; do
    if kubectl exec -n "$GRAFANA_NS" "$GRAFANA_POD" -- wget -qO- "http://localhost:3000/api/health" 2>/dev/null | grep -q "ok"; then
        log "Grafana API is ready."
        break
    fi
    sleep 2
    if [[ "$i" -eq 30 ]]; then
        fail "Timed out waiting for Grafana to be ready."
    fi
done

# 5. Establish Temporary Port-Forward for Admin API Calls
log "Setting up temporary port-forward to Grafana..."
# Check if port 3000 is already in use
LOCAL_API_PORT=3000
if ss -tuln | grep -q ":3000 "; then
    # Test if it's already grafana
    if ! curl -s "http://localhost:3000/api/health" 2>/dev/null | grep -q "ok"; then
        LOCAL_API_PORT=13000
    fi
fi

PF_PID=""
if ! curl -s "http://localhost:$LOCAL_API_PORT/api/health" 2>/dev/null | grep -q "ok"; then
    kubectl port-forward svc/grafana -n "$GRAFANA_NS" "$LOCAL_API_PORT:3000" >/dev/null 2>&1 &
    PF_PID=$!
    sleep 2
fi

cleanup() {
    if [[ -n "$PF_PID" ]]; then
        kill "$PF_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

GRAFANA_URL="http://localhost:$LOCAL_API_PORT"
AUTH_HEADER="Authorization: Basic $(echo -n "${GRAFANA_USER}:${GRAFANA_PASS}" | base64)"

# 6. Verify Kafka Plugin is Loaded
log "Verifying Kafka plugin registration in Grafana..."
PLUGIN_SETTINGS="$(curl -s -H "$AUTH_HEADER" "$GRAFANA_URL/api/plugins/$PLUGIN_NAME/settings" || true)"
if ! echo "$PLUGIN_SETTINGS" | jq -e '.id' >/dev/null 2>&1; then
    fail "Kafka datasource plugin is not recognized by Grafana. Plugin response: $PLUGIN_SETTINGS"
fi
log "Plugin is recognized: $(echo "$PLUGIN_SETTINGS" | jq -r '.name') (version $(echo "$PLUGIN_SETTINGS" | jq -r '.info.version'))"

# 7. Configure Kafka Datasource
log "Configuring Kafka datasource in Grafana..."
EXISTING_DS="$(curl -s -H "$AUTH_HEADER" "$GRAFANA_URL/api/datasources" || echo "[]")"
KAFKA_DS_ID="$(echo "$EXISTING_DS" | jq -r '.[] | select(.name=="Kafka" or .type=="'$PLUGIN_NAME'") | .id' | head -n1)"
CURRENT_UID="$(echo "$EXISTING_DS" | jq -r '.[] | select(.name=="Kafka" or .type=="'$PLUGIN_NAME'") | .uid' | head -n1)"

DS_PAYLOAD=$(cat <<EOF
{
  "name": "Kafka",
  "uid": "$DATASOURCE_UID",
  "type": "$PLUGIN_NAME",
  "access": "proxy",
  "isDefault": false,
  "jsonData": {
    "bootstrapServers": "$KAFKA_BROKER"
  }
}
EOF
)

if [[ -n "$KAFKA_DS_ID" && "$CURRENT_UID" == "$DATASOURCE_UID" ]]; then
    log "Kafka datasource with UID '$DATASOURCE_UID' already exists. Updating..."
    curl -s -X PUT "$GRAFANA_URL/api/datasources/$KAFKA_DS_ID" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "$DS_PAYLOAD" >/dev/null
else
    if [[ -n "$KAFKA_DS_ID" ]]; then
        log "Deleting outdated Kafka datasource (ID $KAFKA_DS_ID, UID $CURRENT_UID)..."
        curl -s -X DELETE "$GRAFANA_URL/api/datasources/$KAFKA_DS_ID" \
            -H "$AUTH_HEADER" >/dev/null
    fi
    log "Creating Kafka datasource (UID: $DATASOURCE_UID, Broker: $KAFKA_BROKER)..."
    CREATE_RESP="$(curl -s -X POST "$GRAFANA_URL/api/datasources" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "$DS_PAYLOAD")"
    if ! echo "$CREATE_RESP" | jq -e '.id' >/dev/null 2>&1; then
        fail "Failed to create Kafka datasource: $CREATE_RESP"
    fi
fi

# Test Datasource Health
log "Testing Kafka datasource connectivity to broker '$KAFKA_BROKER'..."
HEALTH_RESP="$(curl -s -H "$AUTH_HEADER" "$GRAFANA_URL/api/datasources/uid/$DATASOURCE_UID/health" || true)"
if echo "$HEALTH_RESP" | grep -q '"status":"OK"'; then
    log "✔ Kafka datasource is HEALTHY: $(echo "$HEALTH_RESP" | jq -r '.message')"
else
    echo "WARNING: Datasource health check did not return OK: $HEALTH_RESP"
    echo "Note: If no messages have been published to Kafka yet, the datasource will still function once traffic starts."
fi

# 8. Load Dashboard JSON into Grafana
if [[ ! -f "$DASHBOARD_FILE" ]]; then
    fail "Dashboard file not found: $DASHBOARD_FILE"
fi

log "Loading dashboard from: $DASHBOARD_FILE"

# Prepare Dashboard Payload: ensure UID, overwrite, and bind Kafka datasource UID
DASHBOARD_PAYLOAD=$(python3 -c '
import sys, json

dash_file = sys.argv[1]
ds_uid = sys.argv[2]

with open(dash_file, "r") as f:
    raw = json.load(f)

# Support raw dashboard or wrapper with "dashboard" key
if "dashboard" in raw:
    dash = raw["dashboard"]
else:
    dash = raw

# Ensure all kafka panels use the configured datasource UID
for panel in dash.get("panels", []):
    ds = panel.get("datasource", {})
    if isinstance(ds, dict) and ds.get("type") == "hamedkarbasi93-kafka-datasource":
        panel["datasource"]["uid"] = ds_uid
    for target in panel.get("targets", []):
        tds = target.get("datasource", {})
        if isinstance(tds, dict) and tds.get("type") == "hamedkarbasi93-kafka-datasource":
            target["datasource"]["uid"] = ds_uid

dash["id"] = None
payload = {
    "dashboard": dash,
    "overwrite": True
}
print(json.dumps(payload))
' "$DASHBOARD_FILE" "$DATASOURCE_UID")

IMPORT_RESP="$(curl -s -X POST "$GRAFANA_URL/api/dashboards/db" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "$DASHBOARD_PAYLOAD")"

if ! echo "$IMPORT_RESP" | jq -e '.status == "success"' >/dev/null 2>&1; then
    fail "Failed to import dashboard into Grafana: $IMPORT_RESP"
fi

DASH_URL="$(echo "$IMPORT_RESP" | jq -r '.url')"
DASH_UID="$(echo "$IMPORT_RESP" | jq -r '.uid')"

echo ""
echo "=========================================================="
echo " Grafana Kafka Consumer & OpenCNC Dashboard Setup COMPLETE!"
echo "=========================================================="
echo ""
echo "Kafka Datasource:"
echo "  • Name:             Kafka"
echo "  • UID:              $DATASOURCE_UID"
echo "  • Broker:           $KAFKA_BROKER"
echo "  • Plugin:           $PLUGIN_NAME"
echo ""
echo "Imported Dashboard:"
echo "  • UID:              $DASH_UID"
echo "  • Dashboard Path:   $DASH_URL"
echo "  • Direct URL:       http://localhost:3000$DASH_URL"
echo ""
echo "To view the dashboard in your browser:"
echo "  1. Run: ./scripts/open_dashboards.sh"
echo "  2. Open: http://localhost:3000$DASH_URL"
echo ""
