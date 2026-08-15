// servez-control is the ServeEz control plane daemon: API server + object
// store + audit log + reconciler. Run it on the master node, then join
// workers with `servez join` (agent side).
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/apiserver"
	"github.com/Rish3666/ServeEz/internal/audit"
	"github.com/Rish3666/ServeEz/internal/config"
	"github.com/Rish3666/ServeEz/internal/mcp"
	"github.com/Rish3666/ServeEz/internal/orchestrator"
	"github.com/Rish3666/ServeEz/internal/state"
)

func main() {
	var (
		configPath = flag.String("config", "servez-control.json", "path to config file")
		printToken = flag.Bool("print-token", false, "print the join token and exit")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	token, err := ensureJoinToken(cfg)
	if err != nil {
		slog.Error("join token", "err", err)
		os.Exit(1)
	}
	if *printToken {
		fmt.Println(token)
		return
	}

	if err := run(cfg, token); err != nil {
		slog.Error("control plane exited", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, token string) error {
	reg := state.NewRegistry()
	registerSchemas(reg)

	store, err := state.OpenSQLiteWithRegistry(cfg.StatePath, reg)
	if err != nil {
		return err
	}
	defer store.Close()

	auditLog, err := audit.OpenSQLite(cfg.AuditPath)
	if err != nil {
		return err
	}
	defer auditLog.Close()

	sched := orchestrator.NewScheduler(store)
	reconciler := orchestrator.NewReconciler(store, sched)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := reconciler.Run(ctx); err != nil {
			slog.Error("reconciler", "err", err)
		}
	}()
	<-reconciler.Ready

	srv := apiserver.New(store, reg, auditLog, sched, token)
	srv.WithMCP(mcp.New(store, auditLog, srv))
	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("control plane listening",
		"addr", cfg.ListenAddr,
		"state", cfg.StatePath,
		"audit", cfg.AuditPath,
		"join_token", token,
	)

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sig:
		slog.Info("shutting down")
		shutdownCtx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer scancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		cancel()
		return nil
	}
}

// registerSchemas registers the typed object kinds for validation + decode.
func registerSchemas(reg *state.Registry) {
	_ = reg.Register(state.Schema{
		Kind: "Node", Version: "v1",
		NewSpec:   func() any { return &api.NodeSpec{} },
		NewStatus: func() any { return &api.NodeStatus{} },
		Validate: func(spec any) error {
			ns, ok := spec.(*api.NodeSpec)
			if !ok || ns.Runtime == "" {
				return state.ErrValidation
			}
			return nil
		},
	})
	_ = reg.Register(state.Schema{
		Kind: "Workload", Version: "v1",
		NewSpec:   func() any { return &api.WorkloadSpec{} },
		NewStatus: func() any { return &api.WorkloadStatus{} },
		Validate: func(spec any) error {
			ws, ok := spec.(*api.WorkloadSpec)
			if !ok || ws.Image == "" {
				return state.ErrValidation
			}
			return nil
		},
	})
}

// ensureJoinToken returns a configured or generated join token, persisting a
// generated one to TokenFile so restarts are stable.
func ensureJoinToken(cfg *config.Config) (string, error) {
	if cfg.JoinToken != "" {
		return cfg.JoinToken, nil
	}
	if cfg.TokenFile != "" {
		if b, err := os.ReadFile(cfg.TokenFile); err == nil && len(b) > 0 {
			return string(b), nil
		}
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	if cfg.TokenFile != "" {
		if err := os.WriteFile(cfg.TokenFile, []byte(tok), 0o600); err != nil {
			return "", err
		}
	}
	return tok, nil
}
