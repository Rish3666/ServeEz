---
tags:
  - feature
  - cloud
  - cost-optimization
  - arbitrage
status: idea
priority: high
---

# Real-Time Cloud Arbitrage (Cost Routing)

## Concept
Cloud pricing changes constantly (spot instances, reserved vs on-demand, regional pricing). AI continuously scans pricing across providers and ==**seamlessly migrates live containers**== to the ==cheapest + lowest-latency provider== in real-time with zero downtime.

## Key Mechanisms
- **Spot Instance Surfer** — Rides the cheapest spot market across AWS, Azure, GCP, Linode, etc.
- **Price + Latency Optimiser** — Composite score = (price × weight1) + (latency × weight2) + (reliability × weight3)
- **Provider Hopping** — Warm standby containers on alternative providers for instant cutover

## Data Sources
- AWS Spot Price API
- Azure Spot VM API
- GCP Preemptible VM API
- Third-party cloud pricing aggregators
- Real-time latency probes (ICMP / HTTP / gRPC)

## Constraints & Rules
```
IF cost_savings > threshold
AND predicted_stability > 90%
AND migration_time < 5s
THEN → execute arbitrage migration
```

## State Management
| Workload Type | Migration Strategy |
|---------------|-------------------|
| Stateless (web servers) | DNS swap + connection drain |
| Stateful (caches) | Redis replication / DB replication |
| Compute (batch jobs) | Checkpoint + resume |

## Risks
- ==Spot instances can be terminated with 2-minute notice==
- ==Data egress fees between clouds can negate savings==
- Latency varies by time of day — needs constant re-evaluation
- Vendor lock-in escape, but now [[Project serveEz|ServeEz]] lock-in risk

## Integration
- Works with any Docker-compatible runtime via the ServeEz orchestrator
- No vendor lock-in to a specific cloud's native tooling

## Related
- [[01 - Zero Latency Predictive Scaling|Predictive Scaling]] handles capacity, this handles cost
- [[04 - Hardware Compute Distribution|Compute Distribution]] decides *where* workloads run
