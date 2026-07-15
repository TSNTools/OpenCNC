# Chat Handoff: OpenCNC Observability Design

Date: 2026-07-15

## Goal
Design and implement a clean, optional observability pipeline for OpenCNC microservices using Kafka + Protobuf.

## Key Decisions Already Made

1. Observability is optional.
- `observability=off`: core services run without Kafka/observability service.
- `observability=on`: services publish to Kafka, observability service consumes and serves UI/API.

2. Keep existing application logging path (Zap-backed logger wrapper) for service logs/fallback.
- Do not remove logging immediately.
- Add a dedicated producer module for canonical observability events.

3. Use Protobuf for Kafka payloads (not JSON as primary payload format).
- JSON can still be used at UI/API edge if needed.

4. Use one Kafka client for the new producer module.
- Selected direction: `kafka-go` for new producer module.
- Avoid mixed producer behavior across services.

5. No new repository required.
- Implement shared proto + shared producer module in current monorepo.

## What Exists Today

1. Current services already use old logger wrapper via `WithFields(...)`.
- This pre-attaches context fields to logs.
- It is log enrichment, not the new EventEnvelope producer.

2. Existing `opencnc_kafka-exporter` content in this workspace is minimal right now:
- Local Kafka compose.
- New observability proto contract files added during this chat.

3. In the upstream GitHub repo `TSNTools/opencnc_kafka-exporter`:
- There is Kafka-related code for logger sink and gNMI exporter.
- There is no reusable producer SDK for your new `EventEnvelope` contract.

## Artifacts Created in This Chat

1. Canonical proto contract:
- `opencnc_kafka-exporter/proto/opencnc/observability/v1/observability.proto`

2. Proto README:
- `opencnc_kafka-exporter/proto/opencnc/observability/v1/README.md`

3. Event vocabulary catalog (domain/action/subject_type conventions + examples):
- `opencnc_kafka-exporter/proto/opencnc/observability/v1/event-catalog.md`

## Contract Highlights

1. Top-level message: `EventEnvelope`.
2. One payload per event (`oneof`): log/domain/metric/audit/health.
3. Shared fields include:
- `schema_version`
- `event_id`
- `occurred_at`
- `source.*`
- `trace.correlation_id`
- `kind`
- `severity`

4. Topic mapping (recommended):
- `opencnc.logs`
- `opencnc.events`
- `opencnc.metrics`
- `opencnc.audit`
- `opencnc.health`

## Important Clarifications Agreed

1. Kafka is not inside observability service.
- Kafka is shared infrastructure.
- Producers (business services) publish to Kafka.
- Observability service consumes from Kafka.

2. `correlation_id` purpose:
- Same ID across all events for one request/workflow.
- Enables cross-service timeline reconstruction.

3. `kind` purpose:
- Coarse category for routing/filtering.
- Must match selected payload in `oneof`.

## Local Development Strategy Agreed

1. Fast loop:
- Run Kafka locally via Docker Compose.
- Run services directly (go run/go test) against local Kafka.
- Avoid full Kubernetes cycle for every small change.

2. Integration loop:
- Validate in Kubernetes periodically (not per tiny edit).

## Current Gap to Implement Next

No shared producer module exists yet for the new proto contract.

## Next Implementation Plan (Recommended)

1. Add shared producer module in monorepo.
- Suggested location: `opencnc_kafka-exporter/pkg/producer` (or similar).

2. Module responsibilities:
- Validate required fields.
- Route topic by `EventKind`.
- Select partition key:
  - primary `trace.correlation_id`
  - fallback `source.service`
- Marshal protobuf.
- Publish via `kafka-go`.
- Fail-open behavior flag (do not break business flow if Kafka is down).

3. Integrate producer into one service first (main-service), then tsn-service/config-service.

4. Gradually phase out old Kafka-log-only path where appropriate.

## Optional Config Flags to Add

- `OBS_ENABLED`
- `OBS_KAFKA_ENABLED`
- `OBS_FAIL_OPEN`
- `OBS_BROKERS`

## Notes for New Chat

If continuing in another workspace/chat, paste this file and ask to:
1. scaffold producer module,
2. generate proto go code,
3. integrate one service end-to-end,
4. provide smoke-test commands for local Kafka.
