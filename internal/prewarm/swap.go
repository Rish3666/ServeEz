package prewarm

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

// Ops is the subset of container.Manager the swapper needs. Kept narrow so
// blue-green logic is testable with a small fake instead of the full manager.
type Ops interface {
	Create(ctx context.Context, workload string, spec api.WorkloadSpec, replica int) (api.ContainerStatus, error)
	Inspect(ctx context.Context, id string) (api.ContainerStatus, error)
	Stop(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	List(ctx context.Context, workload string) ([]api.ContainerStatus, error)
}

// SwapResult reports the outcome of a blue-green replace.
type SwapResult struct {
	OldID      string
	NewID      string
	Status     string // "swapped" | "rolled_back"
	Message    string
	DurationMS int64
}

// Swapper orchestrates a zero-downtime container swap:
//
//  1. create a warm clone with the target spec
//  2. wait for it to become healthy within the lead-time budget
//  3. drain the old container (stop), keep it as fallback for the window
//  4. remove the old container
//
// If the clone does not become healthy in time it is removed and the old
// container keeps running (rollback) — the operation has no downtime either way.
type Swapper struct {
	ops      Ops
	runtime  string
	lead     func(runtime, image string) time.Duration
	poll     time.Duration
	fallback time.Duration
}

// NewSwapper returns a Swapper. lead may be nil (defaults to LeadTime); poll
// and fallback default to 1s and 60s when non-positive.
func NewSwapper(ops Ops, runtime string, lead func(string, string) time.Duration) *Swapper {
	if lead == nil {
		lead = LeadTime
	}
	return &Swapper{ops: ops, runtime: runtime, lead: lead, poll: time.Second, fallback: 60 * time.Second}
}

// SetFallback overrides the fallback window (how long the old container stays
// alive after the clone is healthy before being removed).
func (s *Swapper) SetFallback(d time.Duration) { s.fallback = d }

// Swap replaces one instance of workload with a warm clone built from spec.
// If oldID is empty the oldest running instance is chosen automatically.
func (s *Swapper) Swap(ctx context.Context, workload string, spec api.WorkloadSpec, oldID string) (SwapResult, error) {
	start := time.Now()
	res := SwapResult{OldID: oldID}

	current, err := s.ops.List(ctx, workload)
	if err != nil {
		return res, fmt.Errorf("list instances: %w", err)
	}
	if oldID == "" {
		oldID = pickTarget(current)
		if oldID == "" {
			return res, fmt.Errorf("no running instance to replace for workload %q", workload)
		}
	}
	res.OldID = oldID

	next := nextReplica(current)
	clone, err := s.ops.Create(ctx, workload, spec, next)
	if err != nil {
		return res, fmt.Errorf("create warm clone: %w", err)
	}
	res.NewID = clone.ID

	if err := s.waitHealthy(ctx, clone.ID, s.lead(s.runtime, spec.Image)); err != nil {
		_ = s.ops.Remove(ctx, clone.ID)
		res.Status = "rolled_back"
		res.Message = fmt.Sprintf("warm clone %s failed health check, rolled back: %v", clone.ID, err)
		res.DurationMS = time.Since(start).Milliseconds()
		return res, nil
	}

	// Clone is healthy: drain the old container but keep it alive as a
	// fallback for the window before removing it.
	if err := s.ops.Stop(ctx, oldID); err != nil {
		return res, fmt.Errorf("drain old container: %w", err)
	}
	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case <-time.After(s.fallback):
	}
	if err := s.ops.Remove(ctx, oldID); err != nil {
		return res, fmt.Errorf("remove old container: %w", err)
	}

	res.Status = "swapped"
	res.Message = "warm clone active, old container drained and removed"
	res.DurationMS = time.Since(start).Milliseconds()
	return res, nil
}

func (s *Swapper) waitHealthy(ctx context.Context, id string, lead time.Duration) error {
	if lead <= 0 {
		lead = 30 * time.Second
	}
	deadline := time.Now().Add(lead)
	for time.Now().Before(deadline) {
		st, err := s.ops.Inspect(ctx, id)
		if err == nil && st.Health == "healthy" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.poll):
		}
	}
	return fmt.Errorf("clone %s not healthy within %s", id, lead)
}

// pickTarget returns the first running container ID, preferring the highest
// replica number (the one a scale-down would remove first).
func pickTarget(current []api.ContainerStatus) string {
	var best string
	bestN := -1
	for _, c := range current {
		if c.State != "" && c.State != "running" {
			continue
		}
		if n := replicaNum(c.Name); n > bestN {
			bestN = n
			best = c.ID
		}
	}
	return best
}

func nextReplica(current []api.ContainerStatus) int {
	next := 1
	for _, c := range current {
		if n := replicaNum(c.Name); n >= next {
			next = n + 1
		}
	}
	return next
}

func replicaNum(name string) int {
	i := strings.LastIndexByte(name, '-')
	if i < 0 {
		return 0
	}
	n, err := strconv.Atoi(name[i+1:])
	if err != nil {
		return 0
	}
	return n
}
