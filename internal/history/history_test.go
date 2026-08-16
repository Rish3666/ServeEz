package history

import (
	"context"
	"testing"
	"time"
)

func TestRecordRecent(t *testing.T) {
	s, err := Open(t.TempDir() + "/h.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		if err := s.Record(ctx, "workload:web", Sample{At: base.Add(time.Duration(i) * time.Second), CPU: float64(i), MemPct: 50}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Recent(ctx, "workload:web", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(got))
	}
	// Oldest-first: indices 2,3,4.
	if got[0].CPU != 2 || got[2].CPU != 4 {
		t.Fatalf("unexpected order or values: %+v", got)
	}
	if _, err := s.Recent(ctx, "workload:nope", 5); err != nil {
		t.Fatal(err)
	}
}

func TestPrune(t *testing.T) {
	s, err := Open(t.TempDir() + "/h.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour)
	if err := s.Record(ctx, "workload:web", Sample{At: old, CPU: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(ctx, "workload:web", Sample{At: time.Now(), CPU: 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.Prune(ctx, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	n, err := s.Count(ctx, "workload:web")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 sample after prune, got %d", n)
	}
}