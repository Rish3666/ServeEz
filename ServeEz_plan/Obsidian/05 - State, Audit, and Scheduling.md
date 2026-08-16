---
tags:
  - obsidian
  - state
  - audit
  - scheduling
status: live
priority: high
---

# State, Audit, and Scheduling

## State Store

- [[internal/state/store.go]] defines the typed object store interface.
- [[internal/state/sqlite.go]] is the embedded persistence backend.
- Store objects are wrapped in `api.Object`.
- Object kinds are validated through the registry.

## Audit Log

- [[internal/audit/audit.go]] keeps an append-only action log.
- Every action should leave an audit trail with before/after context.

## Scheduler

- [[internal/orchestrator/scheduler.go]] picks a node for a workload.
- The scheduler uses current reported usage and node capacity.
- The reconciler watches store changes and updates workload assignment.

## Related Notes

- [[Obsidian/03 - Control Plane]]
- [[Obsidian/07 - API Contracts]]

