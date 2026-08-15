---
tags:
  - obsidian
  - api
  - contract
status: live
priority: critical
---

# API Contracts

`internal/api/types.go` is the shared contract for:

- agent <-> control plane communication
- object store payloads
- CLI reads and writes
- container and workload state

## Core Types

- `NodeSpec`
- `NodeStatus`
- `WorkloadSpec`
- `WorkloadStatus`
- `ContainerStatus`
- `RegisterRequest`
- `RegisterResponse`
- `NodeReport`
- `ReportAck`
- `Action`
- `ActionResult`
- `AuditEntry`
- `SimulationRequest`
- `SimulationResult`

## Practical Rule

- Do not duplicate these structs in other packages.
- Treat this file as the source of truth for the system boundary.

## Related Notes

- [[Obsidian/01 - Runtime Overview]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]

