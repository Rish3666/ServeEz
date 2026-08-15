// Package simulate implements the ServeEz simulation sandbox (AI Control/06).
// v1 provides Tier 1 (statistical, <100ms) rule-based dry-runs. Tier 2
// (constraint validation against the object store) and Tier 3 (full sandbox)
// are layered on top as they land.
package simulate

import (
	"context"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

// Engine produces dry-run results for actions without touching real infra.
type Engine struct {
	// Store enables Tier-2 constraint checks (does the target exist, are
	// resources available). May be nil in which case Tier 2 is skipped.
	Store Store
}

// Store is the subset of the object store the simulator needs.
type Store interface {
	Get(ctx context.Context, kind, namespace, name string) (*api.Object, error)
}

// New returns a simulation engine.
func New(s Store) *Engine {
	return &Engine{Store: s}
}

// Simulate dry-runs an action. Tier 1 (statistical) runs unconditionally;
// Tier 2 (constraint validation) runs when the target object exists.
func (e *Engine) Simulate(ctx context.Context, act api.Action) api.SimulationResult {
	now := time.Now().UTC()
	res := api.SimulationResult{
		ID:             "sim_" + formatNano(now),
		RiskScore:      0.1,
		Confidence:     0.8,
		Predicted:      map[string]any{"impact": "low"},
		Recommendation: "proceed",
	}
	if err := e.checkConstraints(ctx, act); err != nil {
		res.Confidence = 0.95
		res.RiskScore = 1.0
		res.Recommendation = "reject"
		res.Predicted = map[string]any{"impact": err.Error()}
		return res
	}
	switch act.Type {
	case "kill", "remove":
		res.RiskScore = 0.7
		res.Confidence = 0.85
		res.Recommendation = "requires_approval"
		res.FailureScenarios = []api.Scenario{
			{Scenario: "service_interruption", Probability: 0.2, Impact: "workload downtime"},
		}
	case "stop":
		res.RiskScore = 0.6
		res.Confidence = 0.8
		res.Recommendation = "requires_approval"
	case "restart":
		res.RiskScore = 0.3
		res.Confidence = 0.85
		res.FailureScenarios = []api.Scenario{
			{Scenario: "startup_failure", Probability: 0.1, Impact: "workload stays down"},
		}
	case "scale", "scale_up":
		res.RiskScore = 0.1
		res.Confidence = 0.9
		res.Predicted = map[string]any{"impact": "more capacity provisioned", "cost": "+"}
	case "scale_down":
		res.RiskScore = 0.4
		res.Confidence = 0.85
		res.FailureScenarios = []api.Scenario{
			{Scenario: "traffic_spike", Probability: 0.15, Impact: "insufficient capacity"},
		}
	case "migrate":
		res.RiskScore = 0.5
		res.Confidence = 0.8
		res.Recommendation = "requires_approval"
	}
	return res
}

// checkConstraints performs Tier-2 validation against the store when available.
func (e *Engine) checkConstraints(ctx context.Context, act api.Action) error {
	if e.Store == nil {
		return nil
	}
	target := act.Target
	kind := "Workload"
	if len(target) > len("workload:") && target[:len("workload:")] == "workload:" {
		target = target[len("workload:"):]
	}
	// Only validate targets that reference known object kinds.
	if _, err := e.Store.Get(ctx, kind, "default", target); err != nil {
		return err
	}
	return nil
}

func formatNano(t time.Time) string {
	return t.Format("20060102150405.000000000")
}
