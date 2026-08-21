// ABOUTME: Tests Markdown history directories, committed-event reads, prefix lookup, and limits.
// ABOUTME: Verifies corrupt event files fail loudly with their path and history sentinel.
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/models"
)

func TestMarkdownHistorySchema(t *testing.T) {
	store := newTestMarkdownStore(t)
	for _, dir := range []string{store.historyEventsDir(), store.historyPendingDir()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Stat(%q): %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", dir)
		}
	}
}

func TestSyncMarkdownHistoryDirectory(t *testing.T) {
	t.Run("syncs existing directory", func(t *testing.T) {
		store := newTestMarkdownStore(t)
		if err := syncMarkdownHistoryDirectory(store.historyEventsDir()); err != nil {
			t.Fatalf("syncMarkdownHistoryDirectory: %v", err)
		}
	})

	t.Run("reports open error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing")
		err := syncMarkdownHistoryDirectory(path)
		if err == nil {
			t.Fatal("syncMarkdownHistoryDirectory() error = nil")
		}
		if !strings.Contains(err.Error(), "open history events directory") {
			t.Fatalf("syncMarkdownHistoryDirectory() error = %q", err)
		}
	})
}

func TestMarkdownListHistoryAndGetHistoryEvent(t *testing.T) {
	store := newTestMarkdownStore(t)
	runHistoryReadContract(t, historyReadBackend{
		list:  store.ListHistory,
		get:   store.GetHistoryEvent,
		write: store.writeCommittedHistoryEvent,
	})
}

func TestMarkdownListHistoryLimits(t *testing.T) {
	store := newTestMarkdownStore(t)
	runHistoryLimitContract(t, historyReadBackend{
		list:  store.ListHistory,
		write: store.writeCommittedHistoryEvent,
	})
}

func TestMarkdownGetHistoryEventReportsCorruptFile(t *testing.T) {
	store := newTestMarkdownStore(t)
	path := filepath.Join(store.historyEventsDir(), "bad-event.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"extra":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := store.GetHistoryEvent("bad-ev")
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("GetHistoryEvent() error = %q, want path %q", err, path)
	}
}

func TestMarkdownGetHistoryEventRejectsUnknownFields(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	data, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	data = []byte(strings.Replace(string(data), `"schema_version": 1`, `"schema_version": 1, "unknown": true`, 1))
	if err := os.WriteFile(path, data, 0o600); err != nil { //nolint:gosec // path is built from the temporary test store.
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
}

func TestMarkdownGetHistoryEventRejectsMismatchedFilename(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	original := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	mismatched := filepath.Join(store.historyEventsDir(), "80000100-0000-0000-0000-000000000008.json")
	if err := os.Rename(original, mismatched); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	_, err := store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
	if !strings.Contains(err.Error(), mismatched) {
		t.Fatalf("GetHistoryEvent() error = %q, want path %q", err, mismatched)
	}
}

func TestMarkdownGetHistoryEventRejectsSymlink(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	committedPath := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	externalPath := filepath.Join(t.TempDir(), event.ID.String()+".json")
	if err := os.Rename(committedPath, externalPath); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := os.Symlink(externalPath, committedPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, err := store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
}

func TestMarkdownWriteCommittedHistoryEventIsIdempotent(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("first writeCommittedHistoryEvent: %v", err)
	}
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("second writeCommittedHistoryEvent: %v", err)
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventPreservesConflictingEvent(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	want, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	conflicting := *event
	conflicting.Source = history.SourceMCP

	err = store.writeCommittedHistoryEvent(&conflicting)
	if !errors.Is(err, ErrHistoryConflict) {
		t.Fatalf("writeCommittedHistoryEvent() error = %v, want ErrHistoryConflict", err)
	}
	got, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile after conflict: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("conflicting write changed committed event")
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventPreservesCorruptEvent(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	want := []byte("truncated event")
	if err := os.WriteFile(path, want, 0o600); err != nil { //nolint:gosec // path is built from the temporary test store.
		t.Fatalf("WriteFile: %v", err)
	}

	err := store.writeCommittedHistoryEvent(event)
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("writeCommittedHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
	got, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile after corrupt conflict: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("write changed corrupt committed event")
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventRejectsFinalSymlink(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	target := filepath.Join(t.TempDir(), "outside.json")
	want := []byte("outside target")
	if err := os.WriteFile(target, want, 0o600); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if err := store.writeCommittedHistoryEvent(event); err == nil {
		t.Fatal("writeCommittedHistoryEvent() error = nil")
	}
	got, err := os.ReadFile(target) //nolint:gosec // target is built from a temporary test directory.
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("committed write changed symlink target")
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("committed write replaced final symlink")
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventUsesPrivateMode(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("committed mode = %04o, want 0600", got)
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownHistoryReadsIgnoreTemporaryFiles(t *testing.T) {
	store := newTestMarkdownStore(t)
	path := filepath.Join(store.historyEventsDir(), ".interrupted-event.tmp")
	if err := os.WriteFile(path, []byte(`{"partial":`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := store.ListHistory(testHistoryEntityA.String(), 0)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListHistory() len = %d, want 0", len(got))
	}
}

func assertNoMarkdownHistoryTempFiles(t *testing.T, store *MarkdownStore) {
	t.Helper()
	entries, err := os.ReadDir(store.historyEventsDir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatalf("temporary history file remains: %s", entry.Name())
		}
	}
}

func TestMarkdownHistoryMutationContactLifecycle(t *testing.T) {
	store := newTestMarkdownStore(t)
	clock := time.Date(2026, 8, 21, 15, 4, 5, 0, time.FixedZone("CDT", -5*60*60))
	store.now = func() time.Time { return clock }
	created := &models.Contact{
		ID:        uuid.New(),
		Name:      "Ada Lovelace",
		Email:     "ada@example.com",
		Fields:    map[string]any{"nested": map[string]any{"exact": int64(9007199254740993)}},
		Tags:      []string{"math"},
		CreatedAt: clock.Add(-time.Hour),
		UpdatedAt: clock.Add(-time.Hour),
	}
	wantInput := cloneContactForTest(created)

	if err := store.CreateContact(created); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	if !reflect.DeepEqual(created, wantInput) {
		t.Fatal("CreateContact mutated its input")
	}
	afterCreate := mustGetContact(t, store, created.ID)
	createEvent := mustOnlyHistoryEvent(t, store, created.ID)
	assertMarkdownHistoryEvent(t, createEvent, history.EntityContact, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, afterCreate), []uuid.UUID{created.ID})

	updated := cloneContactForTest(afterCreate)
	updated.Name = "Ada King"
	updated.Email = "ada@king.example"
	updated.CreatedAt = clock.Add(24 * time.Hour)
	updated.UpdatedAt = time.Time{}
	wantUpdatedInput := cloneContactForTest(updated)
	clock = clock.Add(time.Minute)
	if err := store.UpdateContact(updated); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}
	if !reflect.DeepEqual(updated, wantUpdatedInput) {
		t.Fatal("UpdateContact mutated its input")
	}
	afterUpdate := mustGetContact(t, store, created.ID)
	if !afterUpdate.CreatedAt.Equal(afterCreate.CreatedAt) {
		t.Fatalf("CreatedAt = %v, want %v", afterUpdate.CreatedAt, afterCreate.CreatedAt)
	}
	events := mustHistoryEvents(t, store, created.ID)
	if len(events) != 2 {
		t.Fatalf("history len = %d, want 2", len(events))
	}
	assertMarkdownHistoryEvent(t, events[0], history.EntityContact, history.ActionUpdate, mustMarkdownContactSnapshot(t, afterCreate), mustMarkdownContactSnapshot(t, afterUpdate), []uuid.UUID{created.ID})

	clock = clock.Add(time.Minute)
	if err := store.DeleteContact(created.ID); err != nil {
		t.Fatalf("DeleteContact: %v", err)
	}
	events = mustHistoryEvents(t, store, created.ID)
	if len(events) != 3 {
		t.Fatalf("history len = %d, want 3", len(events))
	}
	assertMarkdownHistoryEvent(t, events[0], history.EntityContact, history.ActionDelete, mustMarkdownContactSnapshot(t, afterUpdate), nil, []uuid.UUID{created.ID})
}

func TestMarkdownHistoryMutationCompanyAndRelationship(t *testing.T) {
	store := newTestMarkdownStore(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	company := &models.Company{
		ID: uuid.New(), Name: "Exact Numbers", Fields: map[string]any{"exact": int64(9007199254740993)},
		Tags: []string{}, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	storedCompany := mustGetCompany(t, store, company.ID)
	companyEvent := mustOnlyHistoryEvent(t, store, company.ID)
	assertMarkdownHistoryEvent(t, companyEvent, history.EntityCompany, history.ActionCreate, nil, mustMarkdownCompanySnapshot(t, storedCompany), []uuid.UUID{company.ID})

	rel := &models.Relationship{ID: uuid.New(), SourceID: uuid.New(), TargetID: company.ID, Type: "knows", Context: "conference", CreatedAt: now}
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}
	relEvent := mustOnlyHistoryEvent(t, store, rel.ID)
	assertMarkdownHistoryEvent(t, relEvent, history.EntityRelationship, history.ActionCreate, nil, mustMarkdownRelationshipSnapshot(t, rel), []uuid.UUID{rel.ID, rel.SourceID, rel.TargetID})
	for _, entityID := range []uuid.UUID{rel.SourceID, rel.TargetID} {
		events := mustHistoryEvents(t, store, entityID)
		found := false
		for _, event := range events {
			found = found || event.ID == relEvent.ID
		}
		if !found {
			t.Fatalf("endpoint %s history = %#v, missing relationship event %s", entityID, events, relEvent.ID)
		}
	}
	now = now.Add(time.Minute)
	if err := store.DeleteRelationship(rel.ID); err != nil {
		t.Fatalf("DeleteRelationship: %v", err)
	}
	events := mustHistoryEvents(t, store, rel.ID)
	if len(events) != 2 || events[0].Action != history.ActionDelete {
		t.Fatalf("relationship history = %#v, want delete then create", events)
	}
	var relationshipYAML []relationshipEntry
	data, err := os.ReadFile(store.relationshipsFile()) //nolint:gosec // test path is under t.TempDir.
	if err != nil {
		t.Fatalf("ReadFile relationships: %v", err)
	}
	if err := yamlUnmarshal(data, &relationshipYAML); err != nil {
		t.Fatalf("relationships YAML after unlink: %v", err)
	}
}

func TestMarkdownHistoryNoOpContactAndCompany(t *testing.T) {
	store := newTestMarkdownStore(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	contact := &models.Contact{ID: uuid.New(), Name: "No Op", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	stored := mustGetContact(t, store, contact.ID)
	noop := cloneContactForTest(stored)
	noop.CreatedAt = now.Add(time.Hour)
	noop.UpdatedAt = now.Add(time.Minute)
	if err := store.UpdateContact(noop); err != nil {
		t.Fatalf("UpdateContact no-op: %v", err)
	}
	got := mustGetContact(t, store, contact.ID)
	if !reflect.DeepEqual(got, stored) {
		t.Fatalf("stored contact changed on timestamp-only update:\n got %#v\nwant %#v", got, stored)
	}
	if events := mustHistoryEvents(t, store, contact.ID); len(events) != 1 {
		t.Fatalf("contact history len = %d, want 1", len(events))
	}

	company := &models.Company{ID: uuid.New(), Name: "No Op Inc", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	storedCompany := mustGetCompany(t, store, company.ID)
	companyNoop := *storedCompany
	companyNoop.CreatedAt = now.Add(time.Hour)
	companyNoop.UpdatedAt = now.Add(time.Minute)
	if err := store.UpdateCompany(&companyNoop); err != nil {
		t.Fatalf("UpdateCompany no-op: %v", err)
	}
	if got := mustGetCompany(t, store, company.ID); !reflect.DeepEqual(got, storedCompany) {
		t.Fatalf("stored company changed on timestamp-only update:\n got %#v\nwant %#v", got, storedCompany)
	}
	if events := mustHistoryEvents(t, store, company.ID); len(events) != 1 {
		t.Fatalf("company history len = %d, want 1", len(events))
	}
}

func TestMarkdownRecoveryAppliesPendingUpdate(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	before := &models.Contact{ID: uuid.New(), Name: "Before", Fields: map[string]any{"exact": int64(9007199254740993)}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := store.writeContact(before, slugForName(before.Name, before.ID.String(), store.contactsDir())); err != nil {
		t.Fatalf("writeContact: %v", err)
	}
	after := cloneContactForTest(before)
	after.Name = "After"
	after.UpdatedAt = now.Add(time.Minute)
	event := mustHistoryEventForTest(t, history.EntityContact, before.ID, []uuid.UUID{before.ID}, history.ActionUpdate, mustMarkdownContactSnapshot(t, before), mustMarkdownContactSnapshot(t, after), now.Add(time.Minute))
	writePendingHistoryForTest(t, store, event)

	recovered, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore recovery: %v", err)
	}
	got := mustGetContact(t, recovered, before.ID)
	if got.Name != "After" || got.Fields["exact"] != int(9007199254740993) && got.Fields["exact"] != int64(9007199254740993) {
		t.Fatalf("recovered contact = %#v", got)
	}
	if _, err := os.Stat(filepath.Join(recovered.historyPendingDir(), event.ID.String()+".json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending event remains: %v", err)
	}
	if committed, err := recovered.GetHistoryEvent(event.ID.String()); err != nil || committed.ID != event.ID {
		t.Fatalf("GetHistoryEvent = %#v, %v", committed, err)
	}
}

func TestMarkdownRecoveryFinalizesPendingAlreadyApplied(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	contact := &models.Contact{ID: uuid.New(), Name: "Applied", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, contact), now)
	if err := store.writeContact(contact, slugForName(contact.Name, contact.ID.String(), store.contactsDir())); err != nil {
		t.Fatalf("writeContact: %v", err)
	}
	writePendingHistoryForTest(t, store, event)

	if _, err := NewMarkdownStore(dataDir, history.SourceCLI); err != nil {
		t.Fatalf("NewMarkdownStore recovery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.historyEventsDir(), event.ID.String()+".json")); err != nil {
		t.Fatalf("committed event: %v", err)
	}
}

func TestMarkdownRecoveryPreservesConflict(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	before := &models.Contact{ID: uuid.New(), Name: "Before", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	after := cloneContactForTest(before)
	after.Name = "After"
	other := cloneContactForTest(before)
	other.Name = "Manual edit"
	if err := store.writeContact(other, slugForName(other.Name, other.ID.String(), store.contactsDir())); err != nil {
		t.Fatalf("writeContact: %v", err)
	}
	event := mustHistoryEventForTest(t, history.EntityContact, before.ID, []uuid.UUID{before.ID}, history.ActionUpdate, mustMarkdownContactSnapshot(t, before), mustMarkdownContactSnapshot(t, after), now.Add(time.Minute))
	writePendingHistoryForTest(t, store, event)
	pendingPath := filepath.Join(store.historyPendingDir(), event.ID.String()+".json")

	_, err = NewMarkdownStore(dataDir, history.SourceCLI)
	if !errors.Is(err, ErrHistoryConflict) {
		t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryConflict", err)
	}
	if _, err := os.Stat(pendingPath); err != nil {
		t.Fatalf("pending event not preserved: %v", err)
	}
}

func TestMarkdownPendingRejectsMalformedAndIgnoresTemp(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.historyPendingDir(), ".publish.tmp"), []byte(`{"partial":`), 0o600); err != nil {
		t.Fatalf("WriteFile temp: %v", err)
	}
	if _, err := NewMarkdownStore(dataDir, history.SourceCLI); err != nil {
		t.Fatalf("NewMarkdownStore with temp: %v", err)
	}
	badPath := filepath.Join(store.historyPendingDir(), uuid.New().String()+".json")
	if err := os.WriteFile(badPath, []byte(`{"schema_version":1,"unknown":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile malformed: %v", err)
	}
	if _, err := NewMarkdownStore(dataDir, history.SourceCLI); !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryCorrupt", err)
	}
}

func TestMarkdownRecoveryCreateAndDeleteWithAbsentCurrent(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	t.Run("create applies from absent", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		contact := &models.Contact{ID: uuid.New(), Name: "Created", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
		event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, contact), now)
		writePendingHistoryForTest(t, store, event)
		recovered, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore recovery: %v", err)
		}
		if got := mustGetContact(t, recovered, contact.ID); got.Name != contact.Name {
			t.Fatalf("recovered contact name = %q, want %q", got.Name, contact.Name)
		}
	})

	t.Run("delete finalizes from absent", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		contact := &models.Contact{ID: uuid.New(), Name: "Deleted", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
		event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionDelete, mustMarkdownContactSnapshot(t, contact), nil, now)
		writePendingHistoryForTest(t, store, event)
		recovered, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore recovery: %v", err)
		}
		if _, err := recovered.GetHistoryEvent(event.ID.String()); err != nil {
			t.Fatalf("GetHistoryEvent: %v", err)
		}
		if _, err := recovered.GetContact(contact.ID); !errors.Is(err, ErrContactNotFound) {
			t.Fatalf("GetContact error = %v, want ErrContactNotFound", err)
		}
	})
}

func TestMarkdownPendingRejectsMismatchedFilenameAndSymlink(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	contact := &models.Contact{ID: uuid.New(), Name: "Pending", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, contact), now)
	data, err := marshalCommittedHistoryEvent(event)
	if err != nil {
		t.Fatalf("marshalCommittedHistoryEvent: %v", err)
	}

	t.Run("filename mismatch", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		path := filepath.Join(store.historyPendingDir(), uuid.New().String()+".json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if _, err := NewMarkdownStore(dataDir, history.SourceCLI); !errors.Is(err, ErrHistoryCorrupt) {
			t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryCorrupt", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		target := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(target, data, 0o600); err != nil {
			t.Fatalf("WriteFile target: %v", err)
		}
		path := filepath.Join(store.historyPendingDir(), event.ID.String()+".json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		if _, err := NewMarkdownStore(dataDir, history.SourceCLI); !errors.Is(err, ErrHistoryCorrupt) {
			t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryCorrupt", err)
		}
		outside, err := os.ReadFile(target) //nolint:gosec // target is under t.TempDir.
		if err != nil || !bytes.Equal(outside, data) {
			t.Fatalf("outside symlink target changed: %q, %v", outside, err)
		}
	})
}

func TestMarkdownRecoveryHandlesCommittedAndPendingCopies(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	contact := &models.Contact{ID: uuid.New(), Name: "Applied", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, contact), now)

	t.Run("identical copy removes pending", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		if err := store.writeContact(contact, slugForName(contact.Name, contact.ID.String(), store.contactsDir())); err != nil {
			t.Fatalf("writeContact: %v", err)
		}
		if err := store.writeCommittedHistoryEvent(event); err != nil {
			t.Fatalf("writeCommittedHistoryEvent: %v", err)
		}
		writePendingHistoryForTest(t, store, event)
		if _, err := NewMarkdownStore(dataDir, history.SourceCLI); err != nil {
			t.Fatalf("NewMarkdownStore recovery: %v", err)
		}
		if _, err := os.Stat(filepath.Join(store.historyPendingDir(), event.ID.String()+".json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("pending event remains: %v", err)
		}
	})

	t.Run("conflicting committed copy preserves pending", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		if err := store.writeContact(contact, slugForName(contact.Name, contact.ID.String(), store.contactsDir())); err != nil {
			t.Fatalf("writeContact: %v", err)
		}
		conflicting := *event
		conflicting.Source = history.SourceMCP
		if err := store.writeCommittedHistoryEvent(&conflicting); err != nil {
			t.Fatalf("writeCommittedHistoryEvent: %v", err)
		}
		writePendingHistoryForTest(t, store, event)
		_, err = NewMarkdownStore(dataDir, history.SourceCLI)
		if !errors.Is(err, ErrHistoryConflict) {
			t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryConflict", err)
		}
		if _, err := os.Stat(filepath.Join(store.historyPendingDir(), event.ID.String()+".json")); err != nil {
			t.Fatalf("pending event not preserved: %v", err)
		}
	})
}

func TestMarkdownRecoveryRelationshipPreservesUnrelatedEntries(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	unrelated := &models.Relationship{ID: uuid.New(), SourceID: uuid.New(), TargetID: uuid.New(), Type: "knows", CreatedAt: now}
	if err := store.writeRelationships([]relationshipEntry{relationshipToEntry(unrelated)}); err != nil {
		t.Fatalf("writeRelationships: %v", err)
	}
	created := &models.Relationship{ID: uuid.New(), SourceID: uuid.New(), TargetID: uuid.New(), Type: "works_at", CreatedAt: now}
	event := mustHistoryEventForTest(t, history.EntityRelationship, created.ID, []uuid.UUID{created.ID, created.SourceID, created.TargetID}, history.ActionCreate, nil, mustMarkdownRelationshipSnapshot(t, created), now)
	writePendingHistoryForTest(t, store, event)
	recovered, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore recovery: %v", err)
	}
	entries, relationships, err := recovered.readRelationshipsStrict()
	if err != nil {
		t.Fatalf("readRelationshipsStrict: %v", err)
	}
	if len(entries) != 2 || len(relationships) != 2 {
		t.Fatalf("relationships len = %d/%d, want 2/2", len(entries), len(relationships))
	}
	ids := map[uuid.UUID]bool{}
	for _, relationship := range relationships {
		ids[relationship.ID] = true
	}
	if !ids[unrelated.ID] || !ids[created.ID] {
		t.Fatalf("relationship IDs = %v, want %s and %s", ids, unrelated.ID, created.ID)
	}
}

func TestMarkdownRecoveryAcceptsRenamedContactAfterStateAtOldFilename(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	before := &models.Contact{ID: uuid.New(), Name: "Old Name", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	after := cloneContactForTest(before)
	after.Name = "New Name"
	after.UpdatedAt = now.Add(time.Minute)
	oldFilename := slugForName(before.Name, before.ID.String(), store.contactsDir())
	if err := store.writeContact(after, oldFilename); err != nil {
		t.Fatalf("writeContact intermediate state: %v", err)
	}
	event := mustHistoryEventForTest(t, history.EntityContact, before.ID, []uuid.UUID{before.ID}, history.ActionUpdate, mustMarkdownContactSnapshot(t, before), mustMarkdownContactSnapshot(t, after), now.Add(time.Minute))
	writePendingHistoryForTest(t, store, event)
	recovered, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore recovery: %v", err)
	}
	if got := mustGetContact(t, recovered, before.ID); got.Name != after.Name {
		t.Fatalf("recovered name = %q, want %q", got.Name, after.Name)
	}
	if _, err := recovered.GetHistoryEvent(event.ID.String()); err != nil {
		t.Fatalf("GetHistoryEvent: %v", err)
	}
}

func mustHistoryEvents(t *testing.T, store *MarkdownStore, id uuid.UUID) []*history.Event {
	t.Helper()
	summaries, err := store.ListHistory(id.String(), 100)
	if err != nil {
		t.Fatalf("ListHistory(%s): %v", id, err)
	}
	events := make([]*history.Event, 0, len(summaries))
	for _, summary := range summaries {
		event, err := store.GetHistoryEvent(summary.ID.String())
		if err != nil {
			t.Fatalf("GetHistoryEvent(%s): %v", summary.ID, err)
		}
		events = append(events, event)
	}
	return events
}

func mustOnlyHistoryEvent(t *testing.T, store *MarkdownStore, id uuid.UUID) *history.Event {
	t.Helper()
	events := mustHistoryEvents(t, store, id)
	if len(events) != 1 {
		t.Fatalf("history len = %d, want 1", len(events))
	}
	return events[0]
}

func assertMarkdownHistoryEvent(t *testing.T, event *history.Event, entityType history.EntityType, action history.Action, before, after json.RawMessage, related []uuid.UUID) {
	t.Helper()
	if event.EntityType != entityType || event.Action != action || event.Source != history.SourceCLI {
		t.Fatalf("event type/action/source = %s/%s/%s, want %s/%s/%s", event.EntityType, event.Action, event.Source, entityType, action, history.SourceCLI)
	}
	beforeEqual, err := history.EqualSnapshots(entityType, event.Before, before)
	if err != nil || !beforeEqual {
		t.Fatalf("before snapshot equality = %v, %v", beforeEqual, err)
	}
	afterEqual, err := history.EqualSnapshots(entityType, event.After, after)
	if err != nil || !afterEqual {
		t.Fatalf("after snapshot equality = %v, %v", afterEqual, err)
	}
	wantRelated := append([]uuid.UUID(nil), related...)
	for i := 0; i < len(wantRelated); i++ {
		for j := i + 1; j < len(wantRelated); j++ {
			if strings.Compare(wantRelated[i].String(), wantRelated[j].String()) > 0 {
				wantRelated[i], wantRelated[j] = wantRelated[j], wantRelated[i]
			}
		}
	}
	if !reflect.DeepEqual(event.RelatedEntityIDs, wantRelated) {
		t.Fatalf("related IDs = %v, want %v", event.RelatedEntityIDs, wantRelated)
	}
}

func mustMarkdownContactSnapshot(t *testing.T, contact *models.Contact) json.RawMessage {
	t.Helper()
	snapshot, err := history.SnapshotContact(contact)
	if err != nil {
		t.Fatalf("SnapshotContact: %v", err)
	}
	return snapshot
}

func mustMarkdownCompanySnapshot(t *testing.T, company *models.Company) json.RawMessage {
	t.Helper()
	snapshot, err := history.SnapshotCompany(company)
	if err != nil {
		t.Fatalf("SnapshotCompany: %v", err)
	}
	return snapshot
}

func mustMarkdownRelationshipSnapshot(t *testing.T, relationship *models.Relationship) json.RawMessage {
	t.Helper()
	snapshot, err := history.SnapshotRelationship(relationship)
	if err != nil {
		t.Fatalf("SnapshotRelationship: %v", err)
	}
	return snapshot
}

func mustGetContact(t *testing.T, store *MarkdownStore, id uuid.UUID) *models.Contact {
	t.Helper()
	contact, err := store.GetContact(id)
	if err != nil {
		t.Fatalf("GetContact(%s): %v", id, err)
	}
	return contact
}

func mustGetCompany(t *testing.T, store *MarkdownStore, id uuid.UUID) *models.Company {
	t.Helper()
	company, err := store.GetCompany(id)
	if err != nil {
		t.Fatalf("GetCompany(%s): %v", id, err)
	}
	return company
}

func cloneContactForTest(contact *models.Contact) *models.Contact {
	clone := *contact
	clone.Tags = append([]string{}, contact.Tags...)
	clone.Fields = cloneMapForTest(contact.Fields)
	return &clone
}

func cloneMapForTest(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		if nested, ok := value.(map[string]any); ok {
			clone[key] = cloneMapForTest(nested)
		} else {
			clone[key] = value
		}
	}
	return clone
}

func mustHistoryEventForTest(t *testing.T, entityType history.EntityType, id uuid.UUID, related []uuid.UUID, action history.Action, before, after json.RawMessage, occurredAt time.Time) *history.Event {
	t.Helper()
	event, err := history.NewEvent(entityType, id, related, action, history.SourceCLI, before, after, occurredAt)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	return event
}

func writePendingHistoryForTest(t *testing.T, store *MarkdownStore, event *history.Event) {
	t.Helper()
	data, err := marshalCommittedHistoryEvent(event)
	if err != nil {
		t.Fatalf("marshalCommittedHistoryEvent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.historyPendingDir(), event.ID.String()+".json"), data, 0o600); err != nil {
		t.Fatalf("WriteFile pending: %v", err)
	}
}
