# CODEX TASKS — Sprint 7: GUI Dashboard (Web)

You own the **web GUI dashboard** (UI/GUI Dashboard). A parallel agent owns the control-plane REST surface (read-only for you). The GUI is read-only for the MVP sprint.

## What the parallel agent is building (read-only, will exist)

- Nothing new for you beyond what's already there — you read the existing control plane API:
  - `GET /v1/state` (nodes, workloads, status)
  - `GET /v1/audit`
  - `POST /v1/cost/compare` (for the cost view)
  - `GET /v1/predict?workload=<name>` (for the scaling view)
- The parallel agent owns `internal/apiserver/`, `internal/api/`, `cmd/servez/`, `internal/apiclient/`, `internal/mcp/`, `internal/tui/` — do not edit those.

## MVP scope (read-only dashboard)

A **single-page web dashboard** served by a new binary `cmd/servez-gui/main.go` that:

1. **Serves static assets** (HTML/CSS/JS) from an embedded FS (`//go:embed`) — no external build step, no node_modules. Plain HTML + vanilla JS or a CDN-free hand-rolled SPA.
2. **Proxies / polls the control plane** — the GUI binary takes `--control=<url>` and `--listen=:8080`; it forwards `/v1/*` to the control plane (or you can fetch directly from the browser via a small `/api/*` proxy in the GUI).
3. **Views** (mirror the TUI panes):
   - **Cluster** — node grid colored by health (healthy/degraded/unhealthy/disconnected), polled every 2-3s.
   - **Services** — workload list: name, image, replicas, state, assigned node.
   - **Cost** — a form (vCPU/mem/runtime) calling `/v1/cost/compare`, rendering the provider comparison table + best recommendation.
   - **Alerts** — derived from node health (like the TUI alerts pane): unhealthy/degraded nodes as alerts, newest first.
   - **AI Chat** (stretch, if time): a read-only chat box mirroring the TUI chat pane's intent parsing against `/v1/simulate` — no execution this sprint.
4. **Polling** — simple `setInterval` + `fetch` against the proxy; no frameworks needed.

## Deliverables

- `cmd/servez-gui/main.go` — server + embedded assets.
- `cmd/servez-gui/static/` (or `web/`) — HTML/CSS/JS files (embedded via `//go:embed`).
- A README section or `--help` documenting flags.
- Tests: a basic `go test` that boots the GUI with a fake/stub control plane (or an in-process `apiserver` like the autoscale test) and asserts `/` serves HTML and `/api/state` proxies.

## Rules

- Edit only: `cmd/servez-gui/` (and your own `web/`/`static/` dir inside it). Do NOT edit `internal/apiserver/`, `internal/api/`, `internal/apiclient/`, `internal/mcp/`, `internal/tui/`, `cmd/servez/`, `cmd/servez-control/`, `internal/cost/`, `go.mod`, `go.sum`.
- If you need shared types, import `github.com/Rish3666/ServeEz/internal/api` (read-only).
- Keep it dependency-free (stdlib only) so it builds anywhere. If you add a dep, coordinate first.
- When done: `go build ./...`, `go vet ./cmd/servez-gui/`, `go test ./cmd/servez-gui/`, then report back a summary.