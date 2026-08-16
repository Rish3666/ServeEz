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
- `PredictResponse` — forecast payload for `/v1/predict`:
  `workload, available, current_replicas, recommended_replicas, current_cpu_pct, forecast_15m_pct, forecast_1h_pct, confidence, recommendation, reason`

## Practical Rule

- Do not duplicate these structs in other packages.
- Treat this file as the source of truth for the system boundary.
- Simulation and prediction types are also consumed by the MCP tool surface (via `apiserver` adapter methods).

## Related Notes

- [[Obsidian/01 - Runtime Overview]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]
- [[Obsidian/08 - Simulation and MCP]]
- [[Obsidian/09 - Predictive Scaling]]

