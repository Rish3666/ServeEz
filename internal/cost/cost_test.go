package cost

import (
	"context"
	"testing"

	"github.com/Rish3666/ServeEz/internal/api"
)

func TestCompareKnownShape(t *testing.T) {
	e := New()
	rep, err := e.Compare(context.Background(), api.CostCompareRequest{VCPU: 4, MemGB: 8, Region: "us-east-1"})
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if rep.Best == nil {
		t.Fatal("expected a best recommendation")
	}
	if len(rep.Providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(rep.Providers))
	}
	for _, p := range rep.Providers {
		if p.EstMonthly <= 0 {
			t.Errorf("provider %s: non-positive monthly cost %v", p.Provider, p.EstMonthly)
		}
		if p.VCPU < 4 || p.MemGB < 8 {
			t.Errorf("provider %s: %s does not satisfy shape (%d vCPU / %d GB)", p.Provider, p.InstanceType, p.VCPU, p.MemGB)
		}
	}
	// Sorted cheapest -> most expensive.
	for i := 1; i < len(rep.Providers); i++ {
		if rep.Providers[i].EstMonthly < rep.Providers[i-1].EstMonthly {
			t.Errorf("providers not sorted: %v before %v", rep.Providers[i-1].EstMonthly, rep.Providers[i].EstMonthly)
		}
	}
}

func TestCompareNoMatch(t *testing.T) {
	e := New()
	_, err := e.Compare(context.Background(), api.CostCompareRequest{VCPU: 1024, MemGB: 4096})
	if err == nil {
		t.Fatal("expected error for unsatisfiable shape")
	}
}

func TestSpotLeqOnDemand(t *testing.T) {
	e := New()
	rep, err := e.Compare(context.Background(), api.CostCompareRequest{VCPU: 2, MemGB: 4})
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	for _, p := range rep.Providers {
		if p.SpotPerMo > p.OnDemandPerMo {
			t.Errorf("provider %s: spot $%.2f > on-demand $%.2f", p.Provider, p.SpotPerMo, p.OnDemandPerMo)
		}
	}
}

func TestCompareInvalidRequest(t *testing.T) {
	e := New()
	if _, err := e.Compare(context.Background(), api.CostCompareRequest{}); err == nil {
		t.Fatal("expected error for empty request")
	}
	if _, err := e.Compare(context.Background(), api.CostCompareRequest{VCPU: 2, MemGB: 4, RuntimePct: 150}); err == nil {
		t.Fatal("expected error for runtime_pct > 100")
	}
}

func TestRuntimePctScalesCost(t *testing.T) {
	e := New()
	full, _ := e.Compare(context.Background(), api.CostCompareRequest{VCPU: 2, MemGB: 4, RuntimePct: 100})
	half, _ := e.Compare(context.Background(), api.CostCompareRequest{VCPU: 2, MemGB: 4, RuntimePct: 50})
	if half.Best == nil || full.Best == nil {
		t.Fatal("expected recommendations")
	}
	if half.Best.EstMonthly >= full.Best.EstMonthly {
		t.Errorf("50%% runtime should cost less: half=%.2f full=%.2f", half.Best.EstMonthly, full.Best.EstMonthly)
	}
}

func BenchmarkCompare(b *testing.B) {
	e := New()
	req := api.CostCompareRequest{VCPU: 4, MemGB: 8}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := e.Compare(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
