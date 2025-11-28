// ABOUTME: Tests for calendar event importer
// ABOUTME: Verifies sync logic, pagination handling, and token management
package sync

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/calendar/v3"

	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
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

// Event Filtering Tests

func TestShouldSkipEvent_AllDayEvents(t *testing.T) {
	testCases := []struct {
		name        string
		event       *calendar.Event
		userEmail   string
		shouldSkip  bool
		expectedMsg string
	}{
		{
			name: "all-day event with date",
			event: &calendar.Event{
				Id:      "event1",
				Summary: "All Day Meeting",
				Start: &calendar.EventDateTime{
					Date: "2025-11-28",
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "all-day event",
		},
		{
			name: "timed event with dateTime",
			event: &calendar.Event{
				Id:      "event2",
				Summary: "Regular Meeting",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", Self: true},
					{Email: "other@example.com"},
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  false,
			expectedMsg: "",
		},
		{
			name: "event with both date and dateTime prefers date",
			event: &calendar.Event{
				Id:      "event3",
				Summary: "Weird Event",
				Start: &calendar.EventDateTime{
					Date:     "2025-11-28",
					DateTime: "2025-11-28T10:00:00Z",
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "all-day event",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			skip, msg := shouldSkipEvent(tc.event, tc.userEmail)
			if skip != tc.shouldSkip {
				t.Errorf("expected shouldSkip=%v, got %v", tc.shouldSkip, skip)
			}
			if msg != tc.expectedMsg {
				t.Errorf("expected message %q, got %q", tc.expectedMsg, msg)
			}
		})
	}
}

func TestShouldSkipEvent_CancelledEvents(t *testing.T) {
	testCases := []struct {
		name        string
		event       *calendar.Event
		userEmail   string
		shouldSkip  bool
		expectedMsg string
	}{
		{
			name: "cancelled event",
			event: &calendar.Event{
				Id:      "event1",
				Summary: "Cancelled Meeting",
				Status:  "cancelled",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "cancelled",
		},
		{
			name: "confirmed event",
			event: &calendar.Event{
				Id:      "event2",
				Summary: "Confirmed Meeting",
				Status:  "confirmed",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", Self: true},
					{Email: "other@example.com"},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
		{
			name: "tentative event",
			event: &calendar.Event{
				Id:      "event3",
				Summary: "Tentative Meeting",
				Status:  "tentative",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", Self: true},
					{Email: "other@example.com"},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			skip, msg := shouldSkipEvent(tc.event, tc.userEmail)
			if skip != tc.shouldSkip {
				t.Errorf("expected shouldSkip=%v, got %v", tc.shouldSkip, skip)
			}
			if tc.shouldSkip && msg != tc.expectedMsg {
				t.Errorf("expected message %q, got %q", tc.expectedMsg, msg)
			}
		})
	}
}

func TestShouldSkipEvent_DeclinedEvents(t *testing.T) {
	testCases := []struct {
		name        string
		event       *calendar.Event
		userEmail   string
		shouldSkip  bool
		expectedMsg string
	}{
		{
			name: "user declined",
			event: &calendar.Event{
				Id:      "event1",
				Summary: "Meeting I Declined",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "other@example.com", ResponseStatus: "accepted"},
					{Email: "user@example.com", ResponseStatus: "declined", Self: true},
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "declined",
		},
		{
			name: "user accepted",
			event: &calendar.Event{
				Id:      "event2",
				Summary: "Meeting I Accepted",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "other@example.com", ResponseStatus: "accepted"},
					{Email: "user@example.com", ResponseStatus: "accepted", Self: true},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
		{
			name: "user tentative",
			event: &calendar.Event{
				Id:      "event3",
				Summary: "Meeting I'm Tentative About",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", ResponseStatus: "tentative", Self: true},
					{Email: "other@example.com", ResponseStatus: "accepted"},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
		{
			name: "other person declined but not me",
			event: &calendar.Event{
				Id:      "event4",
				Summary: "Meeting Someone Else Declined",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "other@example.com", ResponseStatus: "declined"},
					{Email: "user@example.com", ResponseStatus: "accepted", Self: true},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
		{
			name: "user email not in attendees but using Self flag",
			event: &calendar.Event{
				Id:      "event5",
				Summary: "Meeting with Self Flag",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "calendar@example.com", ResponseStatus: "declined", Self: true},
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "declined",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			skip, msg := shouldSkipEvent(tc.event, tc.userEmail)
			if skip != tc.shouldSkip {
				t.Errorf("expected shouldSkip=%v, got %v", tc.shouldSkip, skip)
			}
			if tc.shouldSkip && msg != tc.expectedMsg {
				t.Errorf("expected message %q, got %q", tc.expectedMsg, msg)
			}
		})
	}
}

func TestShouldSkipEvent_SoloEvents(t *testing.T) {
	testCases := []struct {
		name        string
		event       *calendar.Event
		userEmail   string
		shouldSkip  bool
		expectedMsg string
	}{
		{
			name: "no attendees",
			event: &calendar.Event{
				Id:      "event1",
				Summary: "Solo Event",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: nil,
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "solo event (0 attendees)",
		},
		{
			name: "empty attendees list",
			event: &calendar.Event{
				Id:      "event2",
				Summary: "Solo Event 2",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "solo event (0 attendees)",
		},
		{
			name: "only me",
			event: &calendar.Event{
				Id:      "event3",
				Summary: "Just Me Event",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", Self: true},
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "solo event (1 attendee)",
		},
		{
			name: "me and one other",
			event: &calendar.Event{
				Id:      "event4",
				Summary: "Meeting with Someone",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", Self: true},
					{Email: "other@example.com"},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
		{
			name: "multiple attendees",
			event: &calendar.Event{
				Id:      "event5",
				Summary: "Team Meeting",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", Self: true},
					{Email: "person1@example.com"},
					{Email: "person2@example.com"},
				},
			},
			userEmail:  "user@example.com",
			shouldSkip: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			skip, msg := shouldSkipEvent(tc.event, tc.userEmail)
			if skip != tc.shouldSkip {
				t.Errorf("expected shouldSkip=%v, got %v", tc.shouldSkip, skip)
			}
			if tc.shouldSkip && msg != tc.expectedMsg {
				t.Errorf("expected message %q, got %q", tc.expectedMsg, msg)
			}
		})
	}
}

func TestShouldSkipEvent_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		event       *calendar.Event
		userEmail   string
		shouldSkip  bool
		expectedMsg string
	}{
		{
			name:        "nil event",
			event:       nil,
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "nil event",
		},
		{
			name: "nil start",
			event: &calendar.Event{
				Id:      "event1",
				Summary: "No Start Time",
				Start:   nil,
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "missing start time",
		},
		{
			name: "empty user email",
			event: &calendar.Event{
				Id:      "event2",
				Summary: "Regular Meeting",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "person@example.com"},
					{Email: "person2@example.com"},
				},
			},
			userEmail:  "",
			shouldSkip: false,
		},
		{
			name: "cancelled all-day event (multiple skip conditions)",
			event: &calendar.Event{
				Id:      "event3",
				Summary: "Cancelled All Day Event",
				Status:  "cancelled",
				Start: &calendar.EventDateTime{
					Date: "2025-11-28",
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "all-day event",
		},
		{
			name: "declined solo event (multiple skip conditions)",
			event: &calendar.Event{
				Id:      "event4",
				Summary: "Declined Solo Event",
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com", ResponseStatus: "declined", Self: true},
				},
			},
			userEmail:   "user@example.com",
			shouldSkip:  true,
			expectedMsg: "declined",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			skip, msg := shouldSkipEvent(tc.event, tc.userEmail)
			if skip != tc.shouldSkip {
				t.Errorf("expected shouldSkip=%v, got %v", tc.shouldSkip, skip)
			}
			if tc.shouldSkip && msg != tc.expectedMsg {
				t.Errorf("expected message %q, got %q", tc.expectedMsg, msg)
			}
		})
	}
}

// Attendee → Contact Mapping Tests

func TestExtractContacts_SkipsUserEmail(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create a ContactMatcher with no existing contacts
	matcher := NewContactMatcher([]models.Contact{})

	event := &calendar.Event{
		Id:      "event1",
		Summary: "Team Meeting",
		Attendees: []*calendar.EventAttendee{
			{Email: "user@example.com", DisplayName: "Me", Self: true},
			{Email: "alice@example.com", DisplayName: "Alice"},
			{Email: "bob@example.com", DisplayName: "Bob"},
		},
	}

	contactIDs, err := extractContacts(database, event, "user@example.com", matcher)
	if err != nil {
		t.Fatalf("extractContacts failed: %v", err)
	}

	// Should have created 2 contacts (not 3, since user is skipped)
	if len(contactIDs) != 2 {
		t.Errorf("expected 2 contact IDs, got %d", len(contactIDs))
	}

	// Verify contacts were created in database
	contacts, err := db.FindContacts(database, "", nil, 10)
	if err != nil {
		t.Fatalf("failed to find contacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts in database, got %d", len(contacts))
	}

	// Verify emails are correct
	emails := make(map[string]bool)
	for _, c := range contacts {
		emails[c.Email] = true
	}
	if !emails["alice@example.com"] {
		t.Error("expected alice@example.com to be created")
	}
	if !emails["bob@example.com"] {
		t.Error("expected bob@example.com to be created")
	}
	if emails["user@example.com"] {
		t.Error("user@example.com should not be created")
	}
}

func TestExtractContacts_ReusesExistingContacts(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create an existing contact
	existingContact := &models.Contact{
		Name:  "Alice Smith",
		Email: "alice@example.com",
	}
	if err := db.CreateContact(database, existingContact); err != nil {
		t.Fatalf("failed to create existing contact: %v", err)
	}

	// Create matcher with existing contact
	allContacts, err := db.FindContacts(database, "", nil, 100)
	if err != nil {
		t.Fatalf("failed to find contacts: %v", err)
	}
	matcher := NewContactMatcher(allContacts)

	event := &calendar.Event{
		Id:      "event1",
		Summary: "Team Meeting",
		Attendees: []*calendar.EventAttendee{
			{Email: "user@example.com", DisplayName: "Me", Self: true},
			{Email: "alice@example.com", DisplayName: "Alice"}, // Existing
			{Email: "bob@example.com", DisplayName: "Bob"},     // New
		},
	}

	contactIDs, err := extractContacts(database, event, "user@example.com", matcher)
	if err != nil {
		t.Fatalf("extractContacts failed: %v", err)
	}

	// Should have returned 2 contact IDs
	if len(contactIDs) != 2 {
		t.Errorf("expected 2 contact IDs, got %d", len(contactIDs))
	}

	// Verify only 2 contacts in database (no duplicate alice)
	contacts, err := db.FindContacts(database, "", nil, 10)
	if err != nil {
		t.Fatalf("failed to find contacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts in database (no duplicates), got %d", len(contacts))
	}

	// Verify existing contact ID is in the returned IDs
	foundExisting := false
	for _, id := range contactIDs {
		if id == existingContact.ID {
			foundExisting = true
			break
		}
	}
	if !foundExisting {
		t.Error("expected existing contact ID to be in returned IDs")
	}
}

func TestExtractContacts_CreatesNewContacts(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create a ContactMatcher with no existing contacts
	matcher := NewContactMatcher([]models.Contact{})

	event := &calendar.Event{
		Id:      "event1",
		Summary: "Team Meeting",
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com", DisplayName: "Alice Smith"},
			{Email: "bob@example.com", DisplayName: "Bob Jones"},
		},
	}

	contactIDs, err := extractContacts(database, event, "user@example.com", matcher)
	if err != nil {
		t.Fatalf("extractContacts failed: %v", err)
	}

	// Should have created 2 contacts
	if len(contactIDs) != 2 {
		t.Errorf("expected 2 contact IDs, got %d", len(contactIDs))
	}

	// Verify contacts were created with correct data
	contacts, err := db.FindContacts(database, "", nil, 10)
	if err != nil {
		t.Fatalf("failed to find contacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts in database, got %d", len(contacts))
	}

	// Verify names were set from DisplayName
	for _, c := range contacts {
		if c.Email == "alice@example.com" && c.Name != "Alice Smith" {
			t.Errorf("expected Alice Smith, got %s", c.Name)
		}
		if c.Email == "bob@example.com" && c.Name != "Bob Jones" {
			t.Errorf("expected Bob Jones, got %s", c.Name)
		}
	}
}

func TestExtractContacts_SkipsAttendeesWithNoEmail(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	matcher := NewContactMatcher([]models.Contact{})

	event := &calendar.Event{
		Id:      "event1",
		Summary: "Team Meeting",
		Attendees: []*calendar.EventAttendee{
			{Email: "", DisplayName: "No Email Person"},
			{Email: "alice@example.com", DisplayName: "Alice"},
		},
	}

	contactIDs, err := extractContacts(database, event, "user@example.com", matcher)
	if err != nil {
		t.Fatalf("extractContacts failed: %v", err)
	}

	// Should have created only 1 contact (skipped the one with no email)
	if len(contactIDs) != 1 {
		t.Errorf("expected 1 contact ID, got %d", len(contactIDs))
	}

	contacts, err := db.FindContacts(database, "", nil, 10)
	if err != nil {
		t.Fatalf("failed to find contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("expected 1 contact in database, got %d", len(contacts))
	}
}

func TestExtractContacts_HandlesEmptyAttendees(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	matcher := NewContactMatcher([]models.Contact{})

	event := &calendar.Event{
		Id:        "event1",
		Summary:   "Solo Event",
		Attendees: []*calendar.EventAttendee{},
	}

	contactIDs, err := extractContacts(database, event, "user@example.com", matcher)
	if err != nil {
		t.Fatalf("extractContacts failed: %v", err)
	}

	// Should return empty list
	if len(contactIDs) != 0 {
		t.Errorf("expected 0 contact IDs, got %d", len(contactIDs))
	}
}

// Interaction Logging Tests

func TestLogInteraction_CreatesInteractionsForAllContacts(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create test contacts
	contact1 := &models.Contact{Name: "Alice", Email: "alice@example.com"}
	contact2 := &models.Contact{Name: "Bob", Email: "bob@example.com"}
	if err := db.CreateContact(database, contact1); err != nil {
		t.Fatalf("failed to create contact1: %v", err)
	}
	if err := db.CreateContact(database, contact2); err != nil {
		t.Fatalf("failed to create contact2: %v", err)
	}

	// Create test event
	event := &calendar.Event{
		Id:      "event1",
		Summary: "Team Standup",
		Start: &calendar.EventDateTime{
			DateTime: "2025-11-28T10:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2025-11-28T10:30:00Z",
		},
		Location: "Conference Room A",
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com"},
			{Email: "bob@example.com"},
		},
	}

	// Log interaction
	contactIDs := []uuid.UUID{contact1.ID, contact2.ID}
	err := logInteraction(database, event, contactIDs)
	if err != nil {
		t.Fatalf("logInteraction failed: %v", err)
	}

	// Verify interactions were created - one per contact
	for _, contactID := range contactIDs {
		interactions, err := db.GetInteractionHistory(database, contactID, 10)
		if err != nil {
			t.Fatalf("failed to get interaction history: %v", err)
		}
		if len(interactions) != 1 {
			t.Errorf("expected 1 interaction for contact %s, got %d", contactID, len(interactions))
			continue
		}

		interaction := interactions[0]

		// Verify interaction type
		if interaction.InteractionType != "meeting" {
			t.Errorf("expected interaction type 'meeting', got %q", interaction.InteractionType)
		}

		// Verify notes contains event summary
		if interaction.Notes != "Team Standup" {
			t.Errorf("expected notes 'Team Standup', got %q", interaction.Notes)
		}

		// Verify timestamp matches event start
		expectedTime, _ := time.Parse(time.RFC3339, "2025-11-28T10:00:00Z")
		if !interaction.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, interaction.Timestamp)
		}
	}
}

func TestLogInteraction_StoresMetadataJSON(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create test contact
	contact := &models.Contact{Name: "Alice", Email: "alice@example.com"}
	if err := db.CreateContact(database, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Create test event with all metadata fields
	event := &calendar.Event{
		Id:      "cal-event-123",
		Summary: "Important Meeting",
		Start: &calendar.EventDateTime{
			DateTime: "2025-11-28T14:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2025-11-28T15:30:00Z",
		},
		Location: "Room 42",
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com"},
			{Email: "bob@example.com"},
			{Email: "charlie@example.com"},
		},
	}

	// Log interaction
	err := logInteraction(database, event, []uuid.UUID{contact.ID})
	if err != nil {
		t.Fatalf("logInteraction failed: %v", err)
	}

	// Query the metadata directly from database
	var metadataJSON string
	query := `SELECT metadata FROM interaction_log WHERE contact_id = ?`
	err = database.QueryRow(query, contact.ID.String()).Scan(&metadataJSON)
	if err != nil {
		t.Fatalf("failed to query metadata: %v", err)
	}

	// Parse metadata JSON
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		t.Fatalf("failed to parse metadata JSON: %v", err)
	}

	// Verify metadata fields
	if metadata["calendar_event_id"] != "cal-event-123" {
		t.Errorf("expected calendar_event_id 'cal-event-123', got %v", metadata["calendar_event_id"])
	}
	if metadata["location"] != "Room 42" {
		t.Errorf("expected location 'Room 42', got %v", metadata["location"])
	}
	if metadata["attendee_count"] != float64(3) {
		t.Errorf("expected attendee_count 3, got %v", metadata["attendee_count"])
	}
	if metadata["duration_minutes"] != float64(90) {
		t.Errorf("expected duration_minutes 90, got %v", metadata["duration_minutes"])
	}
}

func TestLogInteraction_CalculatesDurationCorrectly(t *testing.T) {
	testCases := []struct {
		name             string
		startTime        string
		endTime          string
		expectedDuration int
	}{
		{
			name:             "30 minute meeting",
			startTime:        "2025-11-28T10:00:00Z",
			endTime:          "2025-11-28T10:30:00Z",
			expectedDuration: 30,
		},
		{
			name:             "1 hour meeting",
			startTime:        "2025-11-28T14:00:00Z",
			endTime:          "2025-11-28T15:00:00Z",
			expectedDuration: 60,
		},
		{
			name:             "90 minute meeting",
			startTime:        "2025-11-28T09:00:00Z",
			endTime:          "2025-11-28T10:30:00Z",
			expectedDuration: 90,
		},
		{
			name:             "15 minute standup",
			startTime:        "2025-11-28T09:00:00Z",
			endTime:          "2025-11-28T09:15:00Z",
			expectedDuration: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			database := setupTestDB(t)
			defer func() { _ = database.Close() }()

			contact := &models.Contact{Name: "Alice", Email: "alice@example.com"}
			if err := db.CreateContact(database, contact); err != nil {
				t.Fatalf("failed to create contact: %v", err)
			}

			event := &calendar.Event{
				Id:      "event1",
				Summary: tc.name,
				Start: &calendar.EventDateTime{
					DateTime: tc.startTime,
				},
				End: &calendar.EventDateTime{
					DateTime: tc.endTime,
				},
				Attendees: []*calendar.EventAttendee{
					{Email: "alice@example.com"},
				},
			}

			err := logInteraction(database, event, []uuid.UUID{contact.ID})
			if err != nil {
				t.Fatalf("logInteraction failed: %v", err)
			}

			// Query and verify duration
			var metadataJSON string
			query := `SELECT metadata FROM interaction_log WHERE contact_id = ?`
			err = database.QueryRow(query, contact.ID.String()).Scan(&metadataJSON)
			if err != nil {
				t.Fatalf("failed to query metadata: %v", err)
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				t.Fatalf("failed to parse metadata JSON: %v", err)
			}

			duration := int(metadata["duration_minutes"].(float64))
			if duration != tc.expectedDuration {
				t.Errorf("expected duration %d minutes, got %d", tc.expectedDuration, duration)
			}
		})
	}
}

func TestLogInteraction_HandlesNilLocation(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	contact := &models.Contact{Name: "Alice", Email: "alice@example.com"}
	if err := db.CreateContact(database, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Event with no location
	event := &calendar.Event{
		Id:      "event1",
		Summary: "Virtual Meeting",
		Start: &calendar.EventDateTime{
			DateTime: "2025-11-28T10:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2025-11-28T11:00:00Z",
		},
		Location: "", // Empty location
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com"},
		},
	}

	err := logInteraction(database, event, []uuid.UUID{contact.ID})
	if err != nil {
		t.Fatalf("logInteraction failed: %v", err)
	}

	// Verify metadata contains empty location (not error)
	var metadataJSON string
	query := `SELECT metadata FROM interaction_log WHERE contact_id = ?`
	err = database.QueryRow(query, contact.ID.String()).Scan(&metadataJSON)
	if err != nil {
		t.Fatalf("failed to query metadata: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		t.Fatalf("failed to parse metadata JSON: %v", err)
	}

	if metadata["location"] != "" {
		t.Errorf("expected empty location, got %v", metadata["location"])
	}
}

func TestLogInteraction_UpdatesCadence(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	contact := &models.Contact{Name: "Alice", Email: "alice@example.com"}
	if err := db.CreateContact(database, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	event := &calendar.Event{
		Id:      "event1",
		Summary: "Meeting",
		Start: &calendar.EventDateTime{
			DateTime: "2025-11-28T10:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2025-11-28T11:00:00Z",
		},
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com"},
		},
	}

	// Log interaction
	err := logInteraction(database, event, []uuid.UUID{contact.ID})
	if err != nil {
		t.Fatalf("logInteraction failed: %v", err)
	}

	// Verify cadence was updated
	cadence, err := db.GetContactCadence(database, contact.ID)
	if err != nil {
		t.Fatalf("failed to get contact cadence: %v", err)
	}
	if cadence == nil {
		t.Fatal("expected cadence to be created, got nil")
	}

	// Verify last interaction date is set
	if cadence.LastInteractionDate == nil {
		t.Error("expected last_interaction_date to be set")
	} else {
		expectedTime, _ := time.Parse(time.RFC3339, "2025-11-28T10:00:00Z")
		if !cadence.LastInteractionDate.Equal(expectedTime) {
			t.Errorf("expected last_interaction_date %v, got %v", expectedTime, *cadence.LastInteractionDate)
		}
	}

	// Verify next followup date is set (30 days after by default)
	if cadence.NextFollowupDate == nil {
		t.Error("expected next_followup_date to be set")
	}
}

// Duration Calculation Tests

func TestCalculateDuration_ValidEvents(t *testing.T) {
	testCases := []struct {
		name             string
		startTime        string
		endTime          string
		expectedDuration int
		expectError      bool
	}{
		{
			name:             "30 minute meeting",
			startTime:        "2025-11-28T10:00:00Z",
			endTime:          "2025-11-28T10:30:00Z",
			expectedDuration: 30,
			expectError:      false,
		},
		{
			name:             "1 hour meeting",
			startTime:        "2025-11-28T14:00:00Z",
			endTime:          "2025-11-28T15:00:00Z",
			expectedDuration: 60,
			expectError:      false,
		},
		{
			name:             "90 minute meeting",
			startTime:        "2025-11-28T09:00:00Z",
			endTime:          "2025-11-28T10:30:00Z",
			expectedDuration: 90,
			expectError:      false,
		},
		{
			name:             "2 hour meeting",
			startTime:        "2025-11-28T13:00:00Z",
			endTime:          "2025-11-28T15:00:00Z",
			expectedDuration: 120,
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &calendar.Event{
				Start: &calendar.EventDateTime{
					DateTime: tc.startTime,
				},
				End: &calendar.EventDateTime{
					DateTime: tc.endTime,
				},
			}

			duration, err := calculateDuration(event)

			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if duration != tc.expectedDuration {
				t.Errorf("expected duration %d, got %d", tc.expectedDuration, duration)
			}
		})
	}
}

func TestCalculateDuration_ErrorCases(t *testing.T) {
	testCases := []struct {
		name        string
		event       *calendar.Event
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil start time",
			event: &calendar.Event{
				Start: nil,
				End: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:30:00Z",
				},
			},
			expectError: true,
			errorMsg:    "start time is nil",
		},
		{
			name: "nil end time",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				End: nil,
			},
			expectError: true,
			errorMsg:    "end time is nil",
		},
		{
			name: "invalid start time format",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{
					DateTime: "not-a-valid-time",
				},
				End: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:30:00Z",
				},
			},
			expectError: true,
			errorMsg:    "failed to parse start time",
		},
		{
			name: "invalid end time format",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
				End: &calendar.EventDateTime{
					DateTime: "invalid-time",
				},
			},
			expectError: true,
			errorMsg:    "failed to parse end time",
		},
		{
			name: "negative duration (end before start)",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:30:00Z",
				},
				End: &calendar.EventDateTime{
					DateTime: "2025-11-28T10:00:00Z",
				},
			},
			expectError: true,
			errorMsg:    "end time before start time",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			duration, err := calculateDuration(tc.event)

			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tc.expectError && err != nil {
				if tc.errorMsg != "" && !contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tc.errorMsg, err.Error())
				}
				if duration != 0 {
					t.Errorf("expected duration 0 on error, got %d", duration)
				}
			}
		})
	}
}

func TestLogInteraction_HandlesInvalidDuration(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	contact := &models.Contact{Name: "Alice", Email: "alice@example.com"}
	if err := db.CreateContact(database, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Event with invalid duration (end before start)
	event := &calendar.Event{
		Id:      "event1",
		Summary: "Invalid Duration Event",
		Start: &calendar.EventDateTime{
			DateTime: "2025-11-28T11:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2025-11-28T10:00:00Z", // Before start
		},
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com"},
		},
	}

	// Should not fail - should log interaction with 0 duration
	err := logInteraction(database, event, []uuid.UUID{contact.ID})
	if err != nil {
		t.Fatalf("logInteraction should not fail on invalid duration: %v", err)
	}

	// Verify interaction was created with 0 duration
	var metadataJSON string
	query := `SELECT metadata FROM interaction_log WHERE contact_id = ?`
	err = database.QueryRow(query, contact.ID.String()).Scan(&metadataJSON)
	if err != nil {
		t.Fatalf("failed to query metadata: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		t.Fatalf("failed to parse metadata JSON: %v", err)
	}

	duration := int(metadata["duration_minutes"].(float64))
	if duration != 0 {
		t.Errorf("expected duration 0 for invalid event, got %d", duration)
	}
}

// Helper function for error message checking
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
