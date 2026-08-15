package metrics

import (
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

func TestBufferWrapAndDrain(t *testing.T) {
	buf := NewBuffer(1, 10*time.Second.Nanoseconds())
	for i := 0; i < 8; i++ {
		buf.Add(usageFor(i))
	}
	snap := buf.Snapshot()
	if len(snap) != 6 {
		t.Fatalf("Snapshot len = %d, want 6", len(snap))
	}
	if snap[0].Usage.CPUPercent != 2 || snap[5].Usage.CPUPercent != 7 {
		t.Fatalf("Snapshot order = %+v", snap)
	}
	drained := buf.Drain()
	if len(drained) != 6 {
		t.Fatalf("Drain len = %d, want 6", len(drained))
	}
	if got := buf.Snapshot(); len(got) != 0 {
		t.Fatalf("Snapshot after drain len = %d, want 0", len(got))
	}
}

func usageFor(v int) Sample {
	return Sample{
		Usage: api.Usage{
			CPUPercent: float64(v),
		},
	}
}
