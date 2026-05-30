package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"hestia/middleware"
)

// JSON sets Content-Type and writes JSON response.
func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// parseTimestamp parses a raw SQLite timestamp into *time.Time.
func parseTimestamp(raw []byte) *time.Time {
	if len(raw) == 0 {
		return nil
	}
	formats := []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range formats {
		t, err := time.Parse(layout, string(raw))
		if err == nil {
			return &t
		}
	}
	return nil
}

// APIKeyResponse represents the JSON response for a single API key (no raw key).
type APIKeyResponse struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Permissions string     `json:"permissions"`
	IPWhitelist string     `json:"ip_whitelist"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// APIKeyCreateResponse wraps a key response with the one-time raw key.
type APIKeyCreateResponse struct {
	RawKey string           `json:"raw_key"`
	Key    *APIKeyResponse  `json:"key"`
}

func RegisterAPIKeys(r chi.Router, db *sql.DB) {
	r.Get("/api/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		listAdminKeys(db, w, r)
	})
	r.Post("/api/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		createAdminKey(db, w, r)
	})
	r.Delete("/api/admin/keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteAdminKey(db, w, r)
	})
	r.Put("/api/admin/keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateAdminKey(db, w, r)
	})
}

// ── List Keys (GET /api/admin/keys) ──

func listAdminKeys(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, name, permissions, ip_whitelist, expires_at, last_used_at, created_at FROM api_keys ORDER BY created_at DESC",
	)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []*APIKeyResponse
	for rows.Next() {
		item, err := scanAdminKeyRow(rows)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		results = append(results, item)
	}

	if results == nil {
		results = []*APIKeyResponse{}
	}
	jsonResponse(w, http.StatusOK, map[string]any{"keys": results})
}

// ── Create Key (POST /api/admin/keys) ──

func createAdminKey(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string   `json:"name"`
		Permissions string   `json:"permissions"`
		IPWhitelist string   `json:"ip_whitelist"`
		ExpiresAt   *ISOTime `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if input.Name == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if input.Permissions == "" || !(input.Permissions == "read" || input.Permissions == "write" || input.Permissions == "full") {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "permissions must be read, write, or full"})
		return
	}

	rawKey, err := middleware.GenerateRawKey()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate key"})
		return
	}

	hashed, err := middleware.HashKey(rawKey)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash key"})
		return
	}

	var expiresAt *time.Time
	if input.ExpiresAt != nil && !input.ExpiresAt.Time.IsZero() {
		expiresAt = &input.ExpiresAt.Time
	}

	now := time.Now()
	res, err := db.ExecContext(
		context.Background(),
		"INSERT INTO api_keys (key_hash, name, permissions, ip_whitelist, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		string(hashed), input.Name, input.Permissions, input.IPWhitelist, expiresAt, now,
	)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	id, _ := res.LastInsertId()

	keyResp := &APIKeyResponse{
		ID:          int(id),
		Name:        input.Name,
		Permissions: input.Permissions,
		IPWhitelist: input.IPWhitelist,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	jsonResponse(w, http.StatusCreated, APIKeyCreateResponse{
		RawKey: rawKey,
		Key:    keyResp,
	})
}

// ── Delete Key (DELETE /api/admin/keys/{id}) ──

func deleteAdminKey(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminID(r)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	res, err := db.ExecContext(context.Background(), "DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Update Key (PUT /api/admin/keys/{id}) ──

func updateAdminKey(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminID(r)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var input struct {
		Permissions *string  `json:"permissions"`
		ExpiresAt   *ISOTime `json:"expires_at"`
		ExpireNow   *bool    `json:"expire_now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	setters := []string{}
	var args []any

	if input.Permissions != nil {
		if !( *input.Permissions == "read" || *input.Permissions == "write" || *input.Permissions == "full" ) {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "permissions must be read, write, or full"})
			return
		}
		setters = append(setters, "permissions = ?")
		args = append(args, *input.Permissions)
	}

	if input.ExpiresAt != nil {
		setters = append(setters, "expires_at = ?")
		args = append(args, input.ExpiresAt.Time)
	}

	if input.ExpireNow != nil && *input.ExpireNow {
		now := time.Now()
		setters = append(setters, "expires_at = ?")
		args = append(args, now)
	}

	if len(setters) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		return
	}

	query := fmt.Sprintf("UPDATE api_keys SET %s WHERE id = ?", strings.Join(setters, ", "))
	args = append(args, id)

	_, err = db.ExecContext(context.Background(), query, args...)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Helpers ──

func parseAdminID(r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return 0, err
	}
	return id, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAdminKeyRow(row rowScanner) (*APIKeyResponse, error) {
	var key APIKeyResponse
	var rawExpires, rawLastUsed, rawIP, rawCreatedAt []byte

	err := row.Scan(&key.ID, &key.Name, &key.Permissions, &rawIP, &rawExpires, &rawLastUsed, &rawCreatedAt)
	if err != nil {
		return nil, err
	}

	key.ExpiresAt = parseTimestamp(rawExpires)
	key.LastUsedAt = parseTimestamp(rawLastUsed)
	if len(rawIP) > 0 {
		key.IPWhitelist = string(rawIP)
	}
	if len(rawCreatedAt) > 0 {
		t := parseTimestamp(rawCreatedAt)
		if t != nil {
			key.CreatedAt = *t
		}
	}

	return &key, nil
}

// ISOTime is a helper that parses ISO 8601 / RFC3339 timestamps from JSON.
type ISOTime struct {
	time.Time
}

func (t *ISOTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	formats := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05"}
	for _, layout := range formats {
		parsed, err := time.Parse(layout, s)
		if err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("cannot parse time: %s", s)
}
