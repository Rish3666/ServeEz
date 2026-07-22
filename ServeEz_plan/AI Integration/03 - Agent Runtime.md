---
tags:
  - ai
  - agent-runtime
  - sandbox
status: draft
priority: high
---

# Agent Runtime

## Role
The runtime for running ==AI agent sessions== as first-class workloads (inspired by Google's Agent Substrate and kagent). Manages sandboxed, hibernatable, stateful agent sessions.

## Why a Dedicated Agent Runtime?
Standard container orchestration (K8s, Nomad) treats every workload as a long-running service. AI agents are fundamentally different:

| Property | Traditional Workloads | AI Agent Sessions |
|----------|----------------------|-------------------|
| Activity | Always-on | ==Bursty, mostly idle== |
| State | Stateless or DB-backed | In-memory context + tool state |
| Code | Fixed binary | ==Model-generated at runtime== |
| Trust | Assumed well-behaved | ==Untrusted by default== |
| Identity | Per container | Stable per session (survives restarts) |
| Density | 10-100 per node | 1000+ per node |

## Architecture
```
┌──────────────────────────────────────────┐
│           Agent Runtime                   │
│  ┌─────────┐ ┌─────────┐ ┌──────────┐   │
│  │ Sandbox │ │ Session │ │   Tool   │   │
│  │ Manager │ │ Manager │ │ Registry │   │
│  └─────────┘ └─────────┘ └──────────┘   │
│  ┌─────────┐ ┌─────────┐ ┌──────────┐   │
│  │  Warm   │ │  State  │ │   A2A    │   │
│  │  Pool   │ │  Store  │ │  Router  │   │
│  └─────────┘ └─────────┘ └──────────┘   │
└──────────────────────────────────────────┘
```

## Components

### Sandbox Manager
- Creates isolated environments for each agent session
- Uses gVisor (default) or Kata Containers for kernel-level isolation
- Enforces: no network by default, no host filesystem access, resource limits
- Sandbox configuration via MCP tool

### Session Manager
- Tracks all agent sessions (active + hibernated)
- Handles: create, wake, hibernate, terminate
- Hibernation: snapshots RAM + filesystem to disk (or S3)
- Resume: restores snapshot in < 100ms from warm pool
- Sessions survive node restarts (state stored in control plane)

### Tool Registry
- MCP-compatible tool registration
- Agents discover available tools dynamically
- System tools (built-in) + user tools (plugins)
- Tool usage is audited per-session

### Warm Pool
- Pre-provisioned sandboxes for instant agent startup
- Configurable pool size (AI adjusts based on demand)
- Cold start: 2-5s → Warm pool: < 50ms
- Similar to [[Core Features/08 - Predictive Duplicate Container (Blue-Green Pre-Warm)]]

### State Store
- Per-session vector store for context memory
- Persists across hibernation cycles
- Agents can query their own history
- Optional: cross-session knowledge sharing

### A2A Router (Agent-to-Agent)
- Agents discover and invoke other agents
- Protocol: A2A (Agent-to-Agent) standard
- Delegation: "analyze this log for anomalies" → logs-agent
- Coordination: multi-agent workflows with shared state

## Use Cases
- **Monitoring agents**: Long-running sessions watching metrics, waking on anomalies
- **Diagnostic agents**: Spawned during incidents to investigate
- **OPS agents**: "Deploy this, verify health, report back"
- **Chat agents**: Interactive troubleshooting sessions

## Related
- [[Orchestration/06 - Workload Types#Agent Session]] — Type definition
- [[AI Control/02 - MCP Tool Interface]] — Tools exposed to agents
- [[AI Control/04 - State Model for AI]] — Agent state in cluster overview
