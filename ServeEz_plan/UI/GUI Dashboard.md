---
tags:
  - ui
  - gui
  - web
status: idea
priority: high
---

# GUI Dashboard (Website / App)

## Philosophy
More native, user-friendly interface for ==visual thinkers, managers, and less technical users==. The GUI exposes the same data as the [[TUI Dashboard|TUI]] but in a richer format.

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

## Mobile App
- React Native / Flutter
- Read-only mode for on-call
- Push notifications for critical alerts
- Quick "acknowledge" / "escalate" actions
