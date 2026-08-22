// ABOUTME: Verifies that SQLite and Markdown expose the same durable CRM history contract.
// ABOUTME: Covers full entity lifecycles and upgrades from stores created before history existed.
package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
	_ "modernc.org/sqlite"
)

var (
	historyContactID      = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	historyCompanyID      = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	historyRelationshipID = uuid.MustParse("33333333-3333-4333-8333-333333333333")
	historyBaseTime       = time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
)

type historyStoreFactory struct {
	name string
	open func(*testing.T) storage.Storage
}

type expectedHistoryEvent struct {
	entityType history.EntityType
	action     history.Action
	before     json.RawMessage
	after      json.RawMessage
}

func TestHistoryParity(t *testing.T) {
	for _, factory := range historyStoreFactories() {
		t.Run(factory.name, func(t *testing.T) {
			runHistoryParityScenario(t, factory.open(t))
		})
	}
}

func TestHistoryNilCollectionParity(t *testing.T) {
	for _, factory := range historyStoreFactories() {
		t.Run(factory.name, func(t *testing.T) {
			runHistoryNilCollectionScenario(t, factory.open(t))
		})
	}
}

func runHistoryNilCollectionScenario(t *testing.T, store storage.Storage) {
	t.Helper()
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Nil Contact",
		CreatedAt: historyBaseTime,
		UpdatedAt: historyBaseTime,
	}
	company := &models.Company{
		ID:        uuid.New(),
		Name:      "Nil Company",
		CreatedAt: historyBaseTime,
		UpdatedAt: historyBaseTime,
	}

	mustStoreMutation(t, "CreateContact", func() error { return store.CreateContact(contact) })
	mustStoreMutation(t, "CreateCompany", func() error { return store.CreateCompany(company) })
	assertCanonicalContactStateAndHistory(t, store, contact.ID, 1)
	assertCanonicalCompanyStateAndHistory(t, store, company.ID, 1)

	contact.Name = "Updated Nil Contact"
	contact.Fields = nil
	contact.Tags = nil
	contact.UpdatedAt = historyBaseTime.Add(time.Minute)
	company.Name = "Updated Nil Company"
	company.Fields = nil
	company.Tags = nil
	company.UpdatedAt = historyBaseTime.Add(time.Minute)
	mustStoreMutation(t, "UpdateContact", func() error { return store.UpdateContact(contact) })
	mustStoreMutation(t, "UpdateCompany", func() error { return store.UpdateCompany(company) })
	assertCanonicalContactStateAndHistory(t, store, contact.ID, 2)
	assertCanonicalCompanyStateAndHistory(t, store, company.ID, 2)
}

func assertCanonicalContactStateAndHistory(t *testing.T, store storage.Storage, id uuid.UUID, wantEvents int) {
	t.Helper()
	contact, err := store.GetContact(id)
	if err != nil {
		t.Fatalf("GetContact(%s): %v", id, err)
	}
	if contact.Fields == nil || len(contact.Fields) != 0 {
		t.Errorf("GetContact(%s).Fields = %#v, want non-nil empty map", id, contact.Fields)
	}
	if contact.Tags == nil || len(contact.Tags) != 0 {
		t.Errorf("GetContact(%s).Tags = %#v, want non-nil empty slice", id, contact.Tags)
	}
	assertCanonicalContactSnapshots(t, store, id, wantEvents)
}

func assertCanonicalContactSnapshots(t *testing.T, store storage.Storage, id uuid.UUID, wantEvents int) {
	t.Helper()
	summaries := mustListHistory(t, store, id)
	if len(summaries) != wantEvents {
		t.Fatalf("ListHistory(%s) len = %d, want %d", id, len(summaries), wantEvents)
	}
	for _, summary := range summaries {
		event, err := store.GetHistoryEvent(summary.ID.String())
		if err != nil {
			t.Fatalf("GetHistoryEvent(%s): %v", summary.ID, err)
		}
		for side, snapshot := range map[string]json.RawMessage{"before": event.Before, "after": event.After} {
			if len(snapshot) == 0 || bytes.Equal(bytes.TrimSpace(snapshot), []byte("null")) {
				continue
			}
			contact, err := history.ContactFromSnapshot(snapshot)
			if err != nil {
				t.Fatalf("decode contact %s snapshot: %v", side, err)
			}
			if contact.Fields == nil || len(contact.Fields) != 0 || contact.Tags == nil || len(contact.Tags) != 0 {
				t.Errorf("contact %s snapshot collections = fields:%#v tags:%#v, want non-nil empty", side, contact.Fields, contact.Tags)
			}
		}
	}
}

func assertCanonicalCompanyStateAndHistory(t *testing.T, store storage.Storage, id uuid.UUID, wantEvents int) {
	t.Helper()
	company, err := store.GetCompany(id)
	if err != nil {
		t.Fatalf("GetCompany(%s): %v", id, err)
	}
	if company.Fields == nil || len(company.Fields) != 0 {
		t.Errorf("GetCompany(%s).Fields = %#v, want non-nil empty map", id, company.Fields)
	}
	if company.Tags == nil || len(company.Tags) != 0 {
		t.Errorf("GetCompany(%s).Tags = %#v, want non-nil empty slice", id, company.Tags)
	}
	assertCanonicalCompanySnapshots(t, store, id, wantEvents)
}

func assertCanonicalCompanySnapshots(t *testing.T, store storage.Storage, id uuid.UUID, wantEvents int) {
	t.Helper()
	summaries := mustListHistory(t, store, id)
	if len(summaries) != wantEvents {
		t.Fatalf("ListHistory(%s) len = %d, want %d", id, len(summaries), wantEvents)
	}
	for _, summary := range summaries {
		event, err := store.GetHistoryEvent(summary.ID.String())
		if err != nil {
			t.Fatalf("GetHistoryEvent(%s): %v", summary.ID, err)
		}
		for side, snapshot := range map[string]json.RawMessage{"before": event.Before, "after": event.After} {
			if len(snapshot) == 0 || bytes.Equal(bytes.TrimSpace(snapshot), []byte("null")) {
				continue
			}
			company, err := history.CompanyFromSnapshot(snapshot)
			if err != nil {
				t.Fatalf("decode company %s snapshot: %v", side, err)
			}
			if company.Fields == nil || len(company.Fields) != 0 || company.Tags == nil || len(company.Tags) != 0 {
				t.Errorf("company %s snapshot collections = fields:%#v tags:%#v, want non-nil empty", side, company.Fields, company.Tags)
			}
		}
	}
}

func historyStoreFactories() []historyStoreFactory {
	return []historyStoreFactory{
		{
			name: "sqlite",
			open: func(t *testing.T) storage.Storage {
				t.Helper()
				store, err := storage.NewSqliteStore(filepath.Join(t.TempDir(), "crm.db"), history.SourceCLI)
				if err != nil {
					t.Fatalf("NewSqliteStore: %v", err)
				}
				t.Cleanup(func() { _ = store.Close() })
				return store
			},
		},
		{
			name: "markdown",
			open: func(t *testing.T) storage.Storage {
				t.Helper()
				store, err := storage.NewMarkdownStore(t.TempDir(), history.SourceCLI)
				if err != nil {
					t.Fatalf("NewMarkdownStore: %v", err)
				}
				t.Cleanup(func() { _ = store.Close() })
				return store
			},
		},
	}
}

func runHistoryParityScenario(t *testing.T, store storage.Storage) {
	t.Helper()
	initialContact := &models.Contact{
		ID: historyContactID, Name: "Ada Lovelace", Email: "ada@example.com",
		Fields: map[string]any{"timezone": "Europe/London"}, Tags: []string{"friend"},
		CreatedAt: historyBaseTime, UpdatedAt: historyBaseTime,
	}
	initialCompany := &models.Company{
		ID: historyCompanyID, Name: "Analytical Engines Ltd", Domain: "engines.example",
		Fields: map[string]any{"stage": "research"}, Tags: []string{"portfolio"},
		CreatedAt: historyBaseTime.Add(time.Minute), UpdatedAt: historyBaseTime.Add(time.Minute),
	}
	relationship := &models.Relationship{
		ID: historyRelationshipID, SourceID: initialContact.ID, TargetID: initialCompany.ID,
		Type: "works_at", Context: "mathematics", CreatedAt: historyBaseTime.Add(2 * time.Minute),
	}

	mustStoreMutation(t, "CreateContact", func() error { return store.CreateContact(initialContact) })
	mustStoreMutation(t, "CreateCompany", func() error { return store.CreateCompany(initialCompany) })
	mustStoreMutation(t, "CreateRelationship", func() error { return store.CreateRelationship(relationship) })

	updatedContact := *initialContact
	updatedContact.Phone = "+44 20 7946 0958"
	updatedContact.UpdatedAt = historyBaseTime.Add(3 * time.Minute)
	mustStoreMutation(t, "UpdateContact", func() error { return store.UpdateContact(&updatedContact) })

	updatedCompany := *initialCompany
	updatedCompany.Domain = "analytical-engines.example"
	updatedCompany.UpdatedAt = historyBaseTime.Add(4 * time.Minute)
	mustStoreMutation(t, "UpdateCompany", func() error { return store.UpdateCompany(&updatedCompany) })

	contactNoop := updatedContact
	contactNoop.UpdatedAt = historyBaseTime.Add(5 * time.Minute)
	mustStoreMutation(t, "timestamp-only UpdateContact", func() error { return store.UpdateContact(&contactNoop) })
	companyNoop := updatedCompany
	companyNoop.UpdatedAt = historyBaseTime.Add(6 * time.Minute)
	mustStoreMutation(t, "timestamp-only UpdateCompany", func() error { return store.UpdateCompany(&companyNoop) })

	mustStoreMutation(t, "DeleteRelationship", func() error { return store.DeleteRelationship(relationship.ID) })
	mustStoreMutation(t, "DeleteContact", func() error { return store.DeleteContact(initialContact.ID) })
	mustStoreMutation(t, "DeleteCompany", func() error { return store.DeleteCompany(initialCompany.ID) })

	assertHistoryTimeline(t, store, initialContact.ID, []string{
		"contact/delete", "relationship/delete", "contact/update", "relationship/create", "contact/create",
	})
	assertHistoryTimeline(t, store, initialCompany.ID, []string{
		"company/delete", "relationship/delete", "company/update", "relationship/create", "company/create",
	})
	assertHistoryTimeline(t, store, relationship.ID, []string{"relationship/delete", "relationship/create"})

	expected := expectedParityEvents(t, initialContact, &updatedContact, initialCompany, &updatedCompany, relationship)
	assertAllHistoryEvents(t, store, expected)
}

func mustStoreMutation(t *testing.T, name string, mutate func() error) {
	t.Helper()
	if err := mutate(); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

func assertHistoryTimeline(t *testing.T, store storage.Storage, entityID uuid.UUID, want []string) {
	t.Helper()
	summaries, err := store.ListHistory(entityID.String(), 100)
	if err != nil {
		t.Fatalf("ListHistory(%s): %v", entityID, err)
	}
	got := make([]string, len(summaries))
	for index, summary := range summaries {
		got[index] = historyEventKey(summary.EntityType, summary.Action)
		if summary.Source != history.SourceCLI {
			t.Errorf("ListHistory(%s)[%d].Source = %q, want %q", entityID, index, summary.Source, history.SourceCLI)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("ListHistory(%s) = %v, want %v", entityID, got, want)
	}
}

func expectedParityEvents(
	t *testing.T,
	initialContact, updatedContact *models.Contact,
	initialCompany, updatedCompany *models.Company,
	relationship *models.Relationship,
) map[string]expectedHistoryEvent {
	t.Helper()
	contactBefore := mustContactSnapshot(t, initialContact)
	contactAfter := mustContactSnapshot(t, updatedContact)
	companyBefore := mustCompanySnapshot(t, initialCompany)
	companyAfter := mustCompanySnapshot(t, updatedCompany)
	relationshipSnapshot := mustRelationshipSnapshot(t, relationship)
	return map[string]expectedHistoryEvent{
		"contact/create":      {history.EntityContact, history.ActionCreate, nil, contactBefore},
		"contact/update":      {history.EntityContact, history.ActionUpdate, contactBefore, contactAfter},
		"contact/delete":      {history.EntityContact, history.ActionDelete, contactAfter, nil},
		"company/create":      {history.EntityCompany, history.ActionCreate, nil, companyBefore},
		"company/update":      {history.EntityCompany, history.ActionUpdate, companyBefore, companyAfter},
		"company/delete":      {history.EntityCompany, history.ActionDelete, companyAfter, nil},
		"relationship/create": {history.EntityRelationship, history.ActionCreate, nil, relationshipSnapshot},
		"relationship/delete": {history.EntityRelationship, history.ActionDelete, relationshipSnapshot, nil},
	}
}

func assertAllHistoryEvents(t *testing.T, store storage.Storage, expected map[string]expectedHistoryEvent) {
	t.Helper()
	summaryGroups := [][]*history.Summary{
		mustListHistory(t, store, historyContactID),
		mustListHistory(t, store, historyCompanyID),
		mustListHistory(t, store, historyRelationshipID),
	}
	seen := make(map[uuid.UUID]struct{}, len(expected))
	for _, summaries := range summaryGroups {
		for _, summary := range summaries {
			if _, ok := seen[summary.ID]; ok {
				continue
			}
			seen[summary.ID] = struct{}{}
			prefix := summary.ID.String()[:6]
			event, err := store.GetHistoryEvent(prefix)
			if err != nil {
				t.Fatalf("GetHistoryEvent(%q): %v", prefix, err)
			}
			if event.ID != summary.ID {
				t.Errorf("GetHistoryEvent(%q).ID = %s, want %s", prefix, event.ID, summary.ID)
			}
			want, ok := expected[historyEventKey(event.EntityType, event.Action)]
			if !ok {
				t.Errorf("unexpected event %s %s %s", event.ID, event.EntityType, event.Action)
				continue
			}
			assertHistoryEvent(t, event, want)
		}
	}
	if len(seen) != len(expected) {
		t.Errorf("unique history event count = %d, want %d", len(seen), len(expected))
	}
}

func mustListHistory(t *testing.T, store storage.Storage, entityID uuid.UUID) []*history.Summary {
	t.Helper()
	summaries, err := store.ListHistory(entityID.String(), 100)
	if err != nil {
		t.Fatalf("ListHistory(%s): %v", entityID, err)
	}
	return summaries
}

func assertHistoryEvent(t *testing.T, got *history.Event, want expectedHistoryEvent) {
	t.Helper()
	if got.EntityType != want.entityType || got.Action != want.action || got.Source != history.SourceCLI {
		t.Errorf("event metadata = %s/%s/%s, want %s/%s/%s", got.EntityType, got.Action, got.Source, want.entityType, want.action, history.SourceCLI)
	}
	assertSnapshotsEqual(t, got.EntityType, "before", got.Before, want.before)
	assertSnapshotsEqual(t, got.EntityType, "after", got.After, want.after)
}

func assertSnapshotsEqual(t *testing.T, entityType history.EntityType, side string, got, want json.RawMessage) {
	t.Helper()
	equal, err := history.EqualSnapshots(entityType, got, want)
	if err != nil {
		t.Fatalf("compare %s %s snapshot: %v", entityType, side, err)
	}
	if !equal {
		t.Errorf("%s %s snapshot = %s, want %s", entityType, side, got, want)
	}
}

func historyEventKey(entityType history.EntityType, action history.Action) string {
	return fmt.Sprintf("%s/%s", entityType, action)
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

func TestHistoryLegacyData(t *testing.T) {
	t.Run("sqlite", testSQLiteHistoryLegacyData)
	t.Run("markdown", testMarkdownHistoryLegacyData)
}

func testSQLiteHistoryLegacyData(t *testing.T) {
	legacy := legacyContact()
	dbPath := filepath.Join(t.TempDir(), "crm.db")
	createLegacySQLiteDatabase(t, dbPath, legacy)
	store, err := storage.NewSqliteStore(dbPath, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewSqliteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	assertLegacyHistorySeed(t, store, legacy)
}

func testMarkdownHistoryLegacyData(t *testing.T) {
	legacy := legacyContact()
	dataDir := t.TempDir()
	contactsDir := filepath.Join(dataDir, "contacts")
	if err := os.Mkdir(contactsDir, 0o750); err != nil {
		t.Fatalf("create contacts directory: %v", err)
	}
	content := fmt.Sprintf(`---
id: %s
name: %s
email: %s
fields: {}
tags: []
created_at: %s
updated_at: %s
---
`, legacy.ID, legacy.Name, legacy.Email, legacy.CreatedAt.Format(time.RFC3339), legacy.UpdatedAt.Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(contactsDir, "legacy-contact.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("write legacy contact: %v", err)
	}
	store, err := storage.NewMarkdownStore(dataDir, history.SourceCLI)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	assertLegacyHistorySeed(t, store, legacy)
}

func legacyContact() *models.Contact {
	return &models.Contact{
		ID:   uuid.MustParse("44444444-4444-4444-8444-444444444444"),
		Name: "Legacy Contact", Email: "legacy@example.com",
		Fields: map[string]any{}, Tags: []string{},
		CreatedAt: historyBaseTime, UpdatedAt: historyBaseTime,
	}
}

func assertLegacyHistorySeed(t *testing.T, store storage.Storage, legacy *models.Contact) {
	t.Helper()
	empty, err := store.ListHistory(legacy.ID.String(), 0)
	if err != nil {
		t.Fatalf("ListHistory before update: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("ListHistory before update = %#v, want empty non-nil slice", empty)
	}

	updated := *legacy
	updated.Phone = "+1 312 555 0199"
	updated.UpdatedAt = historyBaseTime.Add(time.Minute)
	if err := store.UpdateContact(&updated); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}
	summaries := mustListHistory(t, store, legacy.ID)
	if len(summaries) != 1 {
		t.Fatalf("ListHistory after update len = %d, want 1", len(summaries))
	}
	if summaries[0].EntityType != history.EntityContact || summaries[0].Action != history.ActionUpdate {
		t.Fatalf("legacy event = %s/%s, want contact/update", summaries[0].EntityType, summaries[0].Action)
	}
	if summaries[0].Source != history.SourceCLI || summaries[0].EntityID != legacy.ID {
		t.Fatalf("legacy event source/entity = %s/%s, want cli/%s", summaries[0].Source, summaries[0].EntityID, legacy.ID)
	}
	event, err := store.GetHistoryEvent(summaries[0].ID.String()[:6])
	if err != nil {
		t.Fatalf("GetHistoryEvent: %v", err)
	}
	assertSnapshotsEqual(t, history.EntityContact, "before", event.Before, mustContactSnapshot(t, legacy))
	assertSnapshotsEqual(t, history.EntityContact, "after", event.After, mustContactSnapshot(t, &updated))
}

func createLegacySQLiteDatabase(t *testing.T, path string, contact *models.Contact) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open legacy SQLite database: %v", err)
	}
	defer func() { _ = db.Close() }()
	statements := []string{
		`CREATE TABLE contacts (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
			email TEXT DEFAULT '', phone TEXT DEFAULT '', fields TEXT DEFAULT '{}', tags TEXT DEFAULT '[]',
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE companies (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
			domain TEXT DEFAULT '', fields TEXT DEFAULT '{}', tags TEXT DEFAULT '[]',
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE relationships (
			id TEXT PRIMARY KEY, source_id TEXT NOT NULL, target_id TEXT NOT NULL,
			type TEXT NOT NULL, context TEXT DEFAULT '', created_at DATETIME NOT NULL
		)`,
		`CREATE VIRTUAL TABLE contacts_fts USING fts5(name, email, fields, content=contacts, content_rowid=rowid)`,
		`CREATE VIRTUAL TABLE companies_fts USING fts5(name, domain, fields, content=companies, content_rowid=rowid)`,
		`CREATE TRIGGER contacts_ai AFTER INSERT ON contacts BEGIN
			INSERT INTO contacts_fts(rowid, name, email, fields) VALUES (new.rowid, new.name, new.email, new.fields);
		END`,
		`CREATE TRIGGER contacts_ad AFTER DELETE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, name, email, fields) VALUES ('delete', old.rowid, old.name, old.email, old.fields);
		END`,
		`CREATE TRIGGER contacts_au AFTER UPDATE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, name, email, fields) VALUES ('delete', old.rowid, old.name, old.email, old.fields);
			INSERT INTO contacts_fts(rowid, name, email, fields) VALUES (new.rowid, new.name, new.email, new.fields);
		END`,
		`CREATE TRIGGER companies_ai AFTER INSERT ON companies BEGIN
			INSERT INTO companies_fts(rowid, name, domain, fields) VALUES (new.rowid, new.name, new.domain, new.fields);
		END`,
		`CREATE TRIGGER companies_ad AFTER DELETE ON companies BEGIN
			INSERT INTO companies_fts(companies_fts, rowid, name, domain, fields) VALUES ('delete', old.rowid, old.name, old.domain, old.fields);
		END`,
		`CREATE TRIGGER companies_au AFTER UPDATE ON companies BEGIN
			INSERT INTO companies_fts(companies_fts, rowid, name, domain, fields) VALUES ('delete', old.rowid, old.name, old.domain, old.fields);
			INSERT INTO companies_fts(rowid, name, domain, fields) VALUES (new.rowid, new.name, new.domain, new.fields);
		END`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create legacy SQLite schema: %v", err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO contacts (id, name, email, phone, fields, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		contact.ID.String(), contact.Name, contact.Email, contact.Phone, `{}`, `[]`, contact.CreatedAt, contact.UpdatedAt,
	); err != nil {
		t.Fatalf("insert legacy SQLite contact: %v", err)
	}
}
