// ABOUTME: Shared test utilities for sync package
// ABOUTME: Provides common test database setup and helper functions
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/harperreed/pagen/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	if err := db.InitSchema(database); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return database
}
