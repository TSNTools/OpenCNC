# OpenCNC Observability Proto v1

This package defines the canonical event contract for OpenCNC observability.

- Proto file: `observability.proto`
- Package: `opencnc.observability.v1`
- Envelope message: `EventEnvelope`
- Event vocabulary catalog: `event-catalog.md`

## Topic Mapping (recommended)

- `opencnc.logs` -> `EventEnvelope.payload.log`
- `opencnc.events` -> `EventEnvelope.payload.domain`
- `opencnc.metrics` -> `EventEnvelope.payload.metric`
- `opencnc.audit` -> `EventEnvelope.payload.audit`
- `opencnc.health` -> `EventEnvelope.payload.health`

## Partition Key (recommended)

- Primary: `trace.correlation_id`
- Fallback: `source.service`

This keeps events for a workflow ordered when possible.

## Compatibility Rules

- Never reuse field numbers.
- Reserve removed fields and enum values.
- Add new fields as optional additions only.
- Do not change semantic meaning of existing fields.
- Keep `schema_version` populated by producers.

## Producer Requirements

- Always set: `schema_version`, `event_id`, `occurred_at`, `source.service`, `kind`, `severity`.
- For request lifecycle tracking, set `trace.correlation_id`.
- Keep payload-specific fields small and structured.

## Consumer Expectations

- Ignore unknown fields.
- Route by `kind` and by `oneof payload`.
- Treat missing optional fields as unknown, not errors.
