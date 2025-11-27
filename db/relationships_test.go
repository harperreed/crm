// ABOUTME: Test suite for relationship database operations
// ABOUTME: Verifies CRUD operations and bidirectional relationship handling
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func TestCreateRelationship(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	// Test 1: Create relationship with correct ordering
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
		Context:          "Work together at ACME Corp",
	}

	if err := CreateRelationship(db, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Verify UUID was assigned
	if rel.ID == uuid.Nil {
		t.Error("Expected ID to be assigned")
	}

	// Verify timestamps were set
	if rel.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if rel.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Test 2: Verify ordering - contact_id_1 should be < contact_id_2
	if rel.ContactID1.String() >= rel.ContactID2.String() {
		t.Errorf("Expected ContactID1 (%s) < ContactID2 (%s)", rel.ContactID1, rel.ContactID2)
	}

	// Test 3: Create relationship with reverse order - should be normalized
	rel2 := &models.Relationship{
		ContactID1:       contact2.ID, // Deliberately reversed
		ContactID2:       contact1.ID,
		RelationshipType: "friend",
	}

	if err := CreateRelationship(db, rel2); err != nil {
		t.Fatalf("Failed to create relationship with reversed IDs: %v", err)
	}

	// Should still maintain ordering
	if rel2.ContactID1.String() >= rel2.ContactID2.String() {
		t.Errorf("Expected ContactID1 (%s) < ContactID2 (%s) after normalization", rel2.ContactID1, rel2.ContactID2)
	}
}

func TestGetRelationship(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	// Create a relationship
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "mentor",
		Context:          "Bob mentors Alice",
	}

	if err := CreateRelationship(db, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Test 1: Retrieve by ID
	retrieved, err := GetRelationship(db, rel.ID)
	if err != nil {
		t.Fatalf("Failed to get relationship: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected relationship to be found")
	}

	if retrieved.ID != rel.ID {
		t.Errorf("Expected ID %s, got %s", rel.ID, retrieved.ID)
	}

	if retrieved.RelationshipType != "mentor" {
		t.Errorf("Expected type 'mentor', got '%s'", retrieved.RelationshipType)
	}

	if retrieved.Context != "Bob mentors Alice" {
		t.Errorf("Expected context 'Bob mentors Alice', got '%s'", retrieved.Context)
	}

	// Test 2: Non-existent ID
	nonExistent, err := GetRelationship(db, uuid.New())
	if err != nil {
		t.Fatalf("Expected no error for non-existent ID, got %v", err)
	}

	if nonExistent != nil {
		t.Error("Expected nil for non-existent relationship")
	}
}

func TestFindRelationshipsBetween(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	// Test 1: No relationship exists
	rels, err := FindRelationshipsBetween(db, contact1.ID, contact2.ID)
	if err != nil {
		t.Fatalf("Failed to find relationships: %v", err)
	}

	if len(rels) != 0 {
		t.Errorf("Expected 0 relationships, got %d", len(rels))
	}

	// Create a relationship
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "sibling",
	}

	if err := CreateRelationship(db, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Test 2: Find with same order
	rels, err = FindRelationshipsBetween(db, contact1.ID, contact2.ID)
	if err != nil {
		t.Fatalf("Failed to find relationships: %v", err)
	}

	if len(rels) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(rels))
	}

	if rels[0].RelationshipType != "sibling" {
		t.Errorf("Expected type 'sibling', got '%s'", rels[0].RelationshipType)
	}

	// Test 3: Find with reversed order (bidirectional search)
	rels, err = FindRelationshipsBetween(db, contact2.ID, contact1.ID)
	if err != nil {
		t.Fatalf("Failed to find relationships with reversed IDs: %v", err)
	}

	if len(rels) != 1 {
		t.Errorf("Expected 1 relationship with reversed search, got %d", len(rels))
	}
}

func TestFindContactRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create three contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	contact3 := &models.Contact{Name: "Charlie Brown"}
	if err := CreateContact(db, contact3); err != nil {
		t.Fatalf("Failed to create contact3: %v", err)
	}

	// Create multiple relationships for contact1
	rel1 := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
	}
	if err := CreateRelationship(db, rel1); err != nil {
		t.Fatalf("Failed to create relationship 1: %v", err)
	}

	// Sleep to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	rel2 := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact3.ID,
		RelationshipType: "friend",
	}
	if err := CreateRelationship(db, rel2); err != nil {
		t.Fatalf("Failed to create relationship 2: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	rel3 := &models.Relationship{
		ContactID1:       contact2.ID,
		ContactID2:       contact3.ID,
		RelationshipType: "colleague",
	}
	if err := CreateRelationship(db, rel3); err != nil {
		t.Fatalf("Failed to create relationship 3: %v", err)
	}

	// Test 1: Find all relationships for contact1 (no filter)
	rels, err := FindContactRelationships(db, contact1.ID, "")
	if err != nil {
		t.Fatalf("Failed to find contact relationships: %v", err)
	}

	if len(rels) != 2 {
		t.Fatalf("Expected 2 relationships for contact1, got %d", len(rels))
	}

	// Test 2: Filter by relationship type "colleague"
	rels, err = FindContactRelationships(db, contact1.ID, "colleague")
	if err != nil {
		t.Fatalf("Failed to find filtered relationships: %v", err)
	}

	if len(rels) != 1 {
		t.Fatalf("Expected 1 'colleague' relationship for contact1, got %d", len(rels))
	}

	if rels[0].RelationshipType != "colleague" {
		t.Errorf("Expected type 'colleague', got '%s'", rels[0].RelationshipType)
	}

	// Test 3: Find all relationships for contact2
	rels, err = FindContactRelationships(db, contact2.ID, "")
	if err != nil {
		t.Fatalf("Failed to find contact2 relationships: %v", err)
	}

	if len(rels) != 2 {
		t.Fatalf("Expected 2 relationships for contact2, got %d", len(rels))
	}

	// Test 4: Contact with no relationships
	contact4 := &models.Contact{Name: "David Wilson"}
	if err := CreateContact(db, contact4); err != nil {
		t.Fatalf("Failed to create contact4: %v", err)
	}

	rels, err = FindContactRelationships(db, contact4.ID, "")
	if err != nil {
		t.Fatalf("Failed to find contact4 relationships: %v", err)
	}

	if len(rels) != 0 {
		t.Errorf("Expected 0 relationships for contact4, got %d", len(rels))
	}
}

func TestDeleteRelationship(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two contacts
	contact1 := &models.Contact{Name: "Alice Smith"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob Jones"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	// Create a relationship
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "partner",
	}

	if err := CreateRelationship(db, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Verify it exists
	retrieved, err := GetRelationship(db, rel.ID)
	if err != nil {
		t.Fatalf("Failed to get relationship: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected relationship to exist before deletion")
	}

	// Test 1: Delete the relationship
	if err := DeleteRelationship(db, rel.ID); err != nil {
		t.Fatalf("Failed to delete relationship: %v", err)
	}

	// Test 2: Verify it's gone
	retrieved, err = GetRelationship(db, rel.ID)
	if err != nil {
		t.Fatalf("Failed to get relationship after deletion: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected relationship to be deleted")
	}

	// Test 3: Delete non-existent relationship (should not error)
	if err := DeleteRelationship(db, uuid.New()); err != nil {
		t.Fatalf("Expected no error when deleting non-existent relationship, got %v", err)
	}
}
