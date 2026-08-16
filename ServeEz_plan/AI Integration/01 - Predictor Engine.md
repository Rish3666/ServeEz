---
tags:
  - ai
  - prediction
  - scaling
status: implemented
priority: critical
---

# Predictor Engine

## Implementation Status: ✅ Fast Tier Live (2026-08-16)

- `internal/predictor` — least-squares linear-trend forecast with r² confidence (fast tier, <100ms, no ML deps). Cold-start guard (12 samples).
- `internal/history` — SQLite time-series store recording node CPU% from every report.
- `GET /v1/predict` + MCP `predict.scale` + `cmd/servez-autoscale` loop.
- Prophet/LSTM time-series models, anomaly detection, and cross-tenant transfer learning remain the slow-tier / Phase 2 path.
- See [[Obsidian/09 - Predictive Scaling]] for the implementation doc.

## Role
The brain behind ==predictive scaling and proactive scheduling==. Feeds every AI decision with forward-looking data.

## What It Predicts

| Prediction | Source Data | Used By |
|-----------|------------|---------|
| Traffic volume (requests/s) | Historical metrics, time of day, day of week, seasonality | [[Orchestration/03 - AI Scheduler]] |
| Resource demand (CPU/mem) | Current usage + growth trend | [[Core Features/01 - Zero Latency Predictive Scaling]] |
| Cost trajectory | Cloud pricing changes, usage trend | [[04 - Cost Optimizer]] |
| Failure probability | Anomaly signals, hardware age, error rate | [[02 - Anomaly Detection]] |
| Scaling lead time | Image pull time, startup time, warm-up duration | Pre-warm timing |

## Model Architecture
```
Metrics Stream (5s intervals)
    ↓
Feature Pipeline
    ↓
┌─────────────────┐  ┌──────────────────┐
│ Time-Series Model│  │ Anomaly Detector  │
│ (Prophet/LSTM)   │  │ (Isolation Forest)│
└────────┬────────┘  └────────┬─────────┘
         ↓                    ↓
    Traffic Forecast    Anomaly Score
         ↓                    ↓
┌─────────────────────────────────────────┐
│           Decision Engine                │
│  Combines predictions → recommendations │
└─────────────────────────────────────────┘
```

## Prediction Confidence
Every prediction includes a confidence score:

```json
{
  "service": "web-frontend",
  "metric": "requests_per_sec",
  "current": 2340,
  "predicted_15min": 2800,
  "predicted_1hr": 4500,
  "predicted_peak_8pm": 9800,
  "confidence_15min": 0.95,
  "confidence_1hr": 0.88,
  "confidence_peak": 0.72,
  "recommendation": "scale to 12 replicas by 7:45PM"
}
```

## Cold Start Handling
- New services with no history: Use aggregate patterns from similar services
- First 24 hours: Conservative scaling (over-provision slightly)
- After 7 days: Full ML model activated
- Transfer learning from aggregated (anonymized) cross-tenant data

## Feedback Loop
```
Prediction made → Action taken → Actual observed → Compare → Retrain
                                                              ↓
                                            If error > 10% → adjust model
```

← [[Index]]
