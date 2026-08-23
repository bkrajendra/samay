// Command server runs the samay web console: a small HTTP API in front of
// chronyd (see internal/chrony) plus the built React frontend, served from
// the same binary.
package main

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"samay-console/internal/api"
	"samay-console/internal/auth"
	"samay-console/internal/chrony"
	"samay-console/internal/config"
	"samay-console/web"
)

func main() {
	cfg := config.Load(".env")

	if cfg.Password == "" {
		log.Fatal("CONSOLE_PASSWORD is not set (set it in the environment or .env) — refusing to start with no password")
	}

	chronyClient := chrony.NewClient(cfg.ChronySocket)
	sessionStore := auth.NewStore(cfg.Username, cfg.Password)

	apiServer := &api.Server{
		Chrony: chronyClient,
		Auth:   sessionStore,
		Secure: cfg.CookieSecure,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewRouter(apiServer))
	mux.Handle("/", staticHandler())

	log.Printf("samay console listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

// staticHandler serves the embedded frontend build, falling back to
// index.html for any path that isn't a real asset — client-side routing
// needs that fallback to work on a hard refresh of a non-root URL.
func staticHandler() http.Handler {
	sub, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		log.Fatalf("static assets: %v", err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(sub, assetPath(r.URL.Path)); err != nil {
			r2 := new(http.Request)
			*r2 = *r
			u := *r.URL
			u.Path = "/"
			r2.URL = &u
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// assetPath converts a URL path into the relative path fs.Stat expects
// against the embedded FS root.
func assetPath(urlPath string) string {
	if urlPath == "" || urlPath == "/" {
		return "index.html"
	}
	return strings.TrimPrefix(urlPath, "/")
}
