package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/state"
)

func testRegistry() *state.Registry {
	r := state.NewRegistry()
	_ = r.Register(state.Schema{
		Kind:    "Node",
		Version: "v1",
		NewSpec: func() any { return &api.NodeSpec{} },
		NewStatus: func() any { return &api.NodeStatus{} },
	})
	_ = r.Register(state.Schema{
		Kind:    "Workload",
		Version: "v1",
		NewSpec: func() any { return &api.WorkloadSpec{} },
		NewStatus: func() any { return &api.WorkloadStatus{} },
	})
	return r
}

func testStore() state.Store {
	return state.NewMemStoreWithRegistry(testRegistry())
}

func nodeObj(name, nodeState string, cpu, memPct float64) *api.Object {
	return &api.Object{
		Kind: "Node", Namespace: "default", Name: name,
		Spec: &api.NodeSpec{Runtime: "docker", Provider: "local", Capacity: api.Resources{CPUCores: 4, MemBytes: 16 << 30}},
		Status: &api.NodeStatus{State: nodeState, Resources: api.Usage{CPUPercent: cpu, MemPercent: memPct}},
	}
}

func TestSchedulerBestFit(t *testing.T) {
	ctx := context.Background()
	st := testStore()
	_, _ = st.Create(ctx, nodeObj("node-big", "healthy", 10, 10))
	_, _ = st.Create(ctx, nodeObj("node-small", "healthy", 70, 70))

	s := NewScheduler(st)
	w := &api.WorkloadSpec{Image: "nginx:1.25", Resources: &api.Resources{CPUCores: 1, MemBytes: 1 << 30}}
	score, err := s.Schedule(ctx, w)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	// node-big (10% used) has the tightest fit among the two free nodes? Both
	// fit; best-fit picks least free CPU -> node-big has most free. With
	// "least free CPU" sort, node-small has free 1.2 cores < node-big free 3.6,
	// so node-small should win if it fits (1.2 >= 1.0 yes).
	if score.NodeID != "node-small" {
		t.Fatalf("expected node-small, got %s", score.NodeID)
	}
}

func TestSchedulerSkipsUnhealthy(t *testing.T) {
	ctx := context.Background()
	st := testStore()
	_, _ = st.Create(ctx, nodeObj("node-bad", "unhealthy", 5, 5))
	_, _ = st.Create(ctx, nodeObj("node-good", "healthy", 5, 5))

	s := NewScheduler(st)
	_, err := s.Schedule(ctx, &api.WorkloadSpec{Image: "x"})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
}

func TestSchedulerNoFit(t *testing.T) {
	ctx := context.Background()
	st := testStore()
	_, _ = st.Create(ctx, nodeObj("node-full", "healthy", 99, 99))
	s := NewScheduler(st)
	_, err := s.Schedule(ctx, &api.WorkloadSpec{Image: "x", Resources: &api.Resources{CPUCores: 4, MemBytes: 16 << 30}})
	if err == nil {
		t.Fatal("expected error, got none")
	}
}

func TestReconcilerAssignsNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st := testStore()
	_, _ = st.Create(ctx, nodeObj("node-a", "healthy", 10, 10))
	_, _ = st.Create(ctx, nodeObj("node-b", "healthy", 10, 10))

	s := NewScheduler(st)
	r := NewReconciler(st, s)
	go func() { _ = r.Run(ctx) }()
	<-r.Ready

	wObj := &api.Object{
		Kind: "Workload", Namespace: "default", Name: "web",
		Spec: &api.WorkloadSpec{Image: "nginx", Replicas: 1},
	}
	_, err := st.Create(ctx, wObj)
	if err != nil {
		t.Fatalf("create workload: %v", err)
	}

	// Poll until the reconciler assigns a node.
	for i := 0; i < 50; i++ {
		got, err := st.Get(ctx, "Workload", "default", "web")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if wst, ok := got.Status.(*api.WorkloadStatus); ok && wst.AssignedNode != "" {
			if wst.State != "scheduled" {
				t.Fatalf("unexpected state %q", wst.State)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("workload was not scheduled")
}
