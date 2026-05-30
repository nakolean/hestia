package auth

import (
	"context"
	"database/sql"

	"artemis/middleware"
)

// ValidateLogin checks credentials and returns true if valid
func ValidateLogin(db *sql.DB, username, password string) bool {
	var passwordHash string

	err := db.QueryRowContext(
		context.Background(),
		"SELECT password_hash FROM users WHERE username = ?",
		username,
	).Scan(&passwordHash)

	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		return false
	}

	return middleware.CheckPassword(password, passwordHash)
}