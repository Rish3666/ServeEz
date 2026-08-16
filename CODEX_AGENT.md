# CODEX TASKS — Sprint 5: Autoscale Control Loop (Fast Tier)

You own the **autoscale daemon** for Predictive Scaling (Core Features/01, AI Control/07 Native Control Loop, AI Integration/01). A parallel agent owns the history store + forecast engine + predict API (all read-only for you).

## What the parallel agent is building (read-only, will exist)

- `internal/history` — SQLite time-series store; recorded automatically from every node report.
- `internal/predictor` — statistical forecast model (fast tier, <100ms, no ML deps) producing scale recommendations.
- `GET /v1/predict?workload=<name>` on the control plane — returns the forecast + recommendation (below).
- `internal/apiclient` gets `Predict(ctx, workload)` + `Simulate(ctx, act)` + `Execute(ctx, act)` — you use these.

### `/v1/predict` response shape

```json
{
  "workload": "web",
  "current_replicas": 2,
  "recommended_replicas": 4,
  "current_cpu_pct": 61.5,
  "forecast_15m_pct": 74.0,
  "forecast_1h_pct": 88.2,
  "confidence": 0.87,
  "recommendation": "scale to 4 replicas",
  "reason": "cpu trending up: 55.1 -> 88.2 pct by 1h",
  "available": true
}
```

`available:false` means there is not enough history yet (cold start) or the workload is not scheduled — do not act.

## Your deliverables

### 1. `cmd/servez-autoscale/main.go` — the standing control loop

This is the **fast tier** of the native control loop (AI Control/07). It must run forever, continuously, not on user request:

- Every `--interval` (default 60s):
  1. `env.Client.Predict(ctx, workload)` for each workload in the cluster (fetch via `State(ctx,"Workload")`).
  2. If `available` is false, skip (cold start / unscheduled).
  3. If `recommended_replicas > current_replicas` (scale up only for MVP):
     - Dry-run: `Simulate(ctx, Action{Type:"scale", Target:"workload:"+name, Parameters:{replicas}, Confidence:forecast.Confidence, Initiator:"ai-agent:autoscale"})`.
     - Decision gate: only proceed if `sim.Recommendation != "reject"` and the action's confidence passes the API's per-action threshold (scale = 0.70). If the simulation says `requires_approval`, log and skip (no human in the loop yet).
     - Execute: `Execute(ctx, ...)` the scale action.
  4. Cooldown: never scale the same workload twice within `--cooldown` (default 10m), and never scale down (up-only MVP keeps it safe).
- Graceful shutdown on SIGINT/SIGTERM.
- Log every decision to stdout via `log/slog`: action, replicas, confidence, reason.

### 2. Pre-warm lead-time measurement (Core Features/01)

Predictive scaling needs to know how far ahead to act. Measure and expose the **scale lead time**:

- Add `internal/agent` (EDIT ALLOWED on this one method) support or, if you prefer, do it in your own package `internal/prewarm`:
  - `LeadTime(runtime)` returns estimated time from "scale N→M" until all M replicas are ready.
  - For the MVP, use a default table (docker: ~15s/image pull + 5s startup, tuned per `image` size class) but structure it so real timings can be recorded later.
- Expose it as `GET /v1/prewarm/leadtime?image=<ref>` — but since the control-plane handler is the parallel agent's file, instead:
  - Print the lead-time estimate in your loop's decision log (e.g. `"pre-warm lead ~20s, scaling 5m before forecast peak"`).
  - If the forecast peak is sooner than `lead_time`, act immediately; otherwise you may wait one more interval.

### 3. Tests

`cmd/servez-autoscale/` should have a test that runs the loop against an **in-process control plane** (same pattern as `internal/agent/agent_test.go` — spin up `apiserver.New` + mem store + reconciler, register a node, deploy a workload, seed history via the predictor path, and assert the loop issues a scale action). If seeding the history store from a test is awkward because it's the parallel agent's package, test your loop logic against a stub `Predictor` interface instead (see below).

### Recommended structure

```go
type Predictor interface {
    Predict(ctx context.Context, workload string) (api.PredictResponse, error)
}
type Loop struct {
    client   *apiclient.Client // or your own
    interval time.Duration
    cooldown time.Duration
    lastScale map[string]time.Time
}
```

Test `Loop.tick` against a stub `Predictor` + a fake execute/simulate sink. Full e2e against the real control plane is a bonus.

## Rules

- Edit only: `cmd/servez-autoscale/`, and — if you need it — a new `internal/prewarm/` package. Do NOT edit `internal/apiserver/`, `internal/history/`, `internal/predictor/`, `internal/apiclient/`, `internal/api/`, `internal/state/`, `internal/mcp/`, `internal/tui/`, `internal/agent/` (except the single pre-warm helper if you put it there — prefer your own `internal/prewarm`), `internal/agentnet/`, `internal/container/`, `internal/metrics/`, `internal/config/`, `go.mod`, `go.sum`.
- `internal/api` will gain a `PredictResponse` type from the parallel agent (read-only). If it's not there yet, define a compatible local struct in your package and the parallel agent will align the JSON shape.
- The API server already exposes `/v1/predict`; if `apiclient.Predict` doesn't exist yet when you start, call `GET /v1/predict?workload=` directly with `net/http` in your own client helper.
- When done: `go build ./...`, `go vet ./cmd/servez-autoscale/`, `go test ./cmd/servez-autoscale/`, then report back a summary.