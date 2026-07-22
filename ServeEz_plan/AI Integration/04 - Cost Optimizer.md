---
tags:
  - ai
  - cost
  - cloud-arbitrage
status: draft
priority: high
---

# Cost Optimizer

## Role
Continuously analyzes cloud pricing across providers and ==recommends (or auto-executes) cost-saving actions==. The engine behind [[Core Features/03 - Real-Time Cloud Arbitrage]].

## Data Sources
- AWS Spot Price API (real-time)
- Azure Spot VM API (real-time)
- GCP Preemptible VM API (real-time)
- Reserved instance pricing (AWS RIs, Azure Reserved VMs)
- Data egress pricing per provider/region
- Historical usage patterns (for right-sizing)

## Optimization Loop
```
1. Scan: Current pricing + current usage per provider
2. Analyze: Compare cost of current setup vs alternatives
3. Score: Savings × stability × migration_risk
4. Recommend: "Move db-1 to AWS spot: save 62% ($142/mo)"
5. Simulate: Dry-run the migration, verify impact
6. Execute: If savings > threshold AND risk < threshold
```

## Optimization Types

### Instance Right-Sizing
- Detect over-provisioned instances (< 20% CPU for 7 days)
- Suggest downgrade to smaller instance type
- AI considers: peak vs average, growth trend, reserved time

### Spot Arbitrage
- Continuously scan spot prices across providers
- Migrate to cheapest spot market when savings > 15%
- Pre-evacuate before spot termination (2-min warning)
- Maintain on-demand fallback for critical services

### Reserved Instance Planning
- Predict steady-state usage 1-3 months ahead
- Recommend RI purchases when > 60% utilization predicted
- Compare RI vs spot + on-demand hybrid strategies

### Data Transfer Optimization
- Detect high inter-region / inter-provider traffic
- Suggest co-locating services to reduce egress fees
- Show cost impact of data locality decisions

## Savings Report Example
```json
{
  "report_period": "2026-07",
  "current_spend": "$3,420",
  "optimized_spend": "$2,860",
  "potential_savings": "$560 (16.4%)",
  "recommendations": [
    {
      "action": "rightsize",
      "resource": "db-1 (c6g.4xlarge)",
      "current_cost": "$480/mo",
      "target": "c6g.2xlarge",
      "target_cost": "$240/mo",
      "savings": "$240 (50%)",
      "risk": "low",
      "confidence": 0.92,
      "condition": "peak CPU over 7 days < 25%"
    },
    {
      "action": "migrate_to_spot",
      "resource": "worker-3",
      "provider": "AWS → GCP",
      "savings": "$88 (35%)",
      "risk": "medium",
      "confidence": 0.78,
      "condition": "spot termination < 5% probability"
    }
  ]
}
```

## Constraints
Savings are prioritized against:
- Latency requirements (can't move to cheaper region if p95 degrades)
- Availability requirements (critical services stay on-demand)
- Data residency (can't migrate across legal boundaries)
- Stability (new providers get gradual traffic, not all-at-once)

## Related
- [[Core Features/03 - Real-Time Cloud Arbitrage]] — Execution layer
- [[Orchestration/03 - AI Scheduler]] — Cost data feeds placement decisions
- [[AI Control/04 - State Model for AI]] — Cost metrics in state API
