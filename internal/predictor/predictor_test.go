package predictor

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/history"
)

func samples(cpus ...float64) []history.Sample {
	base := time.Now().UTC().Truncate(time.Second)
	out := make([]history.Sample, 0, len(cpus))
	for i, c := range cpus {
		out = append(out, history.Sample{At: base.Add(time.Duration(i) * 10 * time.Second), CPU: c})
	}
	return out
}

type memHist struct{ sm []history.Sample }

func (m *memHist) Recent(ctx context.Context, series string, n int) ([]history.Sample, error) {
	if n > len(m.sm) {
		n = len(m.sm)
	}
	return m.sm[len(m.sm)-n:], nil
}

func TestColdStart(t *testing.T) {
	e := New(&memHist{samples(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)})
	res, err := e.Predict(context.Background(), "node:node-1", "web", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Available {
		t.Fatal("expected unavailable (cold start)")
	}
}

func TestTrendScaleUp(t *testing.T) {
	// Rising series well above the target by extrapolation.
	cpus := []float64{}
	for i := 0; i < 40; i++ {
		cpus = append(cpus, 50+float64(i)*1.2)
	}
	e := New(&memHist{samples(cpus...)})
	res, err := e.Predict(context.Background(), "node:node-1", "web", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Fatalf("expected available: %+v", res)
	}
	if res.RecommendedReplicas <= res.CurrentReplicas {
		t.Fatalf("expected scale-up, got %d -> %d (reason: %s)", res.CurrentReplicas, res.RecommendedReplicas, res.Reason)
	}
	if res.Confidence <= 0.5 {
		t.Fatalf("expected meaningful confidence, got %v", res.Confidence)
	}
}

func TestFlatNoScale(t *testing.T) {
	cpus := []float64{}
	for i := 0; i < 30; i++ {
		cpus = append(cpus, 30)
	}
	e := New(&memHist{samples(cpus...)})
	res, err := e.Predict(context.Background(), "node:node-1", "web", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Fatal("expected available")
	}
	if res.RecommendedReplicas != res.CurrentReplicas {
		t.Fatalf("expected no scale, got %d -> %d", res.CurrentReplicas, res.RecommendedReplicas)
	}
}

func TestFitLine(t *testing.T) {
	slope, intercept, rsq, err := fitLine(samples(10, 20, 30, 40, 50))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(slope-10) > 0.01 {
		t.Fatalf("expected slope ~10, got %v", slope)
	}
	if math.Abs(intercept-10) > 0.01 {
		t.Fatalf("expected intercept ~10, got %v", intercept)
	}
	if rsq < 0.99 {
		t.Fatalf("expected near-perfect fit, got r²=%v", rsq)
	}
}

func TestPredictResponseJSONShape(t *testing.T) {
	// Guard the wire contract the autoscale loop depends on.
	e := New(&memHist{samples(50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100, 105)})
	res, err := e.Predict(context.Background(), "node:node-1", "web", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = api.PredictResponse(res) // must be assignable
	if res.Workload == "" || !res.Available {
		t.Fatalf("unexpected: %+v", res)
	}
}