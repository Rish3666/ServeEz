package container

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Rish3666/ServeEz/internal/api"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestDockerManagerCreate(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		mu.Unlock()
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/containers/create"):
			return jsonResponse(http.StatusCreated, `{"Id":"abc"}`), nil
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			return jsonResponse(http.StatusNoContent, ``), nil
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/json"):
			return jsonResponse(http.StatusOK, `{"Id":"abc","Name":"/web-1","Config":{"Image":"alpine","Labels":{"servez.workload":"web"}},"State":{"Status":"running","Running":true},"NetworkSettings":{"IPAddress":""}}`), nil
		default:
			return jsonResponse(http.StatusOK, `{}`), nil
		}
	})
	m := &DockerManager{
		baseURL:     "http://docker.local",
		client:      &http.Client{Transport: rt},
		probeClient: &http.Client{Transport: rt},
		specs:       map[string]api.WorkloadSpec{},
		work:        map[string]string{},
		version:     "v1.41",
	}
	status, err := m.Create(context.Background(), "web", api.WorkloadSpec{Image: "alpine"}, 1)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if status.ID != "abc" || status.Name != "web-1" || status.State != "running" {
		t.Fatalf("Create() status = %+v", status)
	}
	if len(calls) != 3 {
		t.Fatalf("call count = %d, want 3: %#v", len(calls), calls)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
