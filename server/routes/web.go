package routes

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// WebHandler serves the built web UI (with SPA fallback) or a placeholder if assets are missing.
func WebHandler(mode, staticDir string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve static asset; if missing, fall back to index.html; if dir missing, return placeholder.
		if _, err := os.Stat(staticDir); err != nil {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "webapp placeholder (mode=%s)\n", mode)
			return
		}

		cleanPath := filepath.Clean(r.URL.Path)
		target := filepath.Join(staticDir, cleanPath)

		if info, err := os.Stat(target); err != nil || info.IsDir() {
			// SPA-style fallback to index.html
			r.URL.Path = "/"
		}

		fs.ServeHTTP(w, r)
	})
}

