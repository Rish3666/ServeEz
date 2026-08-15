---
tags:
  - obsidian
  - cli
  - entrypoint
status: live
priority: high
---

# CLI and Entrypoints

## `cmd/servez`

- `main.go` contains the command registry and dispatch loop.
- `init.go` implements control-plane bootstrap config creation.
- `join.go` implements node bootstrap and registration.
- `status.go` reads cluster state from the control plane.
- `deploy.go` handles workload deployment.
- `token.go` handles join token retrieval.

## `cmd/servez-control`

- Boots the control plane and serves the HTTP API.
- Loads config from `internal/config`.
- Wires state, audit, scheduler, and handler stack.

## `cmd/servez-agent`

- Boots the node daemon.
- Installs the metrics collector, buffer, and container manager.
- Starts the agent run loop.

## CLI Behavior

- Commands register via `registerCommand`.
- `servez join` is a bootstrap command, not the long-running daemon.
- `servez-agent` is the persistent process on each node.

## Related Notes

- [[Obsidian/01 - Runtime Overview]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]

