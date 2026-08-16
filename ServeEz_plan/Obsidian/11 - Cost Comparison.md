---
tags:
  - obsidian
  - cost
  - cloud
  - arbitrage
status: live
priority: high
---

# Cost Comparison

Multi-cloud pricing engine ([[Core Features/03 - Real-Time Cloud Arbitrage]], [[AI Integration/04 - Cost Optimizer]]). Prices a workload shape across AWS/Azure/GCP and recommends the cheapest option.

## Components

### `internal/cost` — pricing engine
- `cost.go`: `Engine` implements `api.CostComparer`; offline baseline 2026 catalog (representative per-provider pricing) + `Compare`.
- `Compare(ctx, req)` → `api.CostReport`: cheapest matching offer per provider (best-fit: first shape satisfying vCPU + mem, cheapest on-demand), monthly cost scaled by `runtime_pct`, spot preferred when ≥15% cheaper.
- Region fallback: if the requested region has no matching offer for a provider, that provider's default region is used so all three are always compared.
- Fast tier: benchmarked <10ms (no network).

### Wire contract (`internal/api/types.go`)
- `CostComparer` interface (structural — `internal/cost` need not import apiserver/mcp).
- `CostCompareRequest` (`vcpu`, `mem_gb`, `runtime_pct`, `region`).
- `CostReport` (`best`, `providers` sorted cheapest-first, `potential_savings_pct`).
- `CostRecommendation` (provider, instance_type, on/off spot per month, est monthly).

### Surfaces
- `POST /v1/cost/compare` — control-plane route (`internal/apiserver/cost.go`).
- `servez cost --vcpu=N --mem-gb=M [--runtime-pct] [--region]` — CLI command.
- `apiclient.CostCompare`.

## Verified Live (2026-08-16)
`servez cost --vcpu=4 --mem-gb=16` → gcp e2-standard-4 @ $29.35/mo (spot) as best, 19.4% savings vs most expensive. REST endpoint returns the same JSON.

## Roadmap
- Live spot-pricing scraping (replace static catalog).
- Optimization loop: scan → analyze → score → recommend → simulate → execute (Phase 2, [[AI Integration/04 - Cost Optimizer]]).

## Related Notes

- [[Obsidian/03 - Control Plane]]
- [[Obsidian/07 - API Contracts]]
- [[Obsidian/09 - Predictive Scaling]]
- [[Core Features/03 - Real-Time Cloud Arbitrage]]
- [[AI Integration/04 - Cost Optimizer]]