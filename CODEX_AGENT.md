# CODEX TASKS — Sprint 7: GUI Dashboard (Web)

You own the **web GUI dashboard** (UI/GUI Dashboard). A parallel agent owns the control-plane REST surface (read-only for you). The GUI is read-only for the MVP sprint.

## Current state (2026-08-16)

The GUI shell was **rebuilt from Stitch-generated designs** (project `ServeEz Control Plane`, design system "Calm Instrumentation") by the parallel agent. The previous hand-rolled draft was removed. **Do not rewrite the layout or visual style** — it is the product's design language:

- Flat dark surfaces (`#111319` canvas, `#1d2025` sidebar, `#272a30` cards), hairline borders, single indigo accent `#4c8dff`.
- Inter for UI, JetBrains Mono for data. Status colors only for status (green/amber/red).
- Views: Dashboard (cluster overview), Nodes, Workloads (services), Cost, Alerts, AI Control.
- Source of truth for visuals: `.stitch/designs/*.html` (generated), `cmd/servez-gui/static/{style.css,index.html,app.js}` (shipped).

## What the parallel agent is building (read-only, exists)

- You read the existing control plane API (all live):
  - `GET /v1/state` (nodes, workloads, status)
  - `GET /v1/audit`
  - `POST /v1/cost/compare` (cost view)
  - `GET /v1/predict?workload=<name>` (scaling view)
- The parallel agent owns `internal/apiserver/`, `internal/api/`, `cmd/servez/`, `internal/apiclient/`, `internal/mcp/`, `internal/tui/`, and now `cmd/servez-gui/` — do not edit those.

## Scope you own

`cmd/servez-gui/` (the whole package) is the parallel agent's. Your job for this sprint is limited to:

- **Review** the GUI's correctness against the control-plane API (flag any mismatched field names in `/v1/state` objects, `/v1/audit`, `/v1/cost/compare`).
- **Optional enhancements** (only if the parallel agent hasn't claimed them): wire `GET /v1/predict` into the AI Control confidence meter, add a workload-detail drawer.
- Do **not** change visual design, do **not** introduce external frontend deps (stdlib-only static embed), do **not** edit `go.mod`/`go.sum`.

## Build / test

- `go build ./...`, `go vet ./cmd/servez-gui/`, `go test ./cmd/servez-gui/ -count=1` (boots GUI against a stub control plane).
- When done: report a summary. Do not rewrite the design.