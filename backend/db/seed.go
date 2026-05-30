package db

import (
	"context"
	"database/sql"
	"log"
	"os"

	"hestia/middleware"
)

func SeedInitialUser(db *sql.DB) error {
	username := os.Getenv("USER_NAME")
	password := os.Getenv("USER_PASS")

	if username == "" || password == "" {
		return nil
	}

	var count int
	err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	hash, err := middleware.HashPassword(password)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(context.Background(),
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, hash,
	)
	if err != nil {
		return err
	}

	log.Printf("Seeded initial user: %s", username)
	return nil
}
