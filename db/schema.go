// ABOUTME: Database schema definitions and migrations
// ABOUTME: Handles SQLite table creation and initialization
package db

import (
	"database/sql"
	"time"
)

// Object represents the core entity in Office OS.
type Object struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func InitSchema(db *sql.DB) error {
	// Enable foreign keys for SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS objects (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_objects_type ON objects(type);
	CREATE INDEX IF NOT EXISTS idx_objects_created_at ON objects(created_at);

	CREATE TABLE IF NOT EXISTS relationships (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		type TEXT NOT NULL,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (source_id) REFERENCES objects(id) ON DELETE CASCADE,
		FOREIGN KEY (target_id) REFERENCES objects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_id);
	CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_id);
	CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(type);
	`

	_, err := db.Exec(schema)
	return err
}
