// sync/contacts_importer_test.go
package sync

import (
	"database/sql"
	"testing"

	"github.com/harperreed/pagen/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	if err := db.InitSchema(database); err != nil {
		t.Fatal(err)
	}

	return database
}

func TestContactsImporterCreate(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	importer := NewContactsImporter(database)
	if importer == nil {
		t.Fatal("expected importer, got nil")
	}
}

func TestImportContact(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	importer := NewContactsImporter(database)

	// Simulate Google Contacts person data
	contactData := &GoogleContact{
		ResourceName: "people/123",
		Name:         "Alice Smith",
		Email:        "alice@example.com",
		Phone:        "555-1234",
		Company:      "Acme Corp",
	}

	created, err := importer.ImportContact(contactData)
	if err != nil {
		t.Fatalf("failed to import contact: %v", err)
	}

	if !created {
		t.Error("expected new contact to be created")
	}

	// Verify contact was created
	contacts, err := db.FindContacts(database, "alice@example.com", nil, 10)
	if err != nil {
		t.Fatalf("failed to find contact: %v", err)
	}

	if len(contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(contacts))
	}

	if contacts[0].Email != "alice@example.com" {
		t.Errorf("expected alice@example.com, got %s", contacts[0].Email)
	}
}
