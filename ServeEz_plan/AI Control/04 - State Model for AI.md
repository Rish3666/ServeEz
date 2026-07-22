---
tags:
  - ai-control
  - state
  - data-model
status: draft
priority: high
---

# State Model for AI

## Principle
AI can't read dashboards. It needs ==structured, queryable state== that it can reason over, filter, aggregate, and watch for changes.

## State Structure

### Cluster Overview
```json
{
  "api_version": "v1",
  "timestamp": "2026-07-23T12:00:00Z",
  "cluster": {
    "name": "prod-us-east",
    "node_count": 12,
    "service_count": 24,
    "agent_count": 5,
    "health": "degraded",
    "health_reason": "node-7 memory pressure"
  },
  "summary": {
    "total_cpu": "48 cores",
    "used_cpu": "62%",
    "total_memory": "256 GB",
    "used_memory": "71%",
    "total_cost_monthly": "$3,420",
    "cost_trend": "+12% vs last month"
  }
}
```

### Node Detail
```json
{
  "nodes": [
    {
      "id": "node-7",
      "provider": "aws",
      "instance_type": "c6g.4xlarge",
      "status": "degraded",
      "health_score": 68,
      "health_factors": [
        { "metric": "memory_pressure", "value": 89, "threshold": 80 },
        { "metric": "disk_iowait", "value": 12, "threshold": 15 }
      ],
      "prediction": {
        "failure_probability_24h": 0.23,
        "predicted_issue": "memory_pressure"
      },
      "workloads": [
        { "type": "service", "name": "api-gateway", "cpu": "45%", "mem": "32%" }
      ],
      "hardware": {
        "temperature_cpu": 72,
        "fan_speed_pct": 64,
        "power_watts": 340
      }
    }
  ]
}
```

### Service Detail
```json
{
  "services": [
    {
      "name": "web-frontend",
      "status": "healthy",
      "replicas": 5,
      "desired_replicas": 5,
      "cpu_utilization": "54%",
      "memory_utilization": "41%",
      "latency_p99": "124ms",
      "error_rate": "0.02%",
      "requests_per_sec": 2340,
      "prediction": {
        "traffic_next_hour": "stable",
        "traffic_peak_8pm": "4.2x current",
        "scaling_suggestion": "increase to 8 replicas by 7:45PM"
      },
      "cost": {
        "monthly": "$420",
        "per_request": "$0.00008"
      }
    }
  ]
}
```

## Query Patterns
AI queries state via structured queries (not grep/logs):

```
GET /state?type=node&filter=health_score<70
GET /state?type=service&filter=error_rate>1%
GET /state?type=node&sort=predicted_failure_probability:desc&limit=3
```

## State Subscriptions
AI can subscribe to state changes instead of polling:

```
POST /state/subscribe
{
  "watch": ["node.*.health_score", "service.*.status"],
  "event_types": ["degraded", "failed"]
}
→ Stream of delta events
```

## State Evolution (Diffs)
AI doesn't need the full state every time — it gets ==diffs==:

```json
{
  "timestamp": "2026-07-23T12:00:05Z",
  "deltas": [
    {
      "path": "nodes.node-7.health_score",
      "from": 85,
      "to": 68,
      "reason": "memory_pressure_increased"
    }
  ]
}
```

## Related
- [[02 - MCP Tool Interface]] — Read tools query this state
- [[05 - Action Audit & Safety]] — State diffs feed audit trail
