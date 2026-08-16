package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGUIEmbedsIndexAndProxiesState(t *testing.T) {
	controlClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/state" || r.URL.Query().Get("type") != "Node" {
			return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("unexpected request")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"objects":[{"kind":"Node","name":"edge-1"}]}`)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})}

	app := newServer("http://control.test", controlClient, slog.New(slog.NewTextHandler(io.Discard, nil)))

	index := httptest.NewRecorder()
	app.routes().ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "ServeEz") {
		t.Fatalf("index status/body = %d/%q", index.Code, index.Body.String()[:min(len(index.Body.String()), 80)])
	}

	state := httptest.NewRecorder()
	app.routes().ServeHTTP(state, httptest.NewRequest(http.MethodGet, "/api/state?type=Node", nil))
	if state.Code != http.StatusOK || !strings.Contains(state.Body.String(), "edge-1") {
		t.Fatalf("state status/body = %d/%s", state.Code, state.Body.String())
	}
}

func TestProxyRejectsUnknownEndpoint(t *testing.T) {
	app := newServer("http://127.0.0.1:1", nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
