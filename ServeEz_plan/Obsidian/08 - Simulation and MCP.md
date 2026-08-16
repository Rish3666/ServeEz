---
tags:
  - obsidian
  - mcp
  - simulation
  - ai
status: live
priority: high
---

# Simulation and MCP

The AI surface of ServeEz. Every operation is a discoverable tool the agent can call; every risky action dry-runs first.

## MCP Tool Surface

- `internal/mcp/server.go` defines the `Server`, `Tool`, and `Predictor`/`Simulator` interfaces.
- `internal/apiserver/mcp.go` adapts the apiserver to `Simulator` and `Predictor` so tools share the same engine as the REST endpoints.
- HTTP surface: `GET /v1/mcp/tools` (discovery) and `POST /v1/mcp/call` (invocation).
- Wire: `apiserver.WithMCP(...)`; `mcp.NewWithPredictor(store, audit, simulator, predictor)`.

## Tools

| Tool | Category | Purpose |
|------|----------|---------|
| `state.list` | read | Dump cluster state as JSON |
| `workload.create` | write | Stage a workload object |
| `simulate.action` | simulate | Dry-run an action before execution |
| `state.subscribe` | subscribe | Watch for state changes |
| `audit.query` | audit | Read the action audit trail |
| `predict.scale` | simulate | Forecast + replica recommendation for a workload |

## Simulation Engine

- `internal/simulate/engine.go` implements a two-tier sandbox:
  - **Tier 1** — statistical sanity check of the proposed action.
  - **Tier 2** — constraint validation against live cluster state.
- Returns a `SimulationResult` with a `Recommendation` (approve / reject / requires_approval).
- Shared by `/v1/simulate` and the MCP `simulate.action` tool.

## Design Cross-References

- [[AI Control/02 - MCP Tool Interface]]
- [[AI Control/06 - Simulation Sandbox]]
- [[AI Control/05 - Action Audit & Safety]]
- [[AI Control/07 - Native Control Loop]]

## Related Notes

- [[Obsidian/03 - Control Plane]]
- [[Obsidian/07 - API Contracts]]
- [[Obsidian/09 - Predictive Scaling]]