# OpenCNC Observability Event Catalog (v1)

This document defines the shared event vocabulary to keep producers and consumers aligned.

Scope:
- OpenCNC main-service
- OpenCNC tsn-service
- OpenCNC config-service

## Contract Rules

1. Every record MUST be an `EventEnvelope`.
2. `kind` MUST match the selected payload in `oneof payload`.
3. Producers MUST set: `schema_version`, `event_id`, `occurred_at`, `source.service`, `kind`, `severity`.
4. For request/workflow flows, producers MUST set `trace.correlation_id`.
5. `domain`, `action`, and `subject_type` values MUST come from this catalog.

## Topic Mapping

- `opencnc.logs` -> `payload.log`
- `opencnc.events` -> `payload.domain`
- `opencnc.metrics` -> `payload.metric`
- `opencnc.audit` -> `payload.audit`
- `opencnc.health` -> `payload.health`

Recommended partition key:
1. `trace.correlation_id`
2. fallback: `source.service`

## Domain Vocabulary

### stream.lifecycle

Description: User and controller lifecycle for stream requests.

Allowed actions:
- `requested`
- `validated`
- `admission_checked`
- `accepted`
- `rejected`
- `scheduled`
- `configured`
- `failed`

Allowed subject_type:
- `stream`
- `stream_request`

Typical subject_id:
- `stream_uid`
- `stream_uid:talker_mac`

### topology.lifecycle

Description: Topology ingestion and graph state transitions.

Allowed actions:
- `received`
- `parsed`
- `stored`
- `updated`
- `failed`

Allowed subject_type:
- `topology`
- `node`
- `link`

Typical subject_id:
- topology version id
- node id
- link id

### device_model.lifecycle

Description: Device model registration and validation.

Allowed actions:
- `received`
- `validated`
- `stored`
- `updated`
- `failed`

Allowed subject_type:
- `device_model`
- `module_registry`

Typical subject_id:
- model id
- module name/version

### schedule.lifecycle

Description: TSN schedule creation and persistence.

Allowed actions:
- `default_loaded`
- `computed`
- `stored`
- `selected`
- `failed`

Allowed subject_type:
- `schedule`
- `port_schedule`

Typical subject_id:
- schedule id

### config.apply

Description: Configuration push and result handling.

Allowed actions:
- `prepared`
- `sent`
- `acknowledged`
- `partially_applied`
- `failed`
- `timed_out`

Allowed subject_type:
- `configuration`
- `device`
- `interface`

Typical subject_id:
- configuration id
- device ip/hostname

### optimizer.lifecycle

Description: External optimizer calls and polling.

Allowed actions:
- `check_started`
- `reachable`
- `request_sent`
- `job_started`
- `job_finished`
- `result_fetched`
- `fallback_used`
- `failed`

Allowed subject_type:
- `optimizer_job`
- `optimizer_result`

Typical subject_id:
- optimizer job id

### netconf.session

Description: NETCONF session and RPC outcomes.

Allowed actions:
- `connect_started`
- `connected`
- `hello_sent`
- `get_config_sent`
- `edit_config_sent`
- `reply_received`
- `rejected`
- `failed`

Allowed subject_type:
- `netconf_session`
- `netconf_rpc`
- `device`

Typical subject_id:
- host:port
- rpc name

## Domain Result Usage

Allowed mapping guidance:
- user request accepted -> `DOMAIN_RESULT_ACCEPTED`
- user request rejected -> `DOMAIN_RESULT_REJECTED`
- operation completed -> `DOMAIN_RESULT_SUCCEEDED`
- operation error -> `DOMAIN_RESULT_FAILED`
- operation timeout -> `DOMAIN_RESULT_TIMEOUT`

## Severity Guidance

- `SEVERITY_DEBUG`: verbose diagnostics
- `SEVERITY_INFO`: expected lifecycle milestones
- `SEVERITY_WARN`: degraded but continuing
- `SEVERITY_ERROR`: failed operation
- `SEVERITY_CRITICAL`: system-threatening condition

## Metric Naming Guidance

Prefix metrics by domain to avoid collisions:
- `stream.*`
- `topology.*`
- `schedule.*`
- `config.*`
- `optimizer.*`
- `netconf.*`

Examples:
- `stream.request.count` (counter)
- `stream.admission.latency_ms` (histogram)
- `config.apply.count` (counter)
- `config.apply.latency_ms` (histogram)
- `optimizer.call.latency_ms` (histogram)
- `netconf.rpc.error.count` (counter)

## Audit Naming Guidance

Action examples:
- `stream.create`
- `stream.delete`
- `topology.update`
- `device_model.register`
- `config.apply`

Target type examples:
- `stream`
- `topology`
- `device_model`
- `configuration`

## Health Check Names

Standard check values:
- `kafka_connectivity`
- `etcd_connectivity`
- `config_service_reachability`
- `optimizer_reachability`
- `netconf_reachability`

## Example Domain Events

### main-service stream accepted

- domain: `stream.lifecycle`
- action: `accepted`
- subject_type: `stream`
- result: `DOMAIN_RESULT_ACCEPTED`

Suggested details keys:
- `stream_uid`
- `talker_mac`
- `listener_count`
- `traffic_class`

### tsn-service configuration built

- domain: `config.apply`
- action: `prepared`
- subject_type: `configuration`
- result: `DOMAIN_RESULT_SUCCEEDED`

Suggested details keys:
- `configuration_id`
- `schedule_id`
- `request_count`

### config-service edit-config rejected

- domain: `netconf.session`
- action: `rejected`
- subject_type: `netconf_rpc`
- result: `DOMAIN_RESULT_FAILED`

Suggested details keys:
- `rpc`
- `device`
- `error_tag`
- `error_path`
- `error_message`

## Compatibility and Governance

When adding a new domain/action/subject_type:
1. Update this catalog first.
2. Add producer tests for the new values.
3. Update consumer filters/dashboards.
4. Keep existing values stable; do not rename active values.
