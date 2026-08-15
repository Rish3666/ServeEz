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

## Main Packages

- `internal/apiserver` exposes the HTTP API.
- `internal/state` persists typed objects.
- `internal/audit` stores append-only action history.
- `internal/orchestrator` schedules workloads and reconciles status.
- `internal/apiclient` is the CLI read path to the control plane.

## Important Endpoints

- `POST /v1/nodes/register`
- `POST /v1/nodes/{id}/report`
- `GET /v1/nodes/{id}/commands`
- `POST /v1/nodes/{id}/commands/{action_id}/ack`
- `GET /v1/state`
- `POST /v1/workloads`
- `POST /v1/execute`
- `GET /v1/audit`

## Control Plane Data Flow

1. Register a node into the state store.
2. Persist periodic reports back into the node object.
3. Queue actions per node.
4. Reconcile workloads with the scheduler.
5. Record all actions in audit.

## Related Notes

- [[Obsidian/05 - State, Audit, and Scheduling]]
- [[Obsidian/07 - API Contracts]]

