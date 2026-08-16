// Package apiclient is a lightweight HTTP client for the ServeEz control-plane
// API, used by the servez CLI. It mirrors internal/agentnet but for
// human-driven commands (read state, deploy, scale, audit) rather than the
// agent's register/report loop.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

// Client talks to a ServeEz control plane.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New returns a client bound to the control plane base URL (e.g. http://localhost:8443).
func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// State fetches the cluster state, optionally filtered by object type.
func (c *Client) State(ctx context.Context, kind string) ([]*api.Object, error) {
	path := "/v1/state"
	if kind != "" {
		path += "?type=" + url.QueryEscape(kind)
	}
	var resp struct {
		Objects []*api.Object `json:"objects"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Objects, nil
}

// Deploy creates a workload.
func (c *Client) Deploy(ctx context.Context, name string, spec *api.WorkloadSpec) error {
	body := map[string]any{"name": name, "spec": spec}
	var out api.Object
	return c.do(ctx, http.MethodPost, "/v1/workloads", body, &out)
}

// Execute submits an action and returns the result.
func (c *Client) Execute(ctx context.Context, act api.Action) (*api.ActionResult, error) {
	var out api.ActionResult
	if err := c.do(ctx, http.MethodPost, "/v1/execute", act, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Simulate dry-runs an action.
func (c *Client) Simulate(ctx context.Context, act api.Action) (*api.SimulationResult, error) {
	var out api.SimulationResult
	req := api.SimulationRequest{Action: act, SimulateOnly: true}
	if err := c.do(ctx, http.MethodPost, "/v1/simulate", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Audit lists audit entries.
func (c *Client) Audit(ctx context.Context, initiator, status string, limit int) ([]*api.AuditEntry, error) {
	q := url.Values{}
	if initiator != "" {
		q.Set("initiator", initiator)
	}
	if status != "" {
		q.Set("status", status)
	}
	path := "/v1/audit"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp struct {
		Entries []*api.AuditEntry `json:"entries"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Entries, nil
}

// KillSwitch disables all control-plane write operations.
func (c *Client) KillSwitch(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/v1/emergency/kill", nil, nil)
}

// Predict returns the forecast + scale recommendation for a workload.
func (c *Client) Predict(ctx context.Context, workload string) (*api.PredictResponse, error) {
	path := "/v1/predict?workload=" + url.QueryEscape(workload)
	var out api.PredictResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return fmt.Errorf("control plane: %s", strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
