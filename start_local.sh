#!/bin/bash

set -e

PIDS=()

cleanup() {
    echo ""
    echo "Stopping all services..."

    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done

    echo "All services stopped."
    exit 0
}

trap cleanup SIGINT SIGTERM

echo "======================================"
echo " Starting local testing environment"
echo "======================================"

echo "[1/6] Starting Kafka..."
./start_kafka_local_testing.sh &
PIDS+=($!)

sleep 3

echo "[2/6] Starting Monitor Service..."
go run ./monitor_service/cmd/ &
PIDS+=($!)

echo "[3/6] Starting GUI Service..."
go run ./gui_service/cmd/ &
PIDS+=($!)

echo "[4/6] Starting Config Service..."
go run ./config_service/cmd/ &
PIDS+=($!)

echo "[5/6] Starting TSN Service..."
go run ./tsn_service/cmd/ &
PIDS+=($!)

echo "[6/6] Starting Main Service..."
go run ./main_service/cmd/ &
PIDS+=($!)

echo ""
echo "======================================"
echo " All services are running!"
echo "======================================"
echo ""
echo "Press Ctrl+C to stop everything."

wait
