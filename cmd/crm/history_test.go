// ABOUTME: Tests the CLI commands that list and inspect immutable CRM history.
// ABOUTME: Uses a real temporary store to verify user-visible history output.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/v2/internal/config"
	"github.com/harperreed/crm/v2/internal/history"
	"github.com/harperreed/crm/v2/internal/models"
)

func TestHistorySummaryFormatterSortsChangedFields(t *testing.T) {
	summary := &history.Summary{
		ID:            uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		EntityType:    history.EntityContact,
		Action:        history.ActionUpdate,
		Source:        history.SourceCLI,
		OccurredAt:    time.Date(2026, time.August, 21, 12, 34, 56, 0, time.FixedZone("offset", -5*60*60)),
		ChangedFields: []string{"name", "email"},
	}

	want := "2026-08-21T17:34:56Z  UPDATE contact [cli] aaaaaaaa  email, name"
	if got := formatHistorySummary(summary); got != want {
		t.Errorf("formatHistorySummary() = %q, want %q", got, want)
	}
}

func TestHistoryEventFormatterPrettyPrintsSnapshots(t *testing.T) {
	event := &history.Event{
		ID:               uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		SchemaVersion:    1,
		EntityType:       history.EntityContact,
		EntityID:         uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		RelatedEntityIDs: []uuid.UUID{uuid.MustParse("11111111-2222-3333-4444-555555555555")},
		Action:           history.ActionCreate,
		Source:           history.SourceCLI,
		OccurredAt:       time.Date(2026, time.August, 21, 12, 34, 56, 0, time.FixedZone("offset", -5*60*60)),
		Before:           json.RawMessage(`null`),
		After:            json.RawMessage(`{"name":"Ada","email":"ada@example.com"}`),
	}

	want := "ID: aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\n" +
		"Schema: 1\n" +
		"Occurred: 2026-08-21T17:34:56Z\n" +
		"Entity: contact 11111111-2222-3333-4444-555555555555\n" +
		"Action: CREATE\n" +
		"Source: cli\n" +
		"Related: 11111111-2222-3333-4444-555555555555\n" +
		"Before:\nnull\n" +
		"After:\n{\n  \"name\": \"Ada\",\n  \"email\": \"ada@example.com\"\n}\n"
	got, err := formatHistoryEvent(event)
	if err != nil {
		t.Fatalf("formatHistoryEvent: %v", err)
	}
	if got != want {
		t.Errorf("formatHistoryEvent() = %q, want %q", got, want)
	}
}

func TestHistoryTimelineFormatsSummaries(t *testing.T) {
	dataDir := configureHistoryCommand(t)
	cfg := &config.Config{Backend: "sqlite", DataDir: dataDir}
	testStore, err := cfg.OpenStorage(history.SourceCLI)
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	contact := models.NewContact("Old Name")
	contact.Email = "old@example.com"
	if err := testStore.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	contact.Name = "New Name"
	contact.Email = "new@example.com"
	contact.Touch()
	if err := testStore.UpdateContact(contact); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}
	summaries, err := testStore.ListHistory(contact.ID.String(), 1)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if err := testStore.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, err := runRootCommand(t, "history", contact.ID.String()[:8], "--limit", "1")
	if err != nil {
		t.Fatalf("Execute history: %v", err)
	}
	want := summaries[0].OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00") +
		"  UPDATE contact [cli] " + summaries[0].ID.String()[:8] + "  email, name\n"
	if got != want {
		t.Errorf("history output = %q, want %q", got, want)
	}
}

func TestHistoryTimelineWithoutEvents(t *testing.T) {
	configureHistoryCommand(t)

	got, err := runRootCommand(t, "history", "aaaaaaaa")
	if err != nil {
		t.Fatalf("Execute history: %v", err)
	}
	if want := "No history found.\n"; got != want {
		t.Errorf("history output = %q, want %q", got, want)
	}
}

func TestHistoryCommandRestoresLimitFlagBetweenRuns(t *testing.T) {
	dataDir := configureHistoryCommand(t)
	cfg := &config.Config{Backend: "sqlite", DataDir: dataDir}
	testStore, err := cfg.OpenStorage(history.SourceCLI)
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	contact := models.NewContact("First Name")
	if err := testStore.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	for _, name := range []string{"Second Name", "Third Name"} {
		contact.Name = name
		contact.Touch()
		if err := testStore.UpdateContact(contact); err != nil {
			t.Fatalf("UpdateContact: %v", err)
		}
	}
	if err := testStore.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	limited, err := runRootCommand(t, "history", contact.ID.String(), "--limit", "1")
	if err != nil {
		t.Fatalf("Execute limited history: %v", err)
	}
	if got := len(strings.Split(strings.TrimSpace(limited), "\n")); got != 1 {
		t.Fatalf("limited history line count = %d, want 1", got)
	}
	unlimited, err := runRootCommand(t, "history", contact.ID.String())
	if err != nil {
		t.Fatalf("Execute default history: %v", err)
	}
	if got := len(strings.Split(strings.TrimSpace(unlimited), "\n")); got != 3 {
		t.Errorf("default history line count = %d, want 3:\n%s", got, unlimited)
	}
}

func TestHistoryShowFormatsEvent(t *testing.T) {
	dataDir := configureHistoryCommand(t)
	cfg := &config.Config{Backend: "sqlite", DataDir: dataDir}
	testStore, err := cfg.OpenStorage(history.SourceCLI)
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	contact := models.NewContact("History Person")
	if err := testStore.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	summaries, err := testStore.ListHistory(contact.ID.String(), 1)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	event, err := testStore.GetHistoryEvent(summaries[0].ID.String())
	if err != nil {
		t.Fatalf("GetHistoryEvent: %v", err)
	}
	if err := testStore.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, err := runRootCommand(t, "history", "show", event.ID.String()[:8])
	if err != nil {
		t.Fatalf("Execute history show: %v", err)
	}
	checks := []string{
		"ID: " + event.ID.String() + "\n",
		"Occurred: " + event.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00") + "\n",
		"Entity: contact " + event.EntityID.String() + "\n",
		"Action: CREATE\n",
		"Source: cli\n",
		"Before:\nnull\n",
		"After:\n{\n  \"id\": \"" + contact.ID.String() + "\"",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("history show output missing %q:\n%s", check, got)
		}
	}
}

func configureHistoryCommand(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "config")
	dataHome := filepath.Join(tempDir, "data-home")
	dataDir := filepath.Join(tempDir, "crm-data")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	if err := os.MkdirAll(filepath.Join(configHome, "crm"), 0o700); err != nil {
		t.Fatalf("MkdirAll config: %v", err)
	}
	encoded, err := json.Marshal(&config.Config{Backend: "sqlite", DataDir: dataDir})
	if err != nil {
		t.Fatalf("Marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "crm", "config.json"), encoded, 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	return dataDir
}

func runRootCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	limitFlag := historyCmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("history limit flag is not registered")
	}
	previousLimit := limitFlag.Value.String()
	previousLimitChanged := limitFlag.Changed
	capturePath := filepath.Join(t.TempDir(), "stdout")
	capture, err := os.Create(capturePath) //nolint:gosec // test-owned temporary path
	if err != nil {
		t.Fatalf("Create stdout capture: %v", err)
	}
	previousStdout := os.Stdout
	os.Stdout = capture
	rootCmd.SetArgs(args)
	rootCmd.SetOut(capture)
	rootCmd.SetErr(capture)
	defer func() {
		os.Stdout = previousStdout
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		store = nil
		if err := limitFlag.Value.Set(previousLimit); err != nil {
			t.Errorf("restore history limit flag: %v", err)
		}
		limitFlag.Changed = previousLimitChanged
	}()

	executeErr := Execute()
	if err := capture.Close(); err != nil {
		t.Fatalf("Close stdout capture: %v", err)
	}
	output, err := os.ReadFile(capturePath) //nolint:gosec // test-owned temporary path
	if err != nil {
		t.Fatalf("ReadFile stdout capture: %v", err)
	}
	return string(output), executeErr
}
