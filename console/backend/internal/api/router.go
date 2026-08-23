package api

import (
	"net/http"

	"samay-console/internal/auth"
	"samay-console/internal/chrony"
)

// Server holds the dependencies HTTP handlers need.
type Server struct {
	Chrony *chrony.Client
	Auth   *auth.Store
	// Secure marks session cookies Secure; enable once served over TLS.
	Secure bool
}

func NewRouter(s *Server) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/session", s.handleSession)

	protect := func(h http.HandlerFunc) http.Handler {
		return s.Auth.RequireSession(h)
	}

	mux.Handle("GET /api/status", protect(s.handleStatus))
	mux.Handle("GET /api/tracking", protect(s.handleTracking))
	mux.Handle("GET /api/sources", protect(s.handleSources))
	mux.Handle("GET /api/clients", protect(s.handleClients))
	mux.Handle("GET /api/diagnostics", protect(s.handleDiagnostics))
	mux.Handle("POST /api/sync", protect(s.handleSync))
	mux.Handle("POST /api/clock/step", protect(s.handleStep))
	mux.Handle("POST /api/service/restart", protect(s.handleRestart))

	return mux
}
