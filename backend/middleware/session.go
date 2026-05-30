package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SessionStore holds session tokens in-memory.
type SessionStore struct {
	sessions map[string]time.Time
	mu       sync.RWMutex
	expiry   time.Duration
}

// NewSessionStore creates a session store with the given token lifetime.
func NewSessionStore(expiry time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]time.Time),
		expiry:   expiry,
	}
}

// GenerateSession creates a random hex token (16 bytes = 32 hex chars).
func GenerateSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Add creates a new session token and returns it.
func (s *SessionStore) Add() string {
	token := GenerateSession()
	s.mu.Lock()
	s.sessions[token] = time.Now().Add(s.expiry)
	s.mu.Unlock()
	return token
}

// Validate checks that the token exists and is not expired.
func (s *SessionStore) Validate(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expiresAt) {
		delete(s.sessions, token)
		return false
	}
	s.sessions[token] = time.Now().Add(s.expiry)
	return true
}

// Cleanup removes expired sessions.
func (s *SessionStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, expiresAt := range s.sessions {
		if now.After(expiresAt) {
			delete(s.sessions, token)
		}
	}
}

const sessionCookieName = "hestia_session"

// SetSessionCookie sets the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

// GetSessionCookie retrieves the session cookie from the request.
func GetSessionCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// RequireSession is a middleware that checks for a valid session cookie.
func RequireSession(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			sessionID, err := GetSessionCookie(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error":"unauthorized"}`)
				return
			}
			if !store.Validate(sessionID) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error":"unauthorized"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
