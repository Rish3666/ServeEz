---
tags:
  - obsidian
  - predictive
  - scaling
  - ai
status: live
priority: critical
---

# Predictive Scaling

The fast tier of the native control loop ([[AI Control/07 - Native Control Loop]]). Statistical forecast → replica recommendation → standing autoscale loop.

## Components

### `internal/history` — time-series store
- SQLite-backed; `internal/history/history.go`.
- Every node report records a CPU% sample into the node's series (`"node:"+nodeID`).
- `Record`, `Recent`, `Count`, `Prune`, `Close`.

### `internal/predictor` — forecast engine
- `internal/predictor/predictor.go`.
- Least-squares linear-trend fit + r² confidence score.
- Cold-start guard: needs `MinSamples=12` before forecasting.
- Compares the 1h forecast against `TargetCPUPercent=80` and derives `recommended_replicas` (up-only).
- Signature: `Predict(ctx, series, workload, currentReplicas, cpuNow) (api.PredictResponse, error)`.
- Fast tier: <100ms, no ML dependencies (kept intentionally light).

### `GET /v1/predict?workload=<name>` — API
- Resolves workload → assigned node → node CPU series → forecast.
- Returns `api.PredictResponse` (see [[Obsidian/07 - API Contracts]]).
- `available:false` = cold start or unscheduled — the autoscale loop must skip.

### `internal/prewarm` — lead time
- `internal/prewarm/leadtime.go`: `LeadTime(runtime)` estimate (image pull + startup) used to decide how far ahead to scale.

### `cmd/servez-autoscale` — the loop
- Standing process: every `--interval` (default 60s):
  1. `Predict` each workload.
  2. Skip if `available:false`.
  3. If `recommended_replicas > current_replicas`: dry-run `/v1/simulate`, check decision gate (confidence ≥ 0.70, simulation not "reject"), then execute scale.
  4. Up-only + cooldown (default 10m) per workload.
- Graceful shutdown on SIGINT/SIGTERM; decisions logged via `log/slog`.

## Confidence Gates
| Action | Min confidence |
|--------|----------------|
| scale_up / scale | 0.70 |
| restart | 0.80 |
| migrate | 0.90 |
| kill / stop | 0.95 |

## Verified Live (2026-08-16)
40 rising node reports → `/v1/predict` returned `scale to 5 replicas` (r²=1.00, confidence 0.85) via both the REST endpoint and the MCP `predict.scale` tool.

## Design Cross-References

- [[AI Integration/01 - Predictor Engine]]
- [[Core Features/01 - Zero Latency Predictive Scaling]]
- [[AI Control/07 - Native Control Loop]]
- [[Core Features/03 - Real-Time Cloud Arbitrage]]

## Related Notes

- [[Obsidian/03 - Control Plane]]
- [[Obsidian/08 - Simulation and MCP]]
- [[Obsidian/07 - API Contracts]]