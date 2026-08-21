// ABOUTME: Tests stable JSON snapshots for contacts, companies, and relationships.
// ABOUTME: Verifies round trips, schema keys, comparisons, and history summaries.
package history

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestContactSnapshotRoundTrip(t *testing.T) {
	contact := &models.Contact{
		ID:        uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		Name:      "Ada Lovelace",
		Email:     "ada@example.com",
		Phone:     "+1-555-0100",
		Fields:    map[string]any{"role": "mathematician", "active": true},
		Tags:      []string{"friend", "writer"},
		CreatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
		UpdatedAt: time.Date(2026, 8, 21, 10, 11, 12, 13, time.UTC),
	}

	snapshot, err := SnapshotContact(contact)
	if err != nil {
		t.Fatalf("SnapshotContact() error = %v", err)
	}
	assertSnapshotKeys(t, snapshot, []string{
		"id", "name", "email", "phone", "fields", "tags", "created_at", "updated_at",
	})

	got, err := ContactFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("ContactFromSnapshot() error = %v", err)
	}
	if !reflect.DeepEqual(got, contact) {
		t.Fatalf("ContactFromSnapshot() = %#v, want %#v", got, contact)
	}
}

func TestCompanySnapshotRoundTrip(t *testing.T) {
	company := &models.Company{
		ID:        uuid.MustParse("20000000-0000-0000-0000-000000000001"),
		Name:      "Analytical Engines, Inc.",
		Domain:    "engines.example.com",
		Fields:    map[string]any{"industry": "computing", "public": false},
		Tags:      []string{"prospect", "hardware"},
		CreatedAt: time.Date(2026, 8, 19, 8, 9, 10, 11, time.UTC),
		UpdatedAt: time.Date(2026, 8, 21, 10, 11, 12, 13, time.UTC),
	}

	snapshot, err := SnapshotCompany(company)
	if err != nil {
		t.Fatalf("SnapshotCompany() error = %v", err)
	}
	assertSnapshotKeys(t, snapshot, []string{
		"id", "name", "domain", "fields", "tags", "created_at", "updated_at",
	})

	got, err := CompanyFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("CompanyFromSnapshot() error = %v", err)
	}
	if !reflect.DeepEqual(got, company) {
		t.Fatalf("CompanyFromSnapshot() = %#v, want %#v", got, company)
	}
}

func TestSnapshotRoundTripPreservesJSONNumbers(t *testing.T) {
	t.Run("contact", func(t *testing.T) {
		contact := &models.Contact{
			ID:        uuid.MustParse("10000000-0000-0000-0000-000000000009"),
			Name:      "Exact Number Contact",
			Fields:    nestedLargeNumberFields(),
			Tags:      []string{},
			CreatedAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		}
		before, err := SnapshotContact(contact)
		if err != nil {
			t.Fatalf("SnapshotContact: %v", err)
		}
		decoded, err := ContactFromSnapshot(before)
		if err != nil {
			t.Fatalf("ContactFromSnapshot: %v", err)
		}
		assertNestedLargeNumber(t, decoded.Fields)
		after, err := SnapshotContact(decoded)
		if err != nil {
			t.Fatalf("SnapshotContact decoded: %v", err)
		}
		if !bytes.Equal(after, before) {
			t.Fatalf("round-trip snapshot = %s, want %s", after, before)
		}
	})

	t.Run("company", func(t *testing.T) {
		company := &models.Company{
			ID:        uuid.MustParse("20000000-0000-0000-0000-000000000009"),
			Name:      "Exact Number Company",
			Fields:    nestedLargeNumberFields(),
			Tags:      []string{},
			CreatedAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		}
		before, err := SnapshotCompany(company)
		if err != nil {
			t.Fatalf("SnapshotCompany: %v", err)
		}
		decoded, err := CompanyFromSnapshot(before)
		if err != nil {
			t.Fatalf("CompanyFromSnapshot: %v", err)
		}
		assertNestedLargeNumber(t, decoded.Fields)
		after, err := SnapshotCompany(decoded)
		if err != nil {
			t.Fatalf("SnapshotCompany decoded: %v", err)
		}
		if !bytes.Equal(after, before) {
			t.Fatalf("round-trip snapshot = %s, want %s", after, before)
		}
	})
}

func TestRelationshipSnapshotRoundTrip(t *testing.T) {
	relationship := &models.Relationship{
		ID:        uuid.MustParse("30000000-0000-0000-0000-000000000001"),
		SourceID:  uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		TargetID:  uuid.MustParse("20000000-0000-0000-0000-000000000001"),
		Type:      "works_at",
		Context:   "Designed the first published algorithm",
		CreatedAt: time.Date(2026, 8, 18, 7, 8, 9, 10, time.UTC),
	}

	snapshot, err := SnapshotRelationship(relationship)
	if err != nil {
		t.Fatalf("SnapshotRelationship() error = %v", err)
	}
	assertSnapshotKeys(t, snapshot, []string{
		"id", "source_id", "target_id", "type", "context", "created_at",
	})

	got, err := RelationshipFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("RelationshipFromSnapshot() error = %v", err)
	}
	if !reflect.DeepEqual(got, relationship) {
		t.Fatalf("RelationshipFromSnapshot() = %#v, want %#v", got, relationship)
	}
}

func TestChangedFields(t *testing.T) {
	beforeContact := &models.Contact{
		ID:        uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		Name:      "Ada Lovelace",
		Email:     "ada@old.example.com",
		Phone:     "+1-555-0100",
		Fields:    map[string]any{"status": "prospect"},
		Tags:      []string{"friend"},
		CreatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
		UpdatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
	}
	afterContact := *beforeContact
	afterContact.Email = "ada@new.example.com"
	afterContact.Phone = "+1-555-0199"
	afterContact.Fields = map[string]any{"status": "customer"}
	afterContact.Tags = []string{"friend", "customer"}
	afterContact.UpdatedAt = beforeContact.UpdatedAt.Add(time.Hour)

	before, err := SnapshotContact(beforeContact)
	if err != nil {
		t.Fatalf("SnapshotContact(before) error = %v", err)
	}
	after, err := SnapshotContact(&afterContact)
	if err != nil {
		t.Fatalf("SnapshotContact(after) error = %v", err)
	}

	got, err := ChangedFields(EntityContact, before, after)
	if err != nil {
		t.Fatalf("ChangedFields() error = %v", err)
	}
	want := []string{"email", "fields", "phone", "tags"}
	if !slices.Equal(got, want) {
		t.Fatalf("ChangedFields() = %v, want %v", got, want)
	}

	updatedOnly := *beforeContact
	updatedOnly.UpdatedAt = beforeContact.UpdatedAt.Add(time.Minute)
	updatedSnapshot, err := SnapshotContact(&updatedOnly)
	if err != nil {
		t.Fatalf("SnapshotContact(updated only) error = %v", err)
	}
	got, err = ChangedFields(EntityContact, before, updatedSnapshot)
	if err != nil {
		t.Fatalf("ChangedFields(updated only) error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ChangedFields(updated only) = %v, want no fields", got)
	}
}

func TestEqualSnapshots(t *testing.T) {
	left := json.RawMessage(`{
		"name":"Ada Lovelace",
		"id":"10000000-0000-0000-0000-000000000001",
		"email":"ada@example.com",
		"phone":"",
		"fields":{},
		"tags":[],
		"created_at":"2026-08-20T09:10:11Z",
		"updated_at":"2026-08-21T10:11:12Z"
	}`)
	right := json.RawMessage(`{"id":"10000000-0000-0000-0000-000000000001","name":"Ada Lovelace","email":"ada@example.com","phone":"","fields":{},"tags":[],"created_at":"2026-08-20T09:10:11Z","updated_at":"2026-08-21T10:11:12Z"}`)

	equal, err := EqualSnapshots(EntityContact, left, right)
	if err != nil {
		t.Fatalf("EqualSnapshots() error = %v", err)
	}
	if !equal {
		t.Fatal("EqualSnapshots() = false for equivalent snapshots")
	}

	equal, err = EqualSnapshots(EntityCompany, nil, json.RawMessage("null"))
	if err != nil {
		t.Fatalf("EqualSnapshots(null, null) error = %v", err)
	}
	if !equal {
		t.Fatal("EqualSnapshots(null, null) = false")
	}
}

func TestSnapshotErrors(t *testing.T) {
	if _, err := SnapshotContact(nil); err == nil {
		t.Fatal("SnapshotContact(nil) error = nil")
	}
	if _, err := SnapshotCompany(nil); err == nil {
		t.Fatal("SnapshotCompany(nil) error = nil")
	}
	if _, err := SnapshotRelationship(nil); err == nil {
		t.Fatal("SnapshotRelationship(nil) error = nil")
	}
	if _, err := ContactFromSnapshot(json.RawMessage(`{"id":`)); err == nil {
		t.Fatal("ContactFromSnapshot(invalid JSON) error = nil")
	}
	if _, err := CompanyFromSnapshot(json.RawMessage(`{"id":`)); err == nil {
		t.Fatal("CompanyFromSnapshot(invalid JSON) error = nil")
	}
	if _, err := RelationshipFromSnapshot(json.RawMessage(`{"id":`)); err == nil {
		t.Fatal("RelationshipFromSnapshot(invalid JSON) error = nil")
	}
	if _, err := EqualSnapshots(EntityType("person"), nil, nil); err == nil {
		t.Fatal("EqualSnapshots(unknown entity type) error = nil")
	}
	if _, err := ChangedFields(EntityContact, json.RawMessage(`{"id":`), nil); err == nil {
		t.Fatal("ChangedFields(invalid JSON) error = nil")
	}
	if _, err := ChangedFields(EntityType("person"), nil, nil); err == nil {
		t.Fatal("ChangedFields(unknown entity type) error = nil")
	}
}

func TestSnapshotDecodersRejectStructurallyInvalidSnapshots(t *testing.T) {
	valid := testContactSnapshot(t, uuid.MustParse("10000000-0000-0000-0000-000000000001"))
	tests := []struct {
		name     string
		snapshot json.RawMessage
	}{
		{name: "empty object", snapshot: json.RawMessage(`{}`)},
		{name: "unknown field", snapshot: snapshotWithField(t, valid, "nickname", "Ada")},
		{name: "wrong entity shape", snapshot: testRelationshipSnapshot(t)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ContactFromSnapshot(tt.snapshot); err == nil {
				t.Fatal("ContactFromSnapshot() error = nil")
			}
		})
	}
}

func TestEqualSnapshotsUsesStrictDecoder(t *testing.T) {
	valid := testContactSnapshot(t, uuid.MustParse("10000000-0000-0000-0000-000000000001"))
	unknownField := snapshotWithField(t, valid, "nickname", "Ada")
	if _, err := EqualSnapshots(EntityContact, valid, unknownField); err == nil {
		t.Fatal("EqualSnapshots() error = nil")
	}
}

func TestChangedFieldsUsesStrictDecoder(t *testing.T) {
	valid := testContactSnapshot(t, uuid.MustParse("10000000-0000-0000-0000-000000000001"))
	unknownField := snapshotWithField(t, valid, "nickname", "Ada")
	if _, err := ChangedFields(EntityContact, valid, unknownField); err == nil {
		t.Fatal("ChangedFields() error = nil")
	}
}

func TestEventSummary(t *testing.T) {
	contact := &models.Contact{
		ID:        uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		Name:      "Ada Lovelace",
		Fields:    map[string]any{},
		Tags:      []string{},
		CreatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
		UpdatedAt: time.Date(2026, 8, 20, 9, 10, 11, 12, time.UTC),
	}
	before, err := SnapshotContact(contact)
	if err != nil {
		t.Fatalf("SnapshotContact(before) error = %v", err)
	}
	updated := *contact
	updated.Email = "ada@example.com"
	after, err := SnapshotContact(&updated)
	if err != nil {
		t.Fatalf("SnapshotContact(after) error = %v", err)
	}
	event, err := NewEvent(
		EntityContact,
		contact.ID,
		[]uuid.UUID{contact.ID},
		ActionUpdate,
		SourceMCP,
		before,
		after,
		time.Date(2026, 8, 21, 10, 11, 12, 13, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	summary, err := event.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.ID != event.ID || summary.EntityType != event.EntityType || summary.EntityID != event.EntityID ||
		summary.Action != event.Action || summary.Source != event.Source || !summary.OccurredAt.Equal(event.OccurredAt) {
		t.Fatalf("Summary() identity fields = %#v, want fields from %#v", summary, event)
	}
	if !slices.Equal(summary.ChangedFields, []string{"email"}) {
		t.Fatalf("Summary().ChangedFields = %v, want [email]", summary.ChangedFields)
	}
}

func assertSnapshotKeys(t *testing.T, snapshot json.RawMessage, want []string) {
	t.Helper()
	for _, legacyKey := range []string{"ID", "CreatedAt", "UpdatedAt"} {
		if strings.Contains(string(snapshot), `"`+legacyKey+`"`) {
			t.Fatalf("snapshot contains legacy key %q: %s", legacyKey, snapshot)
		}
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(snapshot, &object); err != nil {
		t.Fatalf("snapshot is invalid JSON: %v", err)
	}
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("snapshot keys = %v, want %v", got, want)
	}
}

func nestedLargeNumberFields() map[string]any {
	return map[string]any{
		"nested": []any{
			map[string]any{"integer": json.Number("9007199254740993")},
		},
	}
}

func assertNestedLargeNumber(t *testing.T, fields map[string]any) {
	t.Helper()
	nested, ok := fields["nested"].([]any)
	if !ok || len(nested) != 1 {
		t.Fatalf("nested field = %#v", fields["nested"])
	}
	object, ok := nested[0].(map[string]any)
	if !ok {
		t.Fatalf("nested object = %#v", nested[0])
	}
	number, ok := object["integer"].(json.Number)
	if !ok || number.String() != "9007199254740993" {
		t.Fatalf("nested integer = %#v, want exact json.Number", object["integer"])
	}
}
