# ============================================================
# Build stage
# ============================================================
FROM golang:1.25 AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all Go services
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/config-service ./config_service/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/monitor-service ./monitor_service/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tsn-service ./tsn_service/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/main-service ./main_service/cmd


# ============================================================
# Config Service
# ============================================================
FROM debian:bookworm-slim AS config_service

COPY --from=go-builder /out/config-service /app/config-service

EXPOSE 5150

ENTRYPOINT ["/app/config-service"]


# ============================================================
# Monitor Service
# ============================================================
FROM debian:bookworm-slim AS monitor_service

COPY --from=go-builder /out/monitor-service /app/monitor-service

RUN mkdir -p /home/opencnc/OpenCNC/monitor_service/pkg/catalog

COPY monitor_service/pkg/catalog/available_counters.json \
    /home/opencnc/OpenCNC/monitor_service/pkg/catalog/available_counters.json

COPY monitor_service/pkg/catalog/available_metrics.json \
    /home/opencnc/OpenCNC/monitor_service/pkg/catalog/available_metrics.json

EXPOSE 5151

ENTRYPOINT ["/app/monitor-service"]


# ============================================================
# TSN Service
# ============================================================
FROM debian:bookworm-slim AS tsn_service

COPY --from=go-builder /out/tsn-service /app/tsn-service

# TSN service expects its default schedule at this absolute path.
RUN mkdir -p /home/opencnc/OpenCNC/tsn_service/configs

COPY tsn_service/configs/default-schedule.yaml \
    /home/opencnc/OpenCNC/tsn_service/configs/default-schedule.yaml

EXPOSE 5152

ENTRYPOINT ["/app/tsn-service"]


# ============================================================
# Main Service
# ============================================================
FROM debian:bookworm-slim AS main_service

COPY --from=go-builder /out/main-service /app/main-service

EXPOSE 5153
EXPOSE 8000
EXPOSE 8081

ENTRYPOINT ["/app/main-service"]


# ============================================================
# GUI Go build
# ============================================================
FROM go-builder AS gui-go-builder

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /out/gui-service \
    ./gui_service/cmd


# ============================================================
# GUI Service
# ============================================================
FROM debian:bookworm-slim AS gui_service

WORKDIR /app

COPY --from=gui-go-builder \
    /out/gui-service \
    /app/gui-service

# Use the exact same static GUI that is served by
# `go run gui_service/cmd/main.go` locally.
COPY gui_service/static \
    /app/gui_service/static

EXPOSE 8080

ENTRYPOINT ["/app/gui-service"]