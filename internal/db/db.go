package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	// Enable WAL for concurrent read/write
	if _, err = db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err = db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer; serialise writes
	return db, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ping_events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			router_id   TEXT    NOT NULL
			             CHECK(length(router_id) BETWEEN 1 AND 64),
			received_at DATETIME NOT NULL DEFAULT (datetime('now')),
			source_ip   TEXT    NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_ping_events_router_time
			ON ping_events (router_id, received_at);
		CREATE INDEX IF NOT EXISTS idx_ping_events_received_at
			ON ping_events (received_at);
	`)
	return err
}
