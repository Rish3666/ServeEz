// Package apiserver implements the ServeEz control-plane HTTP API.
// Endpoints follow the design in Orchestration/02 (API Design Principles)
// plus the agent contract in CODEX_AGENT.md:
//
//	POST /v1/nodes/register
//	POST /v1/nodes/{id}/report
//	GET  /v1/nodes/{id}/commands
//	POST /v1/nodes/{id}/commands/{action_id}/ack
//	GET  /v1/state
//	POST /v1/workloads
//	POST /v1/execute
//	POST /v1/simulate
//	GET  /v1/audit
//	POST /v1/emergency/kill
package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/history"
	"github.com/Rish3666/ServeEz/internal/mcp"
	"github.com/Rish3666/ServeEz/internal/orchestrator"
	"github.com/Rish3666/ServeEz/internal/predictor"
	"github.com/Rish3666/ServeEz/internal/simulate"
	"github.com/Rish3666/ServeEz/internal/state"
)

// Server is the HTTP API server.
type Server struct {
	store      state.Store
	reg        *state.Registry
	audit      audit.Log
	scheduler  *orchestrator.Scheduler
	simulator  *simulate.Engine
	predictor  *predictor.Engine
	hist       *history.Store
	mcp        *mcp.Server
	joinToken  string
	killSwitch bool

	mu       sync.Mutex
	pending  map[string]api.Action // action ID -> queued for a node
	nodeCmds map[string][]string   // nodeID -> pending action IDs
}

// New creates an API server.
func New(st state.Store, reg *state.Registry, al audit.Log, sched *orchestrator.Scheduler, joinToken string) *Server {
	return &Server{
		store:     st,
		reg:       reg,
		audit:     al,
		scheduler: sched,
		simulator: simulate.New(st),
		joinToken: joinToken,
		pending:   map[string]api.Action{},
		nodeCmds:  map[string][]string{},
	}
}

// WithHistory attaches the time-series store (used for forecasts) and enables
// /v1/predict.
func (s *Server) WithHistory(h *history.Store) *Server {
	s.hist = h
	s.predictor = predictor.New(h)
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/nodes/register", s.handleRegister)
	mux.HandleFunc("POST /v1/nodes/{node_id}/report", s.handleReport)
	mux.HandleFunc("GET /v1/nodes/{node_id}/commands", s.handleCommands)
	mux.HandleFunc("POST /v1/nodes/{node_id}/commands/{action_id}/ack", s.handleAck)
	mux.HandleFunc("GET /v1/state", s.handleState)
	mux.HandleFunc("POST /v1/workloads", s.handleCreateWorkload)
	mux.HandleFunc("POST /v1/execute", s.handleExecute)
	mux.HandleFunc("POST /v1/simulate", s.handleSimulate)
	mux.HandleFunc("GET /v1/audit", s.handleAudit)
	mux.HandleFunc("POST /v1/emergency/kill", s.handleKill)
	mux.HandleFunc("GET /v1/mcp/tools", s.handleMCPTools)
	mux.HandleFunc("POST /v1/mcp/call", s.handleMCPCall)
	mux.HandleFunc("GET /v1/predict", s.handlePredict)
	return withLogging(mux)
}

// ===== Node registration =====

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if s.killSwitch {
		writeError(w, http.StatusForbidden, "control is disabled (kill switch)")
		return
	}
	var req api.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid register request")
		return
	}
	if req.Token == "" || req.Token != s.joinToken {
		writeError(w, http.StatusUnauthorized, "invalid join token")
		return
	}
	if req.NodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id required")
		return
	}

	ctx := r.Context()
	obj := &api.Object{
		Kind: "Node", Namespace: "default", Name: req.NodeID,
		Spec: &api.NodeSpec{
			Provider: req.Provider,
			Runtime:  req.Runtime,
			Labels:   req.Labels,
			Capacity: req.Capacity,
		},
		Status: &api.NodeStatus{State: "pending", LastSeen: time.Now().UTC()},
	}
	if err := s.reg.Validate(obj); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := s.store.Get(ctx, "Node", "default", req.NodeID); errors.Is(err, state.ErrNotFound) {
		if _, err := s.store.Create(ctx, obj); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAppend(ctx, &api.AuditEntry{
		Initiator: "node:" + req.NodeID, ActionType: "node_register", Target: req.NodeID,
		Status: "completed",
	})
	writeJSON(w, http.StatusOK, api.RegisterResponse{Approved: true, NodeID: req.NodeID, ControlPlaneURL: ""})
}

// ===== State reports from agents =====

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	var rep api.NodeReport
	if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
		writeError(w, http.StatusBadRequest, "invalid report")
		return
	}
	rep.NodeID = nodeID

	ctx := r.Context()
	obj, err := s.store.Get(ctx, "Node", "default", nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not registered")
		return
	}
	ns, ok := obj.Spec.(*api.NodeSpec)
	if !ok || ns == nil {
		writeError(w, http.StatusInternalServerError, "corrupt node spec")
		return
	}
	status, ok := obj.Status.(*api.NodeStatus)
	if !ok || status == nil {
		status = &api.NodeStatus{}
	}
	status.State = rep.State
	status.HealthScore = rep.HealthScore
	status.Resources = rep.Usage
	status.Hardware = rep.Hardware
	status.Workloads = rep.Workloads
	status.LastSeen = time.Now().UTC()
	obj.Status = status

	if _, err := s.store.Update(ctx, obj); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Record utilization into the time-series store for the predictor.
	if s.hist != nil && rep.Usage.CPUPercent > 0 {
		_ = s.hist.Record(ctx, "node:"+nodeID, history.Sample{
			At:       time.Now().UTC(),
			CPU:      rep.Usage.CPUPercent,
			MemPct:   rep.Usage.MemPercent,
			MemBytes: 0,
		})
	}
	writeJSON(w, http.StatusOK, api.ReportAck{OK: true})
}

// ===== Command queue (control plane -> agent) =====

func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	s.mu.Lock()
	ids := append([]string(nil), s.nodeCmds[nodeID]...)
	s.nodeCmds[nodeID] = nil
	s.mu.Unlock()

	out := make([]api.Action, 0, len(ids))
	for _, id := range ids {
		if a, ok := s.pending[id]; ok {
			out = append(out, a)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAck(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	actionID := r.PathValue("action_id")
	var body struct {
		Status string `json:"status"`
		Result any    `json:"result,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	s.mu.Lock()
	a, ok := s.pending[actionID]
	delete(s.pending, actionID)
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "unknown action")
		return
	}
	s.auditAppend(r.Context(), &api.AuditEntry{
		Initiator: "node:" + nodeID, ActionType: a.Type, Target: a.Target,
		Parameters: a.Parameters, Status: body.Status, StateAfter: body.Result,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ===== State queries =====

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("type")
	objs, err := s.store.List(r.Context(), kind, "default")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"api_version": "v1",
		"timestamp":   time.Now().UTC(),
		"objects":     objs,
	})
}

// ===== Workload management =====

func (s *Server) handleCreateWorkload(w http.ResponseWriter, r *http.Request) {
	if s.killSwitch {
		writeError(w, http.StatusForbidden, "control is disabled (kill switch)")
		return
	}
	var req struct {
		Name   string             `json:"name"`
		Spec   *api.WorkloadSpec  `json:"spec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Spec == nil {
		writeError(w, http.StatusBadRequest, "name and spec required")
		return
	}
	ctx := r.Context()
	obj := &api.Object{
		Kind: "Workload", Namespace: "default", Name: req.Name,
		Spec:   req.Spec,
		Status: &api.WorkloadStatus{DesiredReplicas: req.Spec.Replicas, State: "declared"},
	}
	if err := s.reg.Validate(obj); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := s.store.Create(ctx, obj); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAppend(ctx, &api.AuditEntry{
		Initiator: "api", ActionType: "deploy", Target: req.Name,
		Parameters: map[string]any{"image": req.Spec.Image, "replicas": req.Spec.Replicas},
		Status:     "completed",
	})
	writeJSON(w, http.StatusCreated, obj)
}

// ===== Action execution =====

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if s.killSwitch {
		writeError(w, http.StatusForbidden, "control is disabled (kill switch)")
		return
	}
	var act api.Action
	if err := json.NewDecoder(r.Body).Decode(&act); err != nil {
		writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if err := s.applyConfidenceGate(act); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	// Scale actions resolve to a node and get queued for its agent.
	if act.Type == "scale" {
		if err := s.queueScale(r.Context(), &act); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	// Other action types (deploy, restart, ...) are queued to the workload's
	// assigned node once scheduling has placed it. For MVP, actions targeting
	// a specific node id are routed directly.
	if act.Target != "" && !strings.HasPrefix(act.Target, "workload:") {
		s.queueForNode(act.Target, act)
	}
	writeJSON(w, http.StatusOK, api.ActionResult{ID: "", Status: "queued", Action: act, Message: "action queued for agent"})
}

// applyConfidenceGate enforces per-action confidence thresholds (AI Control/05).
func (s *Server) applyConfidenceGate(act api.Action) error {
	thresholds := map[string]float64{
		"scale_up": 0.70,
		"scale":    0.70,
		"restart":  0.80,
		"migrate":  0.90,
		"kill":     0.95,
		"stop":     0.95,
	}
	th, ok := thresholds[act.Type]
	if !ok {
		return nil
	}
	if act.Confidence < th {
		return fmt.Errorf("confidence %.2f below threshold %.2f for action %q", act.Confidence, th, act.Type)
	}
	return nil
}

func (s *Server) queueScale(ctx context.Context, act *api.Action) error {
	target := act.Target
	if strings.HasPrefix(target, "workload:") {
		target = strings.TrimPrefix(target, "workload:")
	}
	obj, err := s.store.Get(ctx, "Workload", "default", target)
	if err != nil {
		return fmt.Errorf("workload %q not found", target)
	}
	wst, ok := obj.Status.(*api.WorkloadStatus)
	if !ok || wst.AssignedNode == "" {
		return fmt.Errorf("workload %q not yet scheduled", target)
	}
	act.Target = target
	s.queueForNode(wst.AssignedNode, *act)
	return nil
}

func (s *Server) queueForNode(nodeID string, act api.Action) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("act-%d", time.Now().UnixNano())
	s.pending[id] = act
	s.nodeCmds[nodeID] = append(s.nodeCmds[nodeID], id)
}

// ===== Prediction (AI Integration/01) =====

func (s *Server) handlePredict(w http.ResponseWriter, r *http.Request) {
	if s.predictor == nil {
		writeError(w, http.StatusNotFound, "predictor not enabled")
		return
	}
	workload := r.URL.Query().Get("workload")
	if workload == "" {
		writeError(w, http.StatusBadRequest, "workload query param required")
		return
	}
	ctx := r.Context()

	obj, err := s.store.Get(ctx, "Workload", "default", workload)
	if err != nil {
		writeError(w, http.StatusNotFound, "workload not found")
		return
	}
	ws, ok := obj.Spec.(*api.WorkloadSpec)
	if !ok || ws == nil {
		writeError(w, http.StatusInternalServerError, "corrupt workload spec")
		return
	}
	wst, ok := obj.Status.(*api.WorkloadStatus)
	if !ok || wst == nil || wst.AssignedNode == "" {
		writeJSON(w, http.StatusOK, api.PredictResponse{
			Workload: workload, Available: false, Reason: "workload not yet scheduled",
		})
		return
	}

	// Forecast the assigned node's CPU series. Fall back to 0 cpuNow so the
	// model uses the last recorded sample.
	cpuNow := 0.0
	if n, err := s.store.Get(ctx, "Node", "default", wst.AssignedNode); err == nil {
		if ns, ok := n.Status.(*api.NodeStatus); ok && ns != nil {
			cpuNow = ns.Resources.CPUPercent
		}
	}
	res, err := s.predictor.Predict(ctx, "node:"+wst.AssignedNode, workload, ws.Replicas, cpuNow)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ===== Simulation =====

func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	var req api.SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid simulate request")
		return
	}
	res := s.simulator.Simulate(r.Context(), req.Action)
	writeJSON(w, http.StatusOK, res)
}

// ===== Audit =====

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	f := audit.Filter{
		Initiator: r.URL.Query().Get("initiator"),
		Status:    r.URL.Query().Get("status"),
	}
	entries, err := s.audit.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// ===== Kill switch =====

func (s *Server) handleKill(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.killSwitch = true
	s.pending = map[string]api.Action{}
	s.nodeCmds = map[string][]string{}
	s.mu.Unlock()
	s.auditAppend(r.Context(), &api.AuditEntry{
		Initiator: "emergency", ActionType: "kill_switch", Target: "cluster", Status: "completed",
	})
	writeJSON(w, http.StatusOK, map[string]any{"killed": true})
}

// ===== helpers =====

func (s *Server) auditAppend(ctx context.Context, e *api.AuditEntry) {
	if s.audit == nil {
		return
	}
	if _, err := s.audit.Append(ctx, e); err != nil {
		slog.Warn("audit append failed", "err", err)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Debug("http", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start))
	})
}
