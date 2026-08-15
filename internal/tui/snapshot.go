// Package tui implements the ServeEz terminal dashboard (UI/TUI Dashboard.md):
// a htop-meets-k9s view of cluster nodes, workloads, alerts, and AI chat.
// The app shell (layout, keybindings, polling) lives in app.go; panels are
// pluggable components registered via RegisterPanel so agents can add panes
// without touching the shell.
package tui

import (
	"context"
	"time"

	"github.com/Rish3666/ServeEz/internal/apiclient"
)

// Snapshot is a point-in-time view of cluster state, refreshed by the app
// shell's poller and pushed to every panel via SetSnapshot.
type Snapshot struct {
	ControlURL string
	Nodes      []NodeRow
	Workloads  []WorkloadRow
	FetchedAt  time.Time
	Err        error
}

// NodeRow is a flattened node for terminal rendering.
type NodeRow struct {
	Name    string
	State   string
	Health  int
	CPU     float64
	MemPct  float64
	MemUsed uint64
	MemCap  uint64
}

// WorkloadRow is a flattened workload for terminal rendering.
type WorkloadRow struct {
	Name     string
	Image    string
	Replicas int
	State    string
	Node     string
	CPU      float64
	MemPct   float64
}

// fetchState polls the control plane and flattens objects into a Snapshot.
func fetchState(ctx context.Context, c *apiclient.Client, controlURL string) *Snapshot {
	snap := &Snapshot{ControlURL: controlURL, FetchedAt: time.Now()}
	now := time.Now()

	nodes, err := c.State(ctx, "Node")
	if err != nil {
		snap.Err = err
		return snap
	}
	for _, n := range nodes {
		row := NodeRow{Name: n.Name}
		if st, ok := n.Status.(map[string]any); ok {
			row.State = strOf(st["state"])
			row.Health = intOf(st["health_score"])
			if res, ok := st["resources"].(map[string]any); ok {
				row.CPU = fltOf(res["cpu_pct"])
				row.MemPct = fltOf(res["mem_pct"])
			}
		}
		if sp, ok := n.Spec.(map[string]any); ok {
			if cap, ok := sp["capacity"].(map[string]any); ok {
				row.MemCap = u64Of(cap["mem_bytes"])
			}
		}
		snap.Nodes = append(snap.Nodes, row)
	}

	workloads, err := c.State(ctx, "Workload")
	if err != nil {
		snap.Err = err
		return snap
	}
	for _, w := range workloads {
		row := WorkloadRow{Name: w.Name}
		if sp, ok := w.Spec.(map[string]any); ok {
			row.Image = strOf(sp["image"])
			row.Replicas = intOf(sp["replicas"])
		}
		if st, ok := w.Status.(map[string]any); ok {
			row.State = strOf(st["state"])
			row.Node = strOf(st["assigned_node"])
		}
		snap.Workloads = append(snap.Workloads, row)
	}
	_ = now
	return snap
}

// strOf / fltOf / intOf / u64Of are lenient map accessors for decoded JSON.
func strOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func fltOf(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	}
	return 0
}

func intOf(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}

func u64Of(v any) uint64 {
	switch t := v.(type) {
	case float64:
		return uint64(t)
	case int:
		return uint64(t)
	case uint64:
		return t
	}
	return 0
}
