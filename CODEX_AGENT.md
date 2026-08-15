# CODEX TASKS — Sprint 4: TUI Chat + Alerts Panes

You own the **chat + alerts panes** for the terminal dashboard. A parallel agent owns the app shell (`internal/tui/app.go`, `panel.go`, `snapshot.go`, `cluster.go`, `services.go`, `status.go`) and the `cmd/servez-tui` entrypoint — **do not edit those files.** Your work lives in two new files that register themselves via `RegisterPanel`.

## What exists already (read-only for you)

- `internal/tui/panel.go` — the panel contract + registry you build against.
- `internal/tui/snapshot.go` — the `Snapshot` struct the shell pushes to your panes every ~2s via `SetSnapshot`.
- `internal/tui/app.go` — the shell: polls `/v1/state`, handles tab/q keys, renders a 2x2 grid. Key messages are forwarded to the focused panel.
- `internal/apiclient` — HTTP client for the control plane (has `Execute`, `Deploy`, `Simulate`, `State`). Use it for any action your panes trigger.
- `internal/mcp` — MCP tool server, already wired into the control plane at `POST /v1/mcp/call` (use if you want tool-style invocation).

## The panel contract (from `internal/tui/panel.go`)

```go
type Env struct {
    Client     *apiclient.Client
    ControlURL string
    Logger     *log.Logger
}

type Panel interface {
    tea.Model
    Title() string
    SetSnapshot(*Snapshot)
    Focused() bool
    SetFocused(bool)
}

func RegisterPanel(name string, f func(env Env) Panel)
```

A panel is a Bubble Tea model: `Init() tea.Cmd`, `Update(tea.Msg) (tea.Model, tea.Cmd)`, `View() string`. Register it from `init()`:

```go
func init() {
    RegisterPanel("Chat", func(env Env) Panel { return &chatPane{env: env} })
}
```

## Your deliverables

### 1. `internal/tui/chat.go` — AI Chat sidebar (Core Features/06, AI Control/03 Intent API)

- A focused chat input (text field) + message history rendered in `View()`.
- Enter submits the typed command; Esc/arrows toggle between history scroll and input.
- **MVP intent handling (no LLM yet)**: parse the command into an action and execute it via `env.Client.Execute(...)`, mirroring `servez` CLI behavior. Support at minimum:
  - `scale <workload> <n>` → `api.Action{Type:"scale", Target:"workload:<name>", Parameters:{replicas:n}, Confidence:0.9, Initiator:"human:tui"}` then print the `ActionResult.Status` as a chat reply.
  - `deploy <name> image=<img>` → build an `api.WorkloadSpec` and call `env.Client.Deploy(...)`.
  - `status` / `help` → print node/workload summary from the latest `Snapshot`.
  - Unknown command → reply with the help text.
- Each submitted action should first be dry-run through `env.Client.Simulate(...)` and shown as "simulated: <recommendation>" before executing, when the action type is one the simulate engine gates (kill/stop/remove/restart/migrate/scale). If the recommendation is `requires_approval` and the user didn't confirm (reply `yes`/`confirm`), do NOT execute.
- Store the message history in the pane (ring buffer, ~100 entries).

### 2. `internal/tui/alerts.go` — Alerts pane

- Derives alerts from the latest `Snapshot` (read-only, no HTTP): 
  - node state in {unhealthy, disconnected, cordoned} → red alert "node X is <state>".
  - node state `degraded` → yellow alert.
  - workload state `unschedulable` → red alert.
  - workload state `declared`/`pending` for more than 0 poll cycles → informational "workload X not yet scheduled" (track first-seen cycles in the pane).
- Render newest-first, capped at ~20, each prefixed by a `●` colored by severity.
- Make the pane scrollable with j/k when focused.

## Rules

- Edit only: `internal/tui/chat.go` and `internal/tui/alerts.go`. If you need a helper, put it in one of those files.
- Do NOT edit: `internal/tui/app.go`, `panel.go`, `snapshot.go`, `cluster.go`, `services.go`, `status.go`, `tui_test.go`, `cmd/`, `internal/apiserver/`, `internal/apiclient/`, `internal/mcp/`, `internal/state/`, `internal/api/`, `internal/config/`, `internal/orchestrator/`, `internal/audit/`, `internal/agent/`, `internal/agentnet/`, `internal/metrics/`, `internal/container/`, `go.mod`, `go.sum`.
- `go.mod` already has `bubbletea` + `lipgloss`; you may use `github.com/charmbracelet/bubbles` for the text input if it's available, otherwise implement a minimal text field in `chat.go`.
- Your panes must render an empty state when `Snapshot` is nil (first poll hasn't arrived).
- When done: `go build ./internal/tui/ ./cmd/servez-tui/`, `go vet ./internal/tui/`, `go test ./internal/tui/`, then report back a summary.
