# OpenCNC GUI Service

This directory contains a separate GUI service for OpenCNC.

Current scope:
- multi-screen GUI aligned with the operator mockups
- static frontend (no Node required to run the current GUI)
- backend-driven view data endpoints (dashboard, models, nodes, links, streams, logs, recent events)
- operation endpoints that call real backend functions and currently return hardcoded stub results
- clean 4-layer service structure for incremental CNC integration

Run from the repository root:

```bash
go run ./gui_service/cmd
```

Then open `http://localhost:8080`.

## 4-Layer Structure

1. Transport layer
- Path: `internal/transport/http`
- Responsibility: HTTP routing, request decoding, response encoding.
- No business logic.

2. Application layer
- Path: `internal/app`
- Responsibility: use-case functions per screen/action such as `AddNode`, `DeleteLink`, `GetDashboard`.
- Calls domain ports.

3. Domain layer
- Path: `internal/domain`
- Responsibility: core entities and ports/interfaces shared across layers.
- Proto contracts are defined under `internal/domain/protos`.

4. Adapter layer
- Path: `internal/adapters`
- Responsibility: infrastructure implementation of domain ports.
- Current implementation: `internal/adapters/stub` with hardcoded responses.

## Proto Contracts

Proto files for entities and operations are under:
- `internal/domain/protos/gui_entities.proto`
- `internal/domain/protos/gui_operations.proto`

These define the long-term contracts for:
- dashboard data
- models, nodes, links, streams, logs
- operation request/response envelopes

## API Surface (Current)

Read endpoints:
- `GET /api/v1/dashboard`
- `GET /api/v1/device-models`
- `GET /api/v1/nodes`
- `GET /api/v1/links`
- `GET /api/v1/streams`
- `GET /api/v1/logs`
- `GET /api/v1/events/recent`

Operation endpoints:
- `POST /api/v1/dashboard/refresh`
- `POST /api/v1/device-models/upload`
- `POST /api/v1/nodes`
- `PATCH /api/v1/nodes/:id`
- `DELETE /api/v1/nodes/:id`
- `POST /api/v1/links`
- `PATCH /api/v1/links/:id`
- `DELETE /api/v1/links/:id`
- `POST /api/v1/streams`
- `DELETE /api/v1/streams/:id`
- `POST /api/v1/logs/filter`
- `POST /api/v1/logs/order`

Health endpoint:
- `GET /api/health`

## Behavior Today

- UI fields are loaded from backend endpoints and default to empty values when data is unavailable.
- Button actions call application-layer functions through transport handlers.
- Stub adapter functions print function invocation to CLI and return hardcoded responses.
- This keeps call flow real while CNC-specific logic is still pending.

React frontend:

```bash
cd gui_service/frontend
npm install
npm run dev
```

React build output:
- after `npm run build`, the Go server will automatically prefer `gui_service/frontend/dist`
- until then, it continues to serve the existing static prototype