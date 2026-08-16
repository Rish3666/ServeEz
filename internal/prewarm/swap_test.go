package prewarm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

type fakeOps struct {
	mu        sync.Mutex
	created   []api.ContainerStatus
	stopped   []string
	removed   []string
	health    func(id string) string
	createErr error
}

func newFakeOps() *fakeOps {
	return &fakeOps{health: func(string) string { return "healthy" }}
}

func (f *fakeOps) Create(ctx context.Context, workload string, spec api.WorkloadSpec, replica int) (api.ContainerStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return api.ContainerStatus{}, f.createErr
	}
	st := api.ContainerStatus{ID: "new-" + workload, Name: workload + "-" + itoa(replica), Image: spec.Image, State: "running", Health: "unknown"}
	f.created = append(f.created, st)
	return st, nil
}

func (f *fakeOps) Inspect(ctx context.Context, id string) (api.ContainerStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return api.ContainerStatus{ID: id, Name: id, Image: "img", State: "running", Health: f.health(id)}, nil
}

func (f *fakeOps) Stop(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, id)
	return nil
}

func (f *fakeOps) Remove(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, id)
	return nil
}

func (f *fakeOps) List(ctx context.Context, workload string) ([]api.ContainerStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return []api.ContainerStatus{
		{ID: "old-1", Name: workload + "-1", State: "running", Health: "healthy"},
		{ID: "old-2", Name: workload + "-2", State: "running", Health: "healthy"},
	}, nil
}

func (f *fakeOps) wasRemoved(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.removed {
		if r == id {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	return string(rune('0' + n))
}

func TestSwapSuccessful(t *testing.T) {
	ops := newFakeOps()
	s := NewSwapper(ops, "docker", func(string, string) time.Duration { return 2 * time.Second })
	s.SetFallback(0)
	out, err := s.Swap(context.Background(), "web", api.WorkloadSpec{Image: "img:v2"}, "old-1")
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if out.Status != "swapped" {
		t.Fatalf("status = %q, want swapped", out.Status)
	}
	if out.OldID != "old-1" {
		t.Fatalf("oldID = %q, want old-1", out.OldID)
	}
	if len(ops.created) != 1 {
		t.Fatalf("created %d clones, want 1", len(ops.created))
	}
	if ops.created[0].Name != "web-3" {
		t.Fatalf("clone name = %q, want web-3 (next replica)", ops.created[0].Name)
	}
	// Old container is stopped (drained) then removed after the fallback window.
	if len(ops.stopped) != 1 || ops.stopped[0] != "old-1" {
		t.Fatalf("stopped = %v, want [old-1]", ops.stopped)
	}
	if !ops.wasRemoved("old-1") {
		t.Fatal("old container was not removed")
	}
	// New clone survives.
	if ops.wasRemoved(out.NewID) {
		t.Fatal("warm clone was removed on success")
	}
}

func TestSwapRollbackWhenCloneUnhealthy(t *testing.T) {
	ops := newFakeOps()
	ops.health = func(string) string { return "unhealthy" }
	s := NewSwapper(ops, "docker", func(string, string) time.Duration { return 50 * time.Millisecond })
	s.SetFallback(0)
	// Bump the poll interval so the test finishes quickly.
	s.poll = 10 * time.Millisecond
	out, err := s.Swap(context.Background(), "web", api.WorkloadSpec{Image: "img"}, "old-1")
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if out.Status != "rolled_back" {
		t.Fatalf("status = %q, want rolled_back", out.Status)
	}
	if !ops.wasRemoved(out.NewID) {
		t.Fatal("warm clone should be removed on rollback")
	}
	if ops.wasRemoved("old-1") {
		t.Fatal("old container must keep running on rollback")
	}
}

func TestSwapAutoPickOldestTarget(t *testing.T) {
	ops := newFakeOps()
	s := NewSwapper(ops, "docker", func(string, string) time.Duration { return time.Second })
	s.SetFallback(0)
	out, err := s.Swap(context.Background(), "web", api.WorkloadSpec{Image: "img"}, "")
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	// Picks the highest replica (old-2), which a scale-down would remove first.
	if out.OldID != "old-2" {
		t.Fatalf("auto target = %q, want old-2", out.OldID)
	}
}

func TestSwapCreateFailure(t *testing.T) {
	ops := newFakeOps()
	ops.createErr = errors.New("docker down")
	s := NewSwapper(ops, "docker", LeadTime)
	s.SetFallback(0)
	if _, err := s.Swap(context.Background(), "web", api.WorkloadSpec{Image: "img"}, "old-1"); err == nil {
		t.Fatal("expected error on create failure")
	}
}

func TestLeadTimeTiering(t *testing.T) {
	if got := LeadTime("docker", "web"); got <= 0 {
		t.Fatalf("lead time for plain image = %v", got)
	}
	if got := LeadTime("docker", "img:cuda-12"); got <= LeadTime("docker", "web") {
		t.Fatalf("gpu image should take longer: %v vs %v", got, LeadTime("docker", "web"))
	}
}
