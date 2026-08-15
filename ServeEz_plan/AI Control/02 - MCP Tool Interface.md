---
tags:
  - ai-control
  - mcp
  - interface
status: draft
priority: critical
---

# MCP Tool Interface

## Overview
Every operation in ServeEz is an ==MCP (Model Context Protocol) tool==. The AI discovers available tools dynamically, reads their schemas, and calls them with structured arguments. No hardcoded API knowledge needed.

## Why MCP Over REST/gRPC Alone?
| Feature | MCP | REST |
|---------|-----|------|
| Dynamic discovery | ✅ Tools list with schemas | ❌ Requires docs |
| Type-safe schemas | ✅ JSON Schema per tool | ✅ Possible but not standard |
| Tool categorization | ✅ Read/Write/Simulate/Subscribe | ❌ Manual |
| Streaming results | ✅ SSE | ✅ Possible |
| Human-in-the-loop | ✅ Built-in approval gates | ❌ Custom |

## Tool Categories

### Read Tools (No side effects)
```json
{
  "name": "get_cluster_state",
  "description": "Returns current cluster state for AI analysis",
  "inputSchema": {
    "type": "object",
    "properties": {
      "scope": { "type": "string", "enum": ["all", "nodes", "services", "agents"] },
      "filter": { "type": "string", "description": "Optional label filter" }
    }
  },
  "category": "read"
}
```

### Write Tools (Changes state)
```json
{
  "name": "scale_service",
  "description": "Change replica count for a service",
  "inputSchema": {
    "type": "object",
    "properties": {
      "service": { "type": "string" },
      "replicas": { "type": "integer", "minimum": 0 },
      "reason": { "type": "string", "description": "Why this change is needed" }
    },
    "required": ["service", "replicas", "reason"]
  },
  "category": "write",
  "confidence_required": 0.85
}
```

### Simulate Tools (Dry-run)
```json
{
  "name": "simulate_scale_service",
  "description": "Preview impact of scaling before executing",
  "inputSchema": {
    "type": "object",
    "properties": {
      "service": { "type": "string" },
      "replicas": { "type": "integer" }
    },
    "required": ["service", "replicas"]
  },
  "category": "simulate"
}
```

### Subscribe Tools (Event streams)
```json
{
  "name": "subscribe_node_events",
  "description": "Stream node health events in real-time",
  "inputSchema": {
    "type": "object",
    "properties": {
      "nodes": { "type": "array", "items": { "type": "string" } },
      "event_types": { "type": "array", "items": { "type": "string" } }
    }
  },
  "category": "subscribe"
}
```

## Tool Discovery Flow
```
AI: "What can you do?"
MCP Server: { tools: [list of all tools with schemas] }
AI: "I need to scale web-service. Matching tool: scale_service"
AI: "Let me simulate first."
MCP Server: { simulated_impact: { risk: "low", cost: "+$0.12/hr" } }
AI: "Safe to proceed. Executing."
MCP Server: { status: "completed", before: {...}, after: {...} }
```

## Tool Registry
- Every tool has an owner (which component registers it)
- Tools are versioned
- Deprecated tools return warnings
- Custom tools from plugins are auto-discovered

## Object Store Backing

MCP tools don't read and write files — they ==read and write directly to the typed object store== (Postgres/etcd-style, JSON Schema-validated). Every tool call is a transaction against stored objects, so what the AI reads, what it writes, and what is persisted are always the same thing.

- **Optimistic concurrency.** Each object carries a ==resourceVersion-style token==. Concurrent writes from multiple agents or humans include the version they based their change on; the store rejects stale writes with a conflict response so the writer can re-read and retry. No last-writer-wins blind overwrites.
- **Schema registration (CRD-equivalent).** New object types are added via ==schema registration==, not core code changes. Third-party tools and plugins register their own object types and validation rules, extending the type system without touching the core — the store, the MCP tool layer, and the state model pick them up automatically.

← [[Index]]
