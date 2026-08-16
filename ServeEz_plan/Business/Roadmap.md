---
tags:
  - planning
  - roadmap
status: draft
priority: critical
---

# ServeEz Roadmap

> Decisions applied: Lightweight K8s alternative | AGPL | MVP = Docker scaling on 3 servers + multi-cloud cost | All 3 tiers
> Total timeline: **~8 months** (20 days Phase 0 → 1mo Phase 1 → 1mo Phase 2 → 3mo Phase 3 → 3mo Phase 4)
> ==Status (2026-08-16): Phase 0 complete, Phase 1 ~60% complete==

## Phase 0: Foundation ✅ (Complete)
- [x] ==Define MVP scope (what NOT to build is as important)==
- [x] Choose tech stack (Go for agent/control plane/TUI + Python for AI engine later)
- [x] Design agent architecture
- [x] Build basic agent: metrics collector + health monitor
- [x] Simple TUI showing server list + metrics
- [x] Deploy to a single server (dogfood) — live CLI → daemon → agent smoke test passed
- [x] Lock container orchestration API (Docker-compatible)

## Phase 1: Core MVP (in progress — 75%)
- [x] **Own orchestrator** — Lightweight container lifecycle (Docker-compatible API)
- [x] **Predictive Scaling (fast tier)** — Statistical forecast engine + autoscale control loop (Sprint 5)
- [x] **Multi-cloud cost comparison** — AWS/Azure/GCP pricing engine + savings report (Sprint 6)
- [ ] **Predictive Duplicate Container** — Blue-green pre-warm for zero-downtime ops (prewarm lead-time measured)
- [x] **AI Chat Mode (read-only baseline)** — TUI chat pane with intent parsing + simulate-then-confirm gating
- [ ] **GUI** — Basic dashboard with cluster view + cost comparison + AI chat
- [x] **One-command cluster additions** — [[Core Features/07 - One-Command Cluster Additions]]
- [x] **Real-time monitoring dashboard** — TUI cluster/services/status/alerts panes ([[Core Features/09 - Real-Time Monitoring & AI Suggestions]])

## Phase 2: Intelligence (1 month)
- [ ] **Predictive Self-Healing** — Anomaly detection + auto-migration
- [ ] **Cloud Arbitrage v1** — Spot instance optimization (one provider)
- [ ] **Compute Distribution** — Workload placement engine
- [ ] Audit logging + permission system
- [ ] Open-source release (AGPL)
- [ ] Enterprise tier: SSO, RBAC basics

## Phase 3: Advanced (3 months)
- [ ] **Multi-cloud Arbitrage** — Across providers
- [ ] **Hardware Cooling Control** — IPMI integration
- [ ] **Full enterprise features**: SSO, RBAC, audit compliance
- [ ] **Mobile app** — Read-only monitoring
- [ ] **Plugin marketplace**
- [ ] **Carbon-aware scheduling**

## Phase 4: Scale (3 months)
- [ ] **Chaos Engineering Mode** — AI-proactive failure testing
- [ ] **GitOps integration**
- [ ] **Serverless FaaS layer**
- [ ] **AI-Generated Runbooks**
- [ ] **Cost Forecasting**
- [ ] **Full closed-source Enterprise version**

## Key Milestones
| Timeline | Milestone                             | Success Metric                        |
| -------- | ------------------------------------- | ------------------------------------- |
| Month 1  | Agent + orchestrator runs on 1 server | < 50MB RAM idle, container starts     |
| Month 2  | MVP on 3 servers with cost comparison | Scaling works + cost report generated |
| Month 3  | Open-source launch (AGPL)             | 100 GitHub stars                      |
| Month 6  | First paying enterprise customer      | $10k MRR                              |
| Month 8  | Multi-cloud GA                        | 50+ clusters managed                  |

## Immediate Next Steps
1. ==GUI dashboard (cluster view + cost comparison + AI chat)==
2. Blue-green pre-warm wiring for scale-down
3. AI chat: move from read-only to executed actions (with confirm)
4. Live cloud pricing scraping (replace static catalog)
5. Add regression + e2e coverage across CLI/API/MCP/TUI

## Build Status (2026-08-16)
| Area | Status | Notes |
|------|--------|-------|
| CLI (`servez`) | ✅ Live | init/status/deploy/scale/token/join/cost, command registry |
| Control plane (`servez-control`) | ✅ Live | object store, audit, scheduler, HTTP API, TLS |
| Agent (`servez-agent`) | ✅ Live | metrics, register/report loop, command execution |
| MCP surface | ✅ Live | /v1/mcp/tools + /v1/mcp/call; read/write/simulate/subscribe/audit/predict tools |
| Simulation sandbox | ✅ Live | Tier-1 statistical + Tier-2 constraint validation |
| TUI (`servez-tui`) | ✅ Live | cluster/services/status/chat/alerts panes |
| Predictive scaling | ✅ Live | history store + linear-trend forecast + /v1/predict + autoscale loop |
| Cost comparison | ✅ Live | /v1/cost/compare + `servez cost` CLI (baseline 2026 catalog) |
| GUI | 🚧 Next | — |

← [[Index]]
