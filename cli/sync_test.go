// ABOUTME: Tests for Google sync CLI commands
// ABOUTME: Verifies OAuth setup, calendar sync, and error handling
package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/sync"
	"golang.org/x/oauth2"
)

// TestSyncCalendarCommand_NoToken verifies error when no token exists.
func TestSyncCalendarCommand_NoToken(t *testing.T) {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Ensure no token file exists
	tokenPath := sync.TokenPath()
	_ = os.Remove(tokenPath)

	// Run command - should fail with helpful error
	err = SyncCalendarCommand(database, []string{})
	if err == nil {
		t.Error("Expected error when token doesn't exist, got nil")
	}

	// Verify error message is helpful
	expectedMsg := "no authentication token found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain %q, got: %v", expectedMsg, err)
	}
}

// TestSyncCalendarCommand_ParsesInitialFlag verifies --initial flag parsing.
func TestSyncCalendarCommand_ParsesInitialFlag(t *testing.T) {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create a fake token file for this test
	tokenPath := sync.TokenPath()
	tokenDir := filepath.Dir(tokenPath)
	if err := os.MkdirAll(tokenDir, 0755); err != nil {
		t.Fatalf("Failed to create token directory: %v", err)
	}

	// Create minimal fake token
	fakeToken := &oauth2.Token{
		AccessToken: "fake-access-token",
	}
	if err := sync.SaveToken(fakeToken); err != nil {
		t.Fatalf("Failed to save fake token: %v", err)
	}
	defer func() { _ = os.Remove(tokenPath) }()

	// Test with --initial flag
	// Note: This will fail at client creation since we don't have real credentials
	// but we're just testing flag parsing, so we expect an error anyway
	err = SyncCalendarCommand(database, []string{"--initial"})

	// We expect an error (can't create real client), but not a flag parsing error
	if err == nil {
		t.Error("Expected error (no real credentials), got nil")
	}

	// The error should be about client creation, not flag parsing
	if err != nil && contains(err.Error(), "flag") {
		t.Errorf("Got flag parsing error: %v", err)
	}
}

// TestSyncCalendarCommand_Integration is a full integration test
// This is skipped by default since it requires real OAuth credentials.
func TestSyncCalendarCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if token exists
	tokenPath := sync.TokenPath()
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: no OAuth token found. Run 'pagen sync init' first.")
	}

	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Run calendar sync
	err = SyncCalendarCommand(database, []string{"--initial"})
	if err != nil {
		t.Fatalf("Calendar sync failed: %v", err)
	}

	// Verify sync state was updated
	syncState, err := db.GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("Failed to get sync state: %v", err)
	}

	if syncState == nil {
		t.Error("Calendar sync state not found")
	} else if syncState.Status != "success" && syncState.Status != "syncing" {
		t.Errorf("Expected sync status to be 'success' or 'syncing', got: %s", syncState.Status)
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSyncStatusCommand tests the sync status command with empty database.
func TestSyncStatusCommand(t *testing.T) {
	// Create temp database
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

// TestSyncStatusCommandWithData tests sync status with existing sync states.
func TestSyncStatusCommandWithData(t *testing.T) {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Insert some sync states with all required fields
	now := time.Now()
	syncToken := "test-token"
	_, err = database.Exec(`INSERT INTO sync_state (service, status, last_sync_time, last_sync_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"calendar", "idle", now, syncToken, now, now)
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

// TestSyncResetCommand tests the sync reset command.
func TestSyncResetCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args shows help", []string{}, false},
		{"reset specific service", []string{"calendar"}, false},
		{"reset nonexistent service", []string{"invalid"}, false},
		{"reset all services", []string{"all"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp database
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test.db")
			database, err := db.OpenDatabase(dbPath)
			if err != nil {
				t.Fatalf("Failed to open database: %v", err)
			}
			defer func() { _ = database.Close() }()

			// Insert sync state for testing reset
			_, _ = database.Exec(`INSERT INTO sync_state (service, status, last_sync_time) VALUES (?, ?, ?)`,
				"calendar", "syncing", time.Now())

			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err = SyncResetCommand(database, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Error("SyncResetCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SyncResetCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestSyncContactsCommand_NoToken verifies error when no token exists.
func TestSyncContactsCommand_NoToken(t *testing.T) {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Ensure no token file exists
	tokenPath := sync.TokenPath()
	_ = os.Remove(tokenPath)

	err = SyncContactsCommand(database, []string{})
	if err == nil {
		t.Error("Expected error when token doesn't exist, got nil")
	}

	expectedMsg := "no authentication token found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain %q, got: %v", expectedMsg, err)
	}
}

// TestSyncGmailCommand_NoToken verifies error when no token exists.
func TestSyncGmailCommand_NoToken(t *testing.T) {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Ensure no token file exists
	tokenPath := sync.TokenPath()
	_ = os.Remove(tokenPath)

	err = SyncGmailCommand(database, []string{})
	if err == nil {
		t.Error("Expected error when token doesn't exist, got nil")
	}

	expectedMsg := "no authentication token found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain %q, got: %v", expectedMsg, err)
	}
}

// TestSyncAllCommand_NoToken verifies error when no token exists.
func TestSyncAllCommand_NoToken(t *testing.T) {
	// Create temp database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Ensure no token file exists
	tokenPath := sync.TokenPath()
	_ = os.Remove(tokenPath)

	// Capture stdout (this command prints to stdout)
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SyncAllCommand(database)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Error("Expected error when token doesn't exist, got nil")
	}

	expectedMsg := "no authentication token found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain %q, got: %v", expectedMsg, err)
	}
}
