---
tags:
  - feature
  - scaling
  - reliability
  - blue-green
status: implemented
priority: high
---

# Predictive Duplicate Container (Blue-Green Pre-Warm)

## Concept
Before killing or restarting a live container, AI ==spins up a fully-warmed duplicate== to take over traffic. The old container is only terminated once the new one passes health checks and is handling requests. Zero-downtime operations for updates, scaling down, and remediation.

## Status (2026-08-16)
Implemented in `internal/prewarm/swap.go` (`prewarm.Swapper`) and wired through the agent as a `replace` action:

- **Trigger**: `POST /v1/execute` with `{"type":"replace","target":"workload:web"}` (confidence gate 0.85). The action is routed to the workload's assigned node like `scale`.
- **Swapper flow**: create warm clone at the next replica index → poll health within the `prewarm.LeadTime` budget → drain old container (stop) → keep it alive as fallback (default 60s, `fallback_seconds` param) → remove old.
- **Rollback**: if the clone never becomes healthy in time, the clone is removed and the old container keeps running — the operation has zero downtime in either outcome.
- **Target selection**: a specific container via `parameters.instance`, or the highest-replica running instance automatically (the one a scale-down would remove first).
- **Tests**: `internal/prewarm/swap_test.go` (success, rollback, auto-target, create failure, lead-time tiers).

## Flow
```
1. AI decides: Container A needs replacement
2. Container B (clone) spins up with same config
3. B warms up (cache, connections, app init)
4. Traffic drains from A → routed to B
5. A is kept alive as fallback for 60s
6. A is gracefully terminated
```

## Use Cases
- **Rolling updates** — Replace container with new version
- **Downscaling** — Before removing capacity, confirm remaining can handle load
- **Self-healing** — Replace degraded container before it fails
- **Config changes** — Hot-swap with new config, rollback if issues

## AI Optimization
- AI predicts ==which container is most likely to fail== → pre-warms replacement speculatively
- AI picks optimal time (lowest traffic window) for non-urgent swaps
- Urgent swaps (container already failing) skip warm-up, do fast swap

## Relation to Predictive Scaling
| Feature | When |
|---------|------|
| [[01 - Zero Latency Predictive Scaling|Predictive Scaling]] | Adds capacity *before* demand spikes |
| Predictive Duplicate | Replaces *existing* container with zero downtime |

← [[Index]]
