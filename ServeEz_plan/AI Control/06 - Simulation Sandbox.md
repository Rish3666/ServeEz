---
tags:
  - ai-control
  - simulation
  - safety
status: draft
priority: critical
---

# Simulation Sandbox

## Concept
Before any action, AI can ==simulate it in a sandbox== — no real infrastructure changes. The sandbox returns predicted impact, risk score, and confidence. AI only executes if the simulation passes.

## Why Simulation?
- Prevents mistakes before they happen
- AI learns from simulation results (reinforcement learning)
- Humans can review simulated plans before trusting auto-mode
- "What-if" scenarios for capacity planning

## Simulation Flow
```
AI: "I want to scale web-service from 3→8 replicas"
                     ↓
POST /simulate
{
  "action": "scale_service",
  "parameters": {
    "service": "web",
    "replicas": 8
  }
}
                     ↓
Simulation Engine:
├── Current state snapshot
├── Traffic prediction model
├── Cost calculator
├── Dependency impact analysis
└── Returns:
{
  "simulation_id": "sim_abc123",
  "risk_score": 0.12,
  "confidence": 0.88,
  "predicted_outcomes": [
    { "metric": "p99_latency", "from": "145ms", "to": "98ms" },
    { "metric": "cost_per_hour", "from": "$0.48", "to": "$0.92" }
  ],
  "failure_scenarios": [
    { "scenario": "traffic_doesnt_spike", "impact": "wasted 5 replicas (+$0.44/hr)", "probability": 0.15 }
  ],
  "recommendation": "proceed"
}
```

## Sandbox Tiers

### Tier 1: Statistical Simulation (Fast)
- Uses historical data + ML models
- No actual containers started
- Response: < 100ms
- Accuracy: ~80%

### Tier 2: Dry-Run (Medium)
- Actually runs the API call but with `dry_run: true`
- Orchestrator validates constraints, checks resources
- No actual containers created
- Response: < 2s
- Accuracy: ~95%

### Tier 3: Full Sandbox (Slow but Precise)
- Spins up containers in isolated network namespace
- Runs real health checks
- Measures actual resource usage
- Response: < 30s
- Accuracy: ~99%

## What-If Queries
```
POST /simulate/what-if
{
  "scenario": "What if we move db-1 to a spot instance?",
  "constraints": { "max_downtime": "0s", "min_iops": 10000 }
}
→ Returns cost savings, risk assessment, migration plan
```

## Learning Loop
```
Simulation executed → Compare predicted vs actual → Refine models
                         ↓
              If prediction was wrong → Flag for retraining
```

## Related
- [[03 - Intent API]] — Every intent is simulated before execution
- [[05 - Action Audit & Safety]] — Simulation results stored in audit trail
