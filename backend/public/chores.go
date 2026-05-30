package public

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"artemis/api"
	"artemis/models"
	"github.com/go-chi/chi/v5"
)

// PublicChore represents a chore for the public API
type PublicChore struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Frequency   string     `json:"frequency"`
	NextDue     *time.Time `json:"next_due"`
	Status      string     `json:"status"`
}

// RegisterPublicChores registers public chore read-endpoints
func RegisterPublicChores(r chi.Router, db *sql.DB) {
	r.Get("/api/v1/public/chores", func(w http.ResponseWriter, r *http.Request) {
		listPublicChores(db, w, r)
	})
	r.Get("/api/v1/public/chores/overdue", func(w http.ResponseWriter, r *http.Request) {
		publicOverdueChores(db, w, r)
	})
	r.Get("/api/v1/public/chores/next", func(w http.ResponseWriter, r *http.Request) {
		publicNextChores(db, w, r)
	})
}

// ── GET /api/v1/public/chores ──

func listPublicChores(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, next_due FROM chores WHERE completed = 0 ORDER BY next_due ASC",
	)
	if err != nil {
		api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	chores := []PublicChore{}
	for rows.Next() {
		var chore PublicChore
		var desc []byte
		var freqNum int
		var freqUnit string
		var rawNextDue []byte

		if err := rows.Scan(&chore.ID, &chore.Name, &desc, &freqNum, &freqUnit, &rawNextDue); err != nil {
			api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		chore.Description = string(desc)
		chore.Frequency = models.FormatFrequency(freqNum, freqUnit)
		chore.NextDue = api.ParseTime(rawNextDue)
		chore.Status = computeStatus(chore.NextDue)

		chores = append(chores, chore)
	}

	if chores == nil {
		chores = []PublicChore{}
	}

	api.JSON(w, http.StatusOK, map[string]any{"chores": chores})
}

// ── GET /api/v1/public/chores/overdue ──

func publicOverdueChores(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, next_due FROM chores WHERE next_due <= ? AND completed = 0 ORDER BY next_due ASC",
		time.Now(),
	)
	if err != nil {
		api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	chores := []PublicChore{}
	for rows.Next() {
		var chore PublicChore
		var desc []byte
		var freqNum int
		var freqUnit string
		var rawNextDue []byte

		if err := rows.Scan(&chore.ID, &chore.Name, &desc, &freqNum, &freqUnit, &rawNextDue); err != nil {
			api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		chore.Description = string(desc)
		chore.Frequency = models.FormatFrequency(freqNum, freqUnit)
		chore.NextDue = api.ParseTime(rawNextDue)
		chore.Status = "overdue"

		chores = append(chores, chore)
	}

	if chores == nil {
		chores = []PublicChore{}
	}

	api.JSON(w, http.StatusOK, map[string]any{"chores": chores})
}

// ── GET /api/v1/public/chores/next ──

func publicNextChores(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	n := 5
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		parsed, err := strconv.Atoi(nStr)
		if err != nil {
			api.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid n parameter"})
			return
		}
		if parsed > 50 {
			parsed = 50
		}
		n = parsed
	}

	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, next_due FROM chores WHERE completed = 0 ORDER BY next_due ASC LIMIT ?",
		n,
	)
	if err != nil {
		api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	chores := []PublicChore{}
	for rows.Next() {
		var chore PublicChore
		var desc []byte
		var freqNum int
		var freqUnit string
		var rawNextDue []byte

		if err := rows.Scan(&chore.ID, &chore.Name, &desc, &freqNum, &freqUnit, &rawNextDue); err != nil {
			api.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		chore.Description = string(desc)
		chore.Frequency = models.FormatFrequency(freqNum, freqUnit)
		chore.NextDue = api.ParseTime(rawNextDue)
		chore.Status = computeStatus(chore.NextDue)

		chores = append(chores, chore)
	}

	if chores == nil {
		chores = []PublicChore{}
	}

	api.JSON(w, http.StatusOK, map[string]any{"chores": chores})
}

// computeStatus determines the status of a chore based on next_due
func computeStatus(nextDue *time.Time) string {
	if nextDue == nil {
		return "unknown"
	}
	now := time.Now()
	if nextDue.Before(now) {
		return "overdue"
	}
	if nextDue.Before(now.Add(time.Hour)) {
		return "due_soon"
	}
	return "on_track"
}
