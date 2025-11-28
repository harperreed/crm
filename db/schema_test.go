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

	// Verify tables exist
	tables := []string{"contacts", "companies", "deals", "deal_notes"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s not found: %v", table, err)
		}
	}
}

func TestSchemaIncludesFollowupTables(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Check contact_cadence table exists
	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='contact_cadence'`)
	var tableName string
	err := row.Scan(&tableName)
	if err != nil {
		t.Fatalf("contact_cadence table not found: %v", err)
	}

	// Check interaction_log table exists
	row = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='interaction_log'`)
	err = row.Scan(&tableName)
	if err != nil {
		t.Fatalf("interaction_log table not found: %v", err)
	}

	// Check indexes exist
	row = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_contact_cadence_priority'`)
	err = row.Scan(&tableName)
	if err != nil {
		t.Fatalf("priority index not found: %v", err)
	}
}

func TestSchemaIncludesSyncTables(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Check sync_state table exists
	row := database.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sync_state'`)
	var tableName string
	err := row.Scan(&tableName)
	if err != nil {
		t.Fatalf("sync_state table not found: %v", err)
	}

	// Check sync_log table exists
	row = database.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sync_log'`)
	err = row.Scan(&tableName)
	if err != nil {
		t.Fatalf("sync_log table not found: %v", err)
	}

	// Check suggestions table exists
	row = database.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='suggestions'`)
	err = row.Scan(&tableName)
	if err != nil {
		t.Fatalf("suggestions table not found: %v", err)
	}

	// Verify indexes
	indexes := []string{
		"idx_sync_log_source",
		"idx_sync_log_entity",
		"idx_suggestions_status",
		"idx_suggestions_type",
	}

	for _, idx := range indexes {
		row := database.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx)
		var indexName string
		err := row.Scan(&indexName)
		if err != nil {
			t.Fatalf("index %s not found: %v", idx, err)
		}
	}
}

func TestMigrateInteractionLogMetadata(t *testing.T) {
	// Test migration adds metadata column when it doesn't exist

	// Create a database with the old schema (without metadata column)
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create interaction_log table WITHOUT metadata column
	oldSchema := `
	CREATE TABLE IF NOT EXISTS interaction_log (
		id TEXT PRIMARY KEY,
		contact_id TEXT NOT NULL,
		interaction_type TEXT NOT NULL CHECK(interaction_type IN ('meeting', 'call', 'email', 'message', 'event')),
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		notes TEXT,
		sentiment TEXT CHECK(sentiment IN ('positive', 'neutral', 'negative'))
	);
	`
	_, err = db.Exec(oldSchema)
	if err != nil {
		t.Fatalf("Failed to create old schema: %v", err)
	}

	// Verify metadata column doesn't exist
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('interaction_log')
		WHERE name='metadata'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for metadata column: %v", err)
	}
	if count != 0 {
		t.Fatal("metadata column should not exist before migration")
	}

	// Run migration
	err = migrateInteractionLogMetadata(db)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify metadata column now exists
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('interaction_log')
		WHERE name='metadata'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for metadata column after migration: %v", err)
	}
	if count != 1 {
		t.Fatal("metadata column should exist after migration")
	}

	// Run migration again - should be idempotent
	err = migrateInteractionLogMetadata(db)
	if err != nil {
		t.Fatalf("Migration should be idempotent but failed on second run: %v", err)
	}

	// Verify column still exists and count is still 1
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('interaction_log')
		WHERE name='metadata'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for metadata column after second migration: %v", err)
	}
	if count != 1 {
		t.Fatal("metadata column should still exist after second migration (idempotent)")
	}
}

func TestMigrateInteractionLogMetadata_ExistingColumn(t *testing.T) {
	// Test migration is idempotent when column already exists
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// The setupTestDB already runs InitSchema which includes the metadata column
	// Verify metadata column exists
	var count int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('interaction_log')
		WHERE name='metadata'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for metadata column: %v", err)
	}
	if count != 1 {
		t.Fatal("metadata column should exist in fresh schema")
	}

	// Run migration - should succeed without error
	err = migrateInteractionLogMetadata(database)
	if err != nil {
		t.Fatalf("Migration should be idempotent but failed: %v", err)
	}

	// Verify column still exists (no duplicate)
	err = database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('interaction_log')
		WHERE name='metadata'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for metadata column after migration: %v", err)
	}
	if count != 1 {
		t.Fatal("metadata column should still exist with count of 1 (no duplicate)")
	}
}
