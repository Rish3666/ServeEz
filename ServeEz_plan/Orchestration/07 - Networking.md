---
tags:
  - orchestration
  - networking
status: draft
priority: high
---

# Networking

## Philosophy
Networking should ==just work==. No CNI plugins to install, no kube-proxy to configure, no flannel/calico/weave to decide between. For small clusters, zero-config. For large clusters, API-driven.

## Built-In Service Discovery

### Small Clusters (1-10 nodes)
- ==mDNS (Multicast DNS)== — Services auto-discover each other
- No external DNS or service mesh required
- Works on LAN, homelab, single cloud VPC

### Medium Clusters (10-50 nodes)
- Embedded DNS server (control plane)
- Service names resolve automatically
- Round-robin across healthy replicas

### Large Clusters (50+ nodes)
- Integration with external DNS (Route53, CloudDNS)
- Optional: service mesh integration (Istio, Linkerd)
- All configured via API, not YAML

## Traffic Routing

### Internal
```
web-service:80
  ↓ (resolved via mDNS / embedded DNS)
node-3:30001, node-7:30002, node-12:30005
  ↓ (random or least-loaded)
healthy replica
```

### External
```
api.serveez.example.com
  ↓ (configurable)
Load balancer (cloud LB or HAProxy)
  ↓
Service port mapping
  ↓
Replica
```

## Network Policies
Defined via API, not complex YAML:
```
POST /v1/network/policy
{
  "name": "db-isolation",
  "from": ["api-service", "worker-service"],
  "to": ["db-service"],
  "ports": ["5432"]
}
```

## AI-Managed Networking
- AI continuously monitors latency between nodes
- AI adjusts routing when a node becomes slow
- AI can isolate compromised workloads automatically
- AI detects and blocks anomalous traffic patterns

## Related
- [[04 - Node Agent]] — Applies network rules on each node
- [[AI Control/02 - MCP Tool Interface]] — Network config as MCP tools
