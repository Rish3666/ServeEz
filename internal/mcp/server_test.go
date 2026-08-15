package mcp

import (
	"context"
	"testing"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/state"
)

type fakeSim struct{}

func (fakeSim) Simulate(ctx context.Context, act api.Action) api.SimulationResult {
	return api.SimulationResult{RiskScore: 0.9, Confidence: 0.9, Recommendation: "requires_approval"}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	st := state.NewMemStore()
	al, err := audit.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = al.Close() })
	return New(st, al, fakeSim{})
}

func TestToolDiscovery(t *testing.T) {
	s := newTestServer(t)
	if len(s.Tools()) == 0 {
		t.Fatal("expected tools to be registered")
	}
	if len(s.ToolsByCategory(CatRead)) == 0 {
		t.Fatal("expected read tools")
	}
	if len(s.ToolsByCategory(CatWrite)) == 0 {
		t.Fatal("expected write tools")
	}
	if len(s.ToolsByCategory(CatSimulate)) == 0 {
		t.Fatal("expected simulate tools")
	}
}

func TestCallStateGet(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	_, err := s.Call(ctx, "state.get", map[string]any{"type": "Node", "name": "nope"})
	if err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestCallUnknownTool(t *testing.T) {
	s := newTestServer(t)
	_, err := s.Call(context.Background(), "state.nope", nil)
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestCallSimulate(t *testing.T) {
	s := newTestServer(t)
	res, err := s.Call(context.Background(), "simulate.action", map[string]any{
		"action": `{"type":"scale","target":"web"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	sr, ok := res.(api.SimulationResult)
	if !ok {
		t.Fatalf("expected SimulationResult, got %T", res)
	}
	if sr.Recommendation != "requires_approval" {
		t.Fatalf("expected requires_approval, got %q", sr.Recommendation)
	}
}
