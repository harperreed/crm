// ABOUTME: SQLite storage backend implementing the Storage interface for the CRM.
// ABOUTME: Handles database initialization, schema creation, FTS5 setup, and lifecycle.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// SqliteStore implements Storage using a SQLite database.
type SqliteStore struct {
	db     *sql.DB
	dbPath string
}

// NewSqliteStore creates a new SqliteStore, ensuring parent directories exist,
// opening the database with foreign keys and WAL mode, and initializing the schema.
func NewSqliteStore(dbPath string) (*SqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		return nil, fmt.Errorf("create parent dirs: %w", err)
	}

	dsn := dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	store := &SqliteStore{db: db, dbPath: dbPath}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// initSchema creates all tables, indexes, FTS5 virtual tables, and triggers
// inside a single transaction.
func (s *SqliteStore) initSchema() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmts := append(tableStatements(), ftsStatements()...)
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("exec schema statement: %w", err)
		}
	}

	return tx.Commit()
}

// tableStatements returns DDL for core tables and indexes.
func tableStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS contacts (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			email TEXT DEFAULT '',
			phone TEXT DEFAULT '',
			fields TEXT DEFAULT '{}',
			tags TEXT DEFAULT '[]',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS companies (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			domain TEXT DEFAULT '',
			fields TEXT DEFAULT '{}',
			tags TEXT DEFAULT '[]',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS relationships (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			target_id TEXT NOT NULL,
			type TEXT NOT NULL,
			context TEXT DEFAULT '',
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_id ON contacts(id)`,
		`CREATE INDEX IF NOT EXISTS idx_companies_id ON companies(id)`,
		`CREATE INDEX IF NOT EXISTS idx_relationships_source_id ON relationships(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relationships_target_id ON relationships(target_id)`,
	}
}

// ftsStatements returns DDL for FTS5 virtual tables and sync triggers.
func ftsStatements() []string {
	return []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS contacts_fts USING fts5(name, email, fields, content=contacts, content_rowid=rowid)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS companies_fts USING fts5(name, domain, fields, content=companies, content_rowid=rowid)`,
		`CREATE TRIGGER IF NOT EXISTS contacts_ai AFTER INSERT ON contacts BEGIN
			INSERT INTO contacts_fts(rowid, name, email, fields) VALUES (new.rowid, new.name, new.email, new.fields);
		END`,
		`CREATE TRIGGER IF NOT EXISTS contacts_ad AFTER DELETE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, name, email, fields) VALUES ('delete', old.rowid, old.name, old.email, old.fields);
		END`,
		`CREATE TRIGGER IF NOT EXISTS contacts_au AFTER UPDATE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, name, email, fields) VALUES ('delete', old.rowid, old.name, old.email, old.fields);
			INSERT INTO contacts_fts(rowid, name, email, fields) VALUES (new.rowid, new.name, new.email, new.fields);
		END`,
		`CREATE TRIGGER IF NOT EXISTS companies_ai AFTER INSERT ON companies BEGIN
			INSERT INTO companies_fts(rowid, name, domain, fields) VALUES (new.rowid, new.name, new.domain, new.fields);
		END`,
		`CREATE TRIGGER IF NOT EXISTS companies_ad AFTER DELETE ON companies BEGIN
			INSERT INTO companies_fts(companies_fts, rowid, name, domain, fields) VALUES ('delete', old.rowid, old.name, old.domain, old.fields);
		END`,
		`CREATE TRIGGER IF NOT EXISTS companies_au AFTER UPDATE ON companies BEGIN
			INSERT INTO companies_fts(companies_fts, rowid, name, domain, fields) VALUES ('delete', old.rowid, old.name, old.domain, old.fields);
			INSERT INTO companies_fts(rowid, name, domain, fields) VALUES (new.rowid, new.name, new.domain, new.fields);
		END`,
	}
}

// DataDir returns the XDG data directory for the CRM application.
// Uses $XDG_DATA_HOME/crm if set, otherwise ~/.local/share/crm.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "crm")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "crm")
}

// escapeFTS5Query wraps a query string in double quotes for literal FTS5 search,
// escaping any internal double quotes by doubling them.
func escapeFTS5Query(query string) string {
	escaped := strings.ReplaceAll(query, `"`, `""`)
	return `"` + escaped + `"`
}

// Close closes the underlying database connection.
func (s *SqliteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
