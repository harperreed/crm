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
	defer func() { _ = db.Close() }()

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

	// Verify WAL mode
	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("Expected WAL mode, got %s", mode)
	}
}

func TestOpenDatabaseInvalidPath(t *testing.T) {
	// Try to create database in a path that cannot be created
	// On Unix systems, attempting to write to /root/pagen-crm without permissions should fail
	dbPath := "/invalid/nonexistent/path/that/cannot/be/created/test.db"

	_, err := OpenDatabase(dbPath)
	if err == nil {
		t.Errorf("Expected error for invalid path, but OpenDatabase succeeded")
	}
}

func TestOpenDatabaseSchemaInitFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// First, create a valid database to ensure the file exists
	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Initial OpenDatabase failed: %v", err)
	}
	_ = db.Close()

	// Now try to open the database again - schema initialization should handle existing tables gracefully
	// This tests that "CREATE TABLE IF NOT EXISTS" statements don't fail
	db, err = OpenDatabase(dbPath)
	if err != nil {
		t.Errorf("OpenDatabase should handle re-initialization gracefully, but got error: %v", err)
	}
	if db != nil {
		defer func() { _ = db.Close() }()
	}

	// Verify tables still exist after re-initialization
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query tables after re-initialization: %v", err)
	}
	if count < 4 {
		t.Errorf("Expected at least 4 tables after re-initialization, got %d", count)
	}
}
