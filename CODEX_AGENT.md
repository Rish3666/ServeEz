# CODEX TASKS — Node Agent (Sprint 1)

You are the second agent on the ServeEz build. You own the **node agent** module. A parallel agent owns the control plane. **Do not touch anything outside your directories.** Everything compiles against the shared contract in `internal/api/types.go` (read-only — do not edit it).

## Your deliverables

```
cmd/servez-agent/main.go        — daemon entrypoint
internal/agent/                 — agent core: run loop, registration, reporting
internal/agent/agent.go         — orchestration: collect → report → listen
internal/agent/register.go      — mTLS registration flow
internal/metrics/               — collectors + local 5min buffer
internal/metrics/collector.go   — CPU/mem/disk/net sampling @5s
internal/metrics/buffer.go      — ring buffer (5 min) for offline tolerance
internal/container/             — OCI runtime manager
internal/container/manager.go   — interface + Docker implementation
internal/container/docker.go    — Docker API impl (create/start/stop/remove, probes)
internal/agentnet/              — HTTP client to control plane
internal/agentnet/client.go     — Register / Report / poll commands
```

## Design constraints (from planning docs)

- Pure Go, zero external runtime deps, **idle RAM target < 50MB**.
- Uses `internal/api` types only (`RegisterRequest`, `RegisterResponse`, `NodeReport`, `ReportAck`, `Usage`, `HardwareInfo`, `ContainerStatus`).
- Push-based, not polling: collect metrics every 5s, **report every 10s** (or immediately on state change) to the control plane.
- Local 5-minute metrics buffer for offline tolerance; replay on reconnect.
- Container management via the **Docker Engine API** (v1) — use `github.com/docker/docker/client`. Abstract behind an interface so containerd/runc can be added later.
- mTLS certificate auth: generate a self-signed cert on first run, register with the control plane, use the returned node identity.
- Health probing: liveness/readiness/startup probes per `WorkloadSpec.Probes`; report `ContainerStatus.Health`.
- Node state is one of: `pending`, `healthy`, `degraded`, `unhealthy`, `cordoned`, `disconnected`.

## The control-plane contract you must implement against

Plain HTTP/JSON (gRPC comes later). The control-plane API server (being built in parallel) exposes:

| Endpoint | Request | Response | When |
|----------|---------|----------|------|
| `POST /v1/nodes/register` | `RegisterRequest` | `RegisterResponse` | Agent boot |
| `POST /v1/nodes/{id}/report` | `NodeReport` | `ReportAck` | Every 10s / on change |
| `GET /v1/nodes/{id}/commands` | — | `[]Action` (pending) | Agent polls every 5s |
| `POST /v1/nodes/{id}/commands/{action_id}/ack` | `{status: "completed"\|"failed", result: {...}}` | `{}` | After executing an action |

Register with a join token (`Token` field in `RegisterRequest`). If `RegisterResponse.Approved` is false, log the reason and retry with backoff (max 5 attempts, exponential). After approval, send reports and poll for commands.

**Action types you must handle** (from `internal/api.Action.Type`): `start`, `stop`, `restart`, `remove`, `deploy`, `scale`. For `scale`, the target is a workload; increase/decrease its container count on this node. Return structured `ActionResult`-shaped ack bodies.

## Acceptance criteria

1. `go build ./...` and `go vet ./...` pass for the whole module.
2. `cmd/servez-agent` runs standalone with `--control-plane <url> --token <tok> --node-id <id>`.
3. With a stub control plane, the agent: registers once, sends a `NodeReport` every 10s with real CPU/mem/disk percentages, and buffers metrics when the plane is unreachable.
4. Idle RSS < 50MB (verify with `ps -o rss= -p <pid>`).
5. `go test ./internal/...` — unit tests for metrics collector + buffer + at least one container manager path (use a fake client).

## Rules

- Edit only files under `cmd/servez-agent/`, `internal/agent/`, `internal/metrics/`, `internal/container/`, `internal/agentnet/`. If you need a new file there, create it.
- Do NOT edit `internal/api/`, `internal/state/`, `internal/apiserver/`, `internal/orchestrator/`, `cmd/servez-control/`, `cmd/servez/`, `internal/config/`, `internal/mcp/`, `go.mod`, or `go.sum` without checking with the user first.
- No comments explaining obvious code; match the concise Go style in `internal/api`.
- When done: `go build ./...`, `go vet ./...`, `go test ./internal/...`, then report back a summary of what you built and any deviations from this spec.
