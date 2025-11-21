package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify schema was initialized
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	if count < 4 {
		t.Errorf("Expected at least 4 tables, got %d", count)
	}
}
