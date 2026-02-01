// ABOUTME: Tests for relationship MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
)

func TestLinkContacts(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create two contacts
	contact1Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name":  "Alice Smith",
		"email": "alice@example.com",
	})
	contact1Data := contact1Result.(map[string]interface{})
	contact1ID := contact1Data["id"].(string)

	contact2Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name":  "Bob Jones",
		"email": "bob@example.com",
	})
	contact2Data := contact2Result.(map[string]interface{})
	contact2ID := contact2Data["id"].(string)

	// Link contacts with relationship type and context
	input := map[string]interface{}{
		"contact_id_1":      contact1ID,
		"contact_id_2":      contact2ID,
		"relationship_type": "colleague",
		"context":           "Work together at Acme Corp",
	}

	result, err := handler.LinkContacts_Legacy(input)
	if err != nil {
		t.Fatalf("LinkContacts failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["id"] == nil {
		t.Error("ID was not set")
	}

	if data["relationship_type"] != "colleague" {
		t.Errorf("Expected relationship_type 'colleague', got %v", data["relationship_type"])
	}

	if data["context"] != "Work together at Acme Corp" {
		t.Errorf("Expected context 'Work together at Acme Corp', got %v", data["context"])
	}

	// Verify UUID ordering is handled (should work regardless of input order)
	contactID1UUID, _ := uuid.Parse(contact1ID)
	contactID2UUID, _ := uuid.Parse(contact2ID)

	// Find the relationship in database to verify ordering
	rel, err := client.GetRelationshipBetween(contactID1UUID, contactID2UUID)
	if err != nil {
		t.Fatalf("Failed to find relationship: %v", err)
	}
	if rel == nil {
		t.Error("Expected to find a relationship, got nil")
	}
}

func TestLinkContactsWithoutOptionalFields(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create two contacts
	contact1Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Contact 1",
	})
	contact1Data := contact1Result.(map[string]interface{})
	contact1ID := contact1Data["id"].(string)

	contact2Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Contact 2",
	})
	contact2Data := contact2Result.(map[string]interface{})
	contact2ID := contact2Data["id"].(string)

	// Link without type or context
	input := map[string]interface{}{
		"contact_id_1": contact1ID,
		"contact_id_2": contact2ID,
	}

	result, err := handler.LinkContacts_Legacy(input)
	if err != nil {
		t.Fatalf("LinkContacts failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["id"] == nil {
		t.Error("ID was not set")
	}
}

func TestLinkContactsValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)

	// Missing contact_id_1
	input := map[string]interface{}{
		"contact_id_2": uuid.New().String(),
	}

	_, err := handler.LinkContacts_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for missing contact_id_1")
	}

	// Missing contact_id_2
	input = map[string]interface{}{
		"contact_id_1": uuid.New().String(),
	}

	_, err = handler.LinkContacts_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for missing contact_id_2")
	}

	// Invalid UUID for contact_id_1
	input = map[string]interface{}{
		"contact_id_1": "not-a-uuid",
		"contact_id_2": uuid.New().String(),
	}

	_, err = handler.LinkContacts_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for invalid contact_id_1")
	}

	// Invalid UUID for contact_id_2
	input = map[string]interface{}{
		"contact_id_1": uuid.New().String(),
		"contact_id_2": "not-a-uuid",
	}

	_, err = handler.LinkContacts_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for invalid contact_id_2")
	}
}

func TestFindContactRelationships(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create three contacts
	contact1Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Alice",
	})
	contact1Data := contact1Result.(map[string]interface{})
	contact1ID := contact1Data["id"].(string)

	contact2Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Bob",
	})
	contact2Data := contact2Result.(map[string]interface{})
	contact2ID := contact2Data["id"].(string)

	contact3Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Charlie",
	})
	contact3Data := contact3Result.(map[string]interface{})
	contact3ID := contact3Data["id"].(string)

	// Create multiple relationships for contact1
	_, _ = handler.LinkContacts_Legacy(map[string]interface{}{
		"contact_id_1":      contact1ID,
		"contact_id_2":      contact2ID,
		"relationship_type": "colleague",
	})

	_, _ = handler.LinkContacts_Legacy(map[string]interface{}{
		"contact_id_1":      contact1ID,
		"contact_id_2":      contact3ID,
		"relationship_type": "friend",
	})

	// Find all relationships for contact1
	input := map[string]interface{}{
		"contact_id": contact1ID,
	}

	result, err := handler.FindContactRelationships_Legacy(input)
	if err != nil {
		t.Fatalf("FindContactRelationships failed: %v", err)
	}

	relationships, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(relationships))
	}

	// Verify contact enrichment - each relationship should have contact_1 and contact_2 with names
	for _, rel := range relationships {
		contact1, ok := rel["contact_1"].(map[string]interface{})
		if !ok {
			t.Error("contact_1 is not a map")
			continue
		}

		if contact1["id"] == nil || contact1["name"] == nil {
			t.Error("contact_1 should have id and name")
		}

		contact2, ok := rel["contact_2"].(map[string]interface{})
		if !ok {
			t.Error("contact_2 is not a map")
			continue
		}

		if contact2["id"] == nil || contact2["name"] == nil {
			t.Error("contact_2 should have id and name")
		}
	}
}

func TestFindContactRelationshipsWithTypeFilter(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create contacts
	contact1Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Alice",
	})
	contact1Data := contact1Result.(map[string]interface{})
	contact1ID := contact1Data["id"].(string)

	contact2Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Bob",
	})
	contact2Data := contact2Result.(map[string]interface{})
	contact2ID := contact2Data["id"].(string)

	contact3Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Charlie",
	})
	contact3Data := contact3Result.(map[string]interface{})
	contact3ID := contact3Data["id"].(string)

	// Create relationships with different types
	_, _ = handler.LinkContacts_Legacy(map[string]interface{}{
		"contact_id_1":      contact1ID,
		"contact_id_2":      contact2ID,
		"relationship_type": "colleague",
	})

	_, _ = handler.LinkContacts_Legacy(map[string]interface{}{
		"contact_id_1":      contact1ID,
		"contact_id_2":      contact3ID,
		"relationship_type": "friend",
	})

	// Filter by relationship type
	input := map[string]interface{}{
		"contact_id":        contact1ID,
		"relationship_type": "colleague",
	}

	result, err := handler.FindContactRelationships_Legacy(input)
	if err != nil {
		t.Fatalf("FindContactRelationships failed: %v", err)
	}

	relationships, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(relationships))
	}

	if relationships[0]["relationship_type"] != "colleague" {
		t.Errorf("Expected relationship_type 'colleague', got %v", relationships[0]["relationship_type"])
	}
}

func TestFindContactRelationshipsBidirectional(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create two contacts
	contact1Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Alice",
	})
	contact1Data := contact1Result.(map[string]interface{})
	contact1ID := contact1Data["id"].(string)

	contact2Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Bob",
	})
	contact2Data := contact2Result.(map[string]interface{})
	contact2ID := contact2Data["id"].(string)

	// Create relationship
	_, _ = handler.LinkContacts_Legacy(map[string]interface{}{
		"contact_id_1":      contact1ID,
		"contact_id_2":      contact2ID,
		"relationship_type": "colleague",
	})

	// Find relationships from contact1's perspective
	input := map[string]interface{}{
		"contact_id": contact1ID,
	}

	result1, err := handler.FindContactRelationships_Legacy(input)
	if err != nil {
		t.Fatalf("FindContactRelationships failed: %v", err)
	}

	relationships1, ok := result1.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	// Find relationships from contact2's perspective
	input = map[string]interface{}{
		"contact_id": contact2ID,
	}

	result2, err := handler.FindContactRelationships_Legacy(input)
	if err != nil {
		t.Fatalf("FindContactRelationships failed: %v", err)
	}

	relationships2, ok := result2.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	// Both should find the same relationship
	if len(relationships1) != 1 || len(relationships2) != 1 {
		t.Error("Bidirectional search should find relationship from both contacts")
	}

	if relationships1[0]["id"] != relationships2[0]["id"] {
		t.Error("Should find the same relationship from both perspectives")
	}
}

func TestRemoveRelationship(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create two contacts
	contact1Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Alice",
	})
	contact1Data := contact1Result.(map[string]interface{})
	contact1ID := contact1Data["id"].(string)

	contact2Result, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name": "Bob",
	})
	contact2Data := contact2Result.(map[string]interface{})
	contact2ID := contact2Data["id"].(string)

	// Create relationship
	linkResult, _ := handler.LinkContacts_Legacy(map[string]interface{}{
		"contact_id_1": contact1ID,
		"contact_id_2": contact2ID,
	})
	linkData := linkResult.(map[string]interface{})
	relationshipID := linkData["id"].(string)

	// Remove relationship
	input := map[string]interface{}{
		"relationship_id": relationshipID,
	}

	result, err := handler.RemoveRelationship_Legacy(input)
	if err != nil {
		t.Fatalf("RemoveRelationship failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["success"] != true {
		t.Error("Expected success to be true")
	}

	// Verify relationship is gone
	findInput := map[string]interface{}{
		"contact_id": contact1ID,
	}

	findResult, _ := handler.FindContactRelationships_Legacy(findInput)
	relationships := findResult.([]map[string]interface{})

	if len(relationships) != 0 {
		t.Error("Relationship should have been deleted")
	}
}

func TestRemoveRelationshipValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)

	// Missing relationship_id
	input := map[string]interface{}{}

	_, err := handler.RemoveRelationship_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for missing relationship_id")
	}

	// Invalid UUID
	input = map[string]interface{}{
		"relationship_id": "not-a-uuid",
	}

	_, err = handler.RemoveRelationship_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for invalid relationship_id")
	}
}

// Tests for typed handlers (non-legacy).
func TestLinkContactsTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create two contacts
	_, c1, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Typed Alice"})
	_, c2, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Typed Bob"})

	// Link contacts
	input := LinkContactsInput{
		ContactID1:       c1.ID,
		ContactID2:       c2.ID,
		RelationshipType: "colleague",
		Context:          "Work together",
	}

	_, output, err := handler.LinkContacts(nil, nil, input)
	if err != nil {
		t.Fatalf("LinkContacts failed: %v", err)
	}

	if output.ID == "" {
		t.Error("ID was not set")
	}
	if output.RelationshipType != "colleague" {
		t.Errorf("Expected type 'colleague', got %s", output.RelationshipType)
	}
}

func TestLinkContactsTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)

	// Missing contact_id_1
	_, _, err := handler.LinkContacts(nil, nil, LinkContactsInput{
		ContactID2: uuid.New().String(),
	})
	if err == nil {
		t.Error("Expected error for missing contact_id_1")
	}

	// Missing contact_id_2
	_, _, err = handler.LinkContacts(nil, nil, LinkContactsInput{
		ContactID1: uuid.New().String(),
	})
	if err == nil {
		t.Error("Expected error for missing contact_id_2")
	}

	// Invalid contact_id_1
	_, _, err = handler.LinkContacts(nil, nil, LinkContactsInput{
		ContactID1: "invalid",
		ContactID2: uuid.New().String(),
	})
	if err == nil {
		t.Error("Expected error for invalid contact_id_1")
	}
}

func TestFindContactRelationshipsTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create contacts
	_, c1, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Find Alice"})
	_, c2, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Find Bob"})

	// Link contacts
	_, _, _ = handler.LinkContacts(nil, nil, LinkContactsInput{
		ContactID1:       c1.ID,
		ContactID2:       c2.ID,
		RelationshipType: "friend",
	})

	// Find relationships
	input := FindContactRelationshipsInput{
		ContactID: c1.ID,
	}

	_, output, err := handler.FindContactRelationships(nil, nil, input)
	if err != nil {
		t.Fatalf("FindContactRelationships failed: %v", err)
	}

	if len(output.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(output.Relationships))
	}
}

func TestFindContactRelationshipsTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)

	// Missing contact_id
	_, _, err := handler.FindContactRelationships(nil, nil, FindContactRelationshipsInput{})
	if err == nil {
		t.Error("Expected error for missing contact_id")
	}

	// Invalid contact_id
	_, _, err = handler.FindContactRelationships(nil, nil, FindContactRelationshipsInput{
		ContactID: "invalid",
	})
	if err == nil {
		t.Error("Expected error for invalid contact_id")
	}
}

func TestRemoveRelationshipTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create contacts and link them
	_, c1, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Remove Alice"})
	_, c2, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Remove Bob"})
	_, rel, _ := handler.LinkContacts(nil, nil, LinkContactsInput{
		ContactID1: c1.ID,
		ContactID2: c2.ID,
	})

	// Remove relationship
	input := RemoveRelationshipInput{
		RelationshipID: rel.ID,
	}

	_, output, err := handler.RemoveRelationship(nil, nil, input)
	if err != nil {
		t.Fatalf("RemoveRelationship failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success true")
	}
}

func TestRemoveRelationshipTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)

	// Missing relationship_id
	_, _, err := handler.RemoveRelationship(nil, nil, RemoveRelationshipInput{})
	if err == nil {
		t.Error("Expected error for missing relationship_id")
	}

	// Invalid relationship_id
	_, _, err = handler.RemoveRelationship(nil, nil, RemoveRelationshipInput{
		RelationshipID: "invalid",
	})
	if err == nil {
		t.Error("Expected error for invalid relationship_id")
	}
}

func TestUpdateRelationshipTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)
	contactHandler := NewContactHandlers(client)

	// Create contacts and link them
	_, c1, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Update Alice"})
	_, c2, _ := contactHandler.AddContact(nil, nil, AddContactInput{Name: "Update Bob"})
	_, rel, _ := handler.LinkContacts(nil, nil, LinkContactsInput{
		ContactID1:       c1.ID,
		ContactID2:       c2.ID,
		RelationshipType: "colleague",
	})

	// Update relationship
	input := UpdateRelationshipInput{
		RelationshipID:   rel.ID,
		RelationshipType: "friend",
		Context:          "Became friends",
	}

	_, output, err := handler.UpdateRelationship(nil, nil, input)
	if err != nil {
		t.Fatalf("UpdateRelationship failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success true")
	}
	if output.Message == "" {
		t.Error("Expected a message")
	}
}

func TestUpdateRelationshipTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewRelationshipHandlers(client)

	// Missing relationship_id
	_, _, err := handler.UpdateRelationship(nil, nil, UpdateRelationshipInput{
		RelationshipType: "friend",
	})
	if err == nil {
		t.Error("Expected error for missing relationship_id")
	}

	// Invalid relationship_id
	_, _, err = handler.UpdateRelationship(nil, nil, UpdateRelationshipInput{
		RelationshipID:   "invalid",
		RelationshipType: "friend",
	})
	if err == nil {
		t.Error("Expected error for invalid relationship_id")
	}
}
