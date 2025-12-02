// ABOUTME: Migration utility for transitioning from legacy schema to Office OS foundation.
// ABOUTME: Provides dry-run and backup capabilities for safe schema migration.

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/harperreed/pagen/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "", "Path to database file (required)")
	dryRun := flag.Bool("dry-run", false, "Show what would happen without making changes")
	backup := flag.Bool("backup", true, "Create backup before migration")
	force := flag.Bool("force", false, "Force migration even if data loss may occur")
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("Error: -db flag is required")
	}

	if err := migrate(*dbPath, *dryRun, *backup, *force); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully")
}

func migrate(dbPath string, dryRun, createBackup, force bool) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file does not exist: %s", dbPath)
	}

	if createBackup && !dryRun {
		backupPath := fmt.Sprintf("%s.backup.%s", dbPath, time.Now().Format("20060102-150405"))
		log.Printf("Creating backup: %s", backupPath)

		input, err := os.ReadFile(dbPath)
		if err != nil {
			return fmt.Errorf("failed to read database: %w", err)
		}

		if err := os.WriteFile(backupPath, input, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		log.Printf("Backup created successfully")
	}

	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Check current schema
	tables, err := getCurrentTables(database)
	if err != nil {
		return fmt.Errorf("failed to get current tables: %w", err)
	}

	log.Printf("Current tables: %v", tables)

	hasLegacyTables := false
	for _, table := range tables {
		if isLegacyTable(table) {
			hasLegacyTables = true
			break
		}
	}

	hasNewTables := false
	for _, table := range tables {
		if table == "objects" || table == "relationships" {
			hasNewTables = true
			break
		}
	}

	if hasLegacyTables {
		log.Printf("Found legacy tables (companies, contacts, deals, etc.)")

		if !force {
			log.Printf("WARNING: Migration will drop legacy tables")
			log.Printf("Use -force flag to proceed with migration")
			log.Printf("This will result in data loss of legacy data")
			return fmt.Errorf("migration requires -force flag")
		}
	}

	if dryRun {
		log.Printf("[DRY RUN] Would perform the following actions:")
		if hasLegacyTables {
			log.Printf("[DRY RUN] - Drop legacy tables: companies, contacts, deals, notes, sync_state, sync_log, relationships (old)")
		}
		if !hasNewTables {
			log.Printf("[DRY RUN] - Create new tables: objects, relationships")
			log.Printf("[DRY RUN] - Create indexes for performance")
		} else {
			log.Printf("[DRY RUN] - New Office OS tables already exist")
		}
		return nil
	}

	// Drop legacy tables
	if hasLegacyTables {
		log.Printf("Dropping legacy tables...")
		if err := dropLegacyTables(database); err != nil {
			return fmt.Errorf("failed to drop legacy tables: %w", err)
		}
		log.Printf("Legacy tables dropped")
	}

	// Create new schema
	if !hasNewTables {
		log.Printf("Creating Office OS foundation schema...")
		if err := db.InitSchema(database); err != nil {
			return fmt.Errorf("failed to initialize schema: %w", err)
		}
		log.Printf("Office OS schema created successfully")
	} else {
		log.Printf("Office OS tables already exist, skipping creation")
	}

	return nil
}

func getCurrentTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

func isLegacyTable(name string) bool {
	legacyTables := []string{
		"companies", "contacts", "deals", "notes",
		"sync_state", "sync_log", "contact_cadence",
		"followup_queue", "interactions",
	}

	for _, legacy := range legacyTables {
		if name == legacy {
			return true
		}
	}

	return false
}

func dropLegacyTables(db *sql.DB) error {
	legacyTables := []string{
		"interactions", "followup_queue", "contact_cadence",
		"notes", "deals", "contacts", "companies",
		"sync_log", "sync_state",
	}

	for _, table := range legacyTables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
		log.Printf("Dropped table: %s", table)
	}

	return nil
}

func getBackupDir(dbPath string) (string, error) {
	dir := filepath.Dir(dbPath)
	backupDir := filepath.Join(dir, "backups")

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	return backupDir, nil
}
