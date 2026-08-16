// Package mcp implements the ServeEz MCP (Model Context Protocol) server
// (Core Features/02 - MCP Interface + AI Control/02). It exposes the object
// store, orchestrator, and simulation sandbox as discoverable tools grouped
// into read / write / simulate / subscribe categories so AI agents can only
// do what their role permits.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/state"
)

// Category groups tools by permission level.
type Category string

const (
	CatRead      Category = "read"
	CatWrite     Category = "write"
	CatSimulate  Category = "simulate"
	CatSubscribe Category = "subscribe"
)

// Tool is a discoverable MCP operation.
type Tool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    Category          `json:"category"`
	Parameters  map[string]string `json:"parameters"`
	// Handler runs the tool with the raw parameter map.
	Handler func(ctx context.Context, params map[string]any) (any, error) `json:"-"`
}

// Server is the MCP tool registry + handler.
type Server struct {
	mu    sync.RWMutex
	tools []*Tool
}

// New creates an MCP server wired to the given store/log.
func New(st state.Store, al audit.Log, sim Simulator) *Server {
	return NewWithPredictor(st, al, sim, nil)
}

// NewWithPredictor is New plus an optional predictor engine for the
// predict.scale tool.
func NewWithPredictor(st state.Store, al audit.Log, sim Simulator, pred Predictor) *Server {
	s := &Server{}
	s.register(readTools(st)...)
	s.register(simulateTools(sim)...)
	s.register(writeTools(st)...)
	s.register(subscribeTools(st)...)
	s.register(auditTools(al)...)
	if pred != nil {
		s.register(predictTools(pred)...)
	}
	return s
}

// Simulator is the dry-run surface the MCP expose.
type Simulator interface {
	Simulate(ctx context.Context, act api.Action) api.SimulationResult
}

// Predictor is the forecast surface the MCP expose.
type Predictor interface {
	Predict(ctx context.Context, workload string) (api.PredictResponse, error)
}

func (s *Server) register(ts ...*Tool) {
	s.tools = append(s.tools, ts...)
}

// Tools returns all registered tools.
func (s *Server) Tools() []*Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Tool, len(s.tools))
	copy(out, s.tools)
	return out
}

// ToolsByCategory filters tools by category.
func (s *Server) ToolsByCategory(cat Category) []*Tool {
	var out []*Tool
	for _, t := range s.Tools() {
		if t.Category == cat {
			out = append(out, t)
		}
	}
	return out
}

// Call dispatches a tool by name. Returns ErrUnknownTool if not found.
func (s *Server) Call(ctx context.Context, name string, params map[string]any) (any, error) {
	s.mu.RLock()
	for _, t := range s.tools {
		if t.Name == name {
			fn := t.Handler
			s.mu.RUnlock()
			return fn(ctx, params)
		}
	}
	s.mu.RUnlock()
	return nil, ErrUnknownTool(name)
}

// ErrUnknownTool is returned when a tool name has no handler.
func ErrUnknownTool(name string) error {
	return fmt.Errorf("mcp: unknown tool %q", name)
}

// ===== Tool definitions =====

func readTools(st state.Store) []*Tool {
	return []*Tool{
		{
			Name:        "state.list",
			Description: "List objects in the store by type (Node, Workload, ...).",
			Category:    CatRead,
			Parameters:  map[string]string{"type": "object kind to filter by, optional"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				kind, _ := p["type"].(string)
				return st.List(ctx, kind, "")
			},
		},
		{
			Name:        "state.get",
			Description: "Get a single object by type and name.",
			Category:    CatRead,
			Parameters:  map[string]string{"type": "object kind", "name": "object name"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				kind, _ := p["type"].(string)
				name, _ := p["name"].(string)
				if kind == "" || name == "" {
					return nil, fmt.Errorf("mcp: state.get requires type and name")
				}
				return st.Get(ctx, kind, "default", name)
			},
		},
	}
}

func simulateTools(sim Simulator) []*Tool {
	return []*Tool{
		{
			Name:        "simulate.action",
			Description: "Dry-run an action before it executes (Tier 1/2 sandbox).",
			Category:    CatSimulate,
			Parameters:  map[string]string{"action": "JSON-encoded api.Action"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				var act api.Action
				switch v := p["action"].(type) {
				case string:
					if err := json.Unmarshal([]byte(v), &act); err != nil {
						return nil, fmt.Errorf("mcp: bad action JSON: %w", err)
					}
				case map[string]any:
					b, _ := json.Marshal(v)
					if err := json.Unmarshal(b, &act); err != nil {
						return nil, fmt.Errorf("mcp: bad action: %w", err)
					}
				default:
					return nil, fmt.Errorf("mcp: simulate.action requires an action")
				}
				return sim.Simulate(ctx, act), nil
			},
		},
	}
}

func predictTools(pred Predictor) []*Tool {
	return []*Tool{
		{
			Name:        "predict.scale",
			Description: "Forecast a workload's CPU trend and recommend a replica count.",
			Category:    CatSimulate,
			Parameters:  map[string]string{"workload": "workload name"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				name, _ := p["workload"].(string)
				if name == "" {
					return nil, fmt.Errorf("mcp: predict.scale requires a workload")
				}
				return pred.Predict(ctx, name)
			},
		},
	}
}

func writeTools(st state.Store) []*Tool {
	return []*Tool{
		{
			Name:        "workload.create",
			Description: "Create a workload object (staged, not deployed to a node).",
			Category:    CatWrite,
			Parameters:  map[string]string{"name": "workload name", "spec": "JSON-encoded api.WorkloadSpec"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				name, _ := p["name"].(string)
				specRaw, _ := p["spec"].(string)
				if name == "" || specRaw == "" {
					return nil, fmt.Errorf("mcp: workload.create requires name and spec")
				}
				var spec api.WorkloadSpec
				if err := json.Unmarshal([]byte(specRaw), &spec); err != nil {
					return nil, fmt.Errorf("mcp: bad spec: %w", err)
				}
				obj := &api.Object{
					Kind: "Workload", Namespace: "default", Name: name,
					Spec:   &spec,
					Status: &api.WorkloadStatus{DesiredReplicas: spec.Replicas, State: "declared"},
				}
				return st.Create(ctx, obj)
			},
		},
	}
}

func subscribeTools(st state.Store) []*Tool {
	return []*Tool{
		{
			Name:        "state.subscribe",
			Description: "Watch for changes to objects of a type.",
			Category:    CatSubscribe,
			Parameters:  map[string]string{"type": "object kind"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				kind, _ := p["type"].(string)
				evs, err := st.Watch(ctx)
				if err != nil {
					return nil, err
				}
				select {
				case <-ctx.Done():
					return map[string]any{"stopped": true}, nil
				case ev := <-evs:
					if kind != "" && ev.Kind != kind {
						return map[string]any{"waiting": true}, nil
					}
					return ev, nil
				case <-time.After(5 * time.Second):
					return map[string]any{"timeout": true}, nil
				}
			},
		},
	}
}

func auditTools(al audit.Log) []*Tool {
	return []*Tool{
		{
			Name:        "audit.query",
			Description: "Query the append-only audit log.",
			Category:    CatRead,
			Parameters:  map[string]string{"initiator": "optional", "status": "optional", "limit": "optional int"},
			Handler: func(ctx context.Context, p map[string]any) (any, error) {
				limit := 50
				if l, ok := p["limit"].(float64); ok {
					limit = int(l)
				}
				initiator, _ := p["initiator"].(string)
				status, _ := p["status"].(string)
				return al.List(ctx, audit.Filter{
					Initiator: initiator,
					Status:    status,
					Limit:     limit,
				})
			},
		},
	}
}
