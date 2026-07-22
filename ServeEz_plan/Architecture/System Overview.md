---
tags:
  - architecture
  - design
status: draft
priority: critical
---

# System Architecture Overview

> **Decisions Applied**: Lightweight K8s alternative | ==Hybrid (cloud + bare-metal)== | ==Hybrid LLM (local + API, user-configurable)==

## High-Level Diagram
```
┌─────────────────────────────────────────────────────┐
│                    AI Engine                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐     │
│  │Predictor │ │ Anomaly  │ │  LLM Agent       │     │
│  │(Scaling) │ │ Detector │ │  (Chat + Actions) │    │
│  └──────────┘ └──────────┘ └──────────────────┘     │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────┐
│                   Core Platform                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐      │
│  │Orchestrat│ │  Arbiter │ │  Migration Eng.  │      │
│  │or(Scale) │ │(Cost Opt)│ │  (Self-Heal)     │      │
│  └──────────┘ └──────────┘ └──────────────────┘      │
└──────────────────────┬───────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                  Agents (per node)                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐     │
│  │  Health  │ │  Metrics │ │  Hardware Ctrl   │     │
│  │ Monitor  │ │ Collector│ │  (Cooling/etc)   │     │
│  └──────────┘ └──────────┘ └──────────────────┘     │
└─────────────────────────────────────────────────────┘
```

## Module Breakdown

### AI Engine
- **Predictor** — Time-series forecasting for scaling ([[01 - Zero Latency Predictive Scaling]])
- **Anomaly Detector** — Micro-failure detection ([[02 - Predictive Self-Healing & Seamless Migration]])
- **LLM Agent** — Natural language interface ([[06 - AI Agent & Chat Mode]])

### Core Platform
- **Orchestrator** — Native container/pod lifecycle management (simpler than K8s, AI-native)
- **Arbiter** — Cloud pricing analysis & routing ([[03 - Real-Time Cloud Arbitrage]])
- **Migration Engine** — Live migration of workloads across providers/hardware

### Per-Node Agents
- **Health Monitor** — CPU/mem/disk/network, trend analysis
- **Metrics Collector** — Prometheus-compatible export, local buffer
- **Hardware Controller** — IPMI/BMC fan control, power monitoring ([[05 - AI Server Management & Cooling]])

## Data Flow
```
Agent → Local Buffer → Stream Processor → AI Engine → Decision → Action
         (5min)          (Kafka/NATS)      (infer)      (execute)
```

## Storage
| Data | Storage | Retention |
|------|---------|-----------|
| Metrics | VictoriaDB / InfluxDB | 30d raw, 1yr aggregated |
| Logs | Loki / ClickHouse | 7d hot, 90d cold |
| Predictions | SQLite / Postgres | Forever |
| Audit trail | Postgres / S3 | Forever |

## Infrastructure Model
- ==**Hybrid** (cloud + bare-metal under one management plane)==
- Cloud VMs: AWS, Azure, GCP — auto-discovered via API
- Bare-metal: IPMI/BMC for health, cooling, and power control
- Edge nodes: lightweight agent for low-resource environments

## LLM Strategy
- ==**Hybrid, user-configurable toggle**==
- Local inference (Llama/Mistral, ONNX quantized) for predictions, anomaly detection, basic ops
- Cloud API (OpenAI/Anthropic) for complex chat, multi-step reasoning, deep diagnostics
- Automatic fallback to local when offline

## Tech Stack Considerations
- **Language**: Go (agent) + Python (AI) + Rust (core)
- ==**Agent weight**: < 50MB RAM idle target==
- **Communication**: gRPC + NATS for real-time
- **AI models**: ONNX runtime for local inference, cloud API for complex tasks
- **Orchestration API**: Docker-compatible API surface for easy migration from Compose/K8s
