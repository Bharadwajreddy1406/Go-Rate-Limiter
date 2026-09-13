package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	return open("rate-limiter.db")
}

func OpenInMemory() (*sql.DB, error) {
	db, err := open(":memory:")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func open(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS buckets (
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
