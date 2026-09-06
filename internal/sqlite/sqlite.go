package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// An in-memory SQLite database belongs to one connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	_, err = db.Exec(`
		CREATE TABLE buckets (
			key            TEXT PRIMARY KEY,
			tokens         INTEGER NOT NULL CHECK (tokens >= 0),
			last_refill_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create buckets table: %w", err)
	}

	return db, nil
}
