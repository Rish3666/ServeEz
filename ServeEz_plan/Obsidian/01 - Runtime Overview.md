---
tags:
  - obsidian
  - overview
  - runtime
status: live
priority: high
---

# Runtime Overview

ServeEz has five live runtime surfaces:

- `servez` for user-facing cluster operations (CLI)
- `servez-control` for the control plane
- `servez-agent` for node-local execution
- `servez-tui` for the live terminal dashboard
- `servez-autoscale` for the standing fast-tier scaling loop

## End-to-End Flow

```text
User -> servez CLI -> control plane -> state store / scheduler -> agent -> container runtime
User -> servez-tui  -> control plane (GET /v1/state, 2s poll)
autoscale -> control plane (/v1/predict -> /v1/simulate -> /v1/execute)
```

## Primary Responsibilities

- `servez` loads commands from the registry and talks to the control plane.
- `servez-control` owns cluster state, audit, scheduling, history, MCP surface, and action dispatch.
- `servez-agent` collects telemetry, registers the node, reports state, and executes actions.
- `servez-tui` renders live cluster/services/status/chat/alerts panes from `/v1/state`.
- `servez-autoscale` polls `/v1/predict`, dry-runs `/v1/simulate`, and executes up-only scale actions.

## Core Paths

- CLI entrypoint: [[cmd/servez/main.go]]
- Join flow: [[cmd/servez/join.go]]
- Control plane entrypoint: [[cmd/servez-control/main.go]]
- Agent entrypoint: [[cmd/servez-agent/main.go]]
- TUI entrypoint: [[cmd/servez-tui/main.go]]
- Autoscale entrypoint: [[cmd/servez-autoscale/main.go]]

## Shared Contract

- [[internal/api/types.go]] is the single wire contract for both sides.
- Every report, command, workload/status object, forecast, and simulation result uses those types.

## Related Notes

- [[Obsidian/02 - CLI and Entrypoints]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]
- [[Obsidian/07 - API Contracts]]
- [[Obsidian/08 - Simulation and MCP]]
- [[Obsidian/09 - Predictive Scaling]]
- [[Obsidian/10 - TUI Dashboard]]

