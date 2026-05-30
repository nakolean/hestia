package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"artemis/db"
	"artemis/middleware"
)

type testSetup struct {
	DB             *sql.DB
	SessionStore   *middleware.SessionStore
	SessionToken   string
}

func setupTest(t *testing.T) *testSetup {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	db.Migrate(sqlDB)

	store := middleware.NewSessionStore(86400 * 60 * 60)
	token := store.Add()

	return &testSetup{DB: sqlDB, SessionStore: store, SessionToken: token}
}

func (ts *testSetup) request(t *testing.T, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.AddCookie(&http.Cookie{Name: "artemis_session", Value: ts.SessionToken})
	rec := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Use(middleware.RequireSession(ts.SessionStore))
	RegisterChores(router, ts.DB)
	router.ServeHTTP(rec, r)
	return rec
}

func TestChoreCRUD(t *testing.T) {
	ts := setupTest(t)
	defer ts.DB.Close()

	// ── Test: Create Chore (POST /api/chores) ──
	rec := ts.request(t, http.MethodPost, "/api/chores", []byte(`{"name":"Vacuum","description":"Living room","frequency_num":2,"frequency_unit":"days"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/chores: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	if created["name"] != "Vacuum" {
		t.Fatalf("Created chore name mismatch: %v", created["name"])
	}

	// ── Test: List Chores (GET /api/chores) ──
	rec = ts.request(t, http.MethodGet, "/api/chores", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/chores: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var listResp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&listResp)
	chores := listResp["chores"].([]interface{})
	if len(chores) != 1 {
		t.Fatalf("Expected 1 chore, got %d", len(chores))
	}

	// ── Test: Update Chore (PUT /api/chores/{id}) ──
	rec = ts.request(t, http.MethodPut, "/api/chores/1", []byte(`{"name":"Vacuum Living Room","description":"Updated description","frequency_num":3,"frequency_unit":"days"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/chores/1: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var updated map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&updated)
	if updated["name"] != "Vacuum Living Room" {
		t.Fatalf("Updated chore name mismatch: %v", updated["name"])
	}

	// ── Test: Complete Chore (POST /api/chores/{id}/complete) ──
	rec = ts.request(t, http.MethodPost, "/api/chores/1/complete", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/chores/1/complete: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var completedChore map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&completedChore)
	if completedChore["completed"] != true {
		t.Fatalf("Completed chore should have completed=true")
	}

	// ── Test: Overdue Chores (GET /api/chores/overdue) ──
	rec = ts.request(t, http.MethodGet, "/api/chores/overdue", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/chores/overdue: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// ── Test: Delete Chore (DELETE /api/chores/{id}) ──
	rec = ts.request(t, http.MethodDelete, "/api/chores/1", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/chores/1: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// ── Test: Delete non-existent chore returns 404 ──
	rec = ts.request(t, http.MethodDelete, "/api/chores/999", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE /api/chores/999: expected 404, got %d", rec.Code)
	}
}

func TestChoreValidation(t *testing.T) {
	ts := setupTest(t)
	defer ts.DB.Close()

	// Empty name -> 400
	rec := ts.request(t, http.MethodPost, "/api/chores", []byte(`{"name":"","frequency_num":1,"frequency_unit":"days"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Empty name: expected 400, got %d", rec.Code)
	}

	// Invalid frequency unit -> 400
	rec = ts.request(t, http.MethodPost, "/api/chores", []byte(`{"name":"Test","frequency_num":1,"frequency_unit":"forever"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Invalid frequency_unit: expected 400, got %d", rec.Code)
	}

	// Negative frequency_num -> 400
	rec = ts.request(t, http.MethodPost, "/api/chores", []byte(`{"name":"Test","frequency_num":-1,"frequency_unit":"days"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Negative frequency_num: expected 400, got %d", rec.Code)
	}
}
