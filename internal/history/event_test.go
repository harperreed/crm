// ABOUTME: Tests validation and normalization for immutable CRM history events.
// ABOUTME: Locks down event enums, snapshots, related IDs, timestamps, and limits.
package history

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/v2/internal/models"
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
	before := testContactSnapshot(t, entityID)
	after := testContactSnapshot(t, entityID)
	when := time.Date(2026, 8, 21, 10, 11, 12, 13, time.FixedZone("CDT", -5*60*60))

	tests := validEventCases(t, entityID, otherID, after, when)
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

func validEventCases(t *testing.T, entityID, otherID uuid.UUID, after json.RawMessage, when time.Time) []newEventValidationCase {
	t.Helper()
	companyBefore := testCompanySnapshot(t, entityID)
	companyAfter := testCompanySnapshot(t, entityID)
	targetID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	relationshipBefore := testRelationshipSnapshotFor(t, entityID, otherID, targetID)
	return []newEventValidationCase{
		{name: "create", entityType: EntityContact, related: []uuid.UUID{entityID}, action: ActionCreate, source: SourceCLI, after: after, occurredAt: when},
		{name: "update", entityType: EntityCompany, related: []uuid.UUID{entityID}, action: ActionUpdate, source: SourceMCP, before: companyBefore, after: companyAfter, occurredAt: when},
		{name: "delete", entityType: EntityRelationship, related: []uuid.UUID{otherID, entityID, targetID}, action: ActionDelete, source: SourceCLI, before: relationshipBefore, occurredAt: when},
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
	targetID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	event, err := NewEvent(
		EntityRelationship,
		entityID,
		[]uuid.UUID{entityID, otherID, targetID, entityID, otherID},
		ActionCreate,
		SourceCLI,
		nil,
		testRelationshipSnapshotFor(t, entityID, otherID, targetID),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	want := []uuid.UUID{otherID, entityID, targetID}
	if !slices.Equal(event.RelatedEntityIDs, want) {
		t.Fatalf("RelatedEntityIDs = %v, want %v", event.RelatedEntityIDs, want)
	}
}

func TestEventValidateRejectsInvalidSnapshotsAfterJSONRoundTrip(t *testing.T) {
	entityID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	valid := testContactSnapshot(t, entityID)
	wrongEntity := testRelationshipSnapshot(t)
	mismatched := testContactSnapshot(t, uuid.MustParse("10000000-0000-0000-0000-000000000099"))
	zeroIdentity := testContactSnapshot(t, uuid.Nil)
	zeroTime := snapshotWithField(t, valid, "updated_at", "0001-01-01T00:00:00Z")

	tests := []struct {
		name     string
		snapshot json.RawMessage
	}{
		{name: "empty object", snapshot: json.RawMessage(`{}`)},
		{name: "wrong entity shape", snapshot: wrongEntity},
		{name: "unknown field", snapshot: snapshotWithField(t, valid, "nickname", "Enchantress of Numbers")},
		{name: "mismatched ID", snapshot: mismatched},
		{name: "zero identity", snapshot: zeroIdentity},
		{name: "zero required time", snapshot: zeroTime},
		{name: "missing required time", snapshot: snapshotWithoutField(t, valid, "created_at")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := newContactCreateEvent(t, entityID, valid)
			event.After = tt.snapshot
			if err := validateAfterJSONRoundTrip(t, event); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestEventValidateRejectsNoncanonicalRelatedEntityIDs(t *testing.T) {
	contactID := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	otherID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	contact := newContactCreateEvent(t, contactID, testContactSnapshot(t, contactID))
	relationship := newRelationshipCreateEvent(t)
	relationshipID := relationship.EntityID
	sourceID := relationship.RelatedEntityIDs[0]
	targetID := relationship.RelatedEntityIDs[1]

	tests := []struct {
		name    string
		event   Event
		related []uuid.UUID
	}{
		{name: "nil contact IDs", event: contact, related: nil},
		{name: "duplicate contact ID", event: contact, related: []uuid.UUID{contactID, contactID}},
		{name: "extra contact ID", event: contact, related: []uuid.UUID{otherID, contactID}},
		{name: "unsorted relationship IDs", event: relationship, related: []uuid.UUID{relationshipID, targetID, sourceID}},
		{name: "relationship missing endpoint", event: relationship, related: []uuid.UUID{sourceID, relationshipID}},
		{name: "duplicate relationship ID", event: relationship, related: []uuid.UUID{sourceID, targetID, relationshipID, relationshipID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.event.RelatedEntityIDs = tt.related
			if err := tt.event.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestNewEventOwnsSnapshotBuffers(t *testing.T) {
	entityID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	before := testContactSnapshot(t, entityID)
	after := testContactSnapshot(t, entityID)
	event, err := NewEvent(
		EntityContact, entityID, []uuid.UUID{entityID}, ActionUpdate, SourceCLI,
		before, after, time.Now(),
	)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	wantBefore := slices.Clone(event.Before)
	wantAfter := slices.Clone(event.After)

	before[0] = '['
	after[0] = '['
	if !bytes.Equal(event.Before, wantBefore) || !bytes.Equal(event.After, wantAfter) {
		t.Fatal("NewEvent() retained caller-owned snapshot buffers")
	}
}

func newContactCreateEvent(t *testing.T, entityID uuid.UUID, snapshot json.RawMessage) Event {
	t.Helper()
	event, err := NewEvent(
		EntityContact, entityID, []uuid.UUID{entityID}, ActionCreate, SourceCLI,
		nil, snapshot, time.Now(),
	)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	return *event
}

func newRelationshipCreateEvent(t *testing.T) Event {
	t.Helper()
	snapshot := testRelationshipSnapshot(t)
	relationship, err := RelationshipFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("RelationshipFromSnapshot() error = %v", err)
	}
	event, err := NewEvent(
		EntityRelationship,
		relationship.ID,
		[]uuid.UUID{relationship.ID, relationship.SourceID, relationship.TargetID},
		ActionCreate,
		SourceCLI,
		nil,
		snapshot,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	return *event
}

func testContactSnapshot(t *testing.T, entityID uuid.UUID) json.RawMessage {
	t.Helper()
	snapshot, err := SnapshotContact(&models.Contact{
		ID:        entityID,
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
		UpdatedAt: time.Date(2026, 8, 21, 10, 11, 12, 13, time.UTC),
	})
	if err != nil {
		t.Fatalf("SnapshotContact() error = %v", err)
	}
	return snapshot
}

func testCompanySnapshot(t *testing.T, entityID uuid.UUID) json.RawMessage {
	t.Helper()
	snapshot, err := SnapshotCompany(&models.Company{
		ID:        entityID,
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
		UpdatedAt: time.Date(2026, 8, 21, 10, 11, 12, 13, time.UTC),
	})
	if err != nil {
		t.Fatalf("SnapshotCompany() error = %v", err)
	}
	return snapshot
}

func testRelationshipSnapshot(t *testing.T) json.RawMessage {
	t.Helper()
	return testRelationshipSnapshotFor(
		t,
		uuid.MustParse("30000000-0000-0000-0000-000000000003"),
		uuid.MustParse("30000000-0000-0000-0000-000000000001"),
		uuid.MustParse("30000000-0000-0000-0000-000000000002"),
	)
}

func testRelationshipSnapshotFor(t *testing.T, entityID, sourceID, targetID uuid.UUID) json.RawMessage {
	t.Helper()
	snapshot, err := SnapshotRelationship(&models.Relationship{
		ID:        entityID,
		SourceID:  sourceID,
		TargetID:  targetID,
		Type:      "works_at",
		CreatedAt: time.Date(2026, 8, 21, 10, 11, 12, 13, time.UTC),
	})
	if err != nil {
		t.Fatalf("SnapshotRelationship() error = %v", err)
	}
	return snapshot
}

func snapshotWithField(t *testing.T, snapshot json.RawMessage, key string, value any) json.RawMessage {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(snapshot, &object); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	object[key] = value
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}

func snapshotWithoutField(t *testing.T, snapshot json.RawMessage, key string) json.RawMessage {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(snapshot, &object); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	delete(object, key)
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}

func validateAfterJSONRoundTrip(t *testing.T, event Event) error {
	t.Helper()
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var decoded Event
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return decoded.Validate()
}
