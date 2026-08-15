package state

import (
	"context"
	"errors"
	"testing"

	"github.com/Rish3666/ServeEz/internal/api"
)

func newTestRegistry() *Registry {
	r := NewRegistry()
	_ = r.Register(Schema{
		Kind:    "Node",
		Version: "v1",
		Validate: func(spec any) error {
			ns, ok := spec.(*api.NodeSpec)
			if !ok {
				return ErrValidation
			}
			if ns.Runtime == "" {
				return ErrValidation
			}
			return nil
		},
	})
	return r
}

func testNode(name string) *api.Object {
	return &api.Object{
		Kind:      "Node",
		Namespace: "default",
		Name:      name,
		Spec:      &api.NodeSpec{Runtime: "docker", Provider: "local"},
	}
}

func TestMemStoreCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()

	// Create
	v, err := s.Create(ctx, testNode("node-1"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v == "" {
		t.Fatal("expected resource version")
	}

	// Duplicate create
	if _, err := s.Create(ctx, testNode("node-1")); err == nil {
		t.Fatal("expected error on duplicate create")
	}

	// Get
	got, err := s.Get(ctx, "Node", "default", "node-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ResourceVersion != v {
		t.Fatalf("version mismatch: got %q want %q", got.ResourceVersion, v)
	}

	// List
	items, err := s.List(ctx, "Node", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// Update with stale version -> conflict
	stale := testNode("node-1")
	stale.ResourceVersion = "stale"
	if _, err := s.Update(ctx, stale); err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	// Update with current version -> ok
	upd := testNode("node-1")
	upd.ResourceVersion = v
	upd.Spec = &api.NodeSpec{Runtime: "containerd", Provider: "aws"}
	v2, err := s.Update(ctx, upd)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = s.Get(ctx, "Node", "default", "node-1")
	if got.ResourceVersion != v2 {
		t.Fatalf("version did not change after update")
	}

	// Delete
	if err := s.Delete(ctx, "Node", "default", "node-1", v2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, "Node", "default", "node-1"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestWatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewMemStore()
	ch, err := s.Watch(ctx)
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	_, _ = s.Create(ctx, testNode("node-1"))
	ev := <-ch
	if ev.Action != "create" || ev.Object.Name != "node-1" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestSQLiteStoreCRUD(t *testing.T) {
	ctx := context.Background()
	s, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	v, err := s.Create(ctx, testNode("node-1"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v == "" {
		t.Fatal("expected resource version")
	}
	got, err := s.Get(ctx, "Node", "default", "node-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "node-1" {
		t.Fatalf("bad get: %+v", got)
	}
	if got.ResourceVersion != v {
		t.Fatalf("version mismatch: got %q want %q", got.ResourceVersion, v)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("expected created_at set")
	}
	if _, err := s.Get(ctx, "Node", "default", "missing"); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRegistryValidation(t *testing.T) {
	r := newTestRegistry()

	// Valid node
	if err := r.Validate(testNode("n")); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Unknown kind
	if err := r.Validate(&api.Object{Kind: "Bogus", Name: "x"}); !errors.Is(err, ErrKindUnknown) {
		t.Fatalf("expected ErrKindUnknown, got %v", err)
	}

	// Invalid spec
	bad := &api.Object{Kind: "Node", Name: "x", Spec: &api.NodeSpec{Runtime: ""}}
	if err := r.Validate(bad); err == nil {
		t.Fatal("expected validation error")
	}
}
