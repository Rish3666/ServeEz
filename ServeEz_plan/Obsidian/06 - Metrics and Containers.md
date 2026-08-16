---
tags:
  - obsidian
  - metrics
  - container
status: live
priority: high
---

# Metrics and Containers

## Metrics

- [[internal/metrics/collector.go]] samples CPU, memory, disk, and network usage.
- [[internal/metrics/buffer.go]] keeps a 5-minute local ring buffer.
- [[internal/metrics/capacity.go]] detects node capacity for join/bootstrap.

## Container Runtime

- [[internal/container/manager.go]] defines the runtime abstraction.
- [[internal/container/docker.go]] implements the Docker Engine API path.
- The agent uses this interface so containerd or runc can be added later.

## Runtime Responsibilities

- Create container instances.
- Start, stop, restart, and remove instances.
- Inspect live container state.
- List workload instances for status reporting.
- Run liveness/readiness/startup checks where possible.

## Related Notes

- [[Obsidian/04 - Agent Runtime]]
- [[Obsidian/07 - API Contracts]]

