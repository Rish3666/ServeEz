package agent

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/Rish3666/ServeEz/internal/agentnet"
	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/apiserver"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/container"
	"github.com/Rish3666/ServeEz/internal/metrics"
	"github.com/Rish3666/ServeEz/internal/orchestrator"
	"github.com/Rish3666/ServeEz/internal/state"
)

func TestAgentEndToEnd(t *testing.T) {
	oldCollect, oldReport, oldCommand := collectInterval, reportInterval, commandInterval
	collectInterval = 25 * time.Millisecond
	reportInterval = 40 * time.Millisecond
	commandInterval = 25 * time.Millisecond
	t.Cleanup(func() {
		collectInterval = oldCollect
		reportInterval = oldReport
		commandInterval = oldCommand
	})

	reg := state.NewRegistry()
	mustRegisterTestKind(t, reg)
	st := state.NewMemStoreWithRegistry(reg)
	al, err := audit.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	t.Cleanup(func() { _ = al.Close() })

	sched := orchestrator.NewScheduler(st)
	reconciler := orchestrator.NewReconciler(st, sched)
	rctx, rcancel := context.WithCancel(context.Background())
	go func() { _ = reconciler.Run(rctx) }()
	t.Cleanup(rcancel)
	<-reconciler.Ready

	srv := apiserver.New(st, reg, al, sched, "test-token")
	handler := srv.Handler()

	client, err := agentnet.NewWithHTTPClient("http://control.local", &http.Client{Transport: roundTripper{handler: handler}}, nil)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	mgr := newFakeManager()
	ag := New(Config{
		ControlPlaneURL: "http://control.local",
		Token:           "test-token",
		NodeID:          "node-1",
		Version:         "dev",
		Runtime:         "docker",
		Provider:        "local",
		DataDir:         t.TempDir(),
	}, metrics.NewCollector("/"), metrics.NewBuffer(5, 5*time.Second.Nanoseconds()), mgr, log.New(io.Discard, "", 0))
	ag.client = client

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ag.Run(runCtx)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("agent did not stop")
		}
	})

	waitFor(t, 5*time.Second, func() bool {
		obj, err := st.Get(context.Background(), "Node", "default", "node-1")
		if err != nil {
			return false
		}
		status, ok := obj.Status.(*api.NodeStatus)
		return ok && status != nil && !status.LastSeen.IsZero()
	})

	queueAction(t, srv, "node-1", "start-node-1", api.Action{
		Type:   "start",
		Target: "node-1",
		Parameters: map[string]any{
			"id": "start-node-1",
		},
	})

	waitFor(t, 5*time.Second, func() bool {
		return mgr.started()
	})

	waitFor(t, 5*time.Second, func() bool {
		entries, err := al.List(context.Background(), audit.Filter{Initiator: "node:node-1", Status: "completed"})
		return err == nil && len(entries) > 0
	})

	waitFor(t, 5*time.Second, func() bool {
		obj, err := st.Get(context.Background(), "Node", "default", "node-1")
		if err != nil {
			return false
		}
		status, ok := obj.Status.(*api.NodeStatus)
		if !ok || status == nil {
			return false
		}
		return len(status.Workloads) == 1 && status.Workloads[0].State == "running"
	})
}

func mustRegisterTestKind(t *testing.T, reg *state.Registry) {
	t.Helper()
	if err := reg.Register(state.Schema{
		Kind: "Node", Version: "v1",
		NewSpec:   func() any { return &api.NodeSpec{} },
		NewStatus: func() any { return &api.NodeStatus{} },
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}
	if err := reg.Register(state.Schema{
		Kind: "Workload", Version: "v1",
		NewSpec:   func() any { return &api.WorkloadSpec{} },
		NewStatus: func() any { return &api.WorkloadStatus{} },
	}); err != nil {
		t.Fatalf("register workload: %v", err)
	}
}

func queueAction(t *testing.T, srv *apiserver.Server, nodeID, actionID string, act api.Action) {
	t.Helper()
	v := reflect.ValueOf(srv).Elem()
	pending := makeSettableMap(v.FieldByName("pending"))
	nodeCmds := makeSettableMap(v.FieldByName("nodeCmds"))
	pending.SetMapIndex(reflect.ValueOf(actionID), reflect.ValueOf(act))
	cmds := nodeCmds.MapIndex(reflect.ValueOf(nodeID))
	if !cmds.IsValid() {
		nodeCmds.SetMapIndex(reflect.ValueOf(nodeID), reflect.ValueOf([]string{actionID}))
		return
	}
	list := append(cmds.Interface().([]string), actionID)
	nodeCmds.SetMapIndex(reflect.ValueOf(nodeID), reflect.ValueOf(list))
}

func makeSettableMap(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

type roundTripper struct {
	handler http.Handler
}

func (rt roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rt.handler.ServeHTTP(rec, req.Clone(req.Context()))
	return rec.Result(), nil
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

type fakeManager struct {
	mu      sync.Mutex
	running bool
	startCh chan struct{}
	once    sync.Once
}

func newFakeManager() *fakeManager {
	return &fakeManager{startCh: make(chan struct{}, 1)}
}

func (f *fakeManager) started() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

func (f *fakeManager) Create(ctx context.Context, workload string, spec api.WorkloadSpec, replica int) (api.ContainerStatus, error) {
	return api.ContainerStatus{ID: workload, Name: workload, Image: spec.Image, State: "running", Health: "healthy"}, nil
}

func (f *fakeManager) Start(ctx context.Context, id string) error {
	f.mu.Lock()
	f.running = true
	f.mu.Unlock()
	f.once.Do(func() { f.startCh <- struct{}{} })
	return nil
}

func (f *fakeManager) Stop(ctx context.Context, id string) error {
	f.mu.Lock()
	f.running = false
	f.mu.Unlock()
	return nil
}

func (f *fakeManager) Restart(ctx context.Context, id string) error {
	return f.Start(ctx, id)
}

func (f *fakeManager) Remove(ctx context.Context, id string) error {
	f.mu.Lock()
	f.running = false
	f.mu.Unlock()
	return nil
}

func (f *fakeManager) Inspect(ctx context.Context, id string) (api.ContainerStatus, error) {
	return api.ContainerStatus{ID: id, Name: id, Image: "alpine", State: "running", Health: "healthy"}, nil
}

func (f *fakeManager) List(ctx context.Context, workload string) ([]api.ContainerStatus, error) {
	f.mu.Lock()
	started := f.running
	f.mu.Unlock()
	if !started {
		return nil, nil
	}
	return []api.ContainerStatus{{
		ID: "ctr-1", Name: "web-1", Image: "alpine", State: "running", Health: "healthy", NodeID: "node-1",
	}}, nil
}

var _ container.Manager = (*fakeManager)(nil)
