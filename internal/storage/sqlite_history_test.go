// ABOUTME: Tests SQLite history schema, committed-event reads, prefix lookup, and limits.
// ABOUTME: Shares the backend-neutral history read contract with the Markdown tests.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/models"
)

type historyReadBackend struct {
	list  func(string, int) ([]*history.Summary, error)
	get   func(string) (*history.Event, error)
	write func(*history.Event) error
}

func TestSQLiteHistorySchema(t *testing.T) {
	store := newTestStore(t)
	for _, object := range []struct {
		kind string
		name string
	}{
		{kind: "table", name: "history_events"},
		{kind: "table", name: "history_event_entities"},
		{kind: "index", name: "idx_history_event_entities_entity_id"},
		{kind: "index", name: "idx_history_events_occurred_at"},
	} {
		var name string
		err := store.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = ? AND name = ?",
			object.kind,
			object.name,
		).Scan(&name)
		if err != nil {
			t.Errorf("%s %q not found: %v", object.kind, object.name, err)
		}
	}
}

func TestSQLiteListHistoryAndGetHistoryEvent(t *testing.T) {
	store := newTestStore(t)
	backend := historyReadBackend{
		list: store.ListHistory,
		get:  store.GetHistoryEvent,
		write: func(event *history.Event) error {
			tx, err := store.db.Begin()
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback() }()
			if err := insertSQLiteHistoryEvent(tx, event); err != nil {
				return err
			}
			return tx.Commit()
		},
	}
	runHistoryReadContract(t, backend)
}

func TestSQLiteListHistoryLimits(t *testing.T) {
	store := newTestStore(t)
	backend := historyReadBackend{
		list: store.ListHistory,
		write: func(event *history.Event) error {
			tx, err := store.db.Begin()
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback() }()
			if err := insertSQLiteHistoryEvent(tx, event); err != nil {
				return err
			}
			return tx.Commit()
		},
	}
	runHistoryLimitContract(t, backend)
}

func TestSQLiteGetHistoryEventReportsCorruptRows(t *testing.T) {
	store := newTestStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := insertSQLiteHistoryEvent(tx, event); err != nil {
		t.Fatalf("insertSQLiteHistoryEvent: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if _, err := store.db.Exec("UPDATE history_events SET schema_version = 99 WHERE id = ?", event.ID.String()); err != nil {
		t.Fatalf("corrupt event row: %v", err)
	}

	_, err = store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
}

func TestSQLiteGetHistoryEventReportsCorruptRelatedEntity(t *testing.T) {
	store := newTestStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := insertSQLiteHistoryEvent(tx, event); err != nil {
		t.Fatalf("insertSQLiteHistoryEvent: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if _, err := store.db.Exec(
		"UPDATE history_event_entities SET entity_id = 'not-a-uuid' WHERE event_id = ?",
		event.ID.String(),
	); err != nil {
		t.Fatalf("corrupt related entity row: %v", err)
	}

	_, err = store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
}

func TestSqliteHistoryMutationContact(t *testing.T) {
	store := newTestStore(t)
	eventTimes := fixedSQLiteHistoryTimes(store)
	contact := &models.Contact{
		ID:        uuid.MustParse("11000000-0000-0000-0000-000000000001"),
		Name:      "Ada Lovelace",
		Email:     "ada@example.com",
		Phone:     "+1-555-0101",
		Fields:    map[string]any{"role": "engineer"},
		Tags:      []string{"vip"},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	createdSnapshot := mustContactSnapshot(t, contact)

	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	contact.Name = "Augusta Ada King"
	contact.Email = "ada@lovelace.example"
	contact.Fields = map[string]any{"role": "mathematician"}
	contact.Tags = []string{"vip", "history"}
	contact.UpdatedAt = testHistoryTime.Add(time.Hour)
	updatedSnapshot := mustContactSnapshot(t, contact)
	if err := store.UpdateContact(contact); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}
	if err := store.DeleteContact(contact.ID); err != nil {
		t.Fatalf("DeleteContact: %v", err)
	}

	assertSQLiteMutationHistory(t, store, contact.ID, []sqliteExpectedHistoryEvent{
		{action: history.ActionDelete, occurredAt: eventTimes[2], before: updatedSnapshot, after: nil, related: []uuid.UUID{contact.ID}},
		{action: history.ActionUpdate, occurredAt: eventTimes[1], before: createdSnapshot, after: updatedSnapshot, related: []uuid.UUID{contact.ID}},
		{action: history.ActionCreate, occurredAt: eventTimes[0], before: nil, after: createdSnapshot, related: []uuid.UUID{contact.ID}},
	})
}

func TestSqliteHistoryMutationCompany(t *testing.T) {
	store := newTestStore(t)
	eventTimes := fixedSQLiteHistoryTimes(store)
	company := &models.Company{
		ID:        uuid.MustParse("22000000-0000-0000-0000-000000000002"),
		Name:      "Analytical Engines Ltd",
		Domain:    "engines.example",
		Fields:    map[string]any{"size": float64(12)},
		Tags:      []string{"partner"},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	createdSnapshot := mustCompanySnapshot(t, company)

	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	company.Name = "Difference Engines Ltd"
	company.Domain = "difference.example"
	company.Fields = map[string]any{"size": float64(24)}
	company.Tags = []string{"customer"}
	company.UpdatedAt = testHistoryTime.Add(time.Hour)
	updatedSnapshot := mustCompanySnapshot(t, company)
	if err := store.UpdateCompany(company); err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}
	if err := store.DeleteCompany(company.ID); err != nil {
		t.Fatalf("DeleteCompany: %v", err)
	}

	assertSQLiteMutationHistory(t, store, company.ID, []sqliteExpectedHistoryEvent{
		{action: history.ActionDelete, occurredAt: eventTimes[2], before: updatedSnapshot, after: nil, related: []uuid.UUID{company.ID}},
		{action: history.ActionUpdate, occurredAt: eventTimes[1], before: createdSnapshot, after: updatedSnapshot, related: []uuid.UUID{company.ID}},
		{action: history.ActionCreate, occurredAt: eventTimes[0], before: nil, after: createdSnapshot, related: []uuid.UUID{company.ID}},
	})
}

func TestSqliteHistoryMutationRelationship(t *testing.T) {
	store := newTestStore(t)
	eventTimes := fixedSQLiteHistoryTimes(store)
	relationship := &models.Relationship{
		ID:        uuid.MustParse("33000000-0000-0000-0000-000000000003"),
		SourceID:  uuid.MustParse("11000000-0000-0000-0000-000000000001"),
		TargetID:  uuid.MustParse("22000000-0000-0000-0000-000000000002"),
		Type:      "works_at",
		Context:   "analytical engines",
		CreatedAt: testHistoryTime,
	}
	snapshot := mustRelationshipSnapshot(t, relationship)
	related := []uuid.UUID{relationship.SourceID, relationship.TargetID, relationship.ID}

	if err := store.CreateRelationship(relationship); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}
	if err := store.DeleteRelationship(relationship.ID); err != nil {
		t.Fatalf("DeleteRelationship: %v", err)
	}

	want := []sqliteExpectedHistoryEvent{
		{action: history.ActionDelete, occurredAt: eventTimes[1], before: snapshot, after: nil, related: related},
		{action: history.ActionCreate, occurredAt: eventTimes[0], before: nil, after: snapshot, related: related},
	}
	assertSQLiteMutationHistory(t, store, relationship.ID, want)
	assertSQLiteMutationHistory(t, store, relationship.SourceID, want)
	assertSQLiteMutationHistory(t, store, relationship.TargetID, want)
}

func TestSqliteHistoryNoOpContact(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "No Op Contact",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	persisted, err := store.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	wantUpdatedAt := persisted.UpdatedAt
	persisted.Touch()
	if err := store.UpdateContact(persisted); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}

	assertSQLiteHistoryActions(t, store, contact.ID, []history.Action{history.ActionCreate})
	got, err := store.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("GetContact after no-op: %v", err)
	}
	if !got.UpdatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("UpdatedAt = %s, want unchanged %s", got.UpdatedAt, wantUpdatedAt)
	}
}

func TestSqliteHistoryNoOpCompany(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	company := &models.Company{
		ID:        uuid.New(),
		Name:      "No Op Company",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	persisted, err := store.GetCompany(company.ID)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	wantUpdatedAt := persisted.UpdatedAt
	persisted.Touch()
	if err := store.UpdateCompany(persisted); err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}

	assertSQLiteHistoryActions(t, store, company.ID, []history.Action{history.ActionCreate})
	got, err := store.GetCompany(company.ID)
	if err != nil {
		t.Fatalf("GetCompany after no-op: %v", err)
	}
	if !got.UpdatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("UpdatedAt = %s, want unchanged %s", got.UpdatedAt, wantUpdatedAt)
	}
}

func TestSqliteHistoryRollbackContactCreate(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	installFailHistoryTrigger(t, store)
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Rolled Back Create",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}

	if err := store.CreateContact(contact); err == nil {
		t.Fatal("CreateContact() error = nil")
	}
	if _, err := store.GetContact(contact.ID); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("GetContact() error = %v, want ErrContactNotFound", err)
	}
	assertSQLiteHistoryRowCount(t, store, 0)
}

func TestSqliteHistoryRollbackContactUpdate(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Before Update",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	installFailHistoryTrigger(t, store)
	contact.Name = "After Update"
	contact.UpdatedAt = testHistoryTime.Add(time.Hour)

	if err := store.UpdateContact(contact); err == nil {
		t.Fatal("UpdateContact() error = nil")
	}
	got, err := store.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if got.Name != "Before Update" || !got.UpdatedAt.Equal(testHistoryTime) {
		t.Fatalf("contact after rollback = %#v", got)
	}
	assertSQLiteHistoryRowCount(t, store, 1)
}

func TestSqliteHistoryRollbackContactDelete(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Preserved Delete",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	installFailHistoryTrigger(t, store)

	if err := store.DeleteContact(contact.ID); err == nil {
		t.Fatal("DeleteContact() error = nil")
	}
	if _, err := store.GetContact(contact.ID); err != nil {
		t.Fatalf("GetContact after rollback: %v", err)
	}
	assertSQLiteHistoryRowCount(t, store, 1)
}

func TestSqliteHistoryRollbackCompanyAndRelationship(t *testing.T) {
	t.Run("company", func(t *testing.T) {
		store := newTestStore(t)
		fixedSQLiteHistoryTimes(store)
		installFailHistoryTrigger(t, store)
		company := &models.Company{
			ID:        uuid.New(),
			Name:      "Rolled Back Company",
			Fields:    map[string]any{},
			Tags:      []string{},
			CreatedAt: testHistoryTime,
			UpdatedAt: testHistoryTime,
		}
		if err := store.CreateCompany(company); err == nil {
			t.Fatal("CreateCompany() error = nil")
		}
		if _, err := store.GetCompany(company.ID); !errors.Is(err, ErrCompanyNotFound) {
			t.Fatalf("GetCompany() error = %v, want ErrCompanyNotFound", err)
		}
		assertSQLiteHistoryRowCount(t, store, 0)
	})

	t.Run("relationship", func(t *testing.T) {
		store := newTestStore(t)
		fixedSQLiteHistoryTimes(store)
		installFailHistoryTrigger(t, store)
		relationship := &models.Relationship{
			ID:        uuid.New(),
			SourceID:  uuid.New(),
			TargetID:  uuid.New(),
			Type:      "works_at",
			CreatedAt: testHistoryTime,
		}
		if err := store.CreateRelationship(relationship); err == nil {
			t.Fatal("CreateRelationship() error = nil")
		}
		got, err := store.ListRelationships(relationship.SourceID)
		if err != nil {
			t.Fatalf("ListRelationships: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("ListRelationships() = %v, want empty", got)
		}
		assertSQLiteHistoryRowCount(t, store, 0)
	})
}

func TestSqliteHistoryConflictRejectsStaleBefore(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Original",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: testHistoryTime,
		UpdatedAt: testHistoryTime,
	}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	staleBefore := mustContactSnapshot(t, contact)
	wanted := *contact
	wanted.Name = "Stale Writer"
	wanted.UpdatedAt = testHistoryTime.Add(time.Hour)
	wantedAfter := mustContactSnapshot(t, &wanted)
	event, err := history.NewEvent(
		history.EntityContact,
		contact.ID,
		[]uuid.UUID{contact.ID},
		history.ActionUpdate,
		history.SourceCLI,
		staleBefore,
		wantedAfter,
		testHistoryTime.Add(11*time.Hour),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	newerTime := testHistoryTime.Add(30 * time.Minute)
	if _, err := store.db.Exec(
		"UPDATE contacts SET name = ?, updated_at = ? WHERE id = ?",
		"Concurrent Writer", newerTime, contact.ID.String(),
	); err != nil {
		t.Fatalf("write concurrent state: %v", err)
	}

	if err := store.commitHistoryEvent(event); !errors.Is(err, ErrHistoryConflict) {
		t.Fatalf("commitHistoryEvent() error = %v, want ErrHistoryConflict", err)
	}
	got, err := store.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if got.Name != "Concurrent Writer" || !got.UpdatedAt.Equal(newerTime) {
		t.Fatalf("contact after conflict = %#v", got)
	}
	assertSQLiteHistoryRowCount(t, store, 1)
}

func TestSqliteHistoryMutationSnapshotMatchesPersistedTimes(t *testing.T) {
	t.Run("contact", testSqliteHistoryContactSnapshotTime)
	t.Run("company", testSqliteHistoryCompanySnapshotTime)
	t.Run("relationship", testSqliteHistoryRelationshipSnapshotTime)
}

func testSqliteHistoryContactSnapshotTime(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	modelTime := testHistoryTime.In(time.FixedZone("test-offset", -5*60*60))
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Time Zone Contact",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: modelTime,
		UpdatedAt: modelTime,
	}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	persisted, err := store.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	assertSQLiteHistoryAfterMatches(t, store, contact.ID, mustContactSnapshot(t, persisted))
}

func testSqliteHistoryCompanySnapshotTime(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	modelTime := testHistoryTime.In(time.FixedZone("test-offset", -5*60*60))
	company := &models.Company{
		ID:        uuid.New(),
		Name:      "Time Zone Company",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: modelTime,
		UpdatedAt: modelTime,
	}
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	persisted, err := store.GetCompany(company.ID)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	assertSQLiteHistoryAfterMatches(t, store, company.ID, mustCompanySnapshot(t, persisted))
}

func testSqliteHistoryRelationshipSnapshotTime(t *testing.T) {
	store := newTestStore(t)
	fixedSQLiteHistoryTimes(store)
	relationship := &models.Relationship{
		ID:        uuid.New(),
		SourceID:  uuid.New(),
		TargetID:  uuid.New(),
		Type:      "time_zone",
		CreatedAt: testHistoryTime.In(time.FixedZone("test-offset", -5*60*60)),
	}
	if err := store.CreateRelationship(relationship); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}
	persisted, err := store.getRelationship(relationship.ID)
	if err != nil {
		t.Fatalf("getRelationship: %v", err)
	}
	assertSQLiteHistoryAfterMatches(t, store, relationship.ID, mustRelationshipSnapshot(t, persisted))
}

type sqliteExpectedHistoryEvent struct {
	action     history.Action
	occurredAt time.Time
	before     json.RawMessage
	after      json.RawMessage
	related    []uuid.UUID
}

func fixedSQLiteHistoryTimes(store *SqliteStore) []time.Time {
	times := []time.Time{
		testHistoryTime.Add(10 * time.Hour),
		testHistoryTime.Add(11 * time.Hour),
		testHistoryTime.Add(12 * time.Hour),
	}
	index := 0
	store.now = func() time.Time {
		value := times[index]
		index++
		return value
	}
	return times
}

func installFailHistoryTrigger(t *testing.T, store *SqliteStore) {
	t.Helper()
	_, err := store.db.Exec(`
		CREATE TRIGGER fail_history_insert
		BEFORE INSERT ON history_events
		BEGIN
			SELECT RAISE(ABORT, 'history insert failed');
		END`)
	if err != nil {
		t.Fatalf("install history failure trigger: %v", err)
	}
}

func assertSQLiteHistoryRowCount(t *testing.T, store *SqliteStore, want int) {
	t.Helper()
	var got int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM history_events").Scan(&got); err != nil {
		t.Fatalf("count history events: %v", err)
	}
	if got != want {
		t.Fatalf("history event count = %d, want %d", got, want)
	}
}

func mustContactSnapshot(t *testing.T, contact *models.Contact) json.RawMessage {
	t.Helper()
	snapshot, err := history.SnapshotContact(contact)
	if err != nil {
		t.Fatalf("SnapshotContact: %v", err)
	}
	return snapshot
}

func mustCompanySnapshot(t *testing.T, company *models.Company) json.RawMessage {
	t.Helper()
	snapshot, err := history.SnapshotCompany(company)
	if err != nil {
		t.Fatalf("SnapshotCompany: %v", err)
	}
	return snapshot
}

func mustRelationshipSnapshot(t *testing.T, relationship *models.Relationship) json.RawMessage {
	t.Helper()
	snapshot, err := history.SnapshotRelationship(relationship)
	if err != nil {
		t.Fatalf("SnapshotRelationship: %v", err)
	}
	return snapshot
}

func assertSQLiteMutationHistory(
	t *testing.T,
	store *SqliteStore,
	entityID uuid.UUID,
	want []sqliteExpectedHistoryEvent,
) {
	t.Helper()
	summaries, err := store.ListHistory(entityID.String(), 0)
	if err != nil {
		t.Fatalf("ListHistory(%s): %v", entityID, err)
	}
	if len(summaries) != len(want) {
		t.Fatalf("ListHistory(%s) count = %d, want %d", entityID, len(summaries), len(want))
	}
	for index, expected := range want {
		event, err := store.GetHistoryEvent(summaries[index].ID.String())
		if err != nil {
			t.Fatalf("GetHistoryEvent(%s): %v", summaries[index].ID, err)
		}
		if event.EntityID != wantEntityID(expected, entityID) || event.Action != expected.action ||
			event.Source != history.SourceCLI || !event.OccurredAt.Equal(expected.occurredAt) {
			t.Fatalf("event[%d] metadata = entity:%s action:%s source:%s at:%s", index,
				event.EntityID, event.Action, event.Source, event.OccurredAt)
		}
		assertSQLiteSnapshotEqual(t, event.EntityType, event.Before, expected.before)
		assertSQLiteSnapshotEqual(t, event.EntityType, event.After, expected.after)
		if !slices.Equal(event.RelatedEntityIDs, expected.related) {
			t.Fatalf("event[%d] related IDs = %v, want %v", index, event.RelatedEntityIDs, expected.related)
		}
	}
}

func wantEntityID(expected sqliteExpectedHistoryEvent, timelineID uuid.UUID) uuid.UUID {
	if len(expected.related) == 3 {
		return expected.related[2]
	}
	return timelineID
}

func assertSQLiteSnapshotEqual(t *testing.T, entityType history.EntityType, got, want json.RawMessage) {
	t.Helper()
	equal, err := history.EqualSnapshots(entityType, got, want)
	if err != nil {
		t.Fatalf("EqualSnapshots: %v", err)
	}
	if !equal {
		t.Fatalf("snapshot = %s, want %s", got, want)
	}
}

func assertSQLiteHistoryActions(t *testing.T, store *SqliteStore, entityID uuid.UUID, want []history.Action) {
	t.Helper()
	summaries, err := store.ListHistory(entityID.String(), 0)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(summaries) != len(want) {
		t.Fatalf("history count = %d, want %d", len(summaries), len(want))
	}
	for index := range want {
		if summaries[index].Action != want[index] {
			t.Fatalf("history[%d] action = %s, want %s", index, summaries[index].Action, want[index])
		}
	}
}

func assertSQLiteHistoryAfterMatches(
	t *testing.T,
	store *SqliteStore,
	entityID uuid.UUID,
	want json.RawMessage,
) {
	t.Helper()
	summaries, err := store.ListHistory(entityID.String(), 0)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("history count = %d, want 1", len(summaries))
	}
	event, err := store.GetHistoryEvent(summaries[0].ID.String())
	if err != nil {
		t.Fatalf("GetHistoryEvent: %v", err)
	}
	assertSQLiteSnapshotEqual(t, event.EntityType, event.After, want)
}

var (
	testHistoryEntityA      = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	testHistoryEntityB      = uuid.MustParse("b0000000-0000-0000-0000-000000000002")
	testHistoryRelationship = uuid.MustParse("c0000000-0000-0000-0000-000000000003")
	testHistoryEventA       = uuid.MustParse("d0000100-0000-0000-0000-000000000001")
	testHistoryEventTieA    = uuid.MustParse("10000100-0000-0000-0000-000000000002")
	testHistoryEventTieB    = uuid.MustParse("30000100-0000-0000-0000-000000000003")
	testHistoryTime         = time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
)

func runHistoryReadContract(t *testing.T, backend historyReadBackend) {
	t.Helper()
	older := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	tieA := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventTieA, testHistoryTime.Add(time.Hour))
	tieB := testRelationshipHistoryEvent(t, testHistoryEventTieB, testHistoryTime.Add(time.Hour))
	for _, event := range []*history.Event{older, tieA, tieB} {
		if err := backend.write(event); err != nil {
			t.Fatalf("write event %s: %v", event.ID, err)
		}
	}

	t.Run("full entity ID and deterministic ordering", func(t *testing.T) {
		got, err := backend.list(testHistoryEntityA.String(), 0)
		if err != nil {
			t.Fatalf("ListHistory: %v", err)
		}
		want := []uuid.UUID{testHistoryEventTieA, testHistoryEventTieB, testHistoryEventA}
		assertHistorySummaryIDs(t, got, want)
	})

	t.Run("six character entity prefix", func(t *testing.T) {
		got, err := backend.list(testHistoryEntityA.String()[:6], 0)
		if err != nil {
			t.Fatalf("ListHistory: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("ListHistory() len = %d, want 3", len(got))
		}
	})

	t.Run("relationship visible from target", func(t *testing.T) {
		got, err := backend.list(testHistoryEntityB.String(), 0)
		if err != nil {
			t.Fatalf("ListHistory: %v", err)
		}
		assertHistorySummaryIDs(t, got, []uuid.UUID{testHistoryEventTieB})
	})

	t.Run("event prefix", func(t *testing.T) {
		got, err := backend.get(testHistoryEventA.String()[:6])
		if err != nil {
			t.Fatalf("GetHistoryEvent: %v", err)
		}
		if got.ID != testHistoryEventA {
			t.Fatalf("GetHistoryEvent() ID = %s, want %s", got.ID, testHistoryEventA)
		}
		if len(got.After) == 0 {
			t.Fatal("GetHistoryEvent() omitted snapshot")
		}
	})

	runHistoryUppercaseIDContract(t, backend)
	runHistoryReadErrorContract(t, backend)
}

func runHistoryUppercaseIDContract(t *testing.T, backend historyReadBackend) {
	t.Helper()
	t.Run("uppercase full IDs", func(t *testing.T) {
		got, err := backend.list(strings.ToUpper(testHistoryEntityA.String()), 0)
		if err != nil {
			t.Fatalf("ListHistory: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("ListHistory() len = %d, want 3", len(got))
		}
		event, err := backend.get(strings.ToUpper(testHistoryEventA.String()))
		if err != nil {
			t.Fatalf("GetHistoryEvent: %v", err)
		}
		if event.ID != testHistoryEventA {
			t.Fatalf("GetHistoryEvent() ID = %s, want %s", event.ID, testHistoryEventA)
		}
	})

	t.Run("uppercase prefixes", func(t *testing.T) {
		got, err := backend.list(strings.ToUpper(testHistoryEntityA.String()[:6]), 0)
		if err != nil {
			t.Fatalf("ListHistory: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("ListHistory() len = %d, want 3", len(got))
		}
		event, err := backend.get(strings.ToUpper(testHistoryEventA.String()[:6]))
		if err != nil {
			t.Fatalf("GetHistoryEvent: %v", err)
		}
		if event.ID != testHistoryEventA {
			t.Fatalf("GetHistoryEvent() ID = %s, want %s", event.ID, testHistoryEventA)
		}
	})

	t.Run("too short prefixes", func(t *testing.T) {
		if _, err := backend.list("10000", 0); !errors.Is(err, ErrPrefixTooShort) {
			t.Fatalf("ListHistory() error = %v, want ErrPrefixTooShort", err)
		}
		if _, err := backend.get("90000"); !errors.Is(err, ErrPrefixTooShort) {
			t.Fatalf("GetHistoryEvent() error = %v, want ErrPrefixTooShort", err)
		}
	})
}

func runHistoryReadErrorContract(t *testing.T, backend historyReadBackend) {
	t.Helper()
	writeAmbiguousHistoryFixtures(t, backend)
	t.Run("ambiguous prefixes", func(t *testing.T) {
		if _, err := backend.list("abcdef", 0); !errors.Is(err, ErrAmbiguousPrefix) {
			t.Fatalf("ListHistory() error = %v, want ErrAmbiguousPrefix", err)
		}
		if _, err := backend.list("ABCDEF", 0); !errors.Is(err, ErrAmbiguousPrefix) {
			t.Fatalf("ListHistory() uppercase error = %v, want ErrAmbiguousPrefix", err)
		}
		if _, err := backend.get("fedcba"); !errors.Is(err, ErrAmbiguousPrefix) {
			t.Fatalf("GetHistoryEvent() error = %v, want ErrAmbiguousPrefix", err)
		}
		if _, err := backend.get("FEDCBA"); !errors.Is(err, ErrAmbiguousPrefix) {
			t.Fatalf("GetHistoryEvent() uppercase error = %v, want ErrAmbiguousPrefix", err)
		}
	})

	t.Run("entity without events", func(t *testing.T) {
		got, err := backend.list("40000000-0000-0000-0000-000000000004", 0)
		if err != nil {
			t.Fatalf("ListHistory: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("ListHistory() = %v, want empty", got)
		}
	})

	t.Run("missing event", func(t *testing.T) {
		_, err := backend.get("40000000-0000-0000-0000-000000000004")
		if !errors.Is(err, ErrHistoryNotFound) {
			t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryNotFound", err)
		}
	})
}

func runHistoryLimitContract(t *testing.T, backend historyReadBackend) {
	t.Helper()
	for i := 0; i < history.MaxLimit+5; i++ {
		eventID := uuid.MustParse(fmt.Sprintf("70000000-0000-0000-0000-%012d", i))
		event := testContactHistoryEvent(t, testHistoryEntityA, eventID, testHistoryTime.Add(time.Duration(i)*time.Minute))
		if err := backend.write(event); err != nil {
			t.Fatalf("write event %d: %v", i, err)
		}
	}

	for _, tt := range []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: history.DefaultLimit},
		{name: "explicit", limit: 3, want: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.list(testHistoryEntityA.String(), tt.limit)
			if err != nil {
				t.Fatalf("ListHistory: %v", err)
			}
			if len(got) != tt.want {
				t.Fatalf("ListHistory() len = %d, want %d", len(got), tt.want)
			}
		})
	}

	if _, err := backend.list(testHistoryEntityA.String(), history.MaxLimit+1); err == nil {
		t.Fatal("ListHistory() over maximum error = nil")
	}
}

func writeAmbiguousHistoryFixtures(t *testing.T, backend historyReadBackend) {
	t.Helper()
	fixtures := []struct {
		entityID uuid.UUID
		eventID  uuid.UUID
	}{
		{uuid.MustParse("abcdef00-0000-0000-0000-000000000001"), uuid.MustParse("fedcba00-0000-0000-0000-000000000001")},
		{uuid.MustParse("abcdef00-0000-0000-0000-000000000002"), uuid.MustParse("fedcba00-0000-0000-0000-000000000002")},
	}
	for _, fixture := range fixtures {
		event := testContactHistoryEvent(t, fixture.entityID, fixture.eventID, testHistoryTime)
		if err := backend.write(event); err != nil {
			t.Fatalf("write ambiguous fixture: %v", err)
		}
	}
}

func testContactHistoryEvent(
	t *testing.T,
	entityID uuid.UUID,
	eventID uuid.UUID,
	occurredAt time.Time,
) *history.Event {
	t.Helper()
	contact := &models.Contact{
		ID:        entityID,
		Name:      "History Contact",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: occurredAt,
		UpdatedAt: occurredAt,
	}
	snapshot, err := history.SnapshotContact(contact)
	if err != nil {
		t.Fatalf("SnapshotContact: %v", err)
	}
	event, err := history.NewEvent(
		history.EntityContact,
		entityID,
		[]uuid.UUID{entityID},
		history.ActionCreate,
		history.SourceCLI,
		nil,
		snapshot,
		occurredAt,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	event.ID = eventID
	return event
}

func testRelationshipHistoryEvent(t *testing.T, eventID uuid.UUID, occurredAt time.Time) *history.Event {
	t.Helper()
	relationship := &models.Relationship{
		ID:        testHistoryRelationship,
		SourceID:  testHistoryEntityA,
		TargetID:  testHistoryEntityB,
		Type:      "works_at",
		CreatedAt: occurredAt,
	}
	snapshot, err := history.SnapshotRelationship(relationship)
	if err != nil {
		t.Fatalf("SnapshotRelationship: %v", err)
	}
	event, err := history.NewEvent(
		history.EntityRelationship,
		relationship.ID,
		[]uuid.UUID{relationship.ID, relationship.SourceID, relationship.TargetID},
		history.ActionCreate,
		history.SourceCLI,
		nil,
		snapshot,
		occurredAt,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	event.ID = eventID
	return event
}

func assertHistorySummaryIDs(t *testing.T, got []*history.Summary, want []uuid.UUID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("summary count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("summary[%d].ID = %s, want %s", i, got[i].ID, want[i])
		}
	}
}
