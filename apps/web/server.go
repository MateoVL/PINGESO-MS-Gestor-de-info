package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	port := os.Getenv("WEB_PORT")
	if port == "" {
		port = "3000"
	}
	apiURL := os.Getenv("API_INTERNAL_URL")
	if apiURL == "" {
		apiURL = "http://backend:8080"
	}
	target, err := url.Parse(apiURL)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	distDir := "/app/dist"
	fs := http.FileServer(http.Dir(distDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			proxy.ServeHTTP(w, r)
			return
		}
		cleanPath := filepath.Clean(r.URL.Path)
		fullPath := filepath.Join(distDir, cleanPath)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})

	_ = http.ListenAndServe(":"+port, mux)
}

