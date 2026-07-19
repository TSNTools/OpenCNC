#!/usr/bin/env bash

set -e

CONTAINER_NAME="local-kafka"
BOOTSTRAP="localhost:9092"

TOPICS=(
  "opencnc.logs"
  "opencnc.events"
  "opencnc.metrics"
  "opencnc.audit"
  "opencnc.health"
)

KAFKA_BIN="/opt/kafka/bin"

echo "== Checking Kafka container =="

if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "Kafka container is not running."

    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo "Starting existing container..."
        docker start "${CONTAINER_NAME}"
    else
        echo "Creating Kafka container..."

        docker run -d \
          --name "${CONTAINER_NAME}" \
          -p 9092:9092 \
          -e KAFKA_NODE_ID=1 \
          -e KAFKA_PROCESS_ROLES=broker,controller \
          -e KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093 \
          -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
          -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
          -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
          -e KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
          -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
          -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1 \
          -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1 \
          apache/kafka:3.9.1
    fi
fi

echo "== Waiting for Kafka =="

until docker exec "${CONTAINER_NAME}" \
    ${KAFKA_BIN}/kafka-topics.sh \
    --bootstrap-server ${BOOTSTRAP} \
    --list >/dev/null 2>&1
do
    sleep 2
done

echo "Kafka ready."

echo "== Creating topics =="

for TOPIC in "${TOPICS[@]}"; do
    docker exec "${CONTAINER_NAME}" \
      ${KAFKA_BIN}/kafka-topics.sh \
      --bootstrap-server ${BOOTSTRAP} \
      --create \
      --if-not-exists \
      --topic "${TOPIC}" \
      --partitions 1 \
      --replication-factor 1 \
      >/dev/null

    echo "Created: ${TOPIC}"
done


echo
echo "== Existing topics =="

docker exec "${CONTAINER_NAME}" \
    ${KAFKA_BIN}/kafka-topics.sh \
    --bootstrap-server ${BOOTSTRAP} \
    --list


echo
echo "== Following all OpenCNC topics =="
echo "Format: <topic:message>"
echo


docker exec -i "${CONTAINER_NAME}" \
${KAFKA_BIN}/kafka-console-consumer.sh \
    --bootstrap-server ${BOOTSTRAP} \
    --include "opencnc\..*" \
    --from-beginning \
    --property print.topic=true \
    --property print.separator=":" |
while IFS=: read -r TOPIC MSG
do
    echo "<${TOPIC}:${MSG}>"
done