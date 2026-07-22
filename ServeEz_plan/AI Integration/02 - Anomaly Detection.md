---
tags:
  - ai
  - anomaly
  - self-healing
status: draft
priority: critical
---

# Anomaly Detection

## Role
The ==early warning system==. Detects micro-failures, performance degradation, and security anomalies before they become incidents. Feeds the self-healing loop and scheduler with real-time health signals.

## Detection Methods

### Statistical (Real-time)
- Moving average deviation (CPU, memory, latency)
- Z-score analysis for metric spikes
- Seasonality decomposition (is this normal for Tuesday at 3PM?)

### ML-Based (Pattern Recognition)
- Isolation Forest for multivariate anomaly scoring
- Autoencoder reconstruction error for log patterns
- Sequential pattern mining for request trace anomalies

### NLP (Log Analysis)
- Log level escalation trends
- New error message types
- Correlation across services ("payment-service errors always follow redis latency spikes")

## Detection Categories

| Category | Examples | Detection Method |
|----------|----------|-----------------|
| Resource | Memory leak, CPU runaway, disk filling | Statistical trend |
| Performance | Latency spike, throughput drop | ML + statistical |
| Failure | Crash loop, OOM, connection refused | Pattern matching |
| Security | Brute force, unusual egress, privilege escalation | ML + NLP |
| Dependency | Upstream API slow, DB connection pool exhausted | Graph correlation |

## Confidence Scoring
```json
{
  "anomaly_id": "anom_8f2a1",
  "type": "memory_leak",
  "severity": "warning",
  "target": "service:payment-api",
  "indicators": [
    { "metric": "rss_memory", "trend": "+15%/hr for 6hr", "z_score": 3.2 },
    { "metric": "gc_pause_time", "trend": "+8%/hr", "z_score": 2.1 }
  ],
  "confidence": 0.87,
  "predicted_impact": {
    "time_to_critical": "4.2 hours",
    "affected_services": ["payment-api", "checkout"]
  },
  "recommended_action": "restart during next low-traffic window"
}
```

## Self-Healing Loop
```
Anomaly Detected (confidence > 0.80)
    ↓
Root Cause Analyzed (log correlation + dependency graph)
    ↓
Action Planned (by AI engine)
    ↓
Simulated (dry-run in sandbox)
    ↓
Executed (auto or approved per policy)
    ↓
Monitored (did the fix work?)
    ↓
If not → Escalate to human + detailed report
```

## Feedback to Scheduler
- Node with repeated anomalies → lower scheduling priority
- Service with frequent restarts → schedule on more reliable hardware
- Detected capacity bottleneck → trigger predictive scale-up

## Related
- [[Core Features/02 - Predictive Self-Healing & Seamless Migration]]
- [[Orchestration/03 - AI Scheduler]] — Feeds failure probability to scheduler
- [[AI Control/04 - State Model for AI]] — Anomaly scores in state API
