---
tags:
  - orchestration
  - node-agent
status: draft
priority: critical
---

# Node Agent

## Role
The per-machine daemon. It ==listens for instructions, executes them, and reports structured state back==. No config file watching, no polling — pure API-driven operation.

## Design Constraints
- **Size**: < 50MB RAM idle target
- **Language**: Go (or Rust for bare-metal)
- **Dependencies**: Zero (statically linked binary)
- **Auth**: mTLS certificate from control plane
- **Connectivity**: gRPC stream + MCP

## Responsibilities

### Container Management
- Create, start, stop, remove containers (via Docker API or containerd)
- Report container status, resource usage, health check results
- Handle image pulls (with caching + pre-pull for predictions)

### Health Probing
- Execute liveness, readiness, startup probes
- Report results as structured data (not logs)
- Detect probe degradation trends (feeds anomaly detection)

### Metrics Collection
- CPU, memory, disk, network metrics at 5s intervals
- Temperature, fan speed, power (if IPMI/BMC available)
- Local buffer (5min) for offline tolerance
- Stream to control plane / AI engine

### Hardware Control
- IPMI/BMC fan control for bare-metal (predicted cooling)
- Power monitoring via PDU SNMP
- Temperature sensor aggregation

### Local AI Cache
- Small ONNX model for offline prediction
- Falls back when control plane unreachable
- Cached decisions for basic ops (restart unhealthy container)

## Lifecycle
```
1. Registration: Node boots → agent starts → mTLS handshake
2. Assignment: Control plane confirms identity → adds to cluster
3. Run: Agent reports state → receives instructions → executes
4. Failure: Agent crashes → node marked unhealthy → workloads migrated
5. Recovery: Agent restarts → syncs missed state → resumes
```

## State Reports
Every 10 seconds, or on state change:
```json
{
  "node_id": "node-3",
  "status": "healthy",
  "health_score": 92,
  "resources": { "cpu_pct": 34, "mem_pct": 56, "disk_pct": 41 },
  "hardware": { "temp_cpu": 62, "fan_pct": 48 },
  "workloads": [
    { "id": "web-1", "status": "running", "cpu_pct": 12, "mem_mb": 256 }
  ],
  "prediction": { "cpu_10min": 45, "mem_10min": 61 }
}
```

## Related
- [[05 - Container Lifecycle]] — What the agent manages
- [[Orchestration/07 - Networking]] — Network config the agent applies
- [[AI Control/02 - MCP Tool Interface]] — Agent exposes its capabilities as MCP tools
