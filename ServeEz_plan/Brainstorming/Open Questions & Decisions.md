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

### ✅ Q1: K8s Add-on vs K8s Alternative?
~~Either:~~
- ~~(A) ServeEz as AI layer on top of K8s~~
- ~~(B) ServeEz as lightweight K8s alternative~~

==**Decision**: B — Lightweight K8s alternative with native AI== — Own container orchestration, simpler than K8s, AI-native from day one. Harder to build but more differentiated.

### ✅ Q2: Open Source License?
- ~~Apache 2.0, BSL, Dual license~~

==**Decision**: AGPL== — Strong copyleft, enterprises must buy commercial license for proprietary use.

### ✅ Q3: MVP Scope — What's the smallest useful thing?
~~Possibilities:~~
- ~~Single-server monitoring + AI chat~~
- ~~Multi-cloud cost comparison~~
- ~~AI chat with read-only commands~~

==**Decision**: Docker auto-scaling on 3 servers + Multi-cloud cost comparison== — Delivers immediate value: cost savings + performance.

### ✅ Q4: Target Market?
- ~~Startups vs Enterprises vs Indie devs~~

==**Decision**: All 3 tiers (Lite/Pro/Enterprise)== — Two versions of the app: open-source Lite for indie devs, paid Pro + Enterprise for companies.

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
| Q1 | K8s Add-on vs Alternative | Lightweight K8s alternative | 2026-07-23 | More differentiated, AI-native from day one |
| Q2 | Open-source license | AGPL | 2026-07-23 | Strong copyleft, forces enterprise to buy commercial |
| Q3 | MVP scope | Docker scaling on 3 servers + multi-cloud cost | 2026-07-23 | Delivers both performance + cost value in MVP |
| Q4 | Target market | All 3 tiers (Lite/Pro/Enterprise) | 2026-07-23 | Two versions of the app serving different segments |
