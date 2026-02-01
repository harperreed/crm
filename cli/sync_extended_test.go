// ABOUTME: Extended tests for sync CLI commands
// ABOUTME: Covers sync status display and reset scenarios
package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harperreed/pagen/db"
)

// Note: TestFormatTimeSince and TestParseServices are in sync_daemon_test.go

func TestSyncStatusCommandEmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncStatusCommand(database, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SyncStatusCommand() unexpected error: %v", err)
	}
}

func TestSyncStatusCommandWithErrorState(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Insert sync state with error
	now := time.Now()
	errMsg := "Connection failed"
	_, err = database.Exec(`INSERT INTO sync_state (service, status, error_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"calendar", "error", errMsg, now, now)
	if err != nil {
		t.Fatalf("Failed to insert sync state: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncStatusCommand(database, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SyncStatusCommand() unexpected error: %v", err)
	}
}

func TestSyncStatusCommandWithSyncingState(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Insert sync state in syncing status
	now := time.Now()
	_, err = database.Exec(`INSERT INTO sync_state (service, status, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"gmail", "syncing", now, now)
	if err != nil {
		t.Fatalf("Failed to insert sync state: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncStatusCommand(database, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SyncStatusCommand() unexpected error: %v", err)
	}
}

func TestSyncResetCommandSpecificService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Insert initial sync state
	now := time.Now()
	_, err = database.Exec(`INSERT INTO sync_state (service, status, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"contacts", "syncing", now, now)
	if err != nil {
		t.Fatalf("Failed to insert sync state: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncResetCommand(database, []string{"contacts"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SyncResetCommand() unexpected error: %v", err)
	}

	// Verify state was reset
	var status string
	err = database.QueryRow(`SELECT status FROM sync_state WHERE service = ?`, "contacts").Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query sync state: %v", err)
	}

	if status != "idle" {
		t.Errorf("expected status 'idle', got %s", status)
	}
}

func TestSyncStatusCommandMultipleServices(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Insert multiple sync states
	now := time.Now()
	syncToken := "token123"
	_, err = database.Exec(`INSERT INTO sync_state (service, status, last_sync_time, last_sync_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"calendar", "idle", now, syncToken, now, now)
	if err != nil {
		t.Fatalf("Failed to insert calendar sync state: %v", err)
	}

	_, err = database.Exec(`INSERT INTO sync_state (service, status, last_sync_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"contacts", "idle", now.Add(-1*time.Hour), now, now)
	if err != nil {
		t.Fatalf("Failed to insert contacts sync state: %v", err)
	}

	errMsg := "API error"
	_, err = database.Exec(`INSERT INTO sync_state (service, status, error_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"gmail", "error", errMsg, now, now)
	if err != nil {
		t.Fatalf("Failed to insert gmail sync state: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncStatusCommand(database, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SyncStatusCommand() unexpected error: %v", err)
	}
}

func TestSyncResetCommandAll(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Insert multiple sync states
	now := time.Now()
	services := []string{"calendar", "contacts", "gmail"}
	for _, svc := range services {
		_, err = database.Exec(`INSERT INTO sync_state (service, status, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			svc, "syncing", now, now)
		if err != nil {
			t.Fatalf("Failed to insert sync state for %s: %v", svc, err)
		}
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncResetCommand(database, []string{"all"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SyncResetCommand(all) unexpected error: %v", err)
	}

	// Verify all states were reset
	for _, svc := range services {
		var status string
		err = database.QueryRow(`SELECT status FROM sync_state WHERE service = ?`, svc).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query sync state for %s: %v", svc, err)
		}

		if status != "idle" {
			t.Errorf("expected status 'idle' for %s, got %s", svc, status)
		}
	}
}
