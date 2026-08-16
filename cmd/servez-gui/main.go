package main

import (
	"embed"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed static/*
var staticFS embed.FS

type server struct {
	controlURL string
	client     *http.Client
	logger     *slog.Logger
}

func newServer(controlURL string, client *http.Client, logger *slog.Logger) *server {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &server{
		controlURL: strings.TrimRight(controlURL, "/"),
		client:     client,
		logger:     logger,
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	assets := http.FileServer(http.FS(staticFS))
	mux.Handle("/static/", assets)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/" {
			if r.URL.Path == "/" {
				w.Header().Set("Allow", http.MethodGet)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			http.NotFound(w, r)
			return
		}
		data, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "dashboard unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/", s.proxy)
	return withSecurityHeaders(mux)
}

func (s *server) proxy(w http.ResponseWriter, r *http.Request) {
	path, ok := proxyPath(strings.TrimPrefix(r.URL.Path, "/api/"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	target, err := url.Parse(s.controlURL + path)
	if err != nil {
		http.Error(w, "invalid control-plane URL", http.StatusInternalServerError)
		return
	}
	target.RawQuery = r.URL.RawQuery
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), r.Body)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusBadGateway)
		return
	}
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	res, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("control-plane request failed", "path", path, "error", err)
		http.Error(w, "control plane unavailable", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}

func proxyPath(path string) (string, bool) {
	switch path {
	case "state":
		return "/v1/state", true
	case "audit":
		return "/v1/audit", true
	case "predict":
		return "/v1/predict", true
	case "cost/compare":
		return "/v1/cost/compare", true
	case "simulate":
		return "/v1/simulate", true
	default:
		return "", false
	}
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func main() {
	controlURL := flag.String("control", "http://localhost:8443", "control-plane URL")
	listen := flag.String("listen", ":8080", "GUI listen address")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	app := newServer(*controlURL, nil, logger)
	logger.Info("ServeEz GUI listening", "address", *listen, "control_plane", app.controlURL)
	if err := http.ListenAndServe(*listen, app.routes()); err != nil {
		logger.Error("GUI stopped", "error", err)
		os.Exit(1)
	}
}
