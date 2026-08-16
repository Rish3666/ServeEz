package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/apiclient"
	"github.com/Rish3666/ServeEz/internal/prewarm"
)

const scaleConfidenceThreshold = 0.70

type controlClient interface {
	State(context.Context, string) ([]*api.Object, error)
	Predict(context.Context, string) (api.PredictResponse, error)
	Simulate(context.Context, api.Action) (*api.SimulationResult, error)
	Execute(context.Context, api.Action) (*api.ActionResult, error)
}

type httpClient struct {
	baseURL string
	client  *http.Client
}

func (c *httpClient) State(ctx context.Context, kind string) ([]*api.Object, error) {
	return apiclient.New(c.baseURL).State(ctx, kind)
}

func (c *httpClient) Simulate(ctx context.Context, action api.Action) (*api.SimulationResult, error) {
	return apiclient.New(c.baseURL).Simulate(ctx, action)
}

func (c *httpClient) Execute(ctx context.Context, action api.Action) (*api.ActionResult, error) {
	return apiclient.New(c.baseURL).Execute(ctx, action)
}

func (c *httpClient) Predict(ctx context.Context, workload string) (api.PredictResponse, error) {
	var out api.PredictResponse
	path := c.baseURL + "/v1/predict?workload=" + url.QueryEscape(workload)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return out, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return out, err
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusMultipleChoices {
		return out, fmt.Errorf("control plane prediction: %s", res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

type Loop struct {
	client    controlClient
	interval  time.Duration
	cooldown  time.Duration
	lastScale map[string]time.Time
	now       func() time.Time
	leadTime  func(string, string) time.Duration
	logger    *slog.Logger
}

func NewLoop(client controlClient, interval, cooldown time.Duration, logger *slog.Logger) *Loop {
	if logger == nil {
		logger = slog.Default()
	}
	return &Loop{
		client:    client,
		interval:  interval,
		cooldown:  cooldown,
		lastScale: make(map[string]time.Time),
		now:       time.Now,
		leadTime:  prewarm.LeadTime,
		logger:    logger,
	}
}

func (l *Loop) Run(ctx context.Context) error {
	if l.client == nil {
		return errors.New("autoscale client is nil")
	}
	if l.interval <= 0 {
		return errors.New("autoscale interval must be positive")
	}
	if err := l.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
		l.logger.Error("autoscale tick failed", "error", err)
	}
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := l.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
				l.logger.Error("autoscale tick failed", "error", err)
			}
		}
	}
}

func (l *Loop) tick(ctx context.Context) error {
	objects, err := l.client.State(ctx, "Workload")
	if err != nil {
		return err
	}
	now := l.now()
	for _, workload := range objects {
		if workload == nil || workload.Name == "" {
			continue
		}
		if last, ok := l.lastScale[workload.Name]; ok && now.Sub(last) < l.cooldown {
			l.logger.Info("autoscale cooldown", "workload", workload.Name, "remaining", l.cooldown-now.Sub(last))
			continue
		}
		forecast, err := l.client.Predict(ctx, workload.Name)
		if err != nil {
			l.logger.Error("prediction failed", "workload", workload.Name, "error", err)
			continue
		}
		if !forecast.Available {
			l.logger.Info("autoscale skipped", "workload", workload.Name, "reason", "prediction unavailable")
			continue
		}
		if forecast.RecommendedReplicas <= forecast.CurrentReplicas {
			l.logger.Info("autoscale skipped", "workload", workload.Name, "reason", "scale-up only", "current", forecast.CurrentReplicas, "recommended", forecast.RecommendedReplicas)
			continue
		}
		image := workloadImage(workload.Spec)
		lead := l.leadTime("docker", image)
		action := api.Action{
			Type:       "scale",
			Target:     "workload:" + workload.Name,
			Parameters: map[string]any{"replicas": forecast.RecommendedReplicas},
			Reason:     forecast.Reason,
			Initiator:  "ai-agent:autoscale",
			Confidence: forecast.Confidence,
		}
		l.logger.Info("autoscale decision", "action", action.Type, "workload", workload.Name, "replicas", forecast.RecommendedReplicas, "confidence", forecast.Confidence, "reason", forecast.Reason, "prewarm_lead", lead)
		if forecast.Confidence < scaleConfidenceThreshold {
			l.logger.Info("autoscale skipped", "workload", workload.Name, "reason", "confidence below threshold", "threshold", scaleConfidenceThreshold)
			continue
		}
		sim, err := l.client.Simulate(ctx, action)
		if err != nil {
			l.logger.Error("scale simulation failed", "workload", workload.Name, "error", err)
			continue
		}
		if strings.EqualFold(sim.Recommendation, "reject") || strings.EqualFold(sim.Recommendation, "requires_approval") {
			l.logger.Info("autoscale skipped", "workload", workload.Name, "recommendation", sim.Recommendation)
			continue
		}
		result, err := l.client.Execute(ctx, action)
		if err != nil {
			l.logger.Error("scale execution failed", "workload", workload.Name, "error", err)
			continue
		}
		l.lastScale[workload.Name] = now
		l.logger.Info("autoscale executed", "workload", workload.Name, "status", result.Status, "replicas", forecast.RecommendedReplicas)
	}
	return nil
}

func workloadImage(spec any) string {
	data, err := json.Marshal(spec)
	if err != nil {
		return ""
	}
	var decoded api.WorkloadSpec
	if json.Unmarshal(data, &decoded) != nil {
		return ""
	}
	return decoded.Image
}

func main() {
	controlURL := flag.String("control-plane", "http://localhost:8443", "control-plane URL")
	interval := flag.Duration("interval", time.Minute, "autoscale evaluation interval")
	cooldown := flag.Duration("cooldown", 10*time.Minute, "minimum time between scale-ups per workload")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client := &httpClient{baseURL: strings.TrimRight(*controlURL, "/"), client: &http.Client{Timeout: 15 * time.Second}}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := NewLoop(client, *interval, *cooldown, logger).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("autoscale stopped", "error", err)
		os.Exit(1)
	}
}
