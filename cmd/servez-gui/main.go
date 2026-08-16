// Command servez-gui serves the ServeEz web dashboard. It embeds the
// single-page static app and proxies /v1/* API calls to the control plane.
//
// Usage:
//
//	servez-gui --control=http://127.0.0.1:7400 --listen=:8080
package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

func parseControl(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, &url.Error{Op: "parse", URL: raw, Err: errUnsupportedScheme}
	}
	return u, nil
}

var errUnsupportedScheme = &unsupportedSchemeError{}

type unsupportedSchemeError struct{}

func (*unsupportedSchemeError) Error() string { return "unsupported scheme" }

func buildProxy(upstream *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, `{"error":"control plane unreachable"}`, http.StatusBadGateway)
	}
	return proxy
}

func buildHandler(proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		switch {
		case r.URL.Path == "/" || r.URL.Path == "" || r.URL.Path == "/index.html":
			data, err := staticFS.ReadFile("static/index.html")
			if err != nil {
				http.Error(w, "index missing", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
		case strings.HasPrefix(r.URL.Path, "/static/"):
			name := strings.TrimPrefix(r.URL.Path, "/static/")
			data, err := staticFS.ReadFile("static/" + name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if strings.HasSuffix(name, ".css") {
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			} else if strings.HasSuffix(name, ".js") {
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			}
			_, _ = w.Write(data)
		case strings.HasPrefix(r.URL.Path, "/v1/"):
			proxy.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func main() {
	var (
		control = flag.String("control", "http://127.0.0.1:7400", "control plane base URL")
		listen  = flag.String("listen", ":8080", "listen address")
	)
	flag.Parse()

	upstream, err := parseControl(*control)
	if err != nil {
		log.Fatalf("invalid --control URL: %v", err)
	}

	srv := &http.Server{
		Addr:              *listen,
		Handler:           buildHandler(buildProxy(upstream)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("ServeEz GUI: listening on %s, proxying /v1/* → %s", *listen, upstream)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}