---
tags:
  - orchestration
  - scheduler
  - ai
status: draft
priority: critical
---

# AI Scheduler

## Concept
Unlike K8s scheduler (filters + scores at deploy time), ServeEz's scheduler is ==continuous and predictive==. It doesn't just decide where a workload starts — it continuously optimizes placement based on predicted future conditions.

## How It Works

### Traditional (K8s)
```
Pod created → Filter nodes by resources → Score by fit → Assign → Done forever
```

### ServeEz (AI)
```
Service declared → AI analyzes patterns → Predicts optimal node
                 → Assigns initially
                 → Continuously monitors
                 → Re-optimizes if conditions change
                 → Pre-migrates before problems
```

## Scheduling Factors

| Factor | Weight | Source |
|--------|--------|--------|
| Resource fit | Required | Current metrics |
| Traffic prediction | High | [[AI Integration/01 - Predictor Engine]] |
| Failure probability | High | [[AI Integration/02 - Anomaly Detection]] |
| Cost per hour | Medium | Cloud pricing API |
| Latency to users | Medium | Proximity data |
| Carbon intensity | Low (configurable) | Grid data |
| Hardware affinity | Optional | GPU, NVMe, etc. |
| Data locality | High | Storage attachment |

## Scheduling Flow
```
1. AI Predictor: "node-3 will have 15% headroom in 30min"
2. Cost Optimizer: "node-3 is $0.08/hr cheaper than node-7"
3. Anomaly Detector: "node-7 has elevated disk error rate"
4. Scheduler: "Place on node-3. Re-evaluate in 15min."
```

## Re-Scheduling
- Not a one-time event — ==continuous optimization==
- Triggers:
  - Node health score changes > 10 points
  - Cost differs by > 5% from projection
  - Traffic pattern shifts detected
  - New node joins cluster
- Migration is zero-downtime (pre-warm + drain)

## Agent-Aware Scheduling
For [[06 - Workload Types#Agent Sessions|Agent workloads]], the scheduler additionally considers:
- Session hibernation compatibility
- Sandbox warm pool availability
- Agent-to-agent affinity (co-locate communicating agents)
- GPU/memory for model inference

## Related
- [[AI Integration/01 - Predictor Engine]] — Feeds traffic predictions
- [[AI Integration/02 - Anomaly Detection]] — Feeds failure probability
- [[AI Control/06 - Simulation Sandbox]] — Scheduler simulates placements before assigning
