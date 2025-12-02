// ABOUTME: Tests for database schema creation and migrations
// ABOUTME: Uses in-memory SQLite for fast isolated tests
package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify objects table exists
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='objects'").Scan(&name)
	if err != nil {
		t.Errorf("Table objects not found: %v", err)
	}

	// Verify relationships table exists
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='relationships'").Scan(&name)
	if err != nil {
		t.Errorf("Table relationships not found: %v", err)
	}

	// Verify objects indexes
	objectIndexes := []string{
		"idx_objects_type",
		"idx_objects_created_at",
	}
	for _, idx := range objectIndexes {
		var indexName string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&indexName)
		if err != nil {
			t.Errorf("Index %s not found: %v", idx, err)
		}
	}

	// Verify relationships indexes
	relationshipIndexes := []string{
		"idx_relationships_source",
		"idx_relationships_target",
		"idx_relationships_type",
	}
	for _, idx := range relationshipIndexes {
		var indexName string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&indexName)
		if err != nil {
			t.Errorf("Index %s not found: %v", idx, err)
		}
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Errorf("Failed to check foreign key status: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("Foreign keys not enabled: got %d, want 1", fkEnabled)
	}
}
