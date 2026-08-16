// Package api defines the shared wire types used across ServeEz:
// node agent -> control plane communication, the object store, and the
// external API surface. This is the single source of truth for the
// ServeEz data model. Do not add runtime logic here — pure types only.
package api

import (
	"context"
	"time"
)

// ===== Object Store =====

// ResourceVersion is the optimistic-concurrency token attached to every
// stored object. Concurrent writers must pass the version they based their
// change on; the store rejects stale writes.
type ResourceVersion string

// Object is the envelope stored in the object store. Every stored object
// (Node, Workload, Service, ...) is wrapped in this envelope so the store
// can manage versioning, schema, and timestamps generically.
type Object struct {
	// Kind is the object type, e.g. "Node", "Workload", "Service".
	Kind string `json:"kind"`

	// SchemaVersion is the version of the Kind's JSON schema this object
	// conforms to (CRD-equivalent schema registration).
	SchemaVersion string `json:"schema_version"`

	// Name is the unique, stable name of the object.
	Name string `json:"name"`

	// Namespace is a grouping label; defaults to "default".
	Namespace string `json:"namespace"`

	// ResourceVersion is the optimistic concurrency token.
	ResourceVersion ResourceVersion `json:"resource_version"`

	// Spec holds the desired state (typed per Kind).
	Spec any `json:"spec"`

	// Status holds the observed state (typed per Kind).
	Status any `json:"status,omitempty"`

	// CreatedAt / UpdatedAt for audit and ordering.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ===== Node =====

// NodeSpec is the desired state of a cluster member.
type NodeSpec struct {
	// Provider is "local", "aws", "azure", "gcp", or "bare_metal".
	Provider string `json:"provider"`

	// Runtime is the OCI runtime the agent manages: "docker", "containerd", "runc".
	Runtime string `json:"runtime"`

	// Labels are free-form key/value metadata for scheduling.
	Labels map[string]string `json:"labels,omitempty"`

	// Capacity is the total resources of the node.
	Capacity Resources `json:"capacity,omitempty"`
}

// Resources describes a resource footprint.
type Resources struct {
	CPUCores  float64 `json:"cpu_cores"`
	MemBytes  uint64  `json:"mem_bytes"`
	DiskBytes uint64  `json:"disk_bytes"`
}

// NodeStatus is the observed state of a cluster member.
type NodeStatus struct {
	// State is one of: "pending", "healthy", "degraded", "unhealthy", "cordoned", "disconnected".
	State string `json:"state"`

	// HealthScore is 0-100, computed by the agent / control plane.
	HealthScore int `json:"health_score"`

	// Resources is the current utilization snapshot.
	Resources Usage `json:"resources,omitempty"`

	// Hardware reports temperature/fan/power when available.
	Hardware *HardwareInfo `json:"hardware,omitempty"`

	// Workloads lists container instances running on this node.
	Workloads []ContainerStatus `json:"workloads,omitempty"`

	// LastSeen is the timestamp of the last state report from the agent.
	LastSeen time.Time `json:"last_seen"`

	// Prediction holds failure probability computed by the AI engine (Phase 1+).
	Prediction *Prediction `json:"prediction,omitempty"`
}

// Usage is a utilization snapshot.
type Usage struct {
	CPUPercent  float64 `json:"cpu_pct"`
	MemPercent  float64 `json:"mem_pct"`
	DiskPercent float64 `json:"disk_pct"`
	// Network bytes per second (rx/tx).
	NetRxBps uint64 `json:"net_rx_bps"`
	NetTxBps uint64 `json:"net_tx_bps"`
}

// HardwareInfo is present for bare-metal / IPMI-capable nodes.
type HardwareInfo struct {
	TempCPU float64 `json:"temp_cpu"`
	TempGPU float64 `json:"temp_gpu,omitempty"`
	FanPct  float64 `json:"fan_pct,omitempty"`
	PowerW  float64 `json:"power_w,omitempty"`
}

// Prediction is a forward-looking health signal (populated by the AI engine).
type Prediction struct {
	FailureProbability24h float64 `json:"failure_probability_24h"`
	PredictedIssue        string  `json:"predicted_issue,omitempty"`
}

// ===== Workload / Container =====

// WorkloadSpec is the desired state of a workload. It mirrors the "Workload"
// primitive from the container lifecycle design.
type WorkloadSpec struct {
	// Image is the OCI image reference.
	Image string `json:"image"`

	// Replicas is the desired number of container instances.
	Replicas int `json:"replicas"`

	// Resources is the per-instance resource request.
	Resources *Resources `json:"resources,omitempty"`

	// Ports maps container ports to exposed protocols.
	Ports []Port `json:"ports,omitempty"`

	// Env are environment variables.
	Env map[string]string `json:"env,omitempty"`

	// Type is "service", "stateful", "batch", or "agent_session".
	Type string `json:"type"`

	// RestartPolicy is "always", "on_failure", or "never".
	RestartPolicy string `json:"restart_policy"`

	// Strategy is "rolling", "blue_green", or "recreate".
	Strategy string `json:"strategy"`

	// Probes define liveness/readiness/startup checks.
	Probes *Probes `json:"probes,omitempty"`
}

// Port maps a container port to a protocol.
type Port struct {
	Container int    `json:"container"`
	Protocol  string `json:"protocol"` // "tcp", "udp"
}

// Probes define health checks.
type Probes struct {
	Liveness  *Probe `json:"liveness,omitempty"`
	Readiness *Probe `json:"readiness,omitempty"`
	Startup   *Probe `json:"startup,omitempty"`
}

// Probe is a single health check definition.
type Probe struct {
	// Path for HTTP probes, or Command for exec probes.
	Path    string   `json:"path,omitempty"`
	Command []string `json:"command,omitempty"`
	// Port is the port to probe (HTTP).
	Port int `json:"port,omitempty"`
	// InitialDelaySeconds, PeriodSeconds, TimeoutSeconds.
	InitialDelaySeconds int `json:"initial_delay_seconds,omitempty"`
	PeriodSeconds       int `json:"period_seconds,omitempty"`
	TimeoutSeconds      int `json:"timeout_seconds,omitempty"`
}

// WorkloadStatus is the observed state of a workload.
type WorkloadStatus struct {
	// State is "declared", "validated", "scheduled", "running", "degraded", "unhealthy", "terminated".
	State string `json:"state"`

	// Instances lists the concrete container instances.
	Instances []ContainerStatus `json:"instances,omitempty"`

	// DesiredReplicas / RunningReplicas.
	DesiredReplicas int `json:"desired_replicas"`
	RunningReplicas int `json:"running_replicas"`

	// AssignedNode for single-replica placement (or the primary node).
	AssignedNode string `json:"assigned_node,omitempty"`

	// Error captures the last failure reason, if any.
	Error string `json:"error,omitempty"`
}

// ContainerStatus is the observed state of a single container instance.
type ContainerStatus struct {
	// ID is the runtime container ID.
	ID string `json:"id"`

	// Name is the instance name, e.g. "web-1".
	Name string `json:"name"`

	// Image is the running image.
	Image string `json:"image"`

	// State is "running", "restarting", "exited", "paused", "created".
	State string `json:"state"`

	// Health is "healthy", "unhealthy", "unknown".
	Health string `json:"health"`

	// Usage is the instance's current resource usage.
	Usage *Usage `json:"usage,omitempty"`

	// NodeID is where this instance is running.
	NodeID string `json:"node_id,omitempty"`
}

// ===== Node Agent -> Control Plane =====

// RegisterRequest is sent by the agent to join the cluster.
type RegisterRequest struct {
	// NodeID is a stable identifier generated by the agent.
	NodeID string `json:"node_id"`

	// Token is the join token issued by `servez token` or `servez init`.
	Token string `json:"token"`

	// Version is the agent version, for compatibility checks.
	Version string `json:"version"`

	// Runtime is "docker", "containerd", or "runc".
	Runtime string `json:"runtime"`

	// Provider is "local", "aws", "azure", "gcp", "bare_metal".
	Provider string `json:"provider"`

	// Capacity describes total node resources.
	Capacity Resources `json:"capacity"`

	// Labels are optional node metadata.
	Labels map[string]string `json:"labels,omitempty"`
}

// RegisterResponse is returned by the control plane after registration.
type RegisterResponse struct {
	// Approved is true if the node may join.
	Approved bool `json:"approved"`

	// NodeID is the confirmed node identity.
	NodeID string `json:"node_id"`

	// Reason explains a rejection.
	Reason string `json:"reason,omitempty"`

	// ControlPlaneURL tells the agent where to report state.
	ControlPlaneURL string `json:"control_plane_url,omitempty"`
}

// NodeReport is the periodic state report sent by the agent (every 10s or on change).
type NodeReport struct {
	NodeID      string            `json:"node_id"`
	State       string            `json:"state"`
	HealthScore int               `json:"health_score"`
	Usage       Usage             `json:"usage"`
	Hardware    *HardwareInfo     `json:"hardware,omitempty"`
	Workloads   []ContainerStatus `json:"workloads,omitempty"`
	ReportedAt  time.Time         `json:"reported_at"`
}

// ReportAck acknowledges a state report.
type ReportAck struct {
	// OK is true when the report was accepted.
	OK bool `json:"ok"`

	// Message may carry a reason on failure.
	Message string `json:"message,omitempty"`
}

// ===== Actions / Execution =====

// Action is a concrete operation request. Mirrors the MCP write tools.
type Action struct {
	// Type is one of: "start", "stop", "restart", "scale", "remove", "deploy", "replace", "migrate", "kill".
	// "replace" performs a blue-green warm-clone swap (zero-downtime); see prewarm.Swapper.
	Type string `json:"type"`

	// Target identifies the workload or container.
	Target string `json:"target"`

	// Parameters are action-specific arguments.
	Parameters map[string]any `json:"parameters,omitempty"`

	// Reason is why this action is being taken (audit + AI context).
	Reason string `json:"reason,omitempty"`

	// Initiator identifies who requested the action ("human", "ai-agent:predictor-v2", ...).
	Initiator string `json:"initiator"`

	// Confidence is the AI's confidence in this action (0-1).
	Confidence float64 `json:"confidence,omitempty"`
}

// ActionResult is the structured outcome of an action.
type ActionResult struct {
	// ID is the audit ID for this action.
	ID string `json:"id"`

	// Status is "completed", "failed", "queued", "rejected", "requires_approval".
	Status string `json:"status"`

	// Action echoes the requested action.
	Action Action `json:"action"`

	// Before / After hold the relevant state snapshots (opaque, typed per action).
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`

	// Message is a human-readable summary.
	Message string `json:"message,omitempty"`

	// DurationMS is execution time.
	DurationMS int64 `json:"duration_ms,omitempty"`
}

// ===== Audit =====

// AuditEntry is an immutable record of every action.
type AuditEntry struct {
	ID          string         `json:"id"`
	Timestamp   time.Time      `json:"timestamp"`
	Initiator   string         `json:"initiator"`
	ActionType  string         `json:"action_type"`
	Target      string         `json:"target"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	StateBefore any            `json:"state_before,omitempty"`
	StateAfter  any            `json:"state_after,omitempty"`
	Status      string         `json:"status"`
	Confidence  float64        `json:"confidence,omitempty"`
	DurationMS  int64          `json:"duration_ms,omitempty"`
}

// ===== Simulation =====

// SimulationRequest is a dry-run of an action.
type SimulationRequest struct {
	Action       Action `json:"action"`
	SimulateOnly bool   `json:"simulate_only"`
}

// SimulationResult is the outcome of a dry-run.
type SimulationResult struct {
	ID               string         `json:"id"`
	RiskScore        float64        `json:"risk_score"`
	Confidence       float64        `json:"confidence"`
	Predicted        map[string]any `json:"predicted_outcomes,omitempty"`
	FailureScenarios []Scenario     `json:"failure_scenarios,omitempty"`
	Recommendation   string         `json:"recommendation"`
}

// Scenario is a predicted failure mode.
type Scenario struct {
	Scenario    string  `json:"scenario"`
	Impact      string  `json:"impact"`
	Probability float64 `json:"probability"`
}

// ===== Prediction (AI Integration/01) =====

// PredictResponse is the forecast + scale recommendation for a workload,
// produced by the fast-tier statistical model (internal/predictor) and
// consumed by the autoscale control loop (cmd/servez-autoscale).
type PredictResponse struct {
	Workload            string  `json:"workload"`
	Available           bool    `json:"available"`
	CurrentReplicas     int     `json:"current_replicas"`
	RecommendedReplicas int     `json:"recommended_replicas"`
	CurrentCPUPercent   float64 `json:"current_cpu_pct"`
	Forecast15mPercent  float64 `json:"forecast_15m_pct"`
	Forecast1hPercent   float64 `json:"forecast_1h_pct"`
	Confidence          float64 `json:"confidence"`
	Recommendation      string  `json:"recommendation"`
	Reason              string  `json:"reason"`
}

// ===== Cost Comparison (Core Features/03, AI Integration/04) =====

// CostComparer is implemented by the pricing engine (internal/cost) and
// consumed by the control plane's /v1/cost/compare route. Kept structural so
// internal/cost can satisfy it without importing apiserver or mcp.
type CostComparer interface {
	Compare(ctx context.Context, req CostCompareRequest) (CostReport, error)
}

// CostCompareRequest is the workload shape to price out across providers.
type CostCompareRequest struct {
	// VCPU is the requested CPU cores (required).
	VCPU int `json:"vcpu"`
	// MemGB is the requested memory in GB (required).
	MemGB int `json:"mem_gb"`
	// RuntimePct is the % of the month the instance runs (default 100).
	RuntimePct float64 `json:"runtime_pct,omitempty"`
	// Region defaults to "us-east-1".
	Region string `json:"region,omitempty"`
}

// CostReport is the comparison result across providers.
type CostReport struct {
	Request CostCompareRequest `json:"request"`
	// Best is the overall cheapest recommendation across all providers.
	Best *CostRecommendation `json:"best,omitempty"`
	// Providers is one recommendation per provider, cheapest-to-expensive.
	Providers []CostRecommendation `json:"providers,omitempty"`
	// PotentialSavingsPct is best vs most-expensive provider monthly spend.
	PotentialSavingsPct float64 `json:"potential_savings_pct"`
}

// CostRecommendation is the cheapest matching instance offer for one provider.
type CostRecommendation struct {
	Provider      string  `json:"provider"` // "aws" | "azure" | "gcp"
	InstanceType  string  `json:"instance_type"`
	Region        string  `json:"region"`
	VCPU          int     `json:"vcpu"`
	MemGB         int     `json:"mem_gb"`
	OnDemandPerMo float64 `json:"on_demand_per_mo"`
	SpotPerMo     float64 `json:"spot_per_mo"`
	Recommended   string  `json:"recommended"` // "spot" | "on_demand"
	EstMonthly    float64 `json:"est_monthly"`
}
