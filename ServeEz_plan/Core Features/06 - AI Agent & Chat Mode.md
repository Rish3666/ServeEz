---
tags:
  - feature
  - ai-agent
  - chat
  - tui
status: idea
priority: high
---

# AI Agent & Chat Mode

## Concept
A ==full-access AI agent== that understands natural language commands for server and cluster management. Available through [[UI/TUI Dashboard|TUI]] and [[UI/GUI Dashboard|GUI]].

## Capabilities
- **Deploy** — "Deploy 3 containers of nginx with 2GB RAM each"
- **Diagnose** — "Why is the database slow right now?"
- **Optimize** — "Find the cheapest way to run this workload"
- **Scale** — "Scale up before the flash sale at 8PM"
- **Report** — "Show me last month's cost breakdown by service"
- **Self-Heal** — AI pro-actively reports: "Detected memory leak in payment-service, migrating now"

## Modes
| Mode | Description |
|------|-------------|
| Chat (Interactive) | Back-and-forth conversation for ops |
| Command (One-shot) | Single instruction → execute |
| Monitor (Passive) | AI watches and suggests without acting |
| Auto (Full trust) | AI acts autonomously, logs everything |

## Safety Concerns
- ==**Prompt injection risk**== — Malicious input could manipulate AI into dangerous actions
- ==**Permission tiers** needed==:
  - `read-only` — View only
  - `suggest` — AI suggests, human confirms
  - `auto-scope` — Auto within defined boundaries
  - `full-access` — No restrictions (dangerous)
- ==**Kill switch**== — Physical or keyboard shortcut that instantly revokes AI access
- **Audit log** — Every AI action recorded with before/after state

## Related
- [[Brainstorming/Flaws & Risks|Flaws & Risks]] — AI safety deep-dive
- [[Architecture/System Overview|System Overview]] — How agent integrates with other modules
- [[Brainstorming/Open Questions & Decisions|Open Questions]] — Q5 (auto-remediation) still unresolved
