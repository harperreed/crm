// ABOUTME: Tests for contact MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/db"
)

func TestAddContactHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Test valid contact creation
	input := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"phone": "555-1234",
		"notes": "Test contact",
	}

	result, err := handler.AddContact_Legacy(input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", data["name"])
	}

	if data["email"] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %v", data["email"])
	}

	if data["id"] == nil {
		t.Error("ID was not set")
	}
}

func TestAddContactWithCompanyName(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// First create a company
	companyHandler := NewCompanyHandlers(database)
	_, _ = companyHandler.AddCompany_Legacy(map[string]interface{}{
		"name": "Acme Corp",
	})

	// Add contact with existing company
	input := map[string]interface{}{
		"name":         "Jane Smith",
		"email":        "jane@acme.com",
		"company_name": "Acme Corp",
	}

	result, err := handler.AddContact_Legacy(input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["company_id"] == nil {
		t.Error("Company ID was not set")
	}
}

func TestAddContactCreatesNewCompany(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Add contact with non-existent company (should create it)
	input := map[string]interface{}{
		"name":         "Bob Jones",
		"email":        "bob@newcorp.com",
		"company_name": "New Corp",
	}

	result, err := handler.AddContact_Legacy(input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["company_id"] == nil {
		t.Error("Company ID was not set")
	}

	// Verify company was created
	company, err := db.FindCompanyByName(database, "New Corp")
	if err != nil {
		t.Fatalf("Failed to find company: %v", err)
	}
	if company == nil {
		t.Error("Company was not created")
	}
}

func TestAddContactValidation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Missing required name
	input := map[string]interface{}{
		"email": "test@example.com",
	}

	_, err := handler.AddContact_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestFindContactsHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Add test contacts
	_, _ = handler.AddContact_Legacy(map[string]interface{}{"name": "Alice Smith", "email": "alice@example.com"})
	_, _ = handler.AddContact_Legacy(map[string]interface{}{"name": "Bob Jones", "email": "bob@test.com"})

	// Search by name
	input := map[string]interface{}{
		"query": "smith",
		"limit": float64(10),
	}

	result, err := handler.FindContacts_Legacy(input)
	if err != nil {
		t.Fatalf("FindContacts failed: %v", err)
	}

	contacts, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(contacts) == 0 {
		t.Error("Expected to find contacts")
	}
}

func TestFindContactsByEmail(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	_, _ = handler.AddContact_Legacy(map[string]interface{}{"name": "Test User", "email": "unique@example.com"})

	input := map[string]interface{}{
		"query": "unique@example.com",
	}

	result, err := handler.FindContacts_Legacy(input)
	if err != nil {
		t.Fatalf("FindContacts failed: %v", err)
	}

	contacts, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(contacts))
	}
}

func TestFindContactsByCompanyID(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Create company and contact
	companyHandler := NewCompanyHandlers(database)
	companyResult, _ := companyHandler.AddCompany_Legacy(map[string]interface{}{"name": "Test Corp"})
	companyData := companyResult.(map[string]interface{})
	companyID := companyData["id"].(string)

	_, _ = handler.AddContact_Legacy(map[string]interface{}{
		"name":         "Company Contact",
		"company_name": "Test Corp",
	})

	input := map[string]interface{}{
		"company_id": companyID,
	}

	result, err := handler.FindContacts_Legacy(input)
	if err != nil {
		t.Fatalf("FindContacts failed: %v", err)
	}

	contacts, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(contacts))
	}
}

func TestUpdateContactHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Create contact
	createResult, _ := handler.AddContact_Legacy(map[string]interface{}{
		"name":  "Original Name",
		"email": "original@example.com",
	})
	contactData := createResult.(map[string]interface{})
	contactID := contactData["id"].(string)

	// Update contact
	input := map[string]interface{}{
		"id":    contactID,
		"name":  "Updated Name",
		"email": "updated@example.com",
		"phone": "555-9999",
	}

	result, err := handler.UpdateContact_Legacy(input)
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["name"] != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %v", data["name"])
	}

	if data["email"] != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got %v", data["email"])
	}

	if data["phone"] != "555-9999" {
		t.Errorf("Expected phone '555-9999', got %v", data["phone"])
	}
}

func TestUpdateContactNotFound(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	input := map[string]interface{}{
		"id":   uuid.New().String(),
		"name": "Updated Name",
	}

	_, err := handler.UpdateContact_Legacy(input)
	if err == nil {
		t.Error("Expected error for non-existent contact")
	}
}

func TestLogContactInteractionHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Create contact
	createResult, _ := handler.AddContact_Legacy(map[string]interface{}{
		"name":  "Test Contact",
		"email": "test@example.com",
	})
	contactData := createResult.(map[string]interface{})
	contactID := contactData["id"].(string)

	// Log interaction with default timestamp
	input := map[string]interface{}{
		"contact_id": contactID,
		"note":       "Had a great call",
	}

	result, err := handler.LogContactInteraction_Legacy(input)
	if err != nil {
		t.Fatalf("LogContactInteraction failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["last_contacted_at"] == nil {
		t.Error("Last contacted at was not set")
	}

	// Verify note was appended
	notes := data["notes"].(string)
	if notes == "" {
		t.Error("Notes should contain the interaction note")
	}
}

func TestLogContactInteractionWithCustomDate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Create contact
	createResult, _ := handler.AddContact_Legacy(map[string]interface{}{
		"name": "Test Contact",
	})
	contactData := createResult.(map[string]interface{})
	contactID := contactData["id"].(string)

	// Log interaction with custom timestamp
	customDate := "2024-01-15T10:00:00Z"
	input := map[string]interface{}{
		"contact_id":       contactID,
		"note":             "Past interaction",
		"interaction_date": customDate,
	}

	result, err := handler.LogContactInteraction_Legacy(input)
	if err != nil {
		t.Fatalf("LogContactInteraction failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["last_contacted_at"] == nil {
		t.Error("Last contacted at was not set")
	}

	// Parse and verify the timestamp
	lastContacted, ok := data["last_contacted_at"].(time.Time)
	if !ok {
		t.Error("Last contacted at is not a time.Time")
	}

	expectedTime, _ := time.Parse(time.RFC3339, customDate)
	if !lastContacted.Equal(expectedTime) {
		t.Errorf("Expected timestamp %v, got %v", expectedTime, lastContacted)
	}
}

func TestLogContactInteractionNotFound(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	input := map[string]interface{}{
		"contact_id": uuid.New().String(),
		"note":       "Test note",
	}

	_, err := handler.LogContactInteraction_Legacy(input)
	if err == nil {
		t.Error("Expected error for non-existent contact")
	}
}
