#!/bin/bash
set -euo pipefail

# === CONFIG ===
APP_NAME="tsn-service"
NAMESPACE="opencnc"
CHART_PATH="$HOME/Desktop/deploy-k8s/tsn-service"
VALUES_FILE="${CHART_PATH}/values.yaml"
DEPLOYMENT_LABEL="app=${APP_NAME}"

# === 1. Determine new image version ===
CURRENT_TAG=$(grep "tag:" "$VALUES_FILE" | awk '{print $2}')
IFS='.' read -r MAJOR MINOR PATCH <<< "${CURRENT_TAG#v}"
NEW_PATCH=$((PATCH + 1))
NEW_TAG="v${MAJOR}.${MINOR}.${NEW_PATCH}"

echo "🚀 Building new Docker image: ${APP_NAME}:${NEW_TAG}"
docker build -t ${APP_NAME}:${NEW_TAG} .

# === 2. Load image into Kind ===
echo "📦 Loading Docker image into kind cluster..."
kind load docker-image ${APP_NAME}:${NEW_TAG}

# === 3. Update image tag in Helm values file ===
echo "✏️ Updating image tag in ${VALUES_FILE}..."
sed -i "s/tag: .*/tag: ${NEW_TAG}/" "$VALUES_FILE"

# === 4. Upgrade Helm release ===
echo "🔄 Upgrading Helm release..."
helm upgrade ${APP_NAME} "${CHART_PATH}" --namespace ${NAMESPACE}

# === 5. Wait for pod to be ready ===
echo "⏳ Waiting for ${APP_NAME} pod to be ready..."
kubectl wait --for=condition=ready pod -l ${DEPLOYMENT_LABEL} -n ${NAMESPACE} --timeout=120s

# === 6. Print logs from new tsn-service pod ===
echo "📋 Fetching logs from ${APP_NAME}..."

TSN_POD=$(kubectl get pod -n ${NAMESPACE} -l app=${APP_NAME} -o jsonpath="{.items[0].metadata.name}")

kubectl logs "$TSN_POD" -n ${NAMESPACE}
