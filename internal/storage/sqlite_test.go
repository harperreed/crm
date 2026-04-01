// ABOUTME: Tests for SQLite store initialization, schema creation, and FTS5 triggers.
// ABOUTME: Verifies table structure, index presence, and full-text search trigger behavior.
package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// newTestStore creates a SqliteStore in a temp directory and registers cleanup.
func newTestStore(t *testing.T) *SqliteStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSqliteStore(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestNewSqliteStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")

	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSqliteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	if store.db == nil {
		t.Fatal("expected non-nil db")
	}

	if store.dbPath != dbPath {
		t.Errorf("expected dbPath %q, got %q", dbPath, store.dbPath)
	}

	// Verify parent directories were created
	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("expected parent directories to be created")
	}
}

func TestStoreTables(t *testing.T) {
	store := newTestStore(t)

	expected := []string{"contacts", "companies", "relationships"}
	for _, table := range expected {
		var name string
		err := store.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// Verify indexes
	indexes := []string{
		"idx_contacts_id",
		"idx_companies_id",
		"idx_relationships_source_id",
		"idx_relationships_target_id",
	}
	for _, idx := range indexes {
		var name string
		err := store.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %q not found: %v", idx, err)
		}
	}

	// Verify FTS5 virtual tables exist
	ftsTables := []string{"contacts_fts", "companies_fts"}
	for _, table := range ftsTables {
		var name string
		err := store.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("FTS5 table %q not found: %v", table, err)
		}
	}
}

func TestStoreFTSTriggers(t *testing.T) {
	store := newTestStore(t)

	triggers := []string{
		"contacts_ai", "contacts_ad", "contacts_au",
		"companies_ai", "companies_ad", "companies_au",
	}
	for _, trig := range triggers {
		var name string
		err := store.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='trigger' AND name=?", trig,
		).Scan(&name)
		if err != nil {
			t.Errorf("trigger %q not found: %v", trig, err)
		}
	}

	// Insert a contact and verify FTS5 sync
	_, err := store.db.Exec(`
		INSERT INTO contacts (id, name, email, phone, fields, tags, created_at, updated_at)
		VALUES ('test-id', 'Alice Smith', 'alice@example.com', '', '{}', '[]', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("insert contact: %v", err)
	}

	var count int
	err = store.db.QueryRow(
		"SELECT count(*) FROM contacts_fts WHERE contacts_fts MATCH ?", `"Alice"`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("FTS query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 FTS match for Alice, got %d", count)
	}

	// Delete the contact and verify FTS5 cleanup
	_, err = store.db.Exec(`DELETE FROM contacts WHERE id = 'test-id'`)
	if err != nil {
		t.Fatalf("delete contact: %v", err)
	}

	err = store.db.QueryRow(
		"SELECT count(*) FROM contacts_fts WHERE contacts_fts MATCH ?", `"Alice"`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("FTS query after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 FTS matches after delete, got %d", count)
	}
}

func TestDataDir(t *testing.T) {
	// With XDG_DATA_HOME set
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-test")
	dir := DataDir()
	expected := filepath.Join("/tmp/xdg-test", "crm")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}

	// Without XDG_DATA_HOME (falls back to ~/.local/share)
	t.Setenv("XDG_DATA_HOME", "")
	dir = DataDir()
	home, _ := os.UserHomeDir()
	expected = filepath.Join(home, ".local", "share", "crm")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestStoreCloseNilDB(t *testing.T) {
	store := &SqliteStore{db: nil}
	err := store.Close()
	if err != nil {
		t.Errorf("Close on nil db should return nil, got %v", err)
	}
}

func TestEscapeFTS5Query(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", `"hello world"`},
		{`test"quote`, `"test""quote"`},
		{"simple", `"simple"`},
	}
	for _, tt := range tests {
		got := escapeFTS5Query(tt.input)
		if got != tt.want {
			t.Errorf("escapeFTS5Query(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStoreForeignKeys(t *testing.T) {
	store := newTestStore(t)

	var fkEnabled int
	err := store.db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fkEnabled)
	}
}

func TestStoreWALMode(t *testing.T) {
	store := newTestStore(t)

	var journalMode string
	err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", journalMode)
	}
}

// tableColumnExists checks if a column exists in a table.
func tableColumnExists(db *sql.DB, table, column string) bool {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

func TestStoreContactsColumns(t *testing.T) {
	store := newTestStore(t)

	cols := []string{"id", "name", "email", "phone", "fields", "tags", "created_at", "updated_at"}
	for _, col := range cols {
		if !tableColumnExists(store.db, "contacts", col) {
			t.Errorf("contacts table missing column %q", col)
		}
	}
}

func TestStoreCompaniesColumns(t *testing.T) {
	store := newTestStore(t)

	cols := []string{"id", "name", "domain", "fields", "tags", "created_at", "updated_at"}
	for _, col := range cols {
		if !tableColumnExists(store.db, "companies", col) {
			t.Errorf("companies table missing column %q", col)
		}
	}
}

func TestStoreRelationshipsColumns(t *testing.T) {
	store := newTestStore(t)

	cols := []string{"id", "source_id", "target_id", "type", "context", "created_at"}
	for _, col := range cols {
		if !tableColumnExists(store.db, "relationships", col) {
			t.Errorf("relationships table missing column %q", col)
		}
	}
}
