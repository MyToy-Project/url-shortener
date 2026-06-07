package index

import (
	"net/http"
	"os"
	"path/filepath"
	"url-shortener/internal/url"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	url.CountUpTotalRequestCounter(r.RequestURI)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Prefer current working directory (common during `go run .` from project root)
	wd, _ := os.Getwd()
	p1 := filepath.Join(wd, "index.html")
	if _, err := os.Stat(p1); err == nil {
		http.ServeFile(w, r, p1)
		return
	}

	http.Error(w, "index.html not found", http.StatusNotFound)
}
