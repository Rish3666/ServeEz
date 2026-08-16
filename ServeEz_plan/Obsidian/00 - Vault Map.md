---
tags:
  - obsidian
  - map
  - architecture
status: live
priority: high
---

# ServeEz Vault Map

This note is the navigation hub for the current implementation structure.

## Main Layers

- [[Obsidian/01 - Runtime Overview]]
- [[Obsidian/02 - CLI and Entrypoints]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/04 - Agent Runtime]]
- [[Obsidian/05 - State, Audit, and Scheduling]]
- [[Obsidian/06 - Metrics and Containers]]
- [[Obsidian/07 - API Contracts]]
- [[Obsidian/08 - Simulation and MCP]]
- [[Obsidian/09 - Predictive Scaling]]
- [[Obsidian/10 - TUI Dashboard]]
- [[Obsidian/11 - Cost Comparison]]

## Design Docs

- [[Index]]
- [[Architecture/System Overview]]
- [[Orchestration/01 - Architecture Overview]]
- [[Orchestration/02 - Control Plane]]
- [[Orchestration/04 - Node Agent]]
- [[Orchestration/05 - Container Lifecycle]]
- [[AI Control/02 - MCP Tool Interface]]
- [[AI Control/05 - Action Audit & Safety]]
- [[AI Control/06 - Simulation Sandbox]]
- [[AI Control/07 - Native Control Loop]]
- [[AI Integration/01 - Predictor Engine]]
- [[Core Features/07 - One-Command Cluster Additions]]
- [[Core Features/01 - Zero Latency Predictive Scaling]]
- [[UI/TUI Dashboard]]

## Code Map

- `cmd/servez/main.go` (command registry + dispatch)
- `cmd/servez/init.go`
- `cmd/servez/join.go`
- `cmd/servez/status.go`
- `cmd/servez/deploy.go`
- `cmd/servez/scale.go`
- `cmd/servez/token.go`
- `cmd/servez-control/main.go`
- `cmd/servez-agent/main.go`
- `cmd/servez-tui/main.go`
- `cmd/servez-autoscale/main.go`
- `internal/api/types.go`
- `internal/apiserver/server.go`
- `internal/apiserver/mcp.go`
- `internal/apiclient/client.go`
- `internal/agent/agent.go`
- `internal/agent/register.go`
- `internal/agent/tls.go`
- `internal/agentnet/client.go`
- `internal/state/store.go`
- `internal/state/sqlite.go`
- `internal/audit/audit.go`
- `internal/orchestrator/scheduler.go`
- `internal/container/manager.go`
- `internal/container/docker.go`
- `internal/metrics/collector.go`
- `internal/metrics/buffer.go`
- `internal/metrics/capacity.go`
- `internal/mcp/server.go`
- `internal/simulate/engine.go`
- `internal/history/history.go`
- `internal/predictor/predictor.go`
- `internal/prewarm/leadtime.go`
- `internal/cost/cost.go`
- `internal/tui/app.go`
- `internal/tui/panel.go`
- `internal/tui/cluster.go`
- `internal/tui/services.go`
- `internal/tui/status.go`
- `internal/tui/chat.go`
- `internal/tui/alerts.go`
- `internal/config/config.go`

## How To Read The System

1. Start with [[Obsidian/01 - Runtime Overview]] for the live flow.
2. Read [[Obsidian/07 - API Contracts]] to understand the shared wire types.
3. Use [[Obsidian/03 - Control Plane]] and [[Obsidian/04 - Agent Runtime]] to follow request paths.
4. Use [[Obsidian/05 - State, Audit, and Scheduling]] for persistence and reconciliation.
5. Use [[Obsidian/06 - Metrics and Containers]] for the node-local runtime work.
6. Use [[Obsidian/08 - Simulation and MCP]] for the AI surface.
7. Use [[Obsidian/09 - Predictive Scaling]] for the forecast + autoscale path.
8. Use [[Obsidian/10 - TUI Dashboard]] for the live terminal UI.
9. Use [[Obsidian/11 - Cost Comparison]] for the pricing engine.

