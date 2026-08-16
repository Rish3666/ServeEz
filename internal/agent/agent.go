package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rish3666/ServeEz/internal/agentnet"
	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/container"
	"github.com/Rish3666/ServeEz/internal/metrics"
	"github.com/Rish3666/ServeEz/internal/prewarm"
)

type Config struct {
	ControlPlaneURL string
	Token           string
	NodeID          string
	Version         string
	Runtime         string
	Provider        string
	DataDir         string
}

type Agent struct {
	cfg       Config
	collector *metrics.Collector
	buffer    *metrics.Buffer
	manager   container.Manager
	logger    *log.Logger
	client    *agentnet.Client
	stateMu   sync.Mutex
	state     string
	health    int
}

var (
	collectInterval = 5 * time.Second
	reportInterval  = 10 * time.Second
	commandInterval = 5 * time.Second
)

func New(cfg Config, collector *metrics.Collector, buffer *metrics.Buffer, manager container.Manager, logger *log.Logger) *Agent {
	if logger == nil {
		logger = log.Default()
	}
	return &Agent{
		cfg:       cfg,
		collector: collector,
		buffer:    buffer,
		manager:   manager,
		logger:    logger,
		state:     "pending",
		health:    0,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	client := a.client
	if client == nil {
		var err error
		client, err = agentnet.New(a.cfg.ControlPlaneURL, tlsConfigForControlPlane(a.cfg.ControlPlaneURL, a.cfg.DataDir, a.cfg.NodeID, a.logger))
		if err != nil {
			return err
		}
		a.client = client
	}

	if err := a.register(ctx); err != nil {
		return err
	}

	collectTicker := time.NewTicker(collectInterval)
	reportTicker := time.NewTicker(reportInterval)
	commandTicker := time.NewTicker(commandInterval)
	defer collectTicker.Stop()
	defer reportTicker.Stop()
	defer commandTicker.Stop()

	latest, _ := a.collector.Collect(ctx)
	a.buffer.Add(latest)
	a.refreshState(latest)
	if err := a.flushReports(ctx, latest); err != nil {
		a.logger.Printf("initial report: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-collectTicker.C:
			sample, err := a.collector.Collect(ctx)
			if err != nil {
				a.logger.Printf("collect: %v", err)
				continue
			}
			a.buffer.Add(sample)
			if a.refreshState(sample) {
				_ = a.flushReports(ctx, sample)
			}
		case <-reportTicker.C:
			sample, err := a.collector.Collect(ctx)
			if err != nil {
				a.logger.Printf("collect: %v", err)
				continue
			}
			a.buffer.Add(sample)
			_ = a.flushReports(ctx, sample)
		case <-commandTicker.C:
			if err := a.pollCommands(ctx); err != nil {
				a.logger.Printf("commands: %v", err)
			}
		}
	}
}

func (a *Agent) register(ctx context.Context) error {
	req := api.RegisterRequest{
		NodeID:   a.cfg.NodeID,
		Token:    a.cfg.Token,
		Version:  a.cfg.Version,
		Runtime:  a.cfg.Runtime,
		Provider: a.cfg.Provider,
	}
	approved := false
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp, err := a.client.Register(ctx, req)
		if err != nil {
			lastErr = err
		} else if !resp.Approved {
			lastErr = errors.New(resp.Reason)
		} else {
			if resp.NodeID != "" {
				a.cfg.NodeID = resp.NodeID
			}
			approved = true
			break
		}
		wait := time.Duration(1<<attempt) * time.Second
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		a.logger.Printf("register attempt %d failed: %v", attempt+1, lastErr)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	if !approved {
		if lastErr == nil {
			lastErr = fmt.Errorf("registration rejected")
		}
		return lastErr
	}
	return nil
}

func (a *Agent) flushReports(ctx context.Context, sample metrics.Sample) error {
	samples := a.buffer.Drain()
	if len(samples) == 0 {
		samples = []metrics.Sample{sample}
	}
	for i, s := range samples {
		report := a.buildReport(ctx, s)
		if err := a.sendReport(ctx, report); err != nil {
			for _, pending := range samples[i:] {
				a.buffer.Add(pending)
			}
			return err
		}
	}
	return nil
}

func (a *Agent) sendReport(ctx context.Context, report api.NodeReport) error {
	ack, err := a.client.Report(ctx, a.cfg.NodeID, report)
	if err != nil {
		a.setState("disconnected")
		return err
	}
	if !ack.OK {
		a.logger.Printf("report rejected: %s", ack.Message)
	}
	return nil
}

func (a *Agent) buildReport(ctx context.Context, sample metrics.Sample) api.NodeReport {
	workloads, _ := a.manager.List(ctx, "")
	for i := range workloads {
		workloads[i].NodeID = a.cfg.NodeID
	}
	a.stateMu.Lock()
	state := a.state
	health := a.health
	a.stateMu.Unlock()
	return api.NodeReport{
		NodeID:      a.cfg.NodeID,
		State:       state,
		HealthScore: health,
		Usage:       sample.Usage,
		Hardware:    sample.Hardware,
		Workloads:   workloads,
		ReportedAt:  sample.At,
	}
}

func (a *Agent) refreshState(sample metrics.Sample) bool {
	health := score(sample.Usage)
	state := "healthy"
	switch {
	case health < 40:
		state = "unhealthy"
	case health < 75:
		state = "degraded"
	}
	a.stateMu.Lock()
	changed := a.state != state || a.health != health
	a.state = state
	a.health = health
	a.stateMu.Unlock()
	return changed
}

func score(usage api.Usage) int {
	avg := (usage.CPUPercent + usage.MemPercent + usage.DiskPercent) / 3
	score := int(100 - avg)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func (a *Agent) pollCommands(ctx context.Context) error {
	actions, err := a.client.Commands(ctx, a.cfg.NodeID)
	if err != nil {
		a.setState("disconnected")
		return err
	}
	for _, action := range actions {
		result, execErr := a.executeAction(ctx, action)
		ackStatus := "completed"
		if execErr != nil {
			ackStatus = "failed"
		}
		if err := a.client.Ack(ctx, a.cfg.NodeID, actionID(action), agentnet.CommandAck{Status: ackStatus, Result: result}); err != nil {
			return err
		}
		if execErr != nil {
			a.logger.Printf("action %s failed: %v", action.Type, execErr)
		}
	}
	return nil
}

func (a *Agent) executeAction(ctx context.Context, action api.Action) (api.ActionResult, error) {
	start := time.Now()
	res := api.ActionResult{ID: actionID(action), Action: action}
	switch strings.ToLower(action.Type) {
	case "start":
		if err := a.manager.Start(ctx, action.Target); err != nil {
			return res, err
		}
	case "stop":
		if err := a.manager.Stop(ctx, action.Target); err != nil {
			return res, err
		}
	case "restart":
		if err := a.manager.Restart(ctx, action.Target); err != nil {
			return res, err
		}
	case "remove":
		if err := a.manager.Remove(ctx, action.Target); err != nil {
			return res, err
		}
	case "replace":
		if err := a.replace(ctx, action, &res); err != nil {
			return res, err
		}
	case "deploy":
		spec, replicas, err := specFromAction(action)
		if err != nil {
			return res, err
		}
		for i := 1; i <= replicas; i++ {
			if _, err := a.manager.Create(ctx, action.Target, spec, i); err != nil {
				return res, err
			}
		}
	case "scale":
		replicas, err := replicasFromAction(action)
		if err != nil {
			return res, err
		}
		current, err := a.manager.List(ctx, action.Target)
		if err != nil {
			return res, err
		}
		switch {
		case len(current) < replicas:
			spec, _, err := specFromAction(action)
			if err != nil {
				return res, err
			}
			for i := len(current) + 1; i <= replicas; i++ {
				if _, err := a.manager.Create(ctx, action.Target, spec, i); err != nil {
					return res, err
				}
			}
		case len(current) > replicas:
			for i := replicas; i < len(current); i++ {
				if err := a.manager.Remove(ctx, current[i].ID); err != nil {
					return res, err
				}
			}
		}
	default:
		return res, fmt.Errorf("unsupported action %q", action.Type)
	}
	res.Status = "completed"
	res.DurationMS = time.Since(start).Milliseconds()
	res.Message = "ok"
	return res, nil
}

func actionID(action api.Action) string {
	if action.Parameters != nil {
		if id, ok := action.Parameters["action_id"].(string); ok && id != "" {
			return id
		}
		if id, ok := action.Parameters["id"].(string); ok && id != "" {
			return id
		}
	}
	if action.Target != "" {
		return action.Type + ":" + action.Target
	}
	return action.Type
}

// replace performs a blue-green swap of one instance of action.Target. The
// target is either a workload ("web" or "workload:web") — the oldest running
// instance is replaced — or a specific container via parameters["instance"].
// The old container is only removed after a warm clone is healthy, so the
// operation has zero downtime (rolled back otherwise).
func (a *Agent) replace(ctx context.Context, action api.Action, res *api.ActionResult) error {
	target := strings.TrimPrefix(action.Target, "workload:")
	spec, _, err := specFromAction(action)
	if err != nil {
		return err
	}
	oldID := ""
	if action.Parameters != nil {
		if id, ok := action.Parameters["instance"].(string); ok && id != "" {
			oldID = id
		}
	}
	swapper := prewarm.NewSwapper(a.manager, a.cfg.Runtime, prewarm.LeadTime)
	if fb, ok := action.Parameters["fallback_seconds"].(float64); ok && fb >= 0 {
		swapper.SetFallback(time.Duration(fb) * time.Second)
	}
	out, err := swapper.Swap(ctx, target, spec, oldID)
	if err != nil {
		return err
	}
	res.Before = map[string]any{"instance": out.OldID}
	res.After = map[string]any{"instance": out.NewID, "status": out.Status}
	res.Message = out.Message
	if out.Status == "rolled_back" {
		return fmt.Errorf("replace rolled back: %s", out.Message)
	}
	return nil
}

func replicasFromAction(action api.Action) (int, error) {
	if action.Parameters == nil {
		return 0, fmt.Errorf("replicas missing")
	}
	switch v := action.Parameters["replicas"].(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("replicas missing")
	}
}

func specFromAction(action api.Action) (api.WorkloadSpec, int, error) {
	var spec api.WorkloadSpec
	replicas := 1
	if action.Parameters == nil {
		return spec, 0, fmt.Errorf("spec missing")
	}
	if v, ok := action.Parameters["replicas"]; ok {
		switch n := v.(type) {
		case int:
			replicas = n
		case float64:
			replicas = int(n)
		case string:
			x, err := strconv.Atoi(n)
			if err != nil {
				return spec, 0, err
			}
			replicas = x
		}
	}
	raw, ok := action.Parameters["spec"]
	if !ok {
		raw = action.Parameters
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return spec, 0, err
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return spec, 0, err
	}
	if spec.Image == "" {
		return spec, 0, fmt.Errorf("image missing")
	}
	if replicas < 1 {
		replicas = 1
	}
	return spec, replicas, nil
}

func (a *Agent) setState(state string) {
	a.stateMu.Lock()
	a.state = state
	a.stateMu.Unlock()
}
