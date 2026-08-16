# ServeEz

> AI-powered server & cluster management — predictive scaling, self-healing, and real-time cloud cost arbitrage.

ServeEz is an intelligent infrastructure management system that makes deployment easier, optimizes cloud costs, improves performance, and provides predictive & self-healing capabilities. Two dashboards — a TUI for power users and a GUI for visual management — both backed by an AI agent you can talk to in natural language.

---

## Features

### Core Platform
| Feature | Description |
|---------|-------------|
| Predictive Scaling | ML models forecast traffic spikes and pre-warm containers — near-zero latency scaling |
| Self-Healing | Detects micro-failures, memory leaks, and degradation before they cause downtime; seamless migration |
| Cloud Arbitrage | Real-time cost routing across AWS, Azure, GCP — rides the cheapest spot market |
| Compute Distribution | AI splits workloads across local servers, cloud VMs, and bare metal optimally |
| Server Cooling Control | Predictive fan/cooling control for bare-metal servers (IPMI/BMC) — saves 15-30% on cooling |
| AI Agent | Natural language ops: "scale up before the flash sale at 8PM" or "why is the database slow?" |
| One-Command Joins | `servez join <url> --token=xxx` — new nodes auto-discovered and provisioned in < 30s |
| Blue-Green Pre-Warm | Predictive duplicate containers — zero-downtime replacement for updates, downscaling, and healing |
| AI Monitoring | Proactive suggestions: "CPU trending to 95% in 10min — scale up?" not just "CPU at 95%" |

### Dashboards
- **TUI** — Terminal interface (`htop` meets `k9s` meets ChatGPT). Vim/Emacs bindings, split panes, AI chat sidebar.
- **GUI** — Web dashboard for visual thinkers and managers. Cluster topology, cost explorer, AI chat, one-click remediation.

---

## Architecture

```
┌──────────────────────────────────────┐
│             AI Engine                 │
│  Predictor | Anomaly Detector | LLM  │
└──────────────────┬───────────────────┘
                   │
┌──────────────────▼───────────────────┐
│          Core Platform                │
│  Orchestrator | Arbiter | Migration  │
└──────────────────┬───────────────────┘
                   │
┌──────────────────▼───────────────────┐
│         Per-Node Agents               │
│  Health Monitor | Metrics | Hardware │
└──────────────────────────────────────┘
```

- **Agent**: Go (< 50MB RAM), collects metrics, executes actions
- **AI Engine**: Python, ONNX runtime for inference
- **Communication**: gRPC + NATS
- **Storage**: VictoriaDB (metrics), Loki (logs), Postgres (audit/config)

---

## Target Audience

| Tier | Users | Pricing | Limits |
|------|-------|---------|--------|
| **Lite** (open-source) | Indie devs, solo founders | Free | Max 3 nodes, basic AI |
| **Pro** | SMB / Startups (10-100 servers) | $99-$499/mo | Full AI, multi-cloud |
| **Enterprise** | 100+ servers, compliance | Custom ($2k-$20k/mo) | SSO, RBAC, SLA, dedicated model |

### Deployment Model
- **Lite** — fully self-hosted: agent + orchestrator + AI engine on customer infrastructure. ==No SaaS component at all.==
- **Pro / Enterprise** — centrally-hosted control plane (orchestrator + AI engine + dashboards hosted by us) with a ==lightweight agent== on customer infra that phones home. The hosted control plane is the SaaS component; ==the agent itself is not.==
- Same ==open-core pattern as GitLab/Grafana/Supabase==.

---

## Roadmap

| Phase | Timeline | Focus | Status |
|-------|----------|-------|--------|
| **0: Foundation** | 20 days | Agent + basic TUI + Docker integration | ✅ Done |
| **1: Core MVP** | 1 month | Predictive scaling, AI chat, GUI, one-command joins, multi-cloud cost | 🚧 In progress (~60%) |
| **2: Intelligence** | 1 month | Self-healing, cloud arbitrage v1, compute distribution, open-source | ⛔ |
| **3: Advanced** | 3 months | Multi-cloud arbitrage, cooling control, enterprise features | ⛔ |
| **4: Scale** | 3 months | Chaos engineering, GitOps, FaaS, cost forecasting | ⛔ |

See [[Business/Roadmap]] for the granular checklist.

---

## Key Decisions

| Question | Decision |
|----------|----------|
| **K8s add-on vs K8s alternative?** | ✅ K8s alternative — own orchestration, AI-native |
| **Open-source license?** | ✅ AGPL |
| **MVP scope?** | ✅ Docker scaling on 3 servers + multi-cloud cost comparison |
| **Target market?** | ✅ All 3 tiers (Lite/Pro/Enterprise) |
| **AI auto-remediation?** | ✅ Per-action-type: scaling=auto, migration=manual, killing=manual+confirm |
| **Cloud vs on-prem?** | ✅ Hybrid (cloud + bare-metal under one plane) |
| **LLM strategy?** | ✅ Hybrid (local for predictions, cloud API for chat, user toggle) |

### Still Unresolved
- Stateful workload migration scope?
- Schema evolution/versioning strategy for the object store?
- Concurrent-write conflict resolution across multiple AI agents?

---

## Design Philosophy: AI-as-Pilot

ServeEz is not a traditional orchestrator with AI bolted on. ==The AI is the pilot, the platform is the plane.== Every component is designed for AI consumption:

- **MCP-native**: Every operation is a discoverable tool the AI can call
- **Intent API**: Express goals, not config — AI figures out the how
- **Structured state**: AI reads cluster state as JSON, not dashboards
- **Simulation-first**: Every action dry-runs before execution
- **Full audit**: Every action logged with before/after state, undo support

## Brainstorming

The full Obsidian vault lives in `ServeEz_plan/` with detailed docs on every feature, architecture, orchestration, AI control plane, business model, roadmap, risks, and 17+ brainstormed extension ideas.

---

## Getting Started

> Phase 0/1 control plane, CLI, agent, TUI, and predictive scaling are implemented. See the [[Obsidian/00 - Vault Map]] for the code map and the live build status in [[Business/Roadmap]].

```bash
# On the master node
servez init --dir=./cluster --addr=:9443
serveez-control --config ./cluster/control.json

# On worker nodes
servez join https://master:8443 --token=$(servez token)

# Deploy an app
servez deploy web --image=nginx:1.25 --replicas=2

# Watch the cluster live
serveez-tui --url=http://localhost:9443

# Chat with your cluster
servez chat  # read-only intent parsing (TUI chat pane)

# Fast-tier autoscaling (standing control loop)
serveez-autoscale --config ./cluster/control.json
```

[[Index]]