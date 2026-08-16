package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testControl returns a stub control plane that answers /v1/state and
// /v1/audit, so the GUI's proxy path can be exercised without a real server.
func testControl(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/state", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"api_version": "v1",
			"timestamp":   time.Now().UTC(),
			"objects": []map[string]any{
				{
					"kind": "Node", "name": "node-a1", "namespace": "default",
					"spec": map[string]any{"provider": "local", "capacity": map[string]any{"cpu_cores": 4, "mem_bytes": 4294967296}},
					"status": map[string]any{"state": "healthy", "health_score": 98,
						"resources": map[string]any{"cpu_pct": 34.2, "mem_used_bytes": 1288490188},
						"last_seen": time.Now().Add(-2 * time.Minute).UTC()},
				},
				{
					"kind": "Node", "name": "node-b2", "namespace": "default",
					"spec": map[string]any{"provider": "local", "capacity": map[string]any{"cpu_cores": 4, "mem_bytes": 4294967296}},
					"status": map[string]any{"state": "degraded", "health_score": 72,
						"resources": map[string]any{"cpu_pct": 91.4, "mem_used_bytes": 4078955136},
						"last_seen": time.Now().Add(-1 * time.Minute).UTC()},
				},
				{
					"kind": "Workload", "name": "api-gateway", "namespace": "default",
					"spec": map[string]any{"image": "ghcr.io/acme/gateway:2.1.0", "replicas": 4},
					"status": map[string]any{"state": "running", "node": "node-a1"},
				},
			},
		})
	})
	mux.HandleFunc("/v1/audit", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": []map[string]any{
			{"description": "scaled svc api-gateway 2 → 4 replicas", "timestamp": time.Now().Add(-2 * time.Minute).UTC()},
		}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newGUI(t *testing.T, control string) *httptest.Server {
	t.Helper()
	upstream, err := parseControl(control)
	if err != nil {
		t.Fatalf("parse control: %v", err)
	}
	proxy := buildProxy(upstream)
	handler := buildHandler(proxy)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestGUIServesIndex(t *testing.T) {
	control := testControl(t)
	gui := newGUI(t, control.URL)

	resp, err := http.Get(gui.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	buf := make([]byte, 1024*64)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "ServeEz") {
		t.Fatalf("index does not contain wordmark ServeEz")
	}
	if !strings.Contains(body, "view-overview") {
		t.Fatalf("index missing overview view")
	}
}

func TestGUIServesStatic(t *testing.T) {
	control := testControl(t)
	gui := newGUI(t, control.URL)

	for _, p := range []string{"/static/style.css", "/static/app.js"} {
		resp, err := http.Get(gui.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", p, resp.StatusCode)
		}
	}
}

func TestGUIProxiesState(t *testing.T) {
	control := testControl(t)
	gui := newGUI(t, control.URL)

	resp, err := http.Get(gui.URL + "/v1/state")
	if err != nil {
		t.Fatalf("GET /v1/state: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var payload struct {
		Objects []map[string]any `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Objects) != 3 {
		t.Fatalf("objects = %d, want 3", len(payload.Objects))
	}
}

func TestGUIUnknownRoute404(t *testing.T) {
	control := testControl(t)
	gui := newGUI(t, control.URL)

	resp, err := http.Get(gui.URL + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestParseControl(t *testing.T) {
	u, err := parseControl("http://127.0.0.1:7400")
	if err != nil || u.String() != "http://127.0.0.1:7400" {
		t.Fatalf("parseControl: %v %v", u, err)
	}
	if _, err := parseControl("://bad"); err == nil {
		t.Fatalf("expected error for bad URL")
	}
}

var _ = context.Background