---
tags:
  - feature
  - cluster
  - operations
status: idea
priority: high
---

# One-Command Cluster Additions

## Concept
Adding a new server to a ServeEz cluster should be a ==single command==. No SSH key fiddling, no manual agent install, no config file editing.

## Flow
```
On new server (or via SSH):
$ servez join https://master:8443 --token=xxx

On master:
✓ Node registered
✓ Agent deployed
✓ Health check passed
✓ Added to pool
✓ Ready in < 30 seconds
```

## How It Works
1. **Discovery** — New server runs lightweight bootstrapper
2. **Auth** — Token-based or auto-approve (configurable)
3. **Provision** — ServeEz agent downloaded + started
4. **Register** — Node joins cluster, appears in TUI/GUI
5. **Readiness** — Automatic health check before accepting traffic

## Auto-Discovery Options
| Method | Use Case |
|--------|----------|
| `servez join <url>` | Interactive/manual add |
| Auto-discovery (mDNS) | Same network / homelab |
| Cloud API polling | AWS/GCP auto-scaling groups |
| DHCP hook | Bare-metal provisioning |
| GitOps | Declare node in repo → auto-join |

## Security
- Tokens expire after 15 minutes
- Each join requires signed certificate
- Auto-approve only within trusted subnets
- Manual approval required for external IPs

## Related
- [[04 - Hardware Compute Distribution|Compute Distribution]] decides workload placement on new nodes
- [[09 - Real-Time Monitoring & AI Suggestions|Monitoring]] tracks new node health after join
