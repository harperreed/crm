// ABOUTME: Tests validation and normalization for immutable CRM history events.
// ABOUTME: Locks down event enums, snapshots, related IDs, timestamps, and limits.
package history

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

type newEventValidationCase struct {
	name       string
	entityType EntityType
	related    []uuid.UUID
	action     Action
	source     Source
	before     json.RawMessage
	after      json.RawMessage
	occurredAt time.Time
	wantErr    bool
}

func TestNormalizeLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{name: "default zero", input: 0, want: DefaultLimit},
		{name: "default negative", input: -1, want: DefaultLimit},
		{name: "explicit", input: 42, want: 42},
		{name: "maximum", input: MaxLimit, want: MaxLimit},
		{name: "over maximum", input: MaxLimit + 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeLimit(%d) error = %v", tt.input, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("NormalizeLimit(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewEventValidation(t *testing.T) {
	entityID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	otherID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	before := json.RawMessage(`{"id":"before"}`)
	after := json.RawMessage(`{"id":"after"}`)
	when := time.Date(2026, 8, 21, 10, 11, 12, 13, time.FixedZone("CDT", -5*60*60))

	tests := validEventCases(entityID, otherID, before, after, when)
	tests = append(tests, invalidEventMetadataCases(entityID, otherID, after, when)...)
	tests = append(tests, invalidEventSnapshotCases(entityID, before, after, when)...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := NewEvent(
				tt.entityType, entityID, tt.related, tt.action, tt.source,
				tt.before, tt.after, tt.occurredAt,
			)
			assertNewEventResult(t, event, err, tt.wantErr, when)
		})
	}
}

func validEventCases(entityID, otherID uuid.UUID, before, after json.RawMessage, when time.Time) []newEventValidationCase {
	return []newEventValidationCase{
		{name: "create", entityType: EntityContact, related: []uuid.UUID{entityID}, action: ActionCreate, source: SourceCLI, after: after, occurredAt: when},
		{name: "update", entityType: EntityCompany, related: []uuid.UUID{entityID}, action: ActionUpdate, source: SourceMCP, before: before, after: after, occurredAt: when},
		{name: "delete", entityType: EntityRelationship, related: []uuid.UUID{entityID, otherID}, action: ActionDelete, source: SourceCLI, before: before, occurredAt: when},
	}
}

func invalidEventMetadataCases(entityID, otherID uuid.UUID, after json.RawMessage, when time.Time) []newEventValidationCase {
	return []newEventValidationCase{
		{name: "invalid source", entityType: EntityContact, related: []uuid.UUID{entityID}, action: ActionCreate, source: Source("CLI"), after: after, occurredAt: when, wantErr: true},
		{name: "invalid entity type", entityType: EntityType("Contact"), related: []uuid.UUID{entityID}, action: ActionCreate, source: SourceCLI, after: after, occurredAt: when, wantErr: true},
		{name: "invalid action", entityType: EntityContact, related: []uuid.UUID{entityID}, action: Action("CREATE"), source: SourceCLI, after: after, occurredAt: when, wantErr: true},
		{name: "missing subject in related IDs", entityType: EntityContact, related: []uuid.UUID{otherID}, action: ActionCreate, source: SourceCLI, after: after, occurredAt: when, wantErr: true},
		{name: "zero timestamp", entityType: EntityContact, related: []uuid.UUID{entityID}, action: ActionCreate, source: SourceCLI, after: after, wantErr: true},
	}
}

func invalidEventSnapshotCases(entityID uuid.UUID, before, after json.RawMessage, when time.Time) []newEventValidationCase {
	base := newEventValidationCase{
		entityType: EntityContact,
		related:    []uuid.UUID{entityID},
		source:     SourceCLI,
		occurredAt: when,
		wantErr:    true,
	}
	invalidJSON := base
	invalidJSON.name = "invalid snapshot JSON"
	invalidJSON.action = ActionUpdate
	invalidJSON.before = json.RawMessage(`{"id":`)
	invalidJSON.after = after
	return []newEventValidationCase{
		invalidJSON,
		withEventSnapshots(base, "create with before snapshot", ActionCreate, before, after),
		withEventSnapshots(base, "create without after snapshot", ActionCreate, nil, nil),
		withEventSnapshots(base, "update without before snapshot", ActionUpdate, nil, after),
		withEventSnapshots(base, "update without after snapshot", ActionUpdate, before, nil),
		withEventSnapshots(base, "delete without before snapshot", ActionDelete, nil, nil),
		withEventSnapshots(base, "delete with after snapshot", ActionDelete, before, after),
	}
}

func withEventSnapshots(base newEventValidationCase, name string, action Action, before, after json.RawMessage) newEventValidationCase {
	base.name = name
	base.action = action
	base.before = before
	base.after = after
	return base
}

func assertNewEventResult(t *testing.T, event *Event, err error, wantErr bool, when time.Time) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Fatalf("NewEvent() error = %v, wantErr %v", err, wantErr)
	}
	if wantErr {
		return
	}
	if event.ID == uuid.Nil {
		t.Fatal("NewEvent() generated a nil ID")
	}
	if event.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", event.SchemaVersion)
	}
	if event.OccurredAt.Location() != time.UTC {
		t.Fatalf("OccurredAt location = %v, want UTC", event.OccurredAt.Location())
	}
	if !event.OccurredAt.Equal(when) {
		t.Fatalf("OccurredAt = %v, want instant %v", event.OccurredAt, when)
	}
}

func TestNewEventCanonicalizesRelatedEntityIDs(t *testing.T) {
	entityID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	otherID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	event, err := NewEvent(
		EntityRelationship,
		entityID,
		[]uuid.UUID{entityID, otherID, entityID, otherID},
		ActionCreate,
		SourceCLI,
		nil,
		json.RawMessage(`{"id":"after"}`),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	want := []uuid.UUID{otherID, entityID}
	if !slices.Equal(event.RelatedEntityIDs, want) {
		t.Fatalf("RelatedEntityIDs = %v, want %v", event.RelatedEntityIDs, want)
	}
}
