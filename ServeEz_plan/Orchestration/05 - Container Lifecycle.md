---
tags:
  - orchestration
  - containers
  - lifecycle
status: draft
priority: high
---

# Container Lifecycle

## Simplified Model
Borrows from K8s Pod concept but ==drastically simplified== for AI control.

## Core Primitive: `Workload`
```
Workload
├── name: string
├── image: string
├── resources: { cpu, memory, gpu? }
├── replicas: int
├── ports: [{ container, protocol }]
├── env: { key: value }
├── volumes: [{ name, path, source }]
├── probes: { liveness, readiness, startup }
├── restart_policy: always | on_failure | never
└── strategy: rolling | blue_green | recreate
```

## Lifecycle Phases
```
Declared (API call received)
   ↓
Validated (AI checks constraints + simulates)
   ↓
Scheduled (AI assigns to node)
   ↓
Created (Node agent pulls image + creates container)
   ↓
Running (Health probes active)
   ↓
Healthy ↔ Degraded ↔ Unhealthy (AI monitors)
   ↓
Terminated (Scaling down, migration, or failure)
```

## Probes (AI-Optimized)
| Probe | Purpose | AI Optimization |
|-------|---------|----------------|
| Liveness | Is it alive? | AI adjusts timeout based on service profile |
| Readiness | Can it serve traffic? | AI pre-marks ready if warm-up is predictable |
| Startup | Has it initialized? | AI predicts startup time, sets threshold dynamically |

## Restart Policy
Not just "always restart" — AI decides:
- Crash loop? → Investigate before restarting (quarantine)
- Memory leak? → Restart during next low traffic window
- OOM? → Increase limit + restart
- Config error? → Don't restart, page human

## Zero-Downtime Operations
All lifecycle changes use [[Core Features/08 - Predictive Duplicate Container (Blue-Green Pre-Warm)|blue-green replacement]]:
1. New container starts alongside old one
2. New passes health checks
3. Traffic drains from old → new
4. Old terminates gracefully

## Container Runtime Interface
Node agent supports any OCI-compatible runtime:
- Docker (most compatible)
- containerd (lighter)
- runc (bare minimum)

Chosen at node registration time — reported to control plane.

## Related
- [[04 - Node Agent]] — Executes container lifecycle on each machine
- [[06 - Workload Types]] — Different lifecycle rules per type
- [[Core Features/08 - Predictive Duplicate Container (Blue-Green Pre-Warm)]]
