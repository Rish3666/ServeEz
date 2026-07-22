---
tags:
  - ai-control
  - safety
  - audit
status: draft
priority: critical
---

# Action Audit & Safety

## Philosophy
AI with control over infrastructure is powerful — and dangerous. Every action must be ==visible, reversible, and verifiable== by both humans and AI.

## Audit Trail
Every action (MCP tool call, intent execution) generates an immutable audit entry:

```json
{
  "audit_id": "aud_7f3a1b",
  "timestamp": "2026-07-23T11:59:59Z",
  "initiator": "ai-agent:predictor-v2",
  "action": "scale_service",
  "parameters": {
    "service": "web-frontend",
    "replicas": 8,
    "reason": "predicted_traffic_spike_8pm"
  },
  "simulation": {
    "id": "sim_7f3a1b",
    "risk_score": 0.12,
    "predicted_impact": { "cost": "+$0.12/hr" }
  },
  "state_before": { "replicas": 5, "cpu": "54%" },
  "state_after": { "replicas": 8, "cpu": "32%" },
  "status": "completed",
  "confidence": 0.92,
  "duration_ms": 3400
}
```

## Safety System

### 1. Confidence Thresholds
| Action Type | Min Confidence | Notes |
|-------------|---------------|-------|
| Read/Monitor | 0% | No risk |
| Scale up | 0.70 | Adding resources is safe |
| Scale down | 0.85 | Removing resources needs caution |
| Config change | 0.80 | Can cause restarts |
| Migration | 0.90 | Moving live workloads |
| Kill/Terminate | 0.95 | Destructive, requires confirmation |

### 2. Permission Tiers
| Tier | Auto? | Scope |
|------|-------|-------|
| `monitor` | ✅ | Read-only, subscribe to events |
| `suggest` | ❌ | AI suggests, human approves |
| `auto-safe` | ✅ | Scale up, config changes within bounds |
| `auto-risky` | ✅ | Scale down, migration (if risk < threshold) |
| `danger` | ❌ | Kill, terminate, provider deletion |

### 3. Kill Switch
- Physical: Dedicated hardware button or USB key
- Keyboard: `Ctrl+Alt+Shift+Escape` at any terminal
- API: `POST /emergency/kill`
- Effect: Immediately revokes all AI tokens, sets system to `suggest` mode, pages on-call
- Can only be re-enabled by physical console access

### 4. Undo Capability
Every write action saves a snapshot before execution:
```
GET /undo/:audit_id
→ Returns state_before + estimated_time_to_restore
POST /undo/:audit_id
→ Reverts to state_before
```

### 5. Rate Limits
| Period | Max Actions | Notes |
|--------|------------|-------|
| Per second | 10 | Prevents runaway loops |
| Per minute | 100 | With circuit breaker |
| Per action type | 5/min | Prevents cascade failures |

## AI Self-Audit
AI can audit its own history:
```
GET /audit?initiator=ai-agent&status=failed
→ "I failed 3 times this week. Most common failure: confidence_threshold_too_low"
```

## Related
- [[02 - MCP Tool Interface]] — Every tool call generates audit entries
- [[06 - Simulation Sandbox]] — Simulation results are stored with audit
