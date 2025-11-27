// ABOUTME: Tests for deal MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
)

func TestCreateDeal(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Test valid deal creation with company_name lookup/create
	input := map[string]interface{}{
		"title":        "Enterprise License Deal",
		"amount":       float64(50000),
		"currency":     "USD",
		"stage":        "prospecting",
		"company_name": "Acme Corp",
	}

	result, err := handler.CreateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["title"] != "Enterprise License Deal" {
		t.Errorf("Expected title 'Enterprise License Deal', got %v", data["title"])
	}

	if data["amount"] != int64(50000) {
		t.Errorf("Expected amount 50000, got %v", data["amount"])
	}

	if data["company_id"] == nil {
		t.Error("Company ID was not set")
	}

	if data["id"] == nil {
		t.Error("ID was not set")
	}
}

func TestCreateDealWithContactName(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create a contact first
	contactHandler := NewContactHandlers(database)
	_, _ = contactHandler.AddContact_Legacy(map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	})

	// Create deal with contact name lookup
	input := map[string]interface{}{
		"title":        "Consulting Deal",
		"amount":       float64(25000),
		"company_name": "Test Corp",
		"contact_name": "John Doe",
	}

	result, err := handler.CreateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["contact_id"] == nil {
		t.Error("Contact ID was not set")
	}
}

func TestCreateDealWithoutContact(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal without contact
	input := map[string]interface{}{
		"title":        "Direct Deal",
		"amount":       float64(10000),
		"company_name": "Solo Corp",
	}

	result, err := handler.CreateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["contact_id"] != nil {
		t.Error("Contact ID should be nil")
	}
}

func TestCreateDealWithInitialNote(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal with initial note
	input := map[string]interface{}{
		"title":        "New Opportunity",
		"amount":       float64(75000),
		"company_name": "Big Corp",
		"initial_note": "Initial contact via LinkedIn",
	}

	result, err := handler.CreateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Verify note was created
	dealID, _ := uuid.Parse(data["id"].(string))
	notes, err := db.GetDealNotes(database, dealID)
	if err != nil {
		t.Fatalf("Failed to get deal notes: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}

	if notes[0].Content != "Initial contact via LinkedIn" {
		t.Errorf("Expected note content 'Initial contact via LinkedIn', got %s", notes[0].Content)
	}
}

func TestCreateDealDefaults(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal with minimal input (defaults should be applied)
	input := map[string]interface{}{
		"title":        "Minimal Deal",
		"company_name": "Default Corp",
	}

	result, err := handler.CreateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["currency"] != "USD" {
		t.Errorf("Expected default currency 'USD', got %v", data["currency"])
	}

	if data["stage"] != "prospecting" {
		t.Errorf("Expected default stage 'prospecting', got %v", data["stage"])
	}
}

func TestCreateDealValidation(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Missing required title
	input := map[string]interface{}{
		"company_name": "Test Corp",
	}

	_, err := handler.CreateDeal_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for missing title")
	}

	// Missing required company_name
	input2 := map[string]interface{}{
		"title": "Test Deal",
	}

	_, err = handler.CreateDeal_Legacy(input2)
	if err == nil {
		t.Error("Expected validation error for missing company_name")
	}
}

func TestUpdateDeal(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal first
	createResult, _ := handler.CreateDeal_Legacy(map[string]interface{}{
		"title":        "Original Deal",
		"amount":       float64(10000),
		"company_name": "Test Corp",
	})
	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Wait a moment to ensure timestamps differ
	time.Sleep(10 * time.Millisecond)

	// Update deal amount
	input := map[string]interface{}{
		"id":     dealID,
		"amount": float64(15000),
	}

	result, err := handler.UpdateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("UpdateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["amount"] != int64(15000) {
		t.Errorf("Expected amount 15000, got %v", data["amount"])
	}

	// Verify last_activity_at was updated
	originalLastActivity := dealData["last_activity_at"].(time.Time)
	updatedLastActivity := data["last_activity_at"].(time.Time)

	if !updatedLastActivity.After(originalLastActivity) {
		t.Error("last_activity_at should be updated")
	}
}

func TestUpdateDealStage(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal first
	createResult, _ := handler.CreateDeal_Legacy(map[string]interface{}{
		"title":        "Pipeline Deal",
		"company_name": "Test Corp",
	})
	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Update to different stage
	input := map[string]interface{}{
		"id":    dealID,
		"stage": "negotiation",
	}

	result, err := handler.UpdateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("UpdateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["stage"] != "negotiation" {
		t.Errorf("Expected stage 'negotiation', got %v", data["stage"])
	}
}

func TestUpdateDealExpectedCloseDate(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal first
	createResult, _ := handler.CreateDeal_Legacy(map[string]interface{}{
		"title":        "Dated Deal",
		"company_name": "Test Corp",
	})
	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Update expected close date
	expectedDate := "2024-12-31T00:00:00Z"
	input := map[string]interface{}{
		"id":                  dealID,
		"expected_close_date": expectedDate,
	}

	result, err := handler.UpdateDeal_Legacy(input)
	if err != nil {
		t.Fatalf("UpdateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["expected_close_date"] == nil {
		t.Error("Expected close date was not set")
	}
}

func TestUpdateDealNotFound(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	input := map[string]interface{}{
		"id":     uuid.New().String(),
		"amount": float64(10000),
	}

	_, err := handler.UpdateDeal_Legacy(input)
	if err == nil {
		t.Error("Expected error for non-existent deal")
	}
}

func TestUpdateDealInvalidStage(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal first
	createResult, _ := handler.CreateDeal_Legacy(map[string]interface{}{
		"title":        "Test Deal",
		"company_name": "Test Corp",
	})
	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Try to update with invalid stage
	input := map[string]interface{}{
		"id":    dealID,
		"stage": "invalid_stage",
	}

	_, err := handler.UpdateDeal_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for invalid stage")
	}
}

func TestAddDealNote(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create deal first
	createResult, _ := handler.CreateDeal_Legacy(map[string]interface{}{
		"title":        "Note Deal",
		"company_name": "Test Corp",
	})
	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Wait to ensure timestamp differs
	time.Sleep(10 * time.Millisecond)

	// Add note
	input := map[string]interface{}{
		"deal_id": dealID,
		"content": "Had a productive call with the client",
	}

	result, err := handler.AddDealNote_Legacy(input)
	if err != nil {
		t.Fatalf("AddDealNote failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["content"] != "Had a productive call with the client" {
		t.Errorf("Expected note content, got %v", data["content"])
	}

	// Verify deal's last_activity_at was updated
	dealUUID, _ := uuid.Parse(dealID)
	deal, _ := db.GetDeal(database, dealUUID)

	originalLastActivity := dealData["last_activity_at"].(time.Time)
	if !deal.LastActivityAt.After(originalLastActivity) {
		t.Error("Deal's last_activity_at should be updated after adding note")
	}
}

func TestAddDealNoteUpdatesContactLastContactedAt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	// Create contact first
	contactHandler := NewContactHandlers(database)
	contactResult, _ := contactHandler.AddContact_Legacy(map[string]interface{}{
		"name":  "Jane Smith",
		"email": "jane@example.com",
	})
	contactData := contactResult.(map[string]interface{})
	contactID := contactData["id"].(string)

	// Create deal with contact
	createResult, _ := handler.CreateDeal_Legacy(map[string]interface{}{
		"title":        "Contact Deal",
		"company_name": "Test Corp",
		"contact_name": "Jane Smith",
	})
	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Wait to ensure timestamp differs
	time.Sleep(10 * time.Millisecond)

	// Add note
	input := map[string]interface{}{
		"deal_id": dealID,
		"content": "Follow-up discussion",
	}

	_, err := handler.AddDealNote_Legacy(input)
	if err != nil {
		t.Fatalf("AddDealNote failed: %v", err)
	}

	// Verify contact's last_contacted_at was updated
	contactUUID, _ := uuid.Parse(contactID)
	contact, _ := db.GetContact(database, contactUUID)

	if contact.LastContactedAt == nil {
		t.Error("Contact's last_contacted_at should be set after adding deal note")
	}
}

func TestAddDealNoteNotFound(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	handler := NewDealHandlers(database)

	input := map[string]interface{}{
		"deal_id": uuid.New().String(),
		"content": "Test note",
	}

	_, err := handler.AddDealNote_Legacy(input)
	if err == nil {
		t.Error("Expected error for non-existent deal")
	}
}
