---
tags:
  - orchestration
  - control-plane
status: draft
priority: critical
---

# Control Plane

## Components

### API Server
- Single entry point for all operations
- Exposes REST + gRPC + MCP endpoints
- ==No YAML parsing== — only structured API calls
- Handles auth, validation, admission control
- MCP tool discovery endpoint

### State Store
| Size | Store | Why |
|------|-------|-----|
| 1-3 nodes | SQLite (embedded) | Zero dependencies, < 10MB |
| 3-50 nodes | Postgres | Durable, familiar |
| 50+ nodes | etcd | Proven at scale, watches |

### Controller Manager
Runs reconciliation loops — but ==AI-driven, not just threshold-based==:

| Controller | Traditional (K8s) | ServeEz (AI) |
|-----------|-------------------|--------------|
| Scaling | HPA: CPU > 80% → scale | Predictor: traffic spike in 20min → pre-scale |
| Healing | Pod crash → restart | Anomaly: mem leak detected → pre-migrate |
| Placement | Scheduler at deploy time | Continuous re-optimization by AI |
| Node health | Node condition → cordon | Health score trending → pre-evacuate |

### MCP Server
- Bundled into control plane
- Registers all control plane operations as MCP tools
- Handles authentication + authorization for each tool call
- Enforces rate limits and confidence thresholds

## API Design Principles
```
GET    /v1/state         ← Read cluster state (structured JSON)
POST   /v1/intent        ← Submit high-level goal
POST   /v1/simulate      ← Dry-run an action
POST   /v1/execute       ← Execute a concrete action
GET    /v1/audit         ← Query audit log
POST   /v1/emergency/kill ← Kill switch
```

## No YAML Policy
- There is no `kubectl apply -f` equivalent
- All config changes happen through API calls
- Humans use the TUI/GUI which makes API calls
- AI uses MCP tools which make API calls
- For GitOps: API calls are recorded as IaC → replayed

## Related
- [[03 - AI Scheduler]] — Sits inside control plane
- [[04 - Node Agent]] — Communicates with control plane via MCP
- [[AI Control/02 - MCP Tool Interface]] — How tools are exposed
