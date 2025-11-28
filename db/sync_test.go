// ABOUTME: Tests for sync_state and sync_log database operations
// ABOUTME: Verifies sync status tracking, token management, and import logging
package db

import (
	"testing"
	"time"
)

func TestGetSyncState(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test getting non-existent sync state
	state, err := GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state, got %+v", state)
	}

	// Create a sync state
	err = UpdateSyncStatus(database, "calendar", "idle", nil)
	if err != nil {
		t.Fatalf("failed to create sync state: %v", err)
	}

	// Get sync state
	state, err = GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("expected state, got nil")
	}
	if state.Service != "calendar" {
		t.Errorf("expected service 'calendar', got %q", state.Service)
	}
	if state.Status != "idle" {
		t.Errorf("expected status 'idle', got %q", state.Status)
	}
}

func TestUpdateSyncStatus(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test creating sync status
	err := UpdateSyncStatus(database, "calendar", "syncing", nil)
	if err != nil {
		t.Fatalf("failed to create sync status: %v", err)
	}

	state, err := GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Status != "syncing" {
		t.Errorf("expected status 'syncing', got %q", state.Status)
	}
	if state.ErrorMessage != nil {
		t.Errorf("expected nil error message, got %q", *state.ErrorMessage)
	}

	// Test updating with error
	errMsg := "test error"
	err = UpdateSyncStatus(database, "calendar", "error", &errMsg)
	if err != nil {
		t.Fatalf("failed to update sync status: %v", err)
	}

	state, err = GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Status != "error" {
		t.Errorf("expected status 'error', got %q", state.Status)
	}
	if state.ErrorMessage == nil || *state.ErrorMessage != "test error" {
		t.Errorf("expected error message 'test error', got %v", state.ErrorMessage)
	}
}

func TestUpdateSyncToken(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test creating sync token
	token := "test-token-123"
	err := UpdateSyncToken(database, "calendar", token)
	if err != nil {
		t.Fatalf("failed to create sync token: %v", err)
	}

	state, err := GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != token {
		t.Errorf("expected sync token %q, got %v", token, state.LastSyncToken)
	}
	if state.Status != "idle" {
		t.Errorf("expected status 'idle', got %q", state.Status)
	}
	if state.LastSyncTime == nil {
		t.Error("expected last_sync_time to be set")
	}

	// Test updating sync token
	newToken := "new-token-456"
	err = UpdateSyncToken(database, "calendar", newToken)
	if err != nil {
		t.Fatalf("failed to update sync token: %v", err)
	}

	state, err = GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != newToken {
		t.Errorf("expected sync token %q, got %v", newToken, state.LastSyncToken)
	}
}

func TestCheckSyncLogExists(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test non-existent sync log
	exists, err := CheckSyncLogExists(database, "calendar", "event-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false, got true")
	}

	// Create sync log entry
	err = CreateSyncLog(database, "log-1", "calendar", "event-123", "interaction", "int-1", "{}")
	if err != nil {
		t.Fatalf("failed to create sync log: %v", err)
	}

	// Test existing sync log
	exists, err = CheckSyncLogExists(database, "calendar", "event-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true, got false")
	}
}

func TestCreateSyncLog(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test creating sync log
	err := CreateSyncLog(database, "log-1", "calendar", "event-123", "interaction", "int-1", `{"test": "data"}`)
	if err != nil {
		t.Fatalf("failed to create sync log: %v", err)
	}

	// Verify sync log was created
	var count int
	err = database.QueryRow(`
		SELECT COUNT(*) FROM sync_log
		WHERE source_service = 'calendar' AND source_id = 'event-123'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query sync log: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 sync log entry, got %d", count)
	}

	// Test duplicate prevention (unique constraint)
	err = CreateSyncLog(database, "log-2", "calendar", "event-123", "interaction", "int-2", "{}")
	if err == nil {
		t.Fatal("expected error for duplicate sync log, got nil")
	}
}

func TestSyncWorkflow(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Simulate a sync workflow

	// 1. Start sync
	err := UpdateSyncStatus(database, "calendar", "syncing", nil)
	if err != nil {
		t.Fatalf("failed to start sync: %v", err)
	}

	state, _ := GetSyncState(database, "calendar")
	if state.Status != "syncing" {
		t.Errorf("expected status 'syncing', got %q", state.Status)
	}

	// 2. Process events (simulated)
	time.Sleep(10 * time.Millisecond)

	// 3. Complete sync with token
	token := "sync-token-abc"
	err = UpdateSyncToken(database, "calendar", token)
	if err != nil {
		t.Fatalf("failed to complete sync: %v", err)
	}

	state, _ = GetSyncState(database, "calendar")
	if state.Status != "idle" {
		t.Errorf("expected status 'idle', got %q", state.Status)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != token {
		t.Errorf("expected sync token %q, got %v", token, state.LastSyncToken)
	}
	if state.LastSyncTime == nil {
		t.Error("expected last_sync_time to be set")
	}

	// 4. Next sync - check for token
	state, _ = GetSyncState(database, "calendar")
	if state.LastSyncToken == nil {
		t.Error("expected sync token to be available for incremental sync")
	}
}
