package db

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS chores (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			name            TEXT    NOT NULL,
			description     TEXT,
			frequency_num   INTEGER NOT NULL,
			frequency_unit  TEXT    NOT NULL,
			completed       BOOLEAN DEFAULT 0,
			last_completed  TIMESTAMP,
			next_due        TIMESTAMP,
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS shopping_items (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			text          TEXT    NOT NULL,
			purchased     BOOLEAN DEFAULT 0,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			purchased_at  TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS api_keys (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			key_hash       TEXT    NOT NULL UNIQUE,
			name           TEXT    NOT NULL,
			permissions    TEXT    NOT NULL,
			ip_whitelist   TEXT,
			expires_at     TIMESTAMP,
			last_used_at   TIMESTAMP,
			created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT    NOT NULL UNIQUE,
			password_hash TEXT   NOT NULL,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	return nil
}
