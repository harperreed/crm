// ABOUTME: Tests Markdown history directories, committed-event reads, prefix lookup, and limits.
// ABOUTME: Verifies corrupt event files fail loudly with their path and history sentinel.
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
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
		if !strings.Contains(err.Error(), "open directory") || !strings.Contains(err.Error(), path) {
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
	updated.UpdatedAt = clock.Add(30 * time.Second)
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

func TestMarkdownHistoryMutationCompanyLifecycle(t *testing.T) {
	store := newTestMarkdownStore(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	company := &models.Company{ID: uuid.New(), Name: "Before Inc", Domain: "before.example", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	before := mustGetCompany(t, store, company.ID)
	updated := cloneCompanyForTest(before)
	updated.Name = "After Inc"
	updated.Domain = "after.example"
	updated.CreatedAt = now.Add(time.Hour)
	updated.UpdatedAt = now.Add(time.Minute)
	now = now.Add(time.Minute)
	if err := store.UpdateCompany(updated); err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}
	after := mustGetCompany(t, store, company.ID)
	updateEvent := mustHistoryEventWithAction(t, store, company.ID, history.ActionUpdate)
	assertMarkdownHistoryEvent(t, updateEvent, history.EntityCompany, history.ActionUpdate, mustMarkdownCompanySnapshot(t, before), mustMarkdownCompanySnapshot(t, after), []uuid.UUID{company.ID})
	now = now.Add(time.Minute)
	if err := store.DeleteCompany(company.ID); err != nil {
		t.Fatalf("DeleteCompany: %v", err)
	}
	deleteEvent := mustHistoryEventWithAction(t, store, company.ID, history.ActionDelete)
	assertMarkdownHistoryEvent(t, deleteEvent, history.EntityCompany, history.ActionDelete, mustMarkdownCompanySnapshot(t, after), nil, []uuid.UUID{company.ID})
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

func TestMarkdownHistoryUpdateTimestampParity(t *testing.T) { //nolint:gocognit,funlen // Contact and company parity share one regression contract.
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	t.Run("contact", func(t *testing.T) {
		store := newTestMarkdownStore(t)
		store.now = func() time.Time { return now.Add(2 * time.Hour) }
		contact := &models.Contact{ID: uuid.New(), Name: "Contact", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
		if err := store.CreateContact(contact); err != nil {
			t.Fatalf("CreateContact: %v", err)
		}
		before := mustGetContact(t, store, contact.ID)

		zero := cloneContactForTest(before)
		zero.Name = "Zero timestamp"
		zero.UpdatedAt = time.Time{}
		zeroInput := cloneContactForTest(zero)
		if err := store.UpdateContact(zero); err == nil {
			t.Fatal("UpdateContact zero timestamp error = nil")
		}
		if !reflect.DeepEqual(zero, zeroInput) {
			t.Fatal("UpdateContact mutated zero-timestamp input")
		}
		if got := mustGetContact(t, store, contact.ID); !reflect.DeepEqual(got, before) {
			t.Fatalf("zero-timestamp update changed state: got %#v want %#v", got, before)
		}
		if got := mustHistoryEvents(t, store, contact.ID); len(got) != 1 {
			t.Fatalf("zero-timestamp history len = %d, want 1", len(got))
		}

		older := cloneContactForTest(before)
		older.Name = "Older timestamp"
		older.UpdatedAt = now.Add(-time.Hour)
		olderInput := cloneContactForTest(older)
		if err := store.UpdateContact(older); err != nil {
			t.Fatalf("UpdateContact older timestamp: %v", err)
		}
		if !reflect.DeepEqual(older, olderInput) {
			t.Fatal("UpdateContact mutated older-timestamp input")
		}
		after := mustGetContact(t, store, contact.ID)
		if !after.UpdatedAt.Equal(older.UpdatedAt) {
			t.Fatalf("UpdatedAt = %v, want %v", after.UpdatedAt, older.UpdatedAt)
		}
		event := mustHistoryEventWithAction(t, store, contact.ID, history.ActionUpdate)
		assertMarkdownHistoryEvent(t, event, history.EntityContact, history.ActionUpdate, mustMarkdownContactSnapshot(t, before), mustMarkdownContactSnapshot(t, after), []uuid.UUID{contact.ID})
	})

	t.Run("company", func(t *testing.T) {
		store := newTestMarkdownStore(t)
		store.now = func() time.Time { return now.Add(2 * time.Hour) }
		company := &models.Company{ID: uuid.New(), Name: "Company", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
		if err := store.CreateCompany(company); err != nil {
			t.Fatalf("CreateCompany: %v", err)
		}
		before := mustGetCompany(t, store, company.ID)
		zero := cloneCompanyForTest(before)
		zero.Name = "Zero timestamp"
		zero.UpdatedAt = time.Time{}
		zeroInput := cloneCompanyForTest(zero)
		if err := store.UpdateCompany(zero); err == nil {
			t.Fatal("UpdateCompany zero timestamp error = nil")
		}
		if !reflect.DeepEqual(zero, zeroInput) {
			t.Fatal("UpdateCompany mutated zero-timestamp input")
		}
		if got := mustGetCompany(t, store, company.ID); !reflect.DeepEqual(got, before) {
			t.Fatalf("zero-timestamp update changed state: got %#v want %#v", got, before)
		}
		if got := mustHistoryEvents(t, store, company.ID); len(got) != 1 {
			t.Fatalf("zero-timestamp history len = %d, want 1", len(got))
		}

		older := cloneCompanyForTest(before)
		older.Name = "Older timestamp"
		older.UpdatedAt = now.Add(-time.Hour)
		olderInput := cloneCompanyForTest(older)
		if err := store.UpdateCompany(older); err != nil {
			t.Fatalf("UpdateCompany older timestamp: %v", err)
		}
		if !reflect.DeepEqual(older, olderInput) {
			t.Fatal("UpdateCompany mutated older-timestamp input")
		}
		after := mustGetCompany(t, store, company.ID)
		if !after.UpdatedAt.Equal(older.UpdatedAt) {
			t.Fatalf("UpdatedAt = %v, want %v", after.UpdatedAt, older.UpdatedAt)
		}
		event := mustHistoryEventWithAction(t, store, company.ID, history.ActionUpdate)
		assertMarkdownHistoryEvent(t, event, history.EntityCompany, history.ActionUpdate, mustMarkdownCompanySnapshot(t, before), mustMarkdownCompanySnapshot(t, after), []uuid.UUID{company.ID})
	})
}

func TestMarkdownHistoryNilCollectionsAreCanonical(t *testing.T) { //nolint:gocognit,funlen // Both entity lifecycles must cover create, update, and restart.
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	t.Run("contact create and update", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		contact := &models.Contact{ID: uuid.New(), Name: "Nil Contact", CreatedAt: now, UpdatedAt: now}
		input := cloneContactPreservingNilForTest(contact)
		if err := store.CreateContact(contact); err != nil {
			t.Fatalf("CreateContact: %v", err)
		}
		if !reflect.DeepEqual(contact, input) {
			t.Fatal("CreateContact mutated nil collections")
		}
		fetched := mustGetContact(t, store, contact.ID)
		if fetched.Fields == nil || fetched.Tags == nil {
			t.Fatalf("fetched collections = %#v/%#v, want nonnil empty", fetched.Fields, fetched.Tags)
		}
		createEvent := mustOnlyHistoryEvent(t, store, contact.ID)
		assertSnapshotExactlyMatchesContact(t, createEvent.After, fetched)
		copyCommittedEventToPending(t, store, createEvent)
		store, err = NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("restart after nil create: %v", err)
		}

		populated := mustGetContact(t, store, contact.ID)
		populated.Fields = map[string]any{"old": "value"}
		populated.Tags = []string{"old"}
		populated.UpdatedAt = now.Add(time.Minute)
		if err := store.UpdateContact(populated); err != nil {
			t.Fatalf("UpdateContact populated: %v", err)
		}
		update := mustGetContact(t, store, contact.ID)
		update.Name = "Nil Contact Updated"
		update.Fields = nil
		update.Tags = nil
		update.UpdatedAt = now.Add(2 * time.Minute)
		updateInput := cloneContactPreservingNilForTest(update)
		if err := store.UpdateContact(update); err != nil {
			t.Fatalf("UpdateContact nil: %v", err)
		}
		if !reflect.DeepEqual(update, updateInput) {
			t.Fatal("UpdateContact mutated nil collections")
		}
		fetched = mustGetContact(t, store, contact.ID)
		if fetched.Fields == nil || fetched.Tags == nil {
			t.Fatalf("updated collections = %#v/%#v, want nonnil empty", fetched.Fields, fetched.Tags)
		}
		events := mustHistoryEvents(t, store, contact.ID)
		updateEvent := events[0]
		assertSnapshotExactlyMatchesContact(t, updateEvent.After, fetched)
		copyCommittedEventToPending(t, store, updateEvent)
		if _, err := NewMarkdownStore(dataDir, history.SourceCLI); err != nil {
			t.Fatalf("restart after nil update: %v", err)
		}
	})

	t.Run("company create and update", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("NewMarkdownStore: %v", err)
		}
		company := &models.Company{ID: uuid.New(), Name: "Nil Company", CreatedAt: now, UpdatedAt: now}
		input := cloneCompanyPreservingNilForTest(company)
		if err := store.CreateCompany(company); err != nil {
			t.Fatalf("CreateCompany: %v", err)
		}
		if !reflect.DeepEqual(company, input) {
			t.Fatal("CreateCompany mutated nil collections")
		}
		fetched := mustGetCompany(t, store, company.ID)
		if fetched.Fields == nil || fetched.Tags == nil {
			t.Fatalf("fetched collections = %#v/%#v, want nonnil empty", fetched.Fields, fetched.Tags)
		}
		createEvent := mustOnlyHistoryEvent(t, store, company.ID)
		assertSnapshotExactlyMatchesCompany(t, createEvent.After, fetched)
		copyCommittedEventToPending(t, store, createEvent)
		store, err = NewMarkdownStore(dataDir, history.SourceCLI)
		if err != nil {
			t.Fatalf("restart after nil create: %v", err)
		}

		populated := mustGetCompany(t, store, company.ID)
		populated.Fields = map[string]any{"old": "value"}
		populated.Tags = []string{"old"}
		populated.UpdatedAt = now.Add(time.Minute)
		if err := store.UpdateCompany(populated); err != nil {
			t.Fatalf("UpdateCompany populated: %v", err)
		}
		update := mustGetCompany(t, store, company.ID)
		update.Name = "Nil Company Updated"
		update.Fields = nil
		update.Tags = nil
		update.UpdatedAt = now.Add(2 * time.Minute)
		updateInput := cloneCompanyPreservingNilForTest(update)
		if err := store.UpdateCompany(update); err != nil {
			t.Fatalf("UpdateCompany nil: %v", err)
		}
		if !reflect.DeepEqual(update, updateInput) {
			t.Fatal("UpdateCompany mutated nil collections")
		}
		fetched = mustGetCompany(t, store, company.ID)
		if fetched.Fields == nil || fetched.Tags == nil {
			t.Fatalf("updated collections = %#v/%#v, want nonnil empty", fetched.Fields, fetched.Tags)
		}
		events := mustHistoryEvents(t, store, company.ID)
		updateEvent := events[0]
		assertSnapshotExactlyMatchesCompany(t, updateEvent.After, fetched)
		copyCommittedEventToPending(t, store, updateEvent)
		if _, err := NewMarkdownStore(dataDir, history.SourceCLI); err != nil {
			t.Fatalf("restart after nil update: %v", err)
		}
	})
}

func TestMarkdownHistoryPreservesExactJSONNumbers(t *testing.T) {
	values := map[string]any{
		"decimal":       json.Number("0.123456789012345678901234567890"),
		"negative_zero": json.Number("-0"),
		"exponent":      json.Number("1e+400"),
		"uint64":        json.Number("18446744073709551615"),
		"beyond":        json.Number("184467440737095516150"),
		"nested":        []any{json.Number("1.0000000000000000001"), map[string]any{"value": json.Number("9e-999")}},
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	store := newTestMarkdownStore(t)
	contact := &models.Contact{ID: uuid.New(), Name: "Numbers", Fields: values, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	company := &models.Company{ID: uuid.New(), Name: "Numbers", Fields: values, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	for entity, fields := range map[string]map[string]any{"contact": mustGetContact(t, store, contact.ID).Fields, "company": mustGetCompany(t, store, company.ID).Fields} {
		for key, want := range values {
			wantJSON, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			gotJSON, err := json.Marshal(fields[key])
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotJSON, wantJSON) {
				t.Fatalf("%s %s = %s, want %s", entity, key, gotJSON, wantJSON)
			}
		}
	}
}

func TestMarkdownFrontmatterAcceptsNullFields(t *testing.T) {
	id := uuid.New().String()
	now := "2026-08-21T20:00:00Z"
	t.Run("contact", func(t *testing.T) {
		var frontmatter contactFrontmatter
		data := []byte(fmt.Sprintf("id: %s\nname: Null Fields\nfields: null\ncreated_at: %s\nupdated_at: %s\n", id, now, now))
		if err := strictYAMLUnmarshal(data, &frontmatter); err != nil {
			t.Fatalf("strictYAMLUnmarshal: %v", err)
		}
		contact, err := frontmatterToContact(frontmatter)
		if err != nil {
			t.Fatalf("frontmatterToContact: %v", err)
		}
		if contact.Fields == nil || len(contact.Fields) != 0 {
			t.Fatalf("Fields = %#v, want nonnil empty map", contact.Fields)
		}
	})

	t.Run("company", func(t *testing.T) {
		var frontmatter companyFrontmatter
		data := []byte(fmt.Sprintf("id: %s\nname: Null Fields\nfields: null\ncreated_at: %s\nupdated_at: %s\n", id, now, now))
		if err := strictYAMLUnmarshal(data, &frontmatter); err != nil {
			t.Fatalf("strictYAMLUnmarshal: %v", err)
		}
		company, err := frontmatterToCompany(frontmatter)
		if err != nil {
			t.Fatalf("frontmatterToCompany: %v", err)
		}
		if company.Fields == nil || len(company.Fields) != 0 {
			t.Fatalf("Fields = %#v, want nonnil empty map", company.Fields)
		}
	})
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

func TestMarkdownRecoveryRejectsInexactCurrentFrontmatterDelimiters(t *testing.T) { //nolint:funlen // The table enumerates each malformed frontmatter form.
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		entityType history.EntityType
		content    func(uuid.UUID) string
	}{
		{
			name:       "contact opening delimiter has comment",
			entityType: history.EntityContact,
			content: func(id uuid.UUID) string {
				return fmt.Sprintf("--- # not exact\nid: %s\nname: Contact\ncreated_at: %s\nupdated_at: %s\n---\n", id, now.Format(time.RFC3339), now.Format(time.RFC3339))
			},
		},
		{
			name:       "company closing delimiter has suffix",
			entityType: history.EntityCompany,
			content: func(id uuid.UUID) string {
				return fmt.Sprintf("---\nid: %s\nname: Company\ncreated_at: %s\nupdated_at: %s\n---junk\n", id, now.Format(time.RFC3339), now.Format(time.RFC3339))
			},
		},
		{
			name:       "contact missing closing delimiter",
			entityType: history.EntityContact,
			content: func(id uuid.UUID) string {
				return fmt.Sprintf("---\nid: %s\nname: Contact\ncreated_at: %s\nupdated_at: %s\n", id, now.Format(time.RFC3339), now.Format(time.RFC3339))
			},
		},
		{
			name:       "company empty frontmatter",
			entityType: history.EntityCompany,
			content: func(uuid.UUID) string {
				return "---\n---\n"
			},
		},
		{
			name:       "contact duplicate required key",
			entityType: history.EntityContact,
			content: func(id uuid.UUID) string {
				return fmt.Sprintf("---\nid: %s\nid: %s\nname: Contact\ncreated_at: %s\nupdated_at: %s\n---\n", id, id, now.Format(time.RFC3339), now.Format(time.RFC3339))
			},
		},
		{
			name:       "company unknown field",
			entityType: history.EntityCompany,
			content: func(id uuid.UUID) string {
				return fmt.Sprintf("---\nid: %s\nname: Company\nunknown: value\ncreated_at: %s\nupdated_at: %s\n---\n", id, now.Format(time.RFC3339), now.Format(time.RFC3339))
			},
		},
		{
			name:       "contact missing frontmatter",
			entityType: history.EntityContact,
			content: func(uuid.UUID) string {
				return "plain markdown\n"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataDir := t.TempDir()
			store, err := NewMarkdownStore(dataDir, history.SourceCLI)
			if err != nil {
				t.Fatalf("NewMarkdownStore: %v", err)
			}
			id := uuid.New()
			var before json.RawMessage
			var currentPath string
			switch test.entityType {
			case history.EntityContact:
				before = mustMarkdownContactSnapshot(t, &models.Contact{ID: id, Name: "Contact", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now})
				currentPath = filepath.Join(store.contactsDir(), "malformed.md")
			case history.EntityCompany:
				before = mustMarkdownCompanySnapshot(t, &models.Company{ID: id, Name: "Company", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now})
				currentPath = filepath.Join(store.companiesDir(), "malformed.md")
			default:
				t.Fatalf("unsupported entity type %s", test.entityType)
			}
			if err := os.WriteFile(currentPath, []byte(test.content(id)), 0o600); err != nil {
				t.Fatalf("WriteFile current: %v", err)
			}
			event := mustHistoryEventForTest(t, test.entityType, id, []uuid.UUID{id}, history.ActionDelete, before, nil, now.Add(time.Minute))
			writePendingHistoryForTest(t, store, event)
			pendingPath := filepath.Join(store.historyPendingDir(), event.ID.String()+".json")
			_, err = NewMarkdownStore(dataDir, history.SourceCLI)
			if !errors.Is(err, ErrHistoryCorrupt) {
				t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryCorrupt", err)
			}
			if !strings.Contains(err.Error(), currentPath) {
				t.Fatalf("error = %q, want current path %q", err, currentPath)
			}
			if _, err := os.Stat(pendingPath); err != nil {
				t.Fatalf("pending event not preserved: %v", err)
			}
		})
	}
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
	if _, err := os.Stat(filepath.Join(recovered.contactsDir(), oldFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old filename remains after recovery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(recovered.contactsDir(), "new-name.md")); err != nil {
		t.Fatalf("repaired filename: %v", err)
	}
}

func TestMarkdownRecoveryRepairsLinkedRenameIntermediate(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	for _, entityType := range []history.EntityType{history.EntityContact, history.EntityCompany} {
		t.Run(string(entityType), func(t *testing.T) {
			dataDir := t.TempDir()
			store, err := NewMarkdownStore(dataDir, history.SourceCLI)
			if err != nil {
				t.Fatal(err)
			}
			id := uuid.New()
			var event *history.Event
			var oldPath, newPath string
			if entityType == history.EntityContact {
				before := &models.Contact{ID: id, Name: "Old", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
				after := cloneContactForTest(before)
				after.Name = "New"
				after.UpdatedAt = now.Add(time.Minute)
				oldPath, newPath = filepath.Join(store.contactsDir(), "old.md"), filepath.Join(store.contactsDir(), "new.md")
				if err := store.writeContact(after, "old.md"); err != nil {
					t.Fatal(err)
				}
				event = mustHistoryEventForTest(t, entityType, id, []uuid.UUID{id}, history.ActionUpdate, mustMarkdownContactSnapshot(t, before), mustMarkdownContactSnapshot(t, after), now.Add(time.Minute))
			} else {
				before := &models.Company{ID: id, Name: "Old", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
				after := cloneCompanyForTest(before)
				after.Name = "New"
				after.UpdatedAt = now.Add(time.Minute)
				oldPath, newPath = filepath.Join(store.companiesDir(), "old.md"), filepath.Join(store.companiesDir(), "new.md")
				if err := store.writeCompany(after, "old.md"); err != nil {
					t.Fatal(err)
				}
				event = mustHistoryEventForTest(t, entityType, id, []uuid.UUID{id}, history.ActionUpdate, mustMarkdownCompanySnapshot(t, before), mustMarkdownCompanySnapshot(t, after), now.Add(time.Minute))
			}
			if err := os.Link(oldPath, newPath); err != nil {
				t.Fatal(err)
			}
			writePendingHistoryForTest(t, store, event)
			if _, err := NewMarkdownStore(dataDir, history.SourceCLI); err != nil {
				t.Fatalf("recovery: %v", err)
			}
			if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("old path remains: %v", err)
			}
			if _, err := os.Stat(newPath); err != nil {
				t.Fatalf("new path: %v", err)
			}
		})
	}
}

func TestEnsureDurableDirectory(t *testing.T) {
	t.Run("creates nested path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "one", "two", "three")
		if err := ensureDurableDirectory(path); err != nil {
			t.Fatalf("ensureDurableDirectory: %v", err)
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("Stat = %#v, %v", info, err)
		}
	})
	t.Run("rejects file component", func(t *testing.T) {
		root := t.TempDir()
		file := filepath.Join(root, "file")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := ensureDurableDirectory(filepath.Join(file, "child")); err == nil {
			t.Fatal("error = nil")
		}
	})
}

func TestMarkdownRecoveryRejectsDuplicateCurrentEntityIDs(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	for _, entityType := range []history.EntityType{history.EntityContact, history.EntityCompany} {
		t.Run(string(entityType), func(t *testing.T) {
			dataDir := t.TempDir()
			store, err := NewMarkdownStore(dataDir, history.SourceCLI)
			if err != nil {
				t.Fatalf("NewMarkdownStore: %v", err)
			}
			id := uuid.New()
			var before json.RawMessage
			switch entityType {
			case history.EntityContact:
				value := &models.Contact{ID: id, Name: "Duplicate", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
				before = mustMarkdownContactSnapshot(t, value)
				if err := store.writeContact(value, "one.md"); err != nil {
					t.Fatal(err)
				}
				if err := store.writeContact(value, "two.md"); err != nil {
					t.Fatal(err)
				}
			case history.EntityCompany:
				value := &models.Company{ID: id, Name: "Duplicate", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
				before = mustMarkdownCompanySnapshot(t, value)
				if err := store.writeCompany(value, "one.md"); err != nil {
					t.Fatal(err)
				}
				if err := store.writeCompany(value, "two.md"); err != nil {
					t.Fatal(err)
				}
			}
			event := mustHistoryEventForTest(t, entityType, id, []uuid.UUID{id}, history.ActionDelete, before, nil, now.Add(time.Minute))
			writePendingHistoryForTest(t, store, event)
			_, err = NewMarkdownStore(dataDir, history.SourceCLI)
			if !errors.Is(err, ErrHistoryConflict) {
				t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryConflict", err)
			}
		})
	}
}

func TestMarkdownRecoveryRejectsMalformedRelationshipCurrent(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	rel := &models.Relationship{ID: uuid.New(), SourceID: uuid.New(), TargetID: uuid.New(), Type: "knows", CreatedAt: now}
	event := mustHistoryEventForTest(t, history.EntityRelationship, rel.ID, []uuid.UUID{rel.ID, rel.SourceID, rel.TargetID}, history.ActionDelete, mustMarkdownRelationshipSnapshot(t, rel), nil, now.Add(time.Minute))
	if err := os.WriteFile(store.relationshipsFile(), []byte("- id: not-a-uuid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writePendingHistoryForTest(t, store, event)
	pending := filepath.Join(store.historyPendingDir(), event.ID.String()+".json")
	_, err = NewMarkdownStore(dataDir, history.SourceCLI)
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("error = %v, want ErrHistoryCorrupt", err)
	}
	if _, err := os.Stat(pending); err != nil {
		t.Fatalf("pending not preserved: %v", err)
	}
}

func TestMarkdownPendingRejectsUnexpectedEntriesAndTrailingJSON(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	for _, variant := range []string{"visible-file", "directory", "trailing-json"} {
		t.Run(variant, func(t *testing.T) {
			dataDir := t.TempDir()
			store, err := NewMarkdownStore(dataDir, history.SourceCLI)
			if err != nil {
				t.Fatal(err)
			}
			switch variant {
			case "visible-file":
				err = os.WriteFile(filepath.Join(store.historyPendingDir(), "README"), []byte("unexpected"), 0o600)
			case "directory":
				err = os.Mkdir(filepath.Join(store.historyPendingDir(), "bad.json"), 0o700)
			case "trailing-json":
				contact := &models.Contact{ID: uuid.New(), Name: "Trailing", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
				event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, contact), now)
				data, marshalErr := marshalCommittedHistoryEvent(event)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				err = os.WriteFile(filepath.Join(store.historyPendingDir(), event.ID.String()+".json"), append(data, []byte("{}")...), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewMarkdownStore(dataDir, history.SourceCLI); !errors.Is(err, ErrHistoryCorrupt) {
				t.Fatalf("error = %v, want ErrHistoryCorrupt", err)
			}
		})
	}
}

func TestMarkdownPendingRejectsFIFOWithTempSuffix(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	path := filepath.Join(store.historyPendingDir(), "interrupted.tmp")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("Mkfifo: %v", err)
	}
	_, err = NewMarkdownStore(dataDir, history.SourceCLI)
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("NewMarkdownStore error = %v, want ErrHistoryCorrupt", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("NewMarkdownStore error = %q, want path %q", err, path)
	}
}

func TestMarkdownMutationDrainsPendingBeforeNextMutation(t *testing.T) {
	store := newTestMarkdownStore(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	contact := &models.Contact{ID: uuid.New(), Name: "Initial", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateContact(contact); err != nil {
		t.Fatal(err)
	}
	eventsDir := store.historyEventsDir()
	backupDir := eventsDir + "-backup"
	if err := os.Rename(eventsDir, backupDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventsDir, []byte("blocks publication"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := cloneContactForTest(contact)
	first.Name = "First"
	first.UpdatedAt = now.Add(time.Minute)
	if err := store.UpdateContact(first); err == nil {
		t.Fatal("first update error = nil")
	}
	if err := os.Remove(eventsDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backupDir, eventsDir); err != nil {
		t.Fatal(err)
	}
	second := cloneContactForTest(first)
	second.Name = "Second"
	second.UpdatedAt = now.Add(2 * time.Minute)
	if err := store.UpdateContact(second); err != nil {
		t.Fatalf("second update: %v", err)
	}
	entries, err := os.ReadDir(store.historyPendingDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("pending entries = %d, want 0", len(entries))
	}
	if got := mustGetContact(t, store, contact.ID); got.Name != "Second" {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestMarkdownDuplicateCreatesLeaveStoreHealthy(t *testing.T) {
	store := newTestMarkdownStore(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	contact := &models.Contact{ID: uuid.New(), Name: "Contact", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	company := &models.Company{ID: uuid.New(), Name: "Company", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	rel := &models.Relationship{ID: uuid.New(), SourceID: contact.ID, TargetID: company.ID, Type: "works_at", CreatedAt: now}
	if err := store.CreateContact(contact); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCompany(company); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatal(err)
	}
	for name, create := range map[string]func() error{
		"contact": func() error {
			duplicate := cloneContactForTest(contact)
			duplicate.Name = "Other"
			return store.CreateContact(duplicate)
		},
		"company": func() error {
			duplicate := cloneCompanyForTest(company)
			duplicate.Name = "Other"
			return store.CreateCompany(duplicate)
		},
		"relationship": func() error { duplicate := *rel; duplicate.Type = "other"; return store.CreateRelationship(&duplicate) },
	} {
		if err := create(); err == nil {
			t.Fatalf("duplicate %s error = nil", name)
		}
	}
	if entries, err := os.ReadDir(store.historyPendingDir()); err != nil || len(entries) != 0 {
		t.Fatalf("pending = %v, %v", entries, err)
	}
	if _, err := NewMarkdownStore(store.dataDir, history.SourceCLI); err != nil {
		t.Fatalf("restart: %v", err)
	}
}

func TestMarkdownCreateFilenameCollisionsDoNotOverwrite(t *testing.T) {
	store := newTestMarkdownStore(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	ids := []uuid.UUID{uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"), uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002"), uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000003")}
	for _, id := range ids {
		if err := store.CreateContact(&models.Contact{ID: id, Name: "Same Name", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatalf("CreateContact(%s): %v", id, err)
		}
	}
	for _, id := range ids {
		if got := mustGetContact(t, store, id); got.ID != id {
			t.Fatalf("contact ID = %s", got.ID)
		}
	}
	entries, err := os.ReadDir(store.contactsDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("contact files = %d, want 3", len(entries))
	}
	for _, id := range ids {
		if err := store.CreateCompany(&models.Company{ID: id, Name: "Same Name", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatalf("CreateCompany(%s): %v", id, err)
		}
	}
	for _, id := range ids {
		if got := mustGetCompany(t, store, id); got.ID != id {
			t.Fatalf("company ID = %s", got.ID)
		}
	}
	entries, err = os.ReadDir(store.companiesDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("company files = %d, want 3", len(entries))
	}
}

func TestMarkdownRecoveryCommittedCopyErrorsPreservePending(t *testing.T) {
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	contact := &models.Contact{ID: uuid.New(), Name: "Committed", Fields: map[string]any{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now}
	event := mustHistoryEventForTest(t, history.EntityContact, contact.ID, []uuid.UUID{contact.ID}, history.ActionCreate, nil, mustMarkdownContactSnapshot(t, contact), now)
	for _, variant := range []string{"corrupt", "current-not-after"} {
		t.Run(variant, func(t *testing.T) {
			dataDir := t.TempDir()
			store, err := NewMarkdownStore(dataDir, history.SourceCLI)
			if err != nil {
				t.Fatal(err)
			}
			writePendingHistoryForTest(t, store, event)
			finalPath := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
			if variant == "corrupt" {
				err = os.WriteFile(finalPath, []byte("corrupt"), 0o600)
			} else {
				err = store.writeCommittedHistoryEvent(event)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = NewMarkdownStore(dataDir, history.SourceCLI)
			want := ErrHistoryCorrupt
			if variant == "current-not-after" {
				want = ErrHistoryConflict
			}
			if !errors.Is(err, want) {
				t.Fatalf("error = %v, want %v", err, want)
			}
			if _, statErr := os.Stat(filepath.Join(store.historyPendingDir(), event.ID.String()+".json")); statErr != nil {
				t.Fatalf("pending not preserved: %v", statErr)
			}
		})
	}
}

func TestMarkdownRecoveryRelationshipDeletePreservesUnrelated(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
	target := &models.Relationship{ID: uuid.New(), SourceID: uuid.New(), TargetID: uuid.New(), Type: "target", CreatedAt: now}
	unrelated := &models.Relationship{ID: uuid.New(), SourceID: uuid.New(), TargetID: uuid.New(), Type: "unrelated", CreatedAt: now}
	if err := store.writeRelationships([]relationshipEntry{relationshipToEntry(target), relationshipToEntry(unrelated)}); err != nil {
		t.Fatal(err)
	}
	event := mustHistoryEventForTest(t, history.EntityRelationship, target.ID, []uuid.UUID{target.ID, target.SourceID, target.TargetID}, history.ActionDelete, mustMarkdownRelationshipSnapshot(t, target), nil, now.Add(time.Minute))
	writePendingHistoryForTest(t, store, event)
	recovered, err := NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("recovery: %v", err)
	}
	_, relationships, err := recovered.readRelationshipsStrict()
	if err != nil {
		t.Fatal(err)
	}
	if len(relationships) != 1 || relationships[0].ID != unrelated.ID {
		t.Fatalf("relationships = %#v, want only %s", relationships, unrelated.ID)
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

func mustHistoryEventWithAction(t *testing.T, store *MarkdownStore, id uuid.UUID, action history.Action) *history.Event {
	t.Helper()
	for _, event := range mustHistoryEvents(t, store, id) {
		if event.Action == action {
			return event
		}
	}
	t.Fatalf("history for %s has no %s event", id, action)
	return nil
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

func cloneCompanyForTest(company *models.Company) *models.Company {
	clone := *company
	clone.Tags = append([]string{}, company.Tags...)
	clone.Fields = cloneMapForTest(company.Fields)
	return &clone
}

func cloneContactPreservingNilForTest(contact *models.Contact) *models.Contact {
	clone := *contact
	if contact.Tags != nil {
		clone.Tags = append([]string{}, contact.Tags...)
	}
	if contact.Fields != nil {
		clone.Fields = cloneMapForTest(contact.Fields)
	}
	return &clone
}

func cloneCompanyPreservingNilForTest(company *models.Company) *models.Company {
	clone := *company
	if company.Tags != nil {
		clone.Tags = append([]string{}, company.Tags...)
	}
	if company.Fields != nil {
		clone.Fields = cloneMapForTest(company.Fields)
	}
	return &clone
}

func assertSnapshotExactlyMatchesContact(t *testing.T, snapshot json.RawMessage, contact *models.Contact) {
	t.Helper()
	want := mustMarkdownContactSnapshot(t, contact)
	equal, err := history.EqualSnapshots(history.EntityContact, snapshot, want)
	if err != nil || !equal {
		t.Fatalf("snapshot = %s, want authoritative %s", snapshot, want)
	}
}

func assertSnapshotExactlyMatchesCompany(t *testing.T, snapshot json.RawMessage, company *models.Company) {
	t.Helper()
	want := mustMarkdownCompanySnapshot(t, company)
	equal, err := history.EqualSnapshots(history.EntityCompany, snapshot, want)
	if err != nil || !equal {
		t.Fatalf("snapshot = %s, want authoritative %s", snapshot, want)
	}
}

func copyCommittedEventToPending(t *testing.T, store *MarkdownStore, event *history.Event) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(store.historyEventsDir(), event.ID.String()+".json")) //nolint:gosec // test path is under t.TempDir.
	if err != nil {
		t.Fatalf("ReadFile committed event: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.historyPendingDir(), event.ID.String()+".json"), data, 0o600); err != nil { //nolint:gosec // Test path is under t.TempDir and event ID is validated.
		t.Fatalf("WriteFile pending copy: %v", err)
	}
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
