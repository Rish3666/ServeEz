---
tags:
  - ai-control
  - control-loop
  - design
status: implemented
priority: critical
---

# Native Control Loop

## Implementation Status: ✅ Fast Tier Live (2026-08-16)

The fast tier of the native loop is implemented as `cmd/servez-autoscale` (see [[Obsidian/09 - Predictive Scaling]]): a standing process that continuously forecasts workloads, dry-runs via the simulation sandbox, passes a confidence decision gate, and executes up-only scale actions with cooldown. The push-based NATS delta stream and LLM escalation (slow tier) remain future work.

## Bolted-On vs Native AI

Most "AI infrastructure" tools are ==bolted-on AI==: a chatbot that answers questions on request. You ask, it replies, the loop ends. The system runs fine (or not) without it.

ServeEz is ==native AI==: a standing control loop that watches state continuously and acts on it — not waiting for a prompt. The difference:

| Aspect | Bolted-On AI | Native AI |
|--------|-------------|-----------|
| Trigger | User asks a question | State changes (push-based) |
| Loop | Request → response, then stops | Standing watch → evaluate → act → repeat |
| Relationship | AI sits outside the system | AI is in every control path |
| Failure mode | You notice problems and ask | The loop notices and self-heals |

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    NATS / Postgres                         │
│   LISTEN-NOTIFY state delta stream (push, not poll)        │
└──────────────────────────┬─────────────────────────────────┘
                           │  delta events
┌──────────────────────────▼─────────────────────────────────┐
│                  Watch Loop (always on)                    │
│  filters & debounces deltas → routes to model tier         │
└──────────────────────────┬─────────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────────┐
│          Fast Tier: statistical / ONNX model               │
│  routine cases (scale, placement, thresholds)              │
│  → decides in <100ms, low cost, no LLM call                │
└──────────────┬───────────────────────────┬─────────────────┘
               │ routine                   │ novel / ambiguous
               │ (high confidence)         │ (low confidence / unseen)
┌──────────────▼───────────┐  ┌────────────▼─────────────────┐
│  simulate (dry-run)      │  │  LLM escalation (slow tier)  │
│  in sandbox (06)         │  │  reasons about edge cases    │
└──────────────┬───────────┘  └────────────┬─────────────────┘
               │                           │
               └───────────┬───────────────┘
                           │ proposed action + confidence score
┌──────────────────────────▼─────────────────────────────────┐
│               Decision Gate                                 │
│  confidence ≥ threshold → execute via MCP tool             │
│  confidence < threshold → queue for human review            │
│  (see 05 - Action Audit & Safety)                          │
└────────────────────────────────────────────────────────────┘
```

The watch loop subscribes to state deltas via ==NATS / Postgres LISTEN-NOTIFY== — a push-based subscription, not polling. The loop only wakes when state actually changes (see [[AI Control/04 - State Model for AI]]).

## Why the Hybrid Split

Calling an LLM on every tick would be ==slow and expensive== — a large model per state change burns budget and adds latency to every routine decision.

- **Fast tier (statistical/ONNX)** runs ==on every delta, continuously==. Routine cases — "CPU trending up, scale the web tier", "one replica unhealthy, replace it" — are pattern-matched and decided in milliseconds with a small, cheap local model. This is the loop's heartbeat.
- **LLM escalation** engages only for ==novel or ambiguous cases== the fast tier can't confidently classify — unseen failure modes, cross-service conflicts, intent-heavy decisions. These are rare, so paying for an LLM on them is fine.

The result: a ==continuous, always-on loop that stays fast and affordable== because the cheap model does the repetitive work and the expensive one only handles the edges.

## Cross-References
- [[AI Control/04 - State Model for AI]] — the delta stream the loop watches
- [[AI Control/05 - Action Audit & Safety]] — decision gate, audit, undo, kill switch
- [[AI Control/06 - Simulation Sandbox]] — dry-run before execute
- [[AI Control/02 - MCP Tool Interface]] — the tools the loop calls on execution
- [[AI Control/01 - Philosophy & Principles]] — why native AI is the core tenet

← [[Index]]
