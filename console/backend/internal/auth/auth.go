// Package auth implements a minimal single-user session system: a hardcoded
// username/password (from environment/.env) gate a server-side session,
// handed to the browser as an HttpOnly cookie. There is no user database
// and no multi-user support — this is an admin tool for one operator.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	CookieName = "samay_session"
	sessionTTL = 24 * time.Hour
)

type session struct {
	username  string
	expiresAt time.Time
}

// Store is an in-memory session store. It is intentionally simple: a
// single-replica admin console does not need a distributed session store,
// and losing sessions on pod restart is an acceptable trade-off.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]session

	username string
	password string
}

func NewStore(username, password string) *Store {
	return &Store{
		sessions: make(map[string]session),
		username: username,
		password: password,
	}
}

// Authenticate checks credentials with constant-time comparison and, if
// valid, issues a new session token.
func (s *Store) Authenticate(username, password string) (token string, ok bool) {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) == 1
	if !userOK || !passOK {
		return "", false
	}

	token = newToken()
	s.mu.Lock()
	s.sessions[token] = session{username: username, expiresAt: time.Now().Add(sessionTTL)}
	s.mu.Unlock()
	return token, true
}

func (s *Store) Invalidate(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *Store) Username(token string) (string, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.expiresAt) {
		return "", false
	}
	return sess.username, true
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: failed to read random bytes: " + err.Error())
	}
	return hex.EncodeToString(b)
}

type contextKey string

const usernameContextKey contextKey = "username"

// RequireSession is middleware that rejects requests without a valid
// session cookie, and otherwise attaches the authenticated username to the
// request context.
func (s *Store) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		username, ok := s.Username(cookie.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), usernameContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UsernameFromContext returns the authenticated username set by
// RequireSession.
func UsernameFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(usernameContextKey).(string); ok {
		return v
	}
	return ""
}
