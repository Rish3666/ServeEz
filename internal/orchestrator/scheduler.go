// Package orchestrator implements the v1 (non-AI) scheduling and reconciliation
// control loop. This is the foundation that the AI scheduler will replace in
// Phase 1. Design docs: Orchestration/03 (AI Scheduler) and Orchestration/05
// (Container Lifecycle).
package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/state"
)

// NodeStore abstracts the persistence the scheduler needs. Kept narrow so the
// scheduler is easy to test without a full Store.
type NodeStore interface {
	List(ctx context.Context, kind, namespace string) ([]*api.Object, error)
}

// Scheduler places workloads onto nodes. v1 is a simple best-fit bin-packing
// on reported utilization; the AI scheduler (predictive placement) replaces it
// in Phase 1.
type Scheduler struct {
	store NodeStore
}

// NewScheduler returns a bin-packing scheduler.
func NewScheduler(s NodeStore) *Scheduler {
	return &Scheduler{store: s}
}

// NodeScore is a candidate placement.
type NodeScore struct {
	NodeID   string
	FreeCPU  float64
	FreeMem  float64
	Capacity api.Resources
	Usage    api.Usage
}

// Schedule picks the node with the most free resources that can fit the workload.
// Returns ErrNoFit when no node can host it.
func (s *Scheduler) Schedule(ctx context.Context, w *api.WorkloadSpec) (*NodeScore, error) {
	nodes, err := s.store.List(ctx, "Node", "")
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}
	req := reqResources(w)
	var candidates []*NodeScore
	for _, obj := range nodes {
		ns, ok := obj.Spec.(*api.NodeSpec)
		if !ok || ns == nil {
			continue
		}
		st, _ := obj.Status.(*api.NodeStatus)
		if st == nil {
			continue
		}
		if st.State == "cordoned" || st.State == "unhealthy" || st.State == "disconnected" {
			continue
		}
		freeCPU := ns.Capacity.CPUCores - (ns.Capacity.CPUCores * st.Resources.CPUPercent / 100)
		freeMem := float64(ns.Capacity.MemBytes) - (float64(ns.Capacity.MemBytes) * st.Resources.MemPercent / 100)
		if freeCPU >= req.CPUCores && freeMem >= float64(req.MemBytes) {
			candidates = append(candidates, &NodeScore{
				NodeID:   obj.Name,
				FreeCPU:  freeCPU,
				FreeMem:  freeMem,
				Capacity: ns.Capacity,
				Usage:    st.Resources,
			})
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no node fits workload %q", w.Image)
	}
	// Best fit: least free CPU (tightest fit) to pack densely.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].FreeCPU < candidates[j].FreeCPU
	})
	return candidates[0], nil
}

func reqResources(w *api.WorkloadSpec) api.Resources {
	if w.Resources != nil {
		return *w.Resources
	}
	// Default request when unspecified.
	return api.Resources{CPUCores: 0.1, MemBytes: 256 << 20}
}

// Reconciler watches the object store and drives Workload objects toward their
// desired state. v1: it assigns an initial node when one isn't set (the agent
// executes the actual container ops in parallel).
type Reconciler struct {
	mu        sync.Mutex
	store     state.Store
	scheduler *Scheduler
	// Ready is closed once the reconciler has subscribed to the store watch.
	Ready chan struct{}
}

// NewReconciler returns a reconciler bound to a store and scheduler.
func NewReconciler(st state.Store, s *Scheduler) *Reconciler {
	return &Reconciler{store: st, scheduler: s, Ready: make(chan struct{})}
}

// Run blocks, processing the state watch stream until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) error {
	ch, err := r.store.Watch(ctx)
	if err != nil {
		return fmt.Errorf("watch: %w", err)
	}
	close(r.Ready)
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if ev.Kind == "Workload" {
				r.reconcileWorkload(ctx, ev)
			}
		}
	}
}

func (r *Reconciler) reconcileWorkload(ctx context.Context, ev state.WatchEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev.Action == "delete" {
		return
	}
	obj := ev.Object
	ws, ok := obj.Spec.(*api.WorkloadSpec)
	if !ok || ws == nil {
		return
	}
	// Re-read from the store to get the latest version for the update.
	current, err := r.store.Get(ctx, "Workload", obj.Namespace, obj.Name)
	if err != nil {
		return
	}
	wst, ok := current.Status.(*api.WorkloadStatus)
	if !ok || wst == nil {
		wst = &api.WorkloadStatus{DesiredReplicas: ws.Replicas}
		current.Status = wst
	}
	if wst.AssignedNode == "" {
		score, err := r.scheduler.Schedule(ctx, ws)
		if err != nil {
			wst.State = "unschedulable"
			wst.Error = err.Error()
		} else {
			wst.State = "scheduled"
			wst.AssignedNode = score.NodeID
			wst.Error = ""
		}
		if _, err := r.store.Update(ctx, current); err != nil {
			_ = err
		}
	}
}
