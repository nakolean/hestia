package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateSessionUniqueness(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		token := GenerateSession()
		if _, exists := seen[token]; exists {
			t.Fatal("duplicate session token generated")
		}
		seen[token] = struct{}{}
	}
}

func TestSessionStoreAddValidate(t *testing.T) {
	store := NewSessionStore(24 * time.Hour)
	token := store.Add()
	if !store.Validate(token) {
		t.Fatal("session should be valid immediately after creation")
	}
	if store.Validate("bogus-token") {
		t.Fatal("invalid token should not validate")
	}
}

func TestSessionStoreExpiry(t *testing.T) {
	store := NewSessionStore(10 * time.Millisecond)
	token := store.Add()
	time.Sleep(50 * time.Millisecond)
	if store.Validate(token) {
		t.Fatal("expired session should not validate")
	}
}

func TestRequireSessionWithCookie(t *testing.T) {
	store := NewSessionStore(24 * time.Hour)
	token := store.Add()
	handler := RequireSession(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/chores", nil)
	req.AddCookie(&http.Cookie{Name: "hestia_session", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireSessionNoCookie(t *testing.T) {
	store := NewSessionStore(24 * time.Hour)
	handler := RequireSession(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/chores", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireSessionHealthBypass(t *testing.T) {
	store := NewSessionStore(24 * time.Hour)
	handler := RequireSession(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health should bypass auth, got %d", rec.Code)
	}
}
