package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/history"
	"github.com/Rish3666/ServeEz/internal/orchestrator"
	"github.com/Rish3666/ServeEz/internal/state"
)

func TestPredictEndpoint(t *testing.T) {
	// Server WITHOUT history wiring: predict returns 404 (predictor off).
	_, h := newTestServer(t)
	rec := doJSON(t, h, "GET", "/v1/predict?workload=web", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 without predictor, got %d", rec.Code)
	}
}

func TestPredictWithHistory(t *testing.T) {
	reg := state.NewRegistry()
	mustRegisterPredictKinds(t, reg)
	st := state.NewMemStoreWithRegistry(reg)
	al, err := audit.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = al.Close() })

	hist, err := history.Open(t.TempDir() + "/h.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = hist.Close() })

	ctx := context.Background()
	_, _ = st.Create(ctx, &api.Object{Kind: "Node", Namespace: "default", Name: "node-1",
		Spec:   &api.NodeSpec{Runtime: "docker", Capacity: api.Resources{CPUCores: 4, MemBytes: 16 << 30}},
		Status: &api.NodeStatus{State: "healthy"}})
	_, _ = st.Create(ctx, &api.Object{Kind: "Workload", Namespace: "default", Name: "web",
		Spec:   &api.WorkloadSpec{Image: "nginx", Replicas: 2},
		Status: &api.WorkloadStatus{AssignedNode: "node-1", State: "scheduled"}})

	// Seed a rising CPU series.
	base := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 40; i++ {
		_ = hist.Record(ctx, "node:node-1", history.Sample{
			At: base.Add(time.Duration(i) * 10 * time.Second),
			CPU: 50 + float64(i)*1.0,
		})
	}

	sched := orchestrator.NewScheduler(st)
	srv := New(st, reg, al, sched, "test-token")
	srv.WithHistory(hist)
	h := srv.Handler()

	rec := doJSON(t, h, "GET", "/v1/predict?workload=web", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("predict: %d %s", rec.Code, rec.Body.String())
	}
	var res api.PredictResponse
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if !res.Available {
		t.Fatalf("expected available: %+v", res)
	}
	if res.RecommendedReplicas <= res.CurrentReplicas {
		t.Fatalf("expected scale-up, got %d -> %d", res.CurrentReplicas, res.RecommendedReplicas)
	}
}

func mustRegisterPredictKinds(t *testing.T, reg *state.Registry) {
	t.Helper()
	if err := reg.Register(state.Schema{Kind: "Node", Version: "v1",
		NewSpec: func() any { return &api.NodeSpec{} }, NewStatus: func() any { return &api.NodeStatus{} }}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(state.Schema{Kind: "Workload", Version: "v1",
		NewSpec: func() any { return &api.WorkloadSpec{} }, NewStatus: func() any { return &api.WorkloadStatus{} }}); err != nil {
		t.Fatal(err)
	}
}