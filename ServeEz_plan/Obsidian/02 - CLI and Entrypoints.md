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

- [[cmd/servez/main.go]] contains the command registry and dispatch loop.
- [[cmd/servez/init.go]] implements control-plane bootstrap config creation.
- [[cmd/servez/join.go]] implements node bootstrap and registration.
- [[cmd/servez/status.go]] reads cluster state from the control plane.
- [[cmd/servez/deploy.go]] handles workload deployment and scaling.
- [[cmd/servez/cost.go]] handles multi-cloud cost comparison.
- [[cmd/servez/token.go]] handles join token retrieval.

## `cmd/servez-control`

- [[cmd/servez-control/main.go]] boots the control plane and serves the HTTP API.
- Loads config from [[internal/config/config.go]].
- Wires state, audit, scheduler, history store, MCP server, and handler stack.

## `cmd/servez-agent`

- [[cmd/servez-agent/main.go]] boots the node daemon.
- Installs the metrics collector, buffer, and container manager.
- Starts the agent run loop.

## `cmd/servez-tui`

- [[cmd/servez-tui/main.go]] boots the live terminal dashboard (Bubble Tea).
- Polls `/v1/state` every 2s and renders panes.

## `cmd/servez-autoscale`

- [[cmd/servez-autoscale/main.go]] boots the standing fast-tier scaling loop.
- Polls `/v1/predict`, dry-runs `/v1/simulate`, executes up-only scale with cooldown.

## CLI Behavior

- Commands register via `registerCommand`.
- `servez join` is a bootstrap command, not the long-running daemon.
- `servez-agent` is the persistent process on each node.
- `servez-autoscale` is the persistent control loop for predictive scaling.

## Related Notes

- [[Obsidian/01 - Runtime Overview]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]
- [[Obsidian/09 - Predictive Scaling]]
- [[Obsidian/10 - TUI Dashboard]]

