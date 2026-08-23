package api

import (
	"log"
	"net/http"

	"samay-console/internal/auth"
)

func (s *Server) audit(r *http.Request, action string, err error) {
	user := "unknown"
	if cookie, cerr := r.Cookie(auth.CookieName); cerr == nil {
		if u, ok := s.Auth.Username(cookie.Value); ok {
			user = u
		}
	}
	if err != nil {
		log.Printf("audit: user=%s action=%s result=error err=%v", user, action, err)
		return
	}
	log.Printf("audit: user=%s action=%s result=ok", user, action)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	err := s.Chrony.ForceSync(r.Context())
	s.audit(r, "force-sync", err)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to force sync", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "ok"})
}

func (s *Server) handleStep(w http.ResponseWriter, r *http.Request) {
	err := s.Chrony.StepClock(r.Context())
	s.audit(r, "step-clock", err)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to step clock", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "ok"})
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	err := s.Chrony.RestartService(r.Context())
	s.audit(r, "restart-service", err)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restart chronyd", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "ok"})
}
