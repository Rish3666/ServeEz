package container

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

type DockerManager struct {
	baseURL     string
	client      *http.Client
	probeClient *http.Client
	mu          sync.Mutex
	specs       map[string]api.WorkloadSpec
	work        map[string]string
	version     string
}

func NewDockerManager(cfg DockerConfig) (*DockerManager, error) {
	host := cfg.Host
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	baseURL, client, err := newDockerTransport(host)
	if err != nil {
		return nil, err
	}
	version := cfg.Version
	if version == "" {
		version = "v1.41"
	}
	return &DockerManager{
		baseURL:     baseURL,
		client:      client,
		probeClient: &http.Client{Timeout: 15 * time.Second},
		specs:       map[string]api.WorkloadSpec{},
		work:        map[string]string{},
		version:     version,
	}, nil
}

func newDockerTransport(host string) (string, *http.Client, error) {
	u, err := url.Parse(host)
	if err != nil {
		return "", nil, err
	}
	switch u.Scheme {
	case "unix":
		socketPath := u.Path
		if socketPath == "" {
			socketPath = "/var/run/docker.sock"
		}
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		}
		return "http://unix", &http.Client{Transport: transport, Timeout: 15 * time.Second}, nil
	case "http", "https":
		return strings.TrimRight(host, "/"), &http.Client{Timeout: 15 * time.Second}, nil
	default:
		return "", nil, fmt.Errorf("unsupported docker host scheme %q", u.Scheme)
	}
}

func (m *DockerManager) Create(ctx context.Context, workload string, spec api.WorkloadSpec, replica int) (api.ContainerStatus, error) {
	name := workload
	if replica > 0 {
		name = fmt.Sprintf("%s-%d", workload, replica)
	}
	reqBody := dockerCreateRequest{
		Image: spec.Image,
		Env:   makeEnv(spec.Env),
		Labels: map[string]string{
			"servez.workload": workload,
			"servez.replica":  strconv.Itoa(replica),
		},
		HostConfig: &dockerHostConfig{},
	}
	if len(spec.Ports) > 0 {
		reqBody.ExposedPorts = map[string]struct{}{}
		reqBody.HostConfig.PortBindings = map[string][]dockerPortBinding{}
		for _, port := range spec.Ports {
			key := fmt.Sprintf("%d/%s", port.Container, strings.ToLower(port.Protocol))
			reqBody.ExposedPorts[key] = struct{}{}
			reqBody.HostConfig.PortBindings[key] = []dockerPortBinding{{HostIP: "0.0.0.0", HostPort: "0"}}
		}
	}
	if spec.Resources != nil {
		reqBody.HostConfig.NanoCPUs = int64(spec.Resources.CPUCores * 1_000_000_000)
		reqBody.HostConfig.Memory = int64(spec.Resources.MemBytes)
	}
	body, _ := json.Marshal(reqBody)
	u := fmt.Sprintf("%s/%s/containers/create?name=%s", m.baseURL, m.version, url.QueryEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return api.ContainerStatus{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := m.client.Do(req)
	if err != nil {
		return api.ContainerStatus{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return api.ContainerStatus{}, fmt.Errorf("docker create: %s", strings.TrimSpace(string(data)))
	}
	var out struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return api.ContainerStatus{}, err
	}
	if err := m.Start(ctx, out.ID); err != nil {
		return api.ContainerStatus{}, err
	}
	m.mu.Lock()
	m.specs[out.ID] = spec
	m.work[out.ID] = workload
	m.mu.Unlock()
	return m.Inspect(ctx, out.ID)
}

func (m *DockerManager) Start(ctx context.Context, id string) error {
	return m.emptyAction(ctx, http.MethodPost, id, "start")
}

func (m *DockerManager) Stop(ctx context.Context, id string) error {
	return m.emptyAction(ctx, http.MethodPost, id, "stop?t=10")
}

func (m *DockerManager) Restart(ctx context.Context, id string) error {
	return m.emptyAction(ctx, http.MethodPost, id, "restart?t=10")
}

func (m *DockerManager) Remove(ctx context.Context, id string) error {
	m.mu.Lock()
	delete(m.specs, id)
	delete(m.work, id)
	m.mu.Unlock()
	u := fmt.Sprintf("%s/%s/containers/%s?force=true", m.baseURL, m.version, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	res, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return fmt.Errorf("docker remove: %s", strings.TrimSpace(string(data)))
	}
	return nil
}

func (m *DockerManager) Inspect(ctx context.Context, id string) (api.ContainerStatus, error) {
	u := fmt.Sprintf("%s/%s/containers/%s/json", m.baseURL, m.version, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return api.ContainerStatus{}, err
	}
	res, err := m.client.Do(req)
	if err != nil {
		return api.ContainerStatus{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return api.ContainerStatus{}, fmt.Errorf("docker inspect: %s", strings.TrimSpace(string(data)))
	}
	var out dockerInspectResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return api.ContainerStatus{}, err
	}
	status := api.ContainerStatus{
		ID:    out.ID,
		Name:  strings.TrimPrefix(out.Name, "/"),
		Image: out.Config.Image,
		State: out.State.Status,
		Health: func() string {
			if out.State.Health != nil && out.State.Health.Status != "" {
				return out.State.Health.Status
			}
			if out.State.Running {
				return "healthy"
			}
			if out.State.Status != "" {
				return out.State.Status
			}
			return "unknown"
		}(),
		NodeID: "",
	}
	if workload := out.Config.Labels["servez.workload"]; workload != "" {
		m.mu.Lock()
		if spec, ok := m.specs[out.ID]; ok && spec.Probes != nil {
			if health, err := m.checkProbe(ctx, out, spec.Probes); err == nil && health != "" {
				status.Health = health
			}
		}
		if node := m.work[out.ID]; node != "" {
			status.NodeID = node
		}
		m.mu.Unlock()
	}
	return status, nil
}

func (m *DockerManager) List(ctx context.Context, workload string) ([]api.ContainerStatus, error) {
	filter := "{}"
	if workload != "" {
		filter = fmt.Sprintf(`{"label":["servez.workload=%s"]}`, workload)
	}
	u := fmt.Sprintf("%s/%s/containers/json?all=1&filters=%s", m.baseURL, m.version, url.QueryEscape(filter))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("docker list: %s", strings.TrimSpace(string(data)))
	}
	var out []dockerListItem
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	statuses := make([]api.ContainerStatus, 0, len(out))
	for _, item := range out {
		name := item.ID
		if len(item.Names) > 0 && item.Names[0] != "" {
			name = strings.TrimPrefix(item.Names[0], "/")
		}
		statuses = append(statuses, api.ContainerStatus{
			ID:    item.ID,
			Name:  name,
			Image: item.Image,
			State: item.State,
			Health: func() string {
				if item.Status == "" {
					return "unknown"
				}
				return item.Status
			}(),
		})
	}
	return statuses, nil
}

func (m *DockerManager) emptyAction(ctx context.Context, method, id, suffix string) error {
	u := fmt.Sprintf("%s/%s/containers/%s/%s", m.baseURL, m.version, url.PathEscape(id), suffix)
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return err
	}
	res, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return fmt.Errorf("docker action: %s", strings.TrimSpace(string(data)))
	}
	return nil
}

func (m *DockerManager) checkProbe(ctx context.Context, info dockerInspectResponse, probes *api.Probes) (string, error) {
	if probes == nil {
		return "", nil
	}
	if probes.Readiness != nil {
		return m.runProbe(ctx, info, probes.Readiness)
	}
	if probes.Liveness != nil {
		return m.runProbe(ctx, info, probes.Liveness)
	}
	if probes.Startup != nil {
		return m.runProbe(ctx, info, probes.Startup)
	}
	return "", nil
}

func (m *DockerManager) runProbe(ctx context.Context, info dockerInspectResponse, probe *api.Probe) (string, error) {
	if probe == nil {
		return "", nil
	}
	timeout := time.Duration(probe.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if len(probe.Command) > 0 {
		return m.execProbe(cctx, info.ID, probe.Command)
	}
	host := info.NetworkSettings.IPAddress
	if host == "" {
		for _, n := range info.NetworkSettings.Networks {
			if n.IPAddress != "" {
				host = n.IPAddress
				break
			}
		}
	}
	if host == "" {
		return "unknown", nil
	}
	url := fmt.Sprintf("http://%s:%d%s", host, probe.Port, probe.Path)
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return "unhealthy", err
	}
	res, err := m.probeClient.Do(req)
	if err != nil {
		return "unhealthy", nil
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 400 {
		return "healthy", nil
	}
	return "unhealthy", nil
}

func (m *DockerManager) execProbe(ctx context.Context, id string, command []string) (string, error) {
	payload := map[string]any{
		"AttachStdout": true,
		"AttachStderr": true,
		"Cmd":          command,
	}
	body, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/%s/containers/%s/exec", m.baseURL, m.version, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "unhealthy", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := m.client.Do(req)
	if err != nil {
		return "unhealthy", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "unhealthy", nil
	}
	var out struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "unhealthy", err
	}
	startBody, _ := json.Marshal(map[string]any{"Detach": false, "Tty": false})
	startURL := fmt.Sprintf("%s/%s/exec/%s/start", m.baseURL, m.version, url.PathEscape(out.ID))
	startReq, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, bytes.NewReader(startBody))
	if err != nil {
		return "unhealthy", err
	}
	startReq.Header.Set("Content-Type", "application/json")
	startRes, err := m.client.Do(startReq)
	if err != nil {
		return "unhealthy", err
	}
	defer startRes.Body.Close()
	if startRes.StatusCode >= 200 && startRes.StatusCode < 400 {
		return "healthy", nil
	}
	return "unhealthy", nil
}

func makeEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for k, v := range values {
		out = append(out, k+"="+v)
	}
	return out
}

type dockerCreateRequest struct {
	Image        string              `json:"Image"`
	Env          []string            `json:"Env,omitempty"`
	Labels       map[string]string   `json:"Labels,omitempty"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts,omitempty"`
	HostConfig   *dockerHostConfig   `json:"HostConfig,omitempty"`
	Cmd          []string            `json:"Cmd,omitempty"`
}

type dockerHostConfig struct {
	NanoCPUs     int64                          `json:"NanoCPUs,omitempty"`
	Memory       int64                          `json:"Memory,omitempty"`
	PortBindings map[string][]dockerPortBinding `json:"PortBindings,omitempty"`
}

type dockerPortBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type dockerInspectResponse struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Status  string `json:"Status"`
		Running bool   `json:"Running"`
		Health  *struct {
			Status string `json:"Status"`
		} `json:"Health,omitempty"`
	} `json:"State"`
	NetworkSettings struct {
		IPAddress string `json:"IPAddress"`
		Networks  map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type dockerListItem struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}
