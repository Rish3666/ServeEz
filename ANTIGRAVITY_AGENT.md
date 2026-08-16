# ANTIGRAVITY TASKS — Sprint 8: Live Pricing + Test Hardening

You are the **parallel agent** on ServeEz (Sprint 8). A primary agent owns the control-plane core (blue-green pre-warm + AI executed actions). Your scope is deliberately isolated to packages the primary agent will **not** touch, so there is zero file overlap.

## Ground rules (read first)

- **Owned by you** (you may edit freely): `internal/cost/`, `cmd/servez/cost.go`, and **test files** under `internal/agent/`, `internal/apiserver/`, `internal/tui/`, `internal/mcp/`, `internal/orchestrator/`, `internal/simulate/`, `cmd/servez/`, `cmd/servez-autoscale/`.
- **Owned by the primary agent — do NOT edit**: `internal/orchestrator/scheduler.go`, `internal/agent/agent.go`, `internal/agent/register.go`, `internal/container/`, `internal/prewarm/`, `internal/apiserver/server.go`, `internal/mcp/mcp.go`, `cmd/servez-gui/`, and **`internal/api/types.go`** (the single shared contract file — the primary agent owns it this sprint).
- No `go.mod`/`go.sum` changes without flagging it.
- **CI gate**: `go build ./...` and `go vet ./...` must stay green. Do not break the build.

## Sprint 8 — your two workstreams

### 1. Live cloud pricing scraping (replaces the static catalog)

`internal/cost/cost.go` currently ships an offline 2026 baseline catalog (`PriceCatalog` + `Engine.Compare`). The interface it implements is unchanged:

```go
// internal/api/types.go (read-only for you — signature stable)
type CostComparer interface {
    Compare(ctx context.Context, req CostCompareRequest) (CostReport, error)
}
```

Goal: fetch **live on-demand + spot pricing** for AWS/Azure/GCP instead of the static table, while keeping the `CostComparer` contract intact so the API server, CLI, and GUI keep working.

Constraints:
- Keep the static catalog as a **fallback** when the live source is unreachable (or a region/provider isn't covered). Do not fail the whole request on a scrape error.
- Cache results (pricing changes rarely; a 24h TTL is reasonable) to avoid hammering provider APIs.
- Structure: introduce the scraping behind the existing `Engine` (e.g. `Engine` wraps a fetcher interface you define), or add a new type that still satisfies `api.CostComparer`. Keep it testable — add tests with a fake fetcher.
- Live sources must be keyed off the existing offer shape (`InstanceOffer`): provider, instance_type, region, vcpu, mem_gb, on_demand_per_hr, spot_per_hr. Preserve the `CostReport`/`CostRecommendation` fields the GUI/CLI already render.
- Update `cmd/servez/cost.go` only if needed to expose the new source; otherwise leave the CLI flags as-is.

### 2. Regression + e2e coverage across CLI/API/MCP/TUI/GUI

Add regression and end-to-end coverage so the existing flakiness and API contracts are locked down. Specifically:

- **Fix the flaky `TestAgentEndToEnd`** (`internal/agent/agent_test.go`, around line 112): it times out under system load. It waits for `len(status.Workloads) == 1 && status.Workloads[0].State == "running"` with 5s timeouts. Harden it (longer/more granular waits, or make the intervals configurable) without changing production code behavior. This test is a known flake — fixing it is explicitly in your scope.
- **e2e smoke** covering the public surface:
  - CLI: `servez status`, `servez deploy`, `servez scale`, `servez cost` against a live `servez-control` (spin it up in-process or as a subprocess with a temp config).
  - API: `/v1/state`, `/v1/audit`, `/v1/workloads`, `/v1/execute`, `/v1/simulate`, `/v1/cost/compare`, `/v1/mcp/tools`, `/v1/predict` round-trips.
  - MCP: tool list + a representative `call` for read and simulate paths.
  - TUI: view rendering smoke (e.g. renders without panic on a populated store).
  - GUI: extend `cmd/servez-gui/main_test.go` if you add coverage — but remember `cmd/servez-gui/` is owned by the primary agent; coordinate before touching it. Prefer CLI/API/MCP/TUI coverage first.
- Use the existing test conventions (table tests, `waitFor` helper, stub control plane pattern from `cmd/servez-gui/main_test.go`).

## How to verify your work

- `go build ./... && go vet ./...`
- `go test ./internal/cost/... -count=1`
- `go test ./internal/agent/ -run TestAgentEndToEnd -count=5` (should be green consistently)
- `go test ./... -count=1` (note: if it still flakes on another package, report it — do not silence it)

## What the primary agent is doing (do not duplicate)

- Blue-green pre-warm (Predictive Duplicate Container) in `internal/orchestrator/`, `internal/container/`, `internal/prewarm/`, `internal/agent/`.
- AI chat executed actions (moving `/v1/simulate` → executed `/v1/execute` with the existing confidence gate) in `internal/apiserver/server.go` + `internal/mcp/`.
- They own `internal/api/types.go` this sprint — treat it as read-only.

## Report back

When done, summarize: what you shipped, the new test names/commands, the flake fix, and any API contract notes. Keep the visual design and the control-plane core untouched.