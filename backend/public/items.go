package public

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"artemis/api"
	"github.com/go-chi/chi/v5"
)

// PublicItem represents a shopping item for the public API
type PublicItem struct {
	ID          int        `json:"id"`
	Text        string     `json:"text"`
	Purchased   bool       `json:"purchased"`
	CreatedAt   time.Time  `json:"created_at"`
	PurchasedAt *time.Time `json:"purchased_at"`
}

// RegisterPublicItems registers public item read-endpoints
func RegisterPublicItems(r chi.Router, db *sql.DB) {
	r.Get("/api/v1/public/items", func(w http.ResponseWriter, r *http.Request) {
		listPublicItems(db, w, r)
	})
	r.Get("/api/v1/public/items/pending", func(w http.ResponseWriter, r *http.Request) {
		publicPendingItems(db, w, r)
	})
}

// ── GET /api/v1/public/items ──

func listPublicItems(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, text, purchased, created_at, purchased_at FROM shopping_items ORDER BY purchased ASC, created_at DESC",
	)
	if err != nil {
		api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []PublicItem{}
	for rows.Next() {
		var item PublicItem
		var purchased int64
		var createdAt []byte
		var purchasedAt []byte

		if err := rows.Scan(&item.ID, &item.Text, &purchased, &createdAt, &purchasedAt); err != nil {
			api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		item.Purchased = purchased == 1
		if t := api.ParseTime(createdAt); t != nil {
			item.CreatedAt = *t
		}
		item.PurchasedAt = api.ParseTime(purchasedAt)

		items = append(items, item)
	}

	if items == nil {
		items = []PublicItem{}
	}

	api.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// ── GET /api/v1/public/items/pending ──

func publicPendingItems(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, text, purchased, created_at, purchased_at FROM shopping_items WHERE purchased = 0 ORDER BY created_at DESC",
	)
	if err != nil {
		api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []PublicItem{}
	for rows.Next() {
		var item PublicItem
		var purchased int64
		var createdAt []byte
		var purchasedAt []byte

		if err := rows.Scan(&item.ID, &item.Text, &purchased, &createdAt, &purchasedAt); err != nil {
			api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		item.Purchased = purchased == 1
		if t := api.ParseTime(createdAt); t != nil {
			item.CreatedAt = *t
		}
		item.PurchasedAt = api.ParseTime(purchasedAt)

		items = append(items, item)
	}

	if items == nil {
		items = []PublicItem{}
	}

	api.JSON(w, http.StatusOK, map[string]any{"items": items})
}
