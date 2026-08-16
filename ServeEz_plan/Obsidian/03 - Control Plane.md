---
tags:
  - obsidian
  - control-plane
  - api
status: live
priority: high
---

# Control Plane

## Responsibilities

- Accept node registration.
- Accept node reports.
- Queue and acknowledge commands.
- Persist state and audit trails.
- Reconcile workload placement.
- Record node CPU samples into history.
- Serve forecasts and MCP tool calls.

## Main Packages

- `internal/apiserver` exposes the HTTP API (incl. MCP + predict routes).
- `internal/state` persists typed objects.
- `internal/audit` stores append-only action history.
- `internal/orchestrator` schedules workloads and reconciles status.
- `internal/apiclient` is the CLI read path to the control plane.
- `internal/mcp` exposes the AI tool surface (read/write/simulate/subscribe/audit/predict).
- `internal/simulate` implements the dry-run sandbox.
- `internal/history` records and queries the time-series store.
- `internal/predictor` produces scale forecasts.

## Important Endpoints

- `POST /v1/nodes/register`
- `POST /v1/nodes/{id}/report`
- `GET /v1/nodes/{id}/commands`
- `POST /v1/nodes/{id}/commands/{action_id}/ack`
- `GET /v1/state`
- `POST /v1/workloads`
- `POST /v1/execute`
- `GET /v1/audit`
- `POST /v1/simulate`
- `GET /v1/predict?workload=<name>`
- `GET /v1/mcp/tools`
- `POST /v1/mcp/call`

## Control Plane Data Flow

1. Register a node into the state store.
2. Persist periodic reports back into the node object; record CPU% into history.
3. Queue actions per node.
4. Reconcile workloads with the scheduler.
5. Record all actions in audit.
6. Serve simulation, forecast, and MCP tool calls from the same store.

## Related Notes

- [[Obsidian/05 - State, Audit, and Scheduling]]
- [[Obsidian/07 - API Contracts]]
- [[Obsidian/08 - Simulation and MCP]]
- [[Obsidian/09 - Predictive Scaling]]

