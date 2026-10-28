package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// spaHandler serves static files from distDir and falls back to index.html
// for any non-file path (SPA routing). /api/* and /healthz are not handled
// here (they are registered separately) but are explicitly 404'd as a safety net.
func spaHandler(distDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/healthz" || strings.HasPrefix(path, "/api/") || path == "/api" {
			http.NotFound(w, r)
			return
		}

		// Clean relative path.
		rel := strings.TrimPrefix(path, "/")
		if rel == "" || rel == "." {
			rel = "index.html"
		}
		rel = filepath.Clean(rel)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			rel = "index.html"
		}

		full := filepath.Join(distDir, rel)
		info, err := os.Stat(full)
		if err == nil && info.Mode().IsRegular() {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			http.ServeFile(w, r, full)
			return
		}

		// SPA fallback.
		index := filepath.Join(distDir, "index.html")
		if _, err := os.Stat(index); err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeFile(w, r, index)
	})
}