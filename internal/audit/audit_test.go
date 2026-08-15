package audit

import (
	"context"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

func TestSQLiteLogAppendList(t *testing.T) {
	ctx := context.Background()
	l, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer l.Close()

	e := &api.AuditEntry{
		Initiator:  "ai-agent:predictor-v2",
		ActionType: "scale",
		Target:     "web-frontend",
		Parameters: map[string]any{"replicas": 8},
		Status:     "completed",
		Confidence: 0.92,
	}
	id, err := l.Append(ctx, e)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if id == "" {
		t.Fatal("expected id")
	}

	got, err := l.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Target != "web-frontend" || got.Status != "completed" {
		t.Fatalf("bad entry: %+v", got)
	}

	all, err := l.List(ctx, Filter{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(all))
	}

	// Filter by initiator
	filtered, _ := l.List(ctx, Filter{Initiator: "ai-agent:predictor-v2"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered, got %d", len(filtered))
	}
	// Filter by mismatch
	none, _ := l.List(ctx, Filter{Status: "failed"})
	if len(none) != 0 {
		t.Fatalf("expected 0, got %d", len(none))
	}
}

func TestSQLiteLogNotFound(t *testing.T) {
	ctx := context.Background()
	l, _ := OpenSQLite(":memory:")
	defer l.Close()
	if _, err := l.Get(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteLogOrdering(t *testing.T) {
	ctx := context.Background()
	l, _ := OpenSQLite(":memory:")
	defer l.Close()

	for i := 0; i < 3; i++ {
		e := &api.AuditEntry{Initiator: "u", ActionType: "x", Target: "t", Status: "done", Timestamp: time.Now().UTC()}
		if _, err := l.Append(ctx, e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	all, _ := l.List(ctx, Filter{Limit: 100})
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
}
