package simulate

import (
	"context"
	"testing"

	"github.com/Rish3666/ServeEz/internal/api"
)

type fakeStore struct {
	obj *api.Object
	err error
}

func (f *fakeStore) Get(ctx context.Context, kind, namespace, name string) (*api.Object, error) {
	return f.obj, f.err
}

func TestSimulateKillRequiresApproval(t *testing.T) {
	e := New(nil)
	res := e.Simulate(context.Background(), api.Action{Type: "kill", Target: "web"})
	if res.Recommendation != "requires_approval" {
		t.Fatalf("expected requires_approval, got %q", res.Recommendation)
	}
	if res.RiskScore < 0.5 {
		t.Fatalf("expected high risk, got %v", res.RiskScore)
	}
}

func TestSimulateScaleProceeds(t *testing.T) {
	e := New(nil)
	res := e.Simulate(context.Background(), api.Action{Type: "scale", Target: "web"})
	if res.Recommendation != "proceed" {
		t.Fatalf("expected proceed, got %q", res.Recommendation)
	}
}

func TestSimulateMissingTargetRejects(t *testing.T) {
	e := New(&fakeStore{err: context.DeadlineExceeded})
	res := e.Simulate(context.Background(), api.Action{Type: "restart", Target: "workload:nope"})
	if res.Recommendation != "reject" {
		t.Fatalf("expected reject, got %q", res.Recommendation)
	}
}
