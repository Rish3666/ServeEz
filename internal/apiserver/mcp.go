package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/mcp"
)

// Simulate implements mcp.Simulator so the MCP server shares the same
// simulation engine as /v1/simulate.
func (s *Server) Simulate(ctx context.Context, act api.Action) api.SimulationResult {
	return s.simulator.Simulate(ctx, act)
}

// Predict implements mcp.Predictor, mirroring /v1/predict.
func (s *Server) Predict(ctx context.Context, workload string) (api.PredictResponse, error) {
	if s.predictor == nil {
		return api.PredictResponse{}, fmt.Errorf("predictor not enabled")
	}
	obj, err := s.store.Get(ctx, "Workload", "default", workload)
	if err != nil {
		return api.PredictResponse{}, err
	}
	ws, ok := obj.Spec.(*api.WorkloadSpec)
	if !ok || ws == nil {
		return api.PredictResponse{}, fmt.Errorf("corrupt workload spec")
	}
	wst, ok := obj.Status.(*api.WorkloadStatus)
	if !ok || wst == nil || wst.AssignedNode == "" {
		return api.PredictResponse{Workload: workload, Available: false, Reason: "not yet scheduled"}, nil
	}
	cpuNow := 0.0
	if n, err := s.store.Get(ctx, "Node", "default", wst.AssignedNode); err == nil {
		if ns, ok := n.Status.(*api.NodeStatus); ok && ns != nil {
			cpuNow = ns.Resources.CPUPercent
		}
	}
	return s.predictor.Predict(ctx, "node:"+wst.AssignedNode, workload, ws.Replicas, cpuNow)
}

// MCP HTTP surface. Exposes tool discovery and invocation as JSON endpoints
// so AI agents (and the servez chat command) can drive the control plane
// without direct object-store access. Wire via Server.WithMCP.

// WithMCP attaches an MCP server to the API server, enabling /v1/mcp routes.
func (s *Server) WithMCP(m *mcp.Server) {
	s.mcp = m
}

func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	if s.mcp == nil {
		writeError(w, http.StatusNotFound, "mcp not enabled")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tools": s.mcp.Tools()})
}

func (s *Server) handleMCPCall(w http.ResponseWriter, r *http.Request) {
	if s.mcp == nil {
		writeError(w, http.StatusNotFound, "mcp not enabled")
		return
	}
	var req struct {
		Tool   string         `json:"tool"`
		Params map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Tool == "" {
		writeError(w, http.StatusBadRequest, "tool required")
		return
	}
	out, err := s.mcp.Call(r.Context(), req.Tool, req.Params)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": out})
}
