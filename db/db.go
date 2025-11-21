// ABOUTME: Database connection management and initialization
// ABOUTME: Handles opening SQLite database with WAL mode at XDG path
package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func OpenDatabase(path string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open database with WAL mode
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Initialize schema
	if err := InitSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
