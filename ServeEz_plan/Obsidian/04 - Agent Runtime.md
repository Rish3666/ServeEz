---
tags:
  - obsidian
  - agent
  - runtime
status: live
priority: high
---

# Agent Runtime

## Responsibilities

- Generate or load node identity.
- Register with the control plane.
- Collect node usage every 5 seconds.
- Report node state every 10 seconds.
- Poll commands and execute them.
- Buffer samples locally during outages.

## Main Packages

- [[internal/agent/agent.go]]
- [[internal/agent/register.go]]
- [[internal/agent/tls.go]]
- [[internal/agentnet/client.go]]

## Agent Loop

1. Build a TLS config if the control plane is HTTPS.
2. Register with the join token.
3. Start the collect/report/command timers.
4. Gather metrics and maintain the local ring buffer.
5. Send reports and poll for queued actions.
6. Execute container actions through the runtime manager.

## State Model

- `pending`
- `healthy`
- `degraded`
- `unhealthy`
- `cordoned`
- `disconnected`

## Related Notes

- [[Obsidian/06 - Metrics and Containers]]
- [[Obsidian/07 - API Contracts]]

