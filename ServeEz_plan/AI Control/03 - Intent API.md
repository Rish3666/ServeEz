---
tags:
  - ai-control
  - intent-api
  - design
status: draft
priority: critical
---

# Intent API

## Concept
Instead of calling individual MCP tools, the AI (or human) can express ==high-level intent==. The system decomposes intent into a plan, simulates it, and executes it.

## Example
```
User/AI: "Make the web tier more resilient for the flash sale tomorrow"
                           ↓
POST /intent
{
  "goal": "increase_resilience",
  "target": "web-tier",
  "context": "flash sale expected tomorrow 8PM-11PM EST",
  "constraints": {
    "max_budget_impact": "+$50",
    "min_availability": "99.9%"
  }
}
                           ↓
Intent Engine:
├── Analyzes context
├── Generates plan:
│   1. Scale web from 3→8 replicas (predict 4x traffic)
│   2. Pre-warm duplicate containers
│   3. Spread across 3 availability zones
│   4. Increase probe frequency to 5s
├── Simulates plan (risk: low, cost: +$34)
└── Returns plan for confirmation
```

## Intent API Endpoints

### POST /intent
Submit a high-level goal. Returns a plan.

```json
{
  "goal": "string (required, e.g., 'scale', 'heal', 'optimize', 'deploy', 'migrate')",
  "target": "string (required, what to act on)",
  "parameters": {},
  "context": "string (optional, free-form context for AI)",
  "constraints": {
    "max_budget_impact": "string",
    "min_availability": "number",
    "max_latency_p99": "string",
    "excluded_nodes": ["string"],
    "preferred_providers": ["string"]
  },
  "simulate_only": false
}
```

### GET /intent/:id
Check status of a previously submitted intent.

### DELETE /intent/:id
Cancel an intent in progress (if still in simulation phase).

## Goal Types

| Goal | What AI Does |
|------|-------------|
| `deploy` | Takes source, figures out build + deploy + config automatically |
| `scale` | Determines optimal replica count based on prediction + constraints |
| `heal` | Diagnoses issue, plans remediation, executes with approval |
| `optimize_cost` | Analyzes spend, generates migration plan |
| `optimize_perf` | Analyzes bottlenecks, suggests config changes |
| `migrate` | Moves workload between nodes/providers with zero downtime |
| `backup` | Creates backup plan with retention policy |
| `comply` | Generates compliance report from audit data |

## Intent Decomposition
```
Intent (high-level)
    ↓
AI breaks into sub-intents
    ↓
Each sub-intent maps to MCP tool(s)
    ↓
Simulated in order
    ↓
If all pass → execute sequence
    ↓
If any fails → rollback + report
```

## Related
- [[02 - MCP Tool Interface]] — Low-level tools that execute intent
- [[06 - Simulation Sandbox]] — How intents are tested before execution
