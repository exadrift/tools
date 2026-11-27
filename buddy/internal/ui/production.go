package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:production
var productionFS embed.FS

// Handler serves the production UI at the root path
func Handler() http.Handler {
	sub, err := fs.Sub(productionFS, "production")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServerFS(sub)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			// Serve production.html at root
			r.URL.Path = "/production.html"
		}
		fileServer.ServeHTTP(w, r)
	})
}
