---
tags:
  - obsidian
  - tui
  - ui
status: live
priority: high
---

# TUI Dashboard

Live terminal dashboard (`servez-tui`) — Bubble Tea + Lipgloss. Polls the control plane `/v1/state` every 2s.

## Entrypoint

- [[cmd/servez-tui/main.go]] — enters the alt screen and starts the app.

## Shell

- [[internal/tui/app.go]] — Bubble Tea model: ticker (2s poll), tab/q keys, generic 2-column grid for any pane count.
- [[internal/tui/panel.go]] — `Panel` interface + `RegisterPanel` + `Env{Client, ControlURL, Logger}`.
- [[internal/tui/snapshot.go]] — `Snapshot`/`NodeRow`/`WorkloadRow` + lenient map accessors for the `/v1/state` JSON.

## Panes

| Pane | File | Content |
|------|------|---------|
| Cluster | [[internal/tui/cluster.go]] | Node health map (healthy/degraded/unhealthy) |
| Services | [[internal/tui/services.go]] | Workload list with replicas |
| Status | [[internal/tui/status.go]] | Control-plane / system status |
| Chat | [[internal/tui/chat.go]] | Natural-language intent parsing: deploy/scale/restart/stop/remove/migrate/kill/status/help; simulate-then-confirm gating for risky actions |
| Alerts | [[internal/tui/alerts.go]] | Severity-derived alerts, scrollable, capped at 20 |

## Verified Live

Rendered 5 panes against a running control plane with registered + deployed workloads.

## Design Cross-References

- [[UI/TUI Dashboard]]
- [[Core Features/06 - AI Agent & Chat Mode]]
- [[Core Features/09 - Real-Time Monitoring & AI Suggestions]]

## Related Notes

- [[Obsidian/01 - Runtime Overview]]
- [[Obsidian/03 - Control Plane]]
- [[Obsidian/08 - Simulation and MCP]]