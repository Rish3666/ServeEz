package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

type fakeControlClient struct {
	objects   []*api.Object
	predict   api.PredictResponse
	simulate  api.SimulationResult
	simCalls  int
	execCalls int
	last      api.Action
}

func (f *fakeControlClient) State(context.Context, string) ([]*api.Object, error) {
	return f.objects, nil
}
func (f *fakeControlClient) Predict(context.Context, string) (api.PredictResponse, error) {
	return f.predict, nil
}
func (f *fakeControlClient) Simulate(context.Context, api.Action) (*api.SimulationResult, error) {
	f.simCalls++
	return &f.simulate, nil
}
func (f *fakeControlClient) Execute(_ context.Context, action api.Action) (*api.ActionResult, error) {
	f.execCalls++
	f.last = action
	return &api.ActionResult{Status: "completed", Action: action}, nil
}

func TestLoopScalesUpOnceAndHonorsCooldown(t *testing.T) {
	now := time.Unix(100, 0)
	fake := &fakeControlClient{
		objects:  []*api.Object{{Kind: "Workload", Name: "web", Spec: map[string]any{"image": "nginx:latest"}}},
		predict:  api.PredictResponse{Available: true, Workload: "web", CurrentReplicas: 2, RecommendedReplicas: 4, Confidence: 0.9, Reason: "cpu trending up"},
		simulate: api.SimulationResult{Recommendation: "proceed"},
	}
	loop := NewLoop(fake, time.Minute, 10*time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	loop.now = func() time.Time { return now }
	if err := loop.tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.simCalls != 1 || fake.execCalls != 1 {
		t.Fatalf("calls = simulate %d execute %d, want 1 and 1", fake.simCalls, fake.execCalls)
	}
	if fake.last.Target != "workload:web" || fake.last.Parameters["replicas"] != 4 {
		t.Fatalf("action = %#v", fake.last)
	}
	if err := loop.tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.execCalls != 1 {
		t.Fatalf("execute calls = %d, want cooldown to prevent second scale", fake.execCalls)
	}
}

func TestLoopSkipsUnavailableLowConfidenceAndApproval(t *testing.T) {
	cases := []struct {
		name       string
		available  bool
		confidence float64
		recommend  string
	}{
		{name: "cold start", available: false, confidence: 1, recommend: "proceed"},
		{name: "low confidence", available: true, confidence: 0.69, recommend: "proceed"},
		{name: "approval", available: true, confidence: 0.9, recommend: "requires_approval"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeControlClient{
				objects:  []*api.Object{{Kind: "Workload", Name: "web"}},
				predict:  api.PredictResponse{Available: tc.available, CurrentReplicas: 1, RecommendedReplicas: 2, Confidence: tc.confidence},
				simulate: api.SimulationResult{Recommendation: tc.recommend},
			}
			loop := NewLoop(fake, time.Minute, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err := loop.tick(context.Background()); err != nil {
				t.Fatal(err)
			}
			if fake.execCalls != 0 {
				t.Fatalf("execute calls = %d, want 0", fake.execCalls)
			}
		})
	}
}
