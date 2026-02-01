// ABOUTME: Extended tests for contact CLI commands
// ABOUTME: Covers queueContactToVault and additional edge cases
package cli

import (
	"os"
	"testing"

	"github.com/harperreed/pagen/repository"
	"github.com/harperreed/sweet/vault"
)

func TestQueueContactToVaultNotConfigured(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Save and restore HOME to ensure vault config doesn't exist
	origHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	tempDir := t.TempDir()
	_ = os.Setenv("HOME", tempDir)

	contact := &repository.Contact{Name: "Test Contact"}

	// This should silently return without error when vault is not configured
	queueContactToVault(db, contact, vault.OpUpsert)
	queueContactToVault(db, contact, vault.OpDelete)

	// No panic = success
}

func TestListContactsEmptyPhoneAndEmail(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contact with no email or phone
	_ = db.CreateContact(&repository.Contact{Name: "No Details Contact"})

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ListContactsCommand(db, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListContactsCommand() unexpected error: %v", err)
	}
}

func TestListContactsWithCompanyName(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company and contact
	company := &repository.Company{Name: "Display Corp"}
	_ = db.CreateCompany(company)
	_ = db.CreateContact(&repository.Contact{
		Name:        "Display Contact",
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	})

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ListContactsCommand(db, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListContactsCommand() unexpected error: %v", err)
	}
}

func TestUpdateContactWithNotes(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	contact := &repository.Contact{Name: "Notes Test"}
	_ = db.CreateContact(contact)

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = UpdateContactCommand(db, []string{"--notes", "Updated notes content", contact.ID.String()})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("UpdateContactCommand() unexpected error: %v", err)
	}

	// Verify notes updated
	updated, _ := db.GetContact(contact.ID)
	if updated.Notes != "Updated notes content" {
		t.Errorf("Notes should be updated, got %s", updated.Notes)
	}
}

func TestAddContactWithAllFields(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = AddContactCommand(db, []string{
		"--name", "Full Details",
		"--email", "full@details.com",
		"--phone", "555-FULL",
		"--notes", "Full notes",
		"--company", "Full Corp",
	})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("AddContactCommand() unexpected error: %v", err)
	}

	// Verify contact created with all fields
	contacts, _ := db.ListContacts(&repository.ContactFilter{Query: "full@details.com", Limit: 10})
	if len(contacts) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(contacts))
	}

	c := contacts[0]
	if c.Name != "Full Details" {
		t.Errorf("Name = %s, want Full Details", c.Name)
	}
	if c.Email != "full@details.com" {
		t.Errorf("Email = %s, want full@details.com", c.Email)
	}
	if c.Phone != "555-FULL" {
		t.Errorf("Phone = %s, want 555-FULL", c.Phone)
	}
	if c.Notes != "Full notes" {
		t.Errorf("Notes = %s, want Full notes", c.Notes)
	}
	if c.CompanyID == nil {
		t.Error("CompanyID should not be nil")
	}
}

func TestListContactsCompanyNotFound(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create a contact without company
	_ = db.CreateContact(&repository.Contact{Name: "Solo Contact"})

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Filter by non-existent company - should return empty results
	err = ListContactsCommand(db, []string{"--company", "NonExistent Corp"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListContactsCommand() unexpected error: %v", err)
	}
}
