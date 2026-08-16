// Package predictor implements the fast-tier statistical forecast model for
// predictive scaling (AI Integration/01, AI Control/07 Fast Tier). It fits a
// least-squares linear trend to recent CPU utilization and forecasts ahead,
// producing a replica recommendation + confidence score in well under 100ms
// with no ML dependencies. The LLM/Prophet tier is a later-phase upgrade.
package predictor

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/history"
)

// TargetCPUPercent is the utilization ceiling we scale toward (headroom for
// spikes and new traffic).
const TargetCPUPercent = 80.0

// MinSamples is the minimum history length before the model returns an
// Available prediction (cold-start guard, Core Features/01).
const MinSamples = 12

// maxSamples is the window the model fits over (~5min at 10s intervals).
const maxSamples = 60

// History is the time-series source the predictor reads from.
type History interface {
	Recent(ctx context.Context, series string, n int) ([]history.Sample, error)
}

// Engine produces forecasts + scale recommendations.
type Engine struct {
	hist History
}

// New returns a predictor bound to a history store.
func New(h History) *Engine {
	return &Engine{hist: h}
}

// Predict returns the forecast + scale recommendation for a workload based on
// the CPU series of the node it is scheduled on (nodeCPU is the freshest
// current reading; pass 0 to rely on the last stored sample).
func (e *Engine) Predict(ctx context.Context, series, workload string, currentReplicas int, cpuNow float64) (api.PredictResponse, error) {
	res := api.PredictResponse{
		Workload:        workload,
		CurrentReplicas: currentReplicas,
		Available:       false,
	}
	samples, err := e.hist.Recent(ctx, series, maxSamples)
	if err != nil {
		return res, fmt.Errorf("predict history: %w", err)
	}
	// Cold start: not enough data to fit a trend.
	if len(samples) < MinSamples {
		res.Reason = fmt.Sprintf("cold start: %d samples (need %d)", len(samples), MinSamples)
		return res, nil
	}

	cur := samples[len(samples)-1].CPU
	if cpuNow > 0 {
		cur = cpuNow
	}
	slope, _, rsq, err := fitLine(samples)
	if err != nil {
		return res, fmt.Errorf("fit trend: %w", err)
	}

	res.Available = true
	res.CurrentCPUPercent = cur
	res.Forecast15mPercent = clamp(forecastAt(cur, slope, 15*time.Minute, samples))
	res.Forecast1hPercent = clamp(forecastAt(cur, slope, time.Hour, samples))
	res.Confidence = confidenceScore(rsq, len(samples), res.Forecast1hPercent)

	// Scale when the 1h forecast threatens the target. Round up so we add
	// whole replicas with headroom.
	need := res.Forecast1hPercent / TargetCPUPercent
	rec := int(math.Ceil(float64(currentReplicas) * need))
	if rec > currentReplicas {
		res.RecommendedReplicas = rec
		res.Recommendation = fmt.Sprintf("scale to %d replicas", rec)
		res.Reason = fmt.Sprintf("cpu trending up: %.1f -> %.1f pct by 1h (r²=%.2f)",
			cur, res.Forecast1hPercent, rsq)
	} else {
		res.RecommendedReplicas = currentReplicas
		res.Recommendation = "no scale needed"
		res.Reason = fmt.Sprintf("cpu stable/safe: %.1f -> %.1f pct by 1h", cur, res.Forecast1hPercent)
	}
	return res, nil
}

// fitLine fits y = a + b*x (x = index within samples) by least squares.
// Returns slope b, intercept a, and coefficient of determination r².
func fitLine(samples []history.Sample) (slope, intercept, rsq float64, err error) {
	n := float64(len(samples))
	if n < 2 {
		return 0, 0, 0, fmt.Errorf("need >= 2 samples")
	}
	var sx, sy, sxx, sxy, syy float64
	for i, s := range samples {
		x := float64(i)
		y := s.CPU
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
		syy += y * y
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		return 0, 0, 0, fmt.Errorf("degenerate input")
	}
	slope = (n*sxy - sx*sy) / denom
	intercept = (sy - slope*sx) / n
	// r² from the correlation coefficient.
	corrNum := n*sxy - sx*sy
	corrDen := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if corrDen == 0 {
		return slope, intercept, 0, nil
	}
	r := corrNum / corrDen
	return slope, intercept, r * r, nil
}

// forecastAt extrapolates the trend for dt ahead of the last sample.
func forecastAt(current, slopePerSample float64, dt time.Duration, samples []history.Sample) float64 {
	interval := sampleInterval(samples)
	if interval <= 0 {
		return current
	}
	steps := float64(dt) / interval.Seconds()
	return current + slopePerSample*steps
}

func sampleInterval(samples []history.Sample) time.Duration {
	if len(samples) < 2 {
		return 0
	}
	return samples[len(samples)-1].At.Sub(samples[len(samples)-2].At)
}

// confidenceScore combines fit quality, history length, and how far the
// forecast overshoots the target (beyond-target extrapolations are less
// certain). Returns 0-1.
func confidenceScore(rsq float64, n int, forecastPct float64) float64 {
	c := 0.35 + 0.35*rsq + 0.30*min(float64(n)/maxSamples, 1.0)
	if forecastPct > 90 {
		c -= 0.05 // extrapolating far beyond observed range
	}
	if c < 0 {
		c = 0
	}
	if c > 0.98 {
		c = 0.98
	}
	return c
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 200 {
		return 200
	}
	return v
}