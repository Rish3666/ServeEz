---
tags:
  - orchestration
  - workloads
  - types
status: draft
priority: high
---

# Workload Types

## Overview
ServeEz supports 4 workload types — including ==Agent Sessions==, which K8s doesn't natively support. AI determines optimal placement, scaling, and lifecycle for each type.

## Type 1: Service
Stateless, replicated, always-on.

| Property | Value |
|----------|-------|
| Scaling | AI predictive (pre-warm) |
| Healing | Replace degraded instance |
| Networking | Load balanced, DNS registered |
| Persistence | None (stateless) |
| Example | Web servers, API gateways, workers |

## Type 2: Stateful
Persistent data, ordered identity.

| Property | Value |
|----------|-------|
| Scaling | Manual-recommendation by AI |
| Healing | Automated backup + restore |
| Networking | Stable DNS names per instance |
| Persistence | Volume attached (local or cloud) |
| Example | Databases, message queues, caches |

## Type 3: Batch
Finite execution, terminates on completion.

| Property | Value |
|----------|-------|
| Scaling | Parallelism controlled by AI |
| Healing | Restart on failure (configurable retries) |
| Networking | None isolation |
| Persistence | Scratch storage (ephemeral) |
| Scheduling | AI picks cheapest window (spot instances) |
| Example | ETL jobs, ML training, data processing |

## Type 4: Agent Session ⭐ (New — K8s doesn't have this)
Long-running, stateful AI agent sessions that spend most of their life idle.

| Property | Value |
|----------|-------|
| Lifecycle | ==Hibernatable== — snapshot RAM, resume on demand |
| Scaling | Many sessions per node (like OS processes) |
| Healing | Migrate session state to new sandbox |
| Isolation | Sandboxed (gVisor/Kata) — assumed untrusted |
| Identity | Stable per session (not per container) |
| Networking | MCP tools, A2A agent-to-agent |
| Memory | Vector store for cross-session context |
| Warm pool | Pre-warmed sandboxes for fast cold starts |

### Agent Session Lifecycle
```
Created (sandbox provisioned)
   ↓
Initializing (warm-up, load tools)
   ↓
Idle (no prompt, hibernated to disk)
   ↓ (wake on event)
Active (processing prompt, running code)
   ↓ (idle timeout)
Hibernated (state saved, memory freed)
   ↓ (wake)
Active (state restored, fast resume)
   ↓ (session end)
Terminated (cleanup)
```

### Why This Matters for AI Control
The AI engine can spawn agent sessions to:
- Monitor cluster health continuously
- Execute operational tasks asynchronously
- Run diagnostic agents during incidents
- Coordinate multi-step remediation plans

## Related
- [[05 - Container Lifecycle]] — All workload types use container lifecycle
- [[AI Integration/03 - Agent Runtime]] — How agent sessions are managed
- [[03 - AI Scheduler]] — Different scheduling rules per type
