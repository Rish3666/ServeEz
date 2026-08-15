# CODEX TASKS — Sprint 2-3: Agent Join + mTLS (Agent Side)

You own the **agent-side** work for Sprint 2-3. A parallel agent owns the control-plane/CLI/MCP/simulation work. **Do not touch anything outside your directories.** The shared contract lives in `internal/api/types.go` and `internal/config/config.go` (read-only for you).

## Your deliverables

```
cmd/servez/join.go            — the `servez join` command (register via command registry)
internal/agent/tls.go         — mTLS identity + cert handling for the agent
internal/agentnet/client.go   — (EDIT ALLOWED) add TLS client config wiring
internal/agent/*_test.go      — end-to-end test vs a running control plane
```

## 1. The `servez join` command

The CLI now uses a **command registry**. `cmd/servez/main.go` is owned by the other agent and is **read-only for you**. You add ONE new file: `cmd/servez/join.go` that registers a command. The registry pattern:

```go
package main

func init() {
    registerCommand(Command{
        Name:  "join",
        Usage: "servez join <url> --token=<tok> [--provider=local]",
        Run:   runJoin,
    })
}
```

The `Command` struct and `registerCommand` are defined in `cmd/servez/commands.go` (owned by the other agent, read-only). `runJoin` has signature `func(args []string) error`.

### Join flow to implement (from `Core Features/07 - One-Command Cluster Additions`)

1. Parse `--token`, `--provider`, `--runtime` (default `docker`), optional `--node-id`.
2. Generate a stable node ID (hostname-based or persisted in `~/.servez/node-id`).
3. Create an `internal/agentnet.Client` pointed at `<url>`.
4. Build a `api.RegisterRequest` (node ID, token, runtime, provider, capacity auto-detected via `internal/metrics`).
5. Call `Register`. If `Approved == false`, print the reason and exit non-zero.
6. On success: print a friendly summary ("✓ Node registered", "✓ Agent started", "✓ Ready") and return nil. The actual daemon start is handled by the caller (`cmd/servez-agent`); `join` is the bootstrap command that validates + registers + prints next step.
7. Join token auth errors (401) must print "invalid or expired join token".

Keep it dependency-light: only `internal/agentnet`, `internal/api`, `internal/metrics`, `internal/config`, and stdlib.

## 2. mTLS for agent ↔ control plane

The control plane will gain TLS (the other agent's task). Wire the agent side:

- In `internal/agent/tls.go`: helper to load-or-generate a self-signed client cert in the agent data dir, returning a `*tls.Config` suitable for `agentnet.New(baseURL, tlsConfig)`.
- **Do not break plain-HTTP operation**: if the control plane URL is `http://`, skip TLS config entirely. Only use mTLS when the scheme is `https://`.
- If `https` is used but cert generation fails, log a warning and continue with system TLS pool (fallback), so local dev keeps working.

## 3. End-to-end test (highest value)

Add `internal/agent/agent_test.go` that:
1. Spins up an in-process control plane using `internal/apiserver.New(...)` + `internal/state.NewMemStoreWithRegistry(...)` + `internal/audit.OpenSQLite(":memory:")` + `internal/orchestrator.NewScheduler(...)` — all already exist, wired in `internal/apiserver/server_test.go` if you need a reference (read-only).
2. Runs the real `Agent` loop against it with a fake `container.Manager` (implement the interface from `internal/container/manager.go`).
3. Asserts: agent registers, reports state every 10s, polls commands, executes a `start` action, and acks.

Test must pass with `go test ./internal/agent/`.

## 4. Also fix: report workload status after actions

In `internal/agent/agent.go` (EDIT ALLOWED): after executing `deploy`/`scale`/`start`/`stop`, refresh the container list and include it in the next `NodeReport.Workloads` so the control plane sees running instances. The `container.Manager.List` method already returns `[]api.ContainerStatus`.

## Rules

- Edit only: `cmd/servez/join.go`, `internal/agent/`, `internal/agentnet/`. If you need new files there, create them.
- Do NOT edit: `cmd/servez/main.go`, `cmd/servez/commands.go`, `cmd/servez/init.go`, `cmd/servez/status.go`, `cmd/servez/deploy.go`, `cmd/servez/token.go`, `internal/apiserver/`, `internal/mcp/`, `internal/simulate/`, `internal/state/`, `internal/orchestrator/`, `internal/api/`, `internal/config/`, `internal/audit/`, `go.mod`, `go.sum`.
- `registerCommand` and `Command` may not exist yet when you start — the other agent is creating them in parallel. If `go build` fails because of a missing `cmd/servez/commands.go`, that is expected mid-sprint; your `join.go` must be compatible with the registry above once it exists.
- When done: `go build ./...`, `go vet ./...`, `go test ./internal/agent/ ./internal/agentnet/`, then report back a summary.
