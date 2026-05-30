package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"hestia/models"
)

func RegisterItems(r chi.Router, db *sql.DB) {
	r.Get("/api/items", func(w http.ResponseWriter, r *http.Request) {
		listItems(db, w, r)
	})
	r.Post("/api/items", func(w http.ResponseWriter, r *http.Request) {
		createItem(db, w, r)
	})
	r.Delete("/api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteItem(db, w, r)
	})
	r.Patch("/api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		togglePurchased(db, w, r)
	})
}

// scanItem scans a row into a ShoppingItem struct.
func scanItem(row *sql.Row) (*models.ShoppingItem, error) {
	var item models.ShoppingItem
	var rawPurchased []byte

	err := row.Scan(
		&item.ID,
		&item.Text,
		&rawPurchased,
		&item.CreatedAt,
		&item.PurchasedAt,
	)
	if err != nil {
		return nil, err
	}

	// SQLite stores booleans as integers; handle raw bytes
	if len(rawPurchased) > 0 {
		item.Purchased = rawPurchased[0] != 0
	}

	return &item, nil
}

// scanItemRows scans a row from *sql.Rows into a ShoppingItem struct.
func scanItemRows(rows *sql.Rows) (*models.ShoppingItem, error) {
	var item models.ShoppingItem
	var rawPurchased []byte

	err := rows.Scan(
		&item.ID,
		&item.Text,
		&rawPurchased,
		&item.CreatedAt,
		&item.PurchasedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(rawPurchased) > 0 {
		item.Purchased = rawPurchased[0] != 0
	}

	return &item, nil
}

// ── List Items (GET /api/items) ──

func listItems(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	pending := r.URL.Query().Get("pending") == "1"
	purchasedOnly := r.URL.Query().Get("purchased_only") == "1"

	query := "SELECT id, text, purchased, created_at, purchased_at FROM shopping_items"
	if pending {
		query += " WHERE purchased = 0"
	} else if purchasedOnly {
		query += " WHERE purchased = 1"
	}
	query += " ORDER BY purchased ASC, created_at DESC"

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []models.ShoppingItem{}
	for rows.Next() {
		item, err := scanItemRows(rows)
		if err != nil {
			JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items = append(items, *item)
	}

	JSON(w, http.StatusOK, map[string]any{"items": items})
}

// ── Create Item (POST /api/items) ──

func createItem(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if input.Text == "" {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	// Truncate to 255 chars to prevent abuse
	if len(input.Text) > 255 {
		input.Text = input.Text[:255]
	}

	now := time.Now()
	res, err := db.ExecContext(
		context.Background(),
		"INSERT INTO shopping_items (text, created_at) VALUES (?, ?)",
		input.Text, now,
	)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	id, _ := res.LastInsertId()

	item := models.ShoppingItem{
		ID:        int(id),
		Text:      input.Text,
		CreatedAt: now,
	}

	JSON(w, http.StatusCreated, item)
}

// ── Delete Item (DELETE /api/items/{id}) ──

func deleteItem(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	res, err := db.ExecContext(context.Background(), "DELETE FROM shopping_items WHERE id = ?", id)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		JSON(w, http.StatusNotFound, map[string]string{"error": "item not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Toggle Purchased (PATCH /api/items/{id}) ──

func togglePurchased(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var input struct {
		Purchased bool `json:"purchased"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Verify item exists
	row := db.QueryRowContext(
		context.Background(),
		"SELECT id FROM shopping_items WHERE id = ?",
		id,
	)
	if err := row.Scan(new(int)); errors.Is(err, sql.ErrNoRows) {
		JSON(w, http.StatusNotFound, map[string]string{"error": "item not found"})
		return
	} else if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var purchasedAt *time.Time
	if input.Purchased {
		now := time.Now()
		purchasedAt = &now
	}

	_, err := db.ExecContext(
		context.Background(),
		"UPDATE shopping_items SET purchased=?, purchased_at=? WHERE id=?",
		input.Purchased, purchasedAt, id,
	)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	updated, err := scanItem(db.QueryRowContext(
		context.Background(),
		"SELECT id, text, purchased, created_at, purchased_at FROM shopping_items WHERE id=?",
		id,
	))
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	JSON(w, http.StatusOK, updated)
}
