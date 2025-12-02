package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestObjectsTableCreation(t *testing.T) {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer func() { _ = database.Close() }()

	if err := InitSchema(database); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Verify objects table exists
	var tableName string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='objects'").Scan(&tableName)
	if err != nil {
		t.Fatalf("objects table not found: %v", err)
	}
	if tableName != "objects" {
		t.Errorf("Expected table name 'objects', got %s", tableName)
	}
}
