---
tags:
  - ui
  - tui
  - terminal
status: idea
priority: high
---

# TUI Dashboard (Terminal User Interface)

## Philosophy
Base-metal control with AI insights. The TUI is the ==**primary**== interface for power users, sysadmins, and those who live in the terminal. It should feel like ==`htop` meets `k9s` meets ChatGPT==.

## Layout Concept
```
┌─────────────────────────────────────────────────────────┐
│ ServeEz v0.1 │ 🔵 Healthy (12 nodes)    │ 💰 $342/mo  │
├──────────┬──────────┬──────────┬────────────────────────┤
│ Cluster  │ Services │  Alerts  │  AI Chat               │
│ Map      │ List     │  (3 new) │  ┌──────────────────┐  │
│          │          │          │  │ "scale up web    │  │
│  🟢🟢🟢 │ web     │ 🟡 mem  │  │  servers by 2"   │  │
│  🟢🟡🟢 │ api     │  leak   │  │                  │  │
│  🟢🔴🟢 │ db      │   in    │  │  ✓ Scaling web   │  │
│          │ cache   │  api    │  │    from 3→5       │  │
│          │ worker  │   svc   │  │                  │  │
│          │         │ 🔴 cpu  │  └──────────────────┘  │
│          │         │  spike  │  [Ask anything...]     │
├──────────┴──────────┴──────────┴────────────────────────┤
│ 📊 web: requests 2.3k/s │ p95 45ms │ CPU 62% │ Mem 4.2G│
└─────────────────────────────────────────────────────────┘
```

## Features
- **Vim/Emacs keybindings**
- **Mouse support** for scroll/click
- **Split panes** (tmux-like)
- **AI Chat sidebar** — natural language commands
- **Real-time graphs** — brail plot / ascii sparklines
- **Dark mode** (terminal-native)

## Tech Options
| Library | Pros | Cons |
|---------|------|------|
| Bubble Tea (Go) | Native Go, great ecosystem | Terminal-only |
| Textual (Python) | Rich widgets, CSS-like | Slower, Python dep |
| Ratatui (Rust) | Fast, async | Rust learning curve |

## MVP Focus
1. Server list with health status
2. Basic metrics view (CPU/RAM/Disk)
3. AI chat prompt
4. Service deployment command

## Related
- [[GUI Dashboard]]
- [[06 - AI Agent & Chat Mode]]
