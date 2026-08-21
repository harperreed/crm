// ABOUTME: Tests SQLite history schema, committed-event reads, prefix lookup, and limits.
// ABOUTME: Shares the backend-neutral history read contract with the Markdown tests.
package storage

import (
	"errors"
	"fmt"
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
