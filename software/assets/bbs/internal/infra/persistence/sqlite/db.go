package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS boards (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			slug       TEXT NOT NULL UNIQUE,
			name       TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS threads (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			board_id      INTEGER NOT NULL REFERENCES boards(id),
			title         TEXT NOT NULL,
			post_count    INTEGER NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
			last_posted_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id  INTEGER NOT NULL REFERENCES threads(id),
			number     INTEGER NOT NULL,
			author     TEXT NOT NULL DEFAULT '名無しさん',
			body       TEXT NOT NULL,
			sage       INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}
