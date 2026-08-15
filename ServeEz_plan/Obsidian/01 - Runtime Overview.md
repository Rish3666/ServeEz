---
tags:
  - obsidian
  - overview
  - runtime
status: live
priority: high
---

# Runtime Overview

ServeEz has three live runtime surfaces:

- `servez` for user-facing cluster operations
- `servez-control` for the control plane
- `servez-agent` for node-local execution

## End-to-End Flow

```text
User -> servez CLI -> control plane -> state store / scheduler -> agent -> container runtime
```

## Primary Responsibilities

- `servez` loads commands from the registry and talks to the control plane.
- `servez-control` owns cluster state, audit, scheduling, and action dispatch.
- `servez-agent` collects telemetry, registers the node, reports state, and executes actions.

## Core Paths

- CLI entrypoint: `cmd/servez/main.go`
- Join flow: `cmd/servez/join.go`
- Control plane entrypoint: `cmd/servez-control/main.go`
- Agent entrypoint: `cmd/servez-agent/main.go`

## Shared Contract

- `internal/api/types.go` is the single wire contract for both sides.
- Every report, command, and workload/status object uses those types.

## Related Notes

- [[Obsidian/02 - CLI and Entrypoints]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]
- [[Obsidian/07 - API Contracts]]

