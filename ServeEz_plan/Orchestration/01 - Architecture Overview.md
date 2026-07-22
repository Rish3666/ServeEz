---
tags:
  - orchestration
  - architecture
  - ai-native
status: draft
priority: critical
---

# Orchestration Architecture Overview

## Design Center
ServeEz is a ==lightweight K8s alternative built AI-native from day one==. Unlike K8s where AI is a bolt-on, ServeEz's orchestrator has AI at the center of every decision — scheduling, scaling, healing, placement, networking.

## High-Level Diagram
```
┌──────────────────────────────────────────────────────────┐
│                      AI Engine                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐  │
│  │Predictor │ │ Anomaly  │ │    LLM   │ │    Cost    │  │
│  │(Scaling) │ │ Detector │ │  Agent   │ │ Optimizer  │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────────┘  │
└──────────────────────┬───────────────────────────────────┘
                       │ MCP / Intent API
┌──────────────────────▼───────────────────────────────────┐
│                   Control Plane                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐  │
│  │   API    │ │  State   │ │  AI      │ │ Controller │  │
│  │  Server  │ │  Store   │ │Scheduler │ │  Manager   │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────────┘  │
└──────────────────────┬───────────────────────────────────┘
                       │ gRPC / MCP
┌──────────────────────▼───────────────────────────────────┐
│                    Node Agent (per machine)                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐  │
│  │Container │ │  Health  │ │ Metrics  │ │Hardware Ctrl│  │
│  │ Manager  │ │  Prober  │ │Collector │ │ (IPMI)     │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────────┘  │
└──────────────────────────────────────────────────────────┘
```

## Key Differences from Kubernetes
| Aspect | Kubernetes | ServeEz |
|--------|-----------|---------|
| Configuration | YAML manifests | API calls / MCP tools |
| Scheduling | Resource fit + constraints | AI predictive placement |
| Scaling | Threshold-based (HPA) | ML prediction + pre-warm |
| Healing | Pod restart (reactive) | Micro-failure pred + pre-migrate |
| Networking | CNI plugins + kube-proxy | Built-in mDNS + proxy |
| State | etcd (heavy) | SQLite (small) → etcd (scale) |
| AI integration | External operators | ==In every control loop== |
| Agent workloads | Not supported natively | First-class (hibernatable) |

## Data Flow
```
Intent → API Server → AI Scheduler → Node Agent → Container Runtime
   ↑          ↓            ↑              ↓             ↓
   └── Audit ← State Store ← ─────────────┴── Metrics → AI Engine
```

## Deployment Models
- **Single node**: All-in-one (agent + control plane + AI on same machine)
- **Small cluster**: 1 control plane + N workers (homelab / SMB)
- **Large cluster**: HA control plane + N workers + dedicated AI nodes (enterprise)

## Related
- [[Orchestration/02 - Control Plane]]
- [[Orchestration/03 - AI Scheduler]]
- [[Orchestration/04 - Node Agent]]
- [[AI Control/01 - Philosophy & Principles]]
