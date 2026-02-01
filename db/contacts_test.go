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

func TestFindContacts(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{Name: "ACME Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contacts
	contact1 := &models.Contact{
		Name:      "Alice Smith",
		Email:     "alice@example.com",
		CompanyID: &company.ID,
	}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contact2 := &models.Contact{
		Name:  "Bob Jones",
		Email: "bob@other.com",
	}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contact3 := &models.Contact{
		Name:      "Charlie Smith",
		Email:     "charlie@example.com",
		CompanyID: &company.ID,
	}
	if err := CreateContact(db, contact3); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Test find all contacts
	contacts, err := FindContacts(db, "", nil, 10)
	if err != nil {
		t.Fatalf("FindContacts failed: %v", err)
	}
	if len(contacts) != 3 {
		t.Errorf("Expected 3 contacts, got %d", len(contacts))
	}

	// Test find by name query
	contacts, err = FindContacts(db, "Smith", nil, 10)
	if err != nil {
		t.Fatalf("FindContacts by name failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts matching 'Smith', got %d", len(contacts))
	}

	// Test find by email query
	contacts, err = FindContacts(db, "example.com", nil, 10)
	if err != nil {
		t.Fatalf("FindContacts by email failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts matching 'example.com', got %d", len(contacts))
	}

	// Test find by company ID
	contacts, err = FindContacts(db, "", &company.ID, 10)
	if err != nil {
		t.Fatalf("FindContacts by company failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts in company, got %d", len(contacts))
	}

	// Test combined query and company filter
	contacts, err = FindContacts(db, "Alice", &company.ID, 10)
	if err != nil {
		t.Fatalf("FindContacts with combined filter failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact matching 'Alice' in company, got %d", len(contacts))
	}

	// Test limit
	contacts, err = FindContacts(db, "", nil, 2)
	if err != nil {
		t.Fatalf("FindContacts with limit failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts with limit, got %d", len(contacts))
	}

	// Test zero limit defaults to 10
	contacts, err = FindContacts(db, "", nil, 0)
	if err != nil {
		t.Fatalf("FindContacts with zero limit failed: %v", err)
	}
	if len(contacts) != 3 {
		t.Errorf("Expected 3 contacts with zero limit, got %d", len(contacts))
	}
}

func TestUpdateContact(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{Name: "ACME Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contact
	contact := &models.Contact{
		Name:  "Original Name",
		Email: "original@example.com",
		Phone: "123-456-7890",
		Notes: "Original notes",
	}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Update contact
	updates := &models.Contact{
		Name:      "Updated Name",
		Email:     "updated@example.com",
		Phone:     "999-888-7777",
		Notes:     "Updated notes",
		CompanyID: &company.ID,
	}
	if err := UpdateContact(db, contact.ID, updates); err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}

	// Verify updates
	found, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if found.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", found.Name)
	}
	if found.Email != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got %s", found.Email)
	}
	if found.Phone != "999-888-7777" {
		t.Errorf("Expected phone '999-888-7777', got %s", found.Phone)
	}
	if found.Notes != "Updated notes" {
		t.Errorf("Expected notes 'Updated notes', got %s", found.Notes)
	}
	if found.CompanyID == nil || *found.CompanyID != company.ID {
		t.Error("Company ID was not updated correctly")
	}

	// Test removing company ID
	updates2 := &models.Contact{
		Name:      found.Name,
		Email:     found.Email,
		Phone:     found.Phone,
		Notes:     found.Notes,
		CompanyID: nil,
	}
	if err := UpdateContact(db, contact.ID, updates2); err != nil {
		t.Fatalf("UpdateContact to remove company failed: %v", err)
	}

	found, err = GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if found.CompanyID != nil {
		t.Error("Company ID should be nil after removal")
	}

	// Test update non-existent contact
	err = UpdateContact(db, uuid.New(), updates)
	if err == nil {
		t.Error("Expected error when updating non-existent contact")
	}
}

func TestDeleteContact(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{Name: "ACME Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contact
	contact := &models.Contact{
		Name:      "John Doe",
		Email:     "john@example.com",
		CompanyID: &company.ID,
	}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Create a deal with this contact
	deal := &models.Deal{
		Title:     "Test Deal",
		CompanyID: company.ID,
		Stage:     "prospecting",
		Currency:  "USD",
	}
	// We need to set contact_id in the deal metadata
	if err := CreateDeal(db, deal); err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	// Delete contact
	if err := DeleteContact(db, contact.ID); err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	// Verify contact is deleted
	found, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact after delete failed: %v", err)
	}
	if found != nil {
		t.Error("Contact should be deleted")
	}

	// Test deleting non-existent contact
	err = DeleteContact(db, uuid.New())
	if err == nil {
		t.Error("Expected error when deleting non-existent contact")
	}
}

func TestGetContactRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contact3 := &models.Contact{Name: "Charlie Brown"}
	if err := CreateContact(db, contact3); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Create relationships
	rel1 := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
	}
	if err := CreateRelationship(db, rel1); err != nil {
		t.Fatalf("CreateRelationship failed: %v", err)
	}

	rel2 := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact3.ID,
		RelationshipType: "friend",
	}
	if err := CreateRelationship(db, rel2); err != nil {
		t.Fatalf("CreateRelationship failed: %v", err)
	}

	// Get relationships for contact1
	rels, err := GetContactRelationships(db, contact1.ID)
	if err != nil {
		t.Fatalf("GetContactRelationships failed: %v", err)
	}
	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(rels))
	}

	// Get relationships for contact with no relationships
	contact4 := &models.Contact{Name: "Dave Wilson"}
	if err := CreateContact(db, contact4); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	rels, err = GetContactRelationships(db, contact4.ID)
	if err != nil {
		t.Fatalf("GetContactRelationships failed: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("Expected 0 relationships for new contact, got %d", len(rels))
	}
}

func TestGetRelationshipsBetween(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Test with no relationship
	rels, err := GetRelationshipsBetween(db, contact1.ID, contact2.ID)
	if err != nil {
		t.Fatalf("GetRelationshipsBetween failed: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("Expected 0 relationships, got %d", len(rels))
	}

	// Create relationship
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
		Context:          "Work together",
	}
	if err := CreateRelationship(db, rel); err != nil {
		t.Fatalf("CreateRelationship failed: %v", err)
	}

	// Test with relationship
	rels, err = GetRelationshipsBetween(db, contact1.ID, contact2.ID)
	if err != nil {
		t.Fatalf("GetRelationshipsBetween failed: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(rels))
	}
	if rels[0].RelationshipType != "colleague" {
		t.Errorf("Expected type 'colleague', got %s", rels[0].RelationshipType)
	}

	// Test with reversed order
	rels, err = GetRelationshipsBetween(db, contact2.ID, contact1.ID)
	if err != nil {
		t.Fatalf("GetRelationshipsBetween with reversed order failed: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("Expected 1 relationship with reversed order, got %d", len(rels))
	}
}
