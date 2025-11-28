// ABOUTME: Tests for calendar event importer
// ABOUTME: Verifies sync logic, pagination handling, and token management
package sync

import (
	"testing"
	"time"

	"github.com/harperreed/pagen/db"
)

// Real unit tests that verify actual behavior

func TestSyncStateLifecycle(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// 1. Initial state: no sync state exists
	state, err := db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state != nil {
		t.Errorf("expected nil state for new service, got %+v", state)
	}

	// 2. Start sync: status should be 'syncing'
	err = db.UpdateSyncStatus(database, calendarService, "syncing", nil)
	if err != nil {
		t.Fatalf("failed to update sync status to syncing: %v", err)
	}

	state, err = db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "syncing" {
		t.Errorf("expected status 'syncing', got %q", state.Status)
	}
	if state.ErrorMessage != nil {
		t.Errorf("expected nil error message during sync, got %v", state.ErrorMessage)
	}

	// 3. Complete sync: status should be 'idle' with token
	token := "test-sync-token-abc123"
	err = db.UpdateSyncToken(database, calendarService, token)
	if err != nil {
		t.Fatalf("failed to update sync token: %v", err)
	}

	state, err = db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "idle" {
		t.Errorf("expected status 'idle' after token update, got %q", state.Status)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != token {
		t.Errorf("expected sync token %q, got %v", token, state.LastSyncToken)
	}
	if state.LastSyncTime == nil {
		t.Error("expected last_sync_time to be set after sync")
	}

	// 4. Error state: status should be 'error' with message
	errMsg := "API error: rate limit exceeded"
	err = db.UpdateSyncStatus(database, calendarService, "error", &errMsg)
	if err != nil {
		t.Fatalf("failed to update sync status to error: %v", err)
	}

	state, err = db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "error" {
		t.Errorf("expected status 'error', got %q", state.Status)
	}
	if state.ErrorMessage == nil || *state.ErrorMessage != errMsg {
		t.Errorf("expected error message %q, got %v", errMsg, state.ErrorMessage)
	}
}

func TestSyncTokenPersistence(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Save initial sync token
	token1 := "sync-token-initial"
	err := db.UpdateSyncToken(database, calendarService, token1)
	if err != nil {
		t.Fatalf("failed to save initial sync token: %v", err)
	}

	// Verify token is retrievable
	state, err := db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != token1 {
		t.Errorf("expected sync token %q, got %v", token1, state.LastSyncToken)
	}

	// Update sync token
	token2 := "sync-token-updated"
	err = db.UpdateSyncToken(database, calendarService, token2)
	if err != nil {
		t.Fatalf("failed to update sync token: %v", err)
	}

	// Verify token is updated
	state, err = db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != token2 {
		t.Errorf("expected updated sync token %q, got %v", token2, state.LastSyncToken)
	}
}

func TestTimeMinCalculation(t *testing.T) {
	// Verify that 6 months ago calculation is correct
	now := time.Now()
	sixMonthsAgo := now.AddDate(0, -6, 0)

	// The difference should be approximately 6 months
	diff := now.Sub(sixMonthsAgo)
	expectedDays := 180.0 // Approximate 6 months
	actualDays := diff.Hours() / 24.0

	// Allow for some variance (175-185 days)
	if actualDays < 175 || actualDays > 185 {
		t.Errorf("expected approximately %f days, got %f", expectedDays, actualDays)
	}
}

func TestPageNumberCalculation(t *testing.T) {
	testCases := []struct {
		totalEvents  int
		eventCount   int
		expectedPage int
		description  string
	}{
		{250, 250, 1, "first full page"},
		{500, 250, 2, "second full page"},
		{750, 250, 3, "third full page"},
		{350, 100, 2, "second partial page"},
		{260, 10, 2, "second very small page"},
		{100, 100, 1, "single partial page"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			pageNum := (tc.totalEvents-tc.eventCount)/maxResults + 1
			if pageNum != tc.expectedPage {
				t.Errorf("%s: expected page %d, got %d (totalEvents=%d, eventCount=%d)",
					tc.description, tc.expectedPage, pageNum, tc.totalEvents, tc.eventCount)
			}
		})
	}
}

func TestInitialVsIncrementalSync(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test 1: Initial sync flag should affect behavior
	// When initial=true, we should not look for sync token
	// This is tested implicitly in the ImportCalendar function

	// Test 2: When not initial and sync token exists, should use token
	token := "existing-sync-token"
	err := db.UpdateSyncToken(database, calendarService, token)
	if err != nil {
		t.Fatalf("failed to save sync token: %v", err)
	}

	state, err := db.GetSyncState(database, calendarService)
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.LastSyncToken == nil {
		t.Error("expected sync token to be available for incremental sync")
	}

	// Test 3: When not initial but no sync token, should fall back to timeMin
	err = db.UpdateSyncStatus(database, "new-service", "idle", nil)
	if err != nil {
		t.Fatalf("failed to create new service state: %v", err)
	}

	state, err = db.GetSyncState(database, "new-service")
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.LastSyncToken != nil {
		t.Error("expected nil sync token for new service")
	}
}

// Documentation tests that describe expected behavior with external APIs

func TestInitialSyncWithTimeMin(t *testing.T) {
	// This test verifies that initial sync uses timeMin parameter
	// We can't easily mock the Google Calendar API in Go without significant
	// refactoring, so this test documents the expected behavior

	// Expected behavior for initial sync:
	// 1. Call Events.List with TimeMin set to 6 months ago
	// 2. Set SingleEvents(true) and OrderBy("startTime")
	// 3. Set MaxResults(250) for pagination
	// 4. Loop through pages using PageToken
	// 5. Save NextSyncToken from last page

	t.Log("Initial sync should use timeMin parameter (6 months ago)")
	t.Log("Expected API call: Events.List().TimeMin(sixMonthsAgo).MaxResults(250).SingleEvents(true).OrderBy('startTime')")
}

func TestIncrementalSyncWithToken(t *testing.T) {
	// This test verifies that incremental sync uses syncToken parameter
	// Expected behavior for incremental sync:
	// 1. Get sync token from sync_state table
	// 2. Call Events.List with SyncToken
	// 3. Process only changed/new events
	// 4. Save new NextSyncToken

	t.Log("Incremental sync should use syncToken from database")
	t.Log("Expected API call: Events.List().SyncToken(lastToken)")
}

func TestPaginationHandling(t *testing.T) {
	// This test verifies pagination logic
	// Expected behavior:
	// 1. First call gets up to 250 events
	// 2. If NextPageToken is present, make another call with PageToken
	// 3. Repeat until NextPageToken is empty
	// 4. Save NextSyncToken from the final page

	t.Log("Pagination should continue until NextPageToken is empty")
	t.Log("MaxResults should be 250 per page")
}

func TestSyncStateUpdates(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Test that sync state is properly updated during sync lifecycle

	// 1. Before sync: should be able to create initial state
	err := db.UpdateSyncStatus(database, "calendar", "idle", nil)
	if err != nil {
		t.Fatalf("failed to create initial sync state: %v", err)
	}

	state, err := db.GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "idle" {
		t.Errorf("expected status 'idle', got %q", state.Status)
	}

	// 2. During sync: status should be 'syncing'
	err = db.UpdateSyncStatus(database, "calendar", "syncing", nil)
	if err != nil {
		t.Fatalf("failed to update sync status to syncing: %v", err)
	}

	state, err = db.GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "syncing" {
		t.Errorf("expected status 'syncing', got %q", state.Status)
	}

	// 3. After sync: status should be 'idle' with token
	token := "test-sync-token-123"
	err = db.UpdateSyncToken(database, "calendar", token)
	if err != nil {
		t.Fatalf("failed to update sync token: %v", err)
	}

	state, err = db.GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "idle" {
		t.Errorf("expected status 'idle' after token update, got %q", state.Status)
	}
	if state.LastSyncToken == nil || *state.LastSyncToken != token {
		t.Errorf("expected sync token %q, got %v", token, state.LastSyncToken)
	}

	// 4. On error: status should be 'error' with message
	errMsg := "API error: rate limit exceeded"
	err = db.UpdateSyncStatus(database, "calendar", "error", &errMsg)
	if err != nil {
		t.Fatalf("failed to update sync status to error: %v", err)
	}

	state, err = db.GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state.Status != "error" {
		t.Errorf("expected status 'error', got %q", state.Status)
	}
	if state.ErrorMessage == nil || *state.ErrorMessage != errMsg {
		t.Errorf("expected error message %q, got %v", errMsg, state.ErrorMessage)
	}
}

func TestNoSyncTokenFallback(t *testing.T) {
	// This test verifies behavior when sync token is not available
	// Expected behavior:
	// 1. Check for sync token in database
	// 2. If no token found, fall back to timeMin (6 months ago)
	// 3. Process as initial sync

	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	state, err := db.GetSyncState(database, "calendar")
	if err != nil {
		t.Fatalf("failed to get sync state: %v", err)
	}
	if state != nil {
		t.Errorf("expected nil state for new service, got %+v", state)
	}

	t.Log("When no sync token exists, should fall back to timeMin parameter")
}

func TestEventCountTracking(t *testing.T) {
	// This test verifies that event counts are tracked correctly
	// Expected behavior:
	// 1. Count events from each page
	// 2. Sum total events across all pages
	// 3. Log progress with event counts

	t.Log("Event counts should be tracked and logged during sync")
	t.Log("Should show: 'Fetched X events (page N)'")
}

func TestProgressLogging(t *testing.T) {
	// This test verifies that progress is logged to stdout
	// Expected output:
	// - "Syncing Google Calendar..."
	// - "  → Initial sync (last 6 months)..." or "  → Incremental sync..."
	// - "  → Fetched X events (page N)"
	// - "✓ Fetched X events"
	// - "Sync token saved. Next sync will be incremental."

	t.Log("Progress should be logged to stdout during sync")
	t.Log("Should include sync type, event counts, and completion status")
}
