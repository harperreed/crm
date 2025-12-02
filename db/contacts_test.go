// ABOUTME: Tests for contact database operations
// ABOUTME: Covers CRUD operations and contact lookups
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func TestCreateContact(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	contact := &models.Contact{
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "+1234567890",
		Notes: "Test contact",
	}

	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	if contact.ID == uuid.Nil {
		t.Error("Contact ID was not set")
	}
}

func TestCreateContactWithCompany(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company first
	company := &models.Company{Name: "Test Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contact with company
	contact := &models.Contact{
		Name:      "Jane Doe",
		Email:     "jane@test.com",
		CompanyID: &company.ID,
	}

	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Verify company ID was set
	found, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if found.CompanyID == nil || *found.CompanyID != company.ID {
		t.Error("Company ID not set correctly")
	}
}

func TestUpdateContactLastContacted(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	contact := &models.Contact{Name: "Test Contact"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	now := time.Now()
	if err := UpdateContactLastContacted(db, contact.ID, now); err != nil {
		t.Fatalf("UpdateContactLastContacted failed: %v", err)
	}

	found, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if found.LastContactedAt == nil {
		t.Fatal("LastContactedAt was not set")
	}

	// Allow for small precision loss from JSON serialization (RFC3339 has second precision)
	diff := found.LastContactedAt.Sub(now)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("LastContactedAt time mismatch: got %v, want %v (diff: %v)", found.LastContactedAt, now, diff)
	}
}
