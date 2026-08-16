---
tags:
  - ui
  - gui
  - web
status: in-progress
priority: high
---

# GUI Dashboard (Website / App)

## Philosophy
More native, user-friendly interface for ==visual thinkers, managers, and less technical users==. The GUI exposes the same data as the [[TUI Dashboard|TUI]] but in a richer format.

## Current Implementation (Sprint 7 — 2026-08-16)
- **Go-served SPA**: `cmd/servez-gui/` — embedded static assets (`//go:embed`), stdlib-only, no build step, no node_modules.
- **Stitch-designed**: all screens generated in Google Stitch (project "ServeEz Control Plane", design system "Calm Instrumentation"); source of truth in `.stitch/designs/*.html`.
- **Design language**: flat dark surfaces (`#111319` canvas, `#1d2025` sidebar, `#272a30` cards), hairline borders, single indigo accent `#4c8dff`, Inter UI + JetBrains Mono data, semantic status colors only. No gradients, no glassmorphism, no neon — deliberately *not* "AI-looking".
- **Views shipped**: Dashboard (cluster overview), Nodes (health table), Workloads/Services (table + filters + deploy), Cost (live `/v1/cost/compare` form + results), Alerts (derived from node health), AI Control (chat + proposed-action rail with confidence meter).
- **Data**: polls `GET /v1/state`, `GET /v1/audit`, `POST /v1/cost/compare` via a `/v1/*` reverse proxy to the control plane.
- **Run**: `servez-gui --control=http://127.0.0.1:7400 --listen=:8080`
- **Design artifacts**: `.stitch/DESIGN.md` (design system source) + `.stitch/designs/{overview,services,cost,alerts,ai}.html/.png`.

## Pages

### Dashboard (Home)
- Health overview of entire cluster (map/grid)
- Cost burn rate (real-time $/hr)
- Recent AI actions & alerts
- Quick actions: "Deploy", "Scale", "Migrate"

### Cluster View
- Interactive topology graph of all nodes + services
- Color-coded by health (green/yellow/red)
- Click a node → detail panel

### Cost Explorer
- Breakdown by provider, service, region
- ==Savings opportunities highlighted by AI==
- "What-if" scenario calculator

### AI Chat
- Full-screen chat interface
- Conversation history
- Code blocks for commands/yaml
- "Explain this" button on any metric

### Alerts & Incidents
- Timeline view
- AI-generated RCA (Root Cause Analysis) for each incident
- One-click remediation

### Settings
- Provider credentials (AWS/Azure/GCP/bare-metal)
- AI permission levels
- Notification config (Slack, email, PagerDuty)

## Tech Options
| Stack | Pros | Cons |
|-------|------|------|
| Next.js + Tailwind | Fast dev, great ecosystem | Full-stack complexity |
| SvelteKit | Lightweight, reactive | Smaller ecosystem |
| Go + HTMX | Simple, server-rendered | Less rich interactions |
| **Go + embedded vanilla SPA (chosen)** | Stdlib-only, no deps, single binary | Less rich interactions |

## Mobile App
- React Native / Flutter
- Read-only mode for on-call
- Push notifications for critical alerts
- Quick "acknowledge" / "escalate" actions

← [[Index]]
