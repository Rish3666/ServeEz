# CODEX TASKS — Sprint 6: Multi-Cloud Cost Comparison

You own the **cost comparison engine** (Core Features/03, AI Integration/04 Cost Optimizer). A parallel agent owns the REST surface + CLI command (all read-only for you).

## What the parallel agent is building (read-only, will exist)

- `internal/cost` — your package: the pricing data model + comparison engine (you own this).
- A REST surface under `/v1/cost/...` and a `servez cost` CLI command (parallel agent owns apiserver + cmd/servez; you own `internal/cost/`).

## MVP scope (Phase 1)

Compare AWS / Azure / GCP **spot (or preemptible) + on-demand** pricing for a workload's CPU/mem shape and produce a **savings report** (`api.CostReport`).

### Deliverable 1: `internal/cost/` — pricing engine (you own)

- `PriceCatalog` / `InstanceOffer` type: provider, region, instance type, vCPU, mem GB, on-demand price ($/hr), spot price ($/hr), available.
- A **static baseline catalog** for 2026 (a curated table per provider of a few dozen common instance types + typical regions). MVP must run offline with zero network calls — the parallel agent builds live scraping later.
- `Compare(ctx, req api.CostCompareRequest) (api.CostReport, error)`:
  - Given vCPU + mem + runtime % (e.g. "4 vCPU / 8 GB / 100% uptime"), find the cheapest `InstanceOffer` per provider for the requested shape.
  - Output per provider: best on-demand $/mo, best spot $/mo, savings vs most-expensive.
  - Pick overall best value: `recommendation` field = provider + instance + estimated monthly spend.
- Keep it self-contained — `internal/cost/` must not import `internal/apiserver` or `internal/mcp`.

### Deliverable 2: Tests

- `internal/cost/cost_test.go`: (1) compare known 4vCPU/8GB shape across all three providers returns sane $/mo numbers; (2) a shape no provider satisfies returns an error; (3) spot ≤ on-demand for the same offer; (4) empty request → error.
- Benchmark the `Compare` path — it must be fast-tier fast (<10ms).

### The wire contract (parallel agent will add to `internal/api` — align to this)

```go
type CostCompareRequest struct {
    VCPU       int     `json:"vcpu"`                  // required
    MemGB      int     `json:"mem_gb"`                // required
    RuntimePct float64 `json:"runtime_pct,omitempty"` // % of month running (default 100)
    Region     string  `json:"region,omitempty"`      // default "us-east-1"
}

type CostReport struct {
    Request       CostCompareRequest      `json:"request"`
    Best          *CostRecommendation     `json:"best"`
    Providers     []*CostRecommendation   `json:"providers"`
    PotentialSavingsPct float64           `json:"potential_savings_pct"` // best vs most expensive
}

type CostRecommendation struct {
    Provider     string  `json:"provider"`     // "aws" | "azure" | "gcp"
    InstanceType string  `json:"instance_type"`
    Region       string  `json:"region"`
    VCPU         int     `json:"vcpu"`
    MemGB        int     `json:"mem_gb"`
    OnDemandPerMo float64 `json:"on_demand_per_mo"`
    SpotPerMo     float64 `json:"spot_per_mo"`
    Recommended  string  `json:"recommended"`   // "spot" | "on_demand"
    EstMonthly   float64 `json:"est_monthly"`
}
```

## Rules

- Edit only: `internal/cost/`. Do NOT edit `internal/apiserver/`, `internal/api/` (types land from the parallel agent — define a compatible local struct and align later), `internal/mcp/`, `internal/tui/`, `internal/apiclient/`, `cmd/`, `go.mod`, `go.sum`.
- Define a compatible local `CostReport`/`CostCompareRequest` in `internal/cost` if `internal/api` types don't exist yet.
- When done: `go build ./...`, `go vet ./internal/cost/`, `go test ./internal/cost/ -bench=.`, then report back a summary.