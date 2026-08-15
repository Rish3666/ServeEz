package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/orchestrator"
	"github.com/Rish3666/ServeEz/internal/state"
)

func newTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	reg := state.NewRegistry()
	_ = reg.Register(state.Schema{
		Kind: "Node", Version: "v1",
		NewSpec:   func() any { return &api.NodeSpec{} },
		NewStatus: func() any { return &api.NodeStatus{} },
	})
	_ = reg.Register(state.Schema{
		Kind: "Workload", Version: "v1",
		NewSpec:   func() any { return &api.WorkloadSpec{} },
		NewStatus: func() any { return &api.WorkloadStatus{} },
	})
	st := state.NewMemStoreWithRegistry(reg)
	al, err := audit.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	t.Cleanup(func() { _ = al.Close() })

	sched := orchestrator.NewScheduler(st)
	reconciler := orchestrator.NewReconciler(st, sched)
	rctx, rcancel := context.WithCancel(context.Background())
	go func() { _ = reconciler.Run(rctx) }()
	t.Cleanup(rcancel)
	<-reconciler.Ready

	srv := New(st, reg, al, sched, "test-token")
	return srv, srv.Handler()
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRegisterRejectsBadToken(t *testing.T) {
	_, h := newTestServer(t)
	rec := doJSON(t, h, "POST", "/v1/nodes/register", api.RegisterRequest{NodeID: "n1", Token: "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestFullFlow(t *testing.T) {
	_, h := newTestServer(t)

	// Register a node.
	regReq := api.RegisterRequest{
		NodeID: "node-1", Token: "test-token", Runtime: "docker", Provider: "local",
		Capacity: api.Resources{CPUCores: 4, MemBytes: 16 << 30},
	}
	rec := doJSON(t, h, "POST", "/v1/nodes/register", regReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}

	// Report state.
	report := api.NodeReport{
		NodeID: "node-1", State: "healthy", HealthScore: 95,
		Usage: api.Usage{CPUPercent: 20, MemPercent: 30, DiskPercent: 10},
	}
	rec = doJSON(t, h, "POST", "/v1/nodes/node-1/report", report)
	if rec.Code != http.StatusOK {
		t.Fatalf("report: %d %s", rec.Code, rec.Body.String())
	}

	// Read state.
	rec = doJSON(t, h, "GET", "/v1/state?type=Node", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("state: %d", rec.Code)
	}
	var stateResp struct {
		Objects []*api.Object `json:"objects"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&stateResp)
	if len(stateResp.Objects) != 1 {
		t.Fatalf("expected 1 node, got %d", len(stateResp.Objects))
	}
	ns, ok := stateResp.Objects[0].Spec.(map[string]any)
	if !ok || ns["runtime"] != "docker" {
		t.Fatalf("unexpected node spec: %#v", stateResp.Objects[0].Spec)
	}

	// Create a workload; reconcile (run synchronously) assigns a node.
	rec = doJSON(t, h, "POST", "/v1/workloads", map[string]any{
		"name": "web",
		"spec": map[string]any{
			"image": "nginx:1.25", "replicas": 1, "type": "service",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create workload: %d %s", rec.Code, rec.Body.String())
	}

	// Simulate a scale action to confirm confidence gate + queuing.
	rec = doJSON(t, h, "POST", "/v1/execute", api.Action{
		Type: "scale", Target: "workload:web", Reason: "test", Initiator: "human",
		Confidence: 0.5,
	})
	if rec.Code == http.StatusOK {
		t.Fatal("expected confidence gate rejection")
	}

	rec = doJSON(t, h, "POST", "/v1/execute", api.Action{
		Type: "scale", Target: "workload:web", Reason: "test", Initiator: "human",
		Confidence: 0.9,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("scale: %d %s", rec.Code, rec.Body.String())
	}

	// Kill switch blocks subsequent writes.
	rec = doJSON(t, h, "POST", "/v1/emergency/kill", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("kill: %d", rec.Code)
	}
	rec = doJSON(t, h, "POST", "/v1/execute", api.Action{Type: "scale", Target: "x"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 after kill switch, got %d", rec.Code)
	}

	// Audit trail has entries.
	rec = doJSON(t, h, "GET", "/v1/audit", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit: %d", rec.Code)
	}
	var auditResp struct {
		Entries []*api.AuditEntry `json:"entries"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&auditResp)
	if len(auditResp.Entries) == 0 {
		t.Fatal("expected audit entries")
	}

	// Ensure enough time for any async work before cleanup.
	time.Sleep(10 * time.Millisecond)
}
