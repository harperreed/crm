// ABOUTME: Tests for the cross-entity search functionality.
// ABOUTME: Verifies that Search returns results from both contacts and companies.
package storage

import (
	"testing"

	"github.com/harperreed/crm/internal/models"
)

func TestSearch(t *testing.T) {
	store := newTestStore(t)

	contact := models.NewContact("Quantum Findable")
	contact.Email = "quantum@findable.com"
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	company := models.NewCompany("Quantum Enterprises")
	company.Domain = "quantum.com"
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	// Create a non-matching contact to verify filtering
	other := models.NewContact("Invisible Person")
	if err := store.CreateContact(other); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	results, err := store.Search("Quantum")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Contacts) != 1 {
		t.Errorf("Contacts len = %d, want 1", len(results.Contacts))
	}
	if len(results.Companies) != 1 {
		t.Errorf("Companies len = %d, want 1", len(results.Companies))
	}

	if len(results.Contacts) == 1 && results.Contacts[0].Name != "Quantum Findable" {
		t.Errorf("Contact name = %q, want %q", results.Contacts[0].Name, "Quantum Findable")
	}
	if len(results.Companies) == 1 && results.Companies[0].Name != "Quantum Enterprises" {
		t.Errorf("Company name = %q, want %q", results.Companies[0].Name, "Quantum Enterprises")
	}
}

func TestSearchNoResults(t *testing.T) {
	store := newTestStore(t)

	results, err := store.Search("nonexistent")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Contacts) != 0 {
		t.Errorf("Contacts len = %d, want 0", len(results.Contacts))
	}
	if len(results.Companies) != 0 {
		t.Errorf("Companies len = %d, want 0", len(results.Companies))
	}
}
