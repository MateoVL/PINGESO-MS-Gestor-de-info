package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var version = "dev"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("WEB_PORT")
	}
	if port == "" {
		port = "8080"
	}

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = os.Getenv("API_INTERNAL_URL")
	}
	if backendURL == "" {
		backendURL = "http://backend:8080"
	}

	target, err := url.Parse(backendURL)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "/srv"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newSPAHandler(staticDir, proxy),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	_ = srv.ListenAndServe()
}

func newSPAHandler(staticDir string, proxy http.Handler) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if r.URL.Path == "/healthz" {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","version":"` + version + `"}`))
			return
		}

		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" || r.URL.Path == "/readyz" {
			proxy.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		cleanPath := filepath.Clean(r.URL.Path)
		fullPath := filepath.Join(staticDir, cleanPath)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})
}
