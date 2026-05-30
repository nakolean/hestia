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

// JSON sets the content type and writes JSON response.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// ParseTime converts raw SQL timestamp bytes to *time.Time.
func ParseTime(raw []byte) *time.Time {
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

// scanChore scans a row into a Chore struct.
func scanChore(row *sql.Row) (*models.Chore, error) {
	var c models.Chore
	var rawDesc, rawLast, rawNext []byte

	err := row.Scan(
		&c.ID,
		&c.Name,
		&rawDesc,
		&c.FrequencyNum,
		&c.FrequencyUnit,
		&c.Completed,
		&rawLast,
		&rawNext,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(rawDesc) > 0 {
		c.Description = string(rawDesc)
	}
	c.LastCompleted = ParseTime(rawLast)
	c.NextDue = ParseTime(rawNext)

	return &c, nil
}

// scanChoreRows scans a row from *sql.Rows into a Chore struct.
func scanChoreRows(rows *sql.Rows) (*models.Chore, error) {
	var c models.Chore
	var rawDesc, rawLast, rawNext []byte

	err := rows.Scan(
		&c.ID,
		&c.Name,
		&rawDesc,
		&c.FrequencyNum,
		&c.FrequencyUnit,
		&c.Completed,
		&rawLast,
		&rawNext,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(rawDesc) > 0 {
		c.Description = string(rawDesc)
	}
	c.LastCompleted = ParseTime(rawLast)
	c.NextDue = ParseTime(rawNext)

	return &c, nil
}

func RegisterChores(r chi.Router, db *sql.DB) {
	r.Get("/api/chores", func(w http.ResponseWriter, r *http.Request) {
		listChores(db, w, r)
	})
	r.Post("/api/chores", func(w http.ResponseWriter, r *http.Request) {
		createChore(db, w, r)
	})
	r.Put("/api/chores/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateChore(db, w, r)
	})
	r.Delete("/api/chores/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteChore(db, w, r)
	})
	r.Post("/api/chores/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		completeChore(db, w, r)
	})
	r.Get("/api/chores/overdue", func(w http.ResponseWriter, r *http.Request) {
		overdueChores(db, w, r)
	})
}

// ── List Chores (GET /api/chores) ──

func listChores(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	query := "SELECT id, name, description, frequency_num, frequency_unit, completed, last_completed, next_due, created_at, updated_at FROM chores"
	args := []any{}
	if r.URL.Query().Get("active_only") == "1" {
		query += " WHERE completed = 0"
	}
	query += " ORDER BY next_due ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	chores := []models.Chore{}
	for rows.Next() {
		chore, err := scanChoreRows(rows)
		if err != nil {
			JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		chores = append(chores, *chore)
	}

	JSON(w, http.StatusOK, map[string]any{"chores": chores})
}

// ── Create Chore (POST /api/chores) ──

func createChore(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		FrequencyNum  int    `json:"frequency_num"`
		FrequencyUnit string `json:"frequency_unit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if input.Name == "" {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if input.FrequencyNum < 1 {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "frequency_num must be at least 1"})
		return
	}
	if !models.ValidateFrequencyUnit(input.FrequencyUnit) {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid frequency_unit, must be hours, days, or weeks"})
		return
	}

	nextDue := models.NextDue(input.FrequencyNum, input.FrequencyUnit)
	now := time.Now()

	res, err := db.ExecContext(
		context.Background(),
		"INSERT INTO chores (name, description, frequency_num, frequency_unit, last_completed, next_due, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		input.Name, input.Description, input.FrequencyNum, input.FrequencyUnit, nil, nextDue, now, now,
	)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	id, _ := res.LastInsertId()

	chore := models.Chore{
		ID:            int(id),
		Name:          input.Name,
		Description:   input.Description,
		FrequencyNum:  input.FrequencyNum,
		FrequencyUnit: input.FrequencyUnit,
		NextDue:       &nextDue,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	JSON(w, http.StatusCreated, chore)
}

// ── Update Chore (PUT /api/chores/{id}) ──

func updateChore(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var input struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		FrequencyNum  int    `json:"frequency_num"`
		FrequencyUnit string `json:"frequency_unit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if input.Name == "" {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if !models.ValidateFrequencyUnit(input.FrequencyUnit) {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid frequency_unit, must be hours, days, or weeks"})
		return
	}

	existing, err := scanChore(db.QueryRowContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, completed, last_completed, next_due, created_at, updated_at FROM chores WHERE id = ?",
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		JSON(w, http.StatusNotFound, map[string]string{"error": "chore not found"})
		return
	}
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var newNextDue *time.Time
	if input.FrequencyNum != existing.FrequencyNum || input.FrequencyUnit != existing.FrequencyUnit {
		base := existing.NextDue
		if base == nil {
			base = &time.Time{}
		}
		nextDue := models.CalculateNextDue(*base, input.FrequencyNum, input.FrequencyUnit)
		newNextDue = &nextDue
	} else {
		newNextDue = existing.NextDue
	}

	now := time.Now()
	_, err = db.ExecContext(
		context.Background(),
		"UPDATE chores SET name=?, description=?, frequency_num=?, frequency_unit=?, next_due=?, updated_at=? WHERE id=?",
		input.Name, input.Description, input.FrequencyNum, input.FrequencyUnit, newNextDue, now, id,
	)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	updated, err := scanChore(db.QueryRowContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, completed, last_completed, next_due, created_at, updated_at FROM chores WHERE id=?",
		id,
	))
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	JSON(w, http.StatusOK, updated)
}

// ── Delete Chore (DELETE /api/chores/{id}) ──

func deleteChore(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	res, err := db.ExecContext(context.Background(), "DELETE FROM chores WHERE id = ?", id)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		JSON(w, http.StatusNotFound, map[string]string{"error": "chore not found"})
		return
	}

	JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ── Complete Chore (POST /api/chores/{id}/complete) ──

func completeChore(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	existing, err := scanChore(db.QueryRowContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, completed, last_completed, next_due, created_at, updated_at FROM chores WHERE id = ?",
		id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			JSON(w, http.StatusNotFound, map[string]string{"error": "chore not found"})
			return
		}
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	chore := *existing
	chore.MarkCompleted()

	_, err = db.ExecContext(
		context.Background(),
		"UPDATE chores SET completed=1, last_completed=?, next_due=?, updated_at=? WHERE id=?",
		chore.LastCompleted, chore.NextDue, time.Now(), id,
	)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	updated, err := scanChore(db.QueryRowContext(
		context.Background(),
		"SELECT id, name, description, frequency_num, frequency_unit, completed, last_completed, next_due, created_at, updated_at FROM chores WHERE id=?",
		id,
	))
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	JSON(w, http.StatusOK, updated)
}

// ── Overdue Chores (GET /api/chores/overdue) ──

type OverdueChore struct {
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	NextDue *time.Time `json:"next_due"`
	Status  string    `json:"status"`
}

func overdueChores(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	upcoming := r.URL.Query().Get("upcoming") == "1"

	if upcoming {
		rows, err := db.QueryContext(
			context.Background(),
			"SELECT id, name, next_due, CASE WHEN next_due <= ? THEN 'overdue' ELSE 'upcoming' END FROM chores WHERE next_due <= ? AND completed = 0 ORDER BY next_due ASC",
			time.Now(), time.Now().Add(24*time.Hour),
		)
		if err != nil {
			JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		results := []OverdueChore{}
		for rows.Next() {
			var c OverdueChore
			var rawNext []byte
			err := rows.Scan(&c.ID, &c.Name, &rawNext, &c.Status)
			if err != nil {
				JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		c.NextDue = ParseTime(rawNext)
			results = append(results, c)
		}

		JSON(w, http.StatusOK, map[string]any{"chores": results})
		return
	}

	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, name, next_due FROM chores WHERE next_due <= ? AND completed = 0 ORDER BY next_due ASC",
		time.Now(),
	)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	results := []OverdueChore{}
	for rows.Next() {
		var c OverdueChore
		var rawNext []byte
		err := rows.Scan(&c.ID, &c.Name, &rawNext)
		if err != nil {
			JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.NextDue = ParseTime(rawNext)
		c.Status = "overdue"
		results = append(results, c)
	}

	JSON(w, http.StatusOK, map[string]any{"chores": results})
}

