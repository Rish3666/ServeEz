---
tags:
  - brainstorming
  - questions
  - decisions
status: needs-decision
priority: high
---

# Open Questions & Decisions

> Questions that need answers before or during development. Update as decisions are made.

---

## 🔴 Must Decide (Pre-Development)

### Q1: K8s Add-on vs K8s Alternative?
Either:
- **(A) ServeEz as AI layer on top of K8s** — Works with existing K8s clusters, adds AI prediction/chat/healing. Less ambitious, faster to market, piggybacks on K8s ecosystem.
- **(B) ServeEz as lightweight K8s alternative** — Own container orchestration, simpler than K8s, AI-native from day one. Harder to build but more differentiated.

==**Decision**: [ ]==

### Q2: Open Source License?
- AGPL (strong copyleft, enterprise must buy commercial license)
- Apache 2.0 (permissive, harder to monetize)
- BSL / Source-available (like HashiCorp / Sentry)
- Dual license (AGPL community + commercial enterprise)

**Decision**: [ ]

### Q3: ==MVP Scope — What's the smallest useful thing?==
Possibilities:
- Single-server monitoring + AI chat (least useful but fastest)
- Docker auto-scaling on 3 servers (more useful, more work)
- Multi-cloud cost comparison (useful for SMBs)
- AI chat with read-only commands (safe intro to the concept)

**Decision**: [ ]

### Q4: Target Market — Startups vs Enterprises first?
- Startups: Faster sales cycle, lower budget, more risk-tolerant
- Enterprises: Longer sales cycle, bigger budget, need compliance/SSO/SLA
- Indie devs: Zero revenue but great for adoption + feedback

**Decision**: [ ]

---

## 🟡 Should Decide (During Development)

### Q5: ==Should AI have automatic remediation or always suggest?==
- **Option 1**: AI suggests → Human confirms (safer, slower)
- **Option 2**: AI acts within boundaries (need clear "boundary" definition)
- **Option 3**: Configurable per-action-type (scaling = auto, migration = manual, killing = manual + confirmation)

**Decision**: [ ]

### Q6: Stateful workload migration — how far do we go?
- Skip stateful entirely for MVP (stateless only)
- Support DB replication-based migration
- Full live migration (carry memory state) — **very hard**

**Decision**: [ ]

### Q7: Cloud-Only vs Hybrid vs On-Prem?
- Cloud-only (simpler, no hardware cooling, no IPMI)
- Hybrid (both cloud + bare-metal)
- On-prem focus (datacenters, colo, homelab)

**Decision**: [ ]

### Q8: ==LLM — Local vs API?==
- Cloud API (OpenAI, Anthropic, etc.): cheaper inference, privacy concerns, requires internet
- Local LLM (Llama, Mistral, Qwen): private, offline-capable, needs GPU, slower
- Hybrid: local for predictions, API for complex chat, fallback for offline

**Decision**: [ ]

### Q9: What language for the agent?
- Go: lightweight, great concurrency, huge DevOps ecosystem
- Rust: fastest, safest, steep learning curve
- Python: best AI/ML ecosystem, heavier
- Go + Python hybrid (agents in Go, AI in Python)

**Decision**: [ ]

---

## 🟢 Good to Answer (Later)

### Q10: Pricing model?
- Per-node per-month
- Per-cluster flat fee
- Percentage of cloud cost saved (value-based)
- Free core + paid enterprise features

### Q11: Mobile app necessity?
- Read-only alerts and monitoring (useful)
- Full management (probably overkill)
- Skip mobile, focus on responsive web

### Q12: Should we build a FaaS layer?
- Expands scope significantly
- Could be a spin-off product
- Probably Phase 4+

### Q13: Data residency / compliance?
- Support for EU-only data storage (GDPR)
- FedRAMP for US government
- HIPAA for healthcare

---

## Decision Log

| # | Question | Decision | Date | Rationale |
|---|----------|----------|------|-----------|
| — | — | — | — | — |
