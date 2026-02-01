// ABOUTME: Tests for contact MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
)

func TestAddContactHandler(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// First create a company
	companyHandler := NewCompanyHandlers(client)
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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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

	// Verify company was created by searching for it
	companies, err := client.ListCompanies(&repository.CompanyFilter{
		Query: "New Corp",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("Failed to find company: %v", err)
	}
	if len(companies) == 0 {
		t.Error("Company was not created")
	}
}

func TestAddContactValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create company and contact
	companyHandler := NewCompanyHandlers(client)
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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	input := map[string]interface{}{
		"contact_id": uuid.New().String(),
		"note":       "Test note",
	}

	_, err := handler.LogContactInteraction_Legacy(input)
	if err == nil {
		t.Error("Expected error for non-existent contact")
	}
}

// Tests for the new typed handlers (non-legacy).
func TestAddContactTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Test valid contact creation
	input := AddContactInput{
		Name:  "John Typed",
		Email: "john.typed@example.com",
		Phone: "555-1234",
		Notes: "Test typed contact",
	}

	_, output, err := handler.AddContact(nil, nil, input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	if output.Name != "John Typed" {
		t.Errorf("Expected name 'John Typed', got %v", output.Name)
	}
	if output.Email != "john.typed@example.com" {
		t.Errorf("Expected email 'john.typed@example.com', got %v", output.Email)
	}
	if output.ID == "" {
		t.Error("ID was not set")
	}
}

func TestAddContactTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Missing required name
	input := AddContactInput{
		Email: "test@example.com",
	}

	_, _, err := handler.AddContact(nil, nil, input)
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestAddContactTypedWithCompany(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Add contact with new company (should create it)
	input := AddContactInput{
		Name:        "Company Contact",
		Email:       "cc@newcorp.com",
		CompanyName: "New Corp Typed",
	}

	_, output, err := handler.AddContact(nil, nil, input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	if output.CompanyID == nil {
		t.Error("Company ID was not set")
	}
}

func TestFindContactsTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Add test contacts
	_, _, _ = handler.AddContact(nil, nil, AddContactInput{Name: "Alice Typed", Email: "alice@typed.com"})
	_, _, _ = handler.AddContact(nil, nil, AddContactInput{Name: "Bob Typed", Email: "bob@typed.com"})

	// Search by name
	input := FindContactsInput{
		Query: "typed",
		Limit: 10,
	}

	_, output, err := handler.FindContacts(nil, nil, input)
	if err != nil {
		t.Fatalf("FindContacts failed: %v", err)
	}

	if len(output.Contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(output.Contacts))
	}
}

func TestFindContactsTypedWithCompanyID(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create contact with company
	_, output, _ := handler.AddContact(nil, nil, AddContactInput{
		Name:        "Company Contact",
		CompanyName: "Test Corp Typed",
	})
	companyID := output.CompanyID

	// Search by company_id
	input := FindContactsInput{
		CompanyID: *companyID,
	}

	_, result, err := handler.FindContacts(nil, nil, input)
	if err != nil {
		t.Fatalf("FindContacts failed: %v", err)
	}

	if len(result.Contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result.Contacts))
	}
}

func TestFindContactsTypedInvalidCompanyID(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	input := FindContactsInput{
		CompanyID: "invalid-uuid",
	}

	_, _, err := handler.FindContacts(nil, nil, input)
	if err == nil {
		t.Error("Expected error for invalid company_id")
	}
}

func TestUpdateContactTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create contact
	_, created, _ := handler.AddContact(nil, nil, AddContactInput{
		Name:  "Original Typed",
		Email: "original@typed.com",
	})

	// Update contact
	input := UpdateContactInput{
		ID:    created.ID,
		Name:  "Updated Typed",
		Email: "updated@typed.com",
		Phone: "555-9999",
		Notes: "Updated notes",
	}

	_, output, err := handler.UpdateContact(nil, nil, input)
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}

	if output.Name != "Updated Typed" {
		t.Errorf("Expected name 'Updated Typed', got %v", output.Name)
	}
	if output.Email != "updated@typed.com" {
		t.Errorf("Expected email 'updated@typed.com', got %v", output.Email)
	}
}

func TestUpdateContactTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Missing ID
	input := UpdateContactInput{
		Name: "Test",
	}

	_, _, err := handler.UpdateContact(nil, nil, input)
	if err == nil {
		t.Error("Expected error for missing ID")
	}
}

func TestUpdateContactTypedInvalidID(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	input := UpdateContactInput{
		ID:   "invalid-uuid",
		Name: "Test",
	}

	_, _, err := handler.UpdateContact(nil, nil, input)
	if err == nil {
		t.Error("Expected error for invalid ID")
	}
}

func TestLogContactInteractionTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create contact
	_, created, _ := handler.AddContact(nil, nil, AddContactInput{
		Name: "Interaction Contact",
	})

	// Log interaction
	input := LogContactInteractionInput{
		ContactID: created.ID,
		Note:      "Had a typed call",
	}

	_, output, err := handler.LogContactInteraction(nil, nil, input)
	if err != nil {
		t.Fatalf("LogContactInteraction failed: %v", err)
	}

	if output.LastContactedAt == nil {
		t.Error("LastContactedAt was not set")
	}
}

func TestLogContactInteractionTypedWithDate(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create contact
	_, created, _ := handler.AddContact(nil, nil, AddContactInput{
		Name: "Date Contact",
	})

	// Log interaction with custom date
	input := LogContactInteractionInput{
		ContactID:       created.ID,
		Note:            "Past interaction",
		InteractionDate: "2024-06-15T14:00:00Z",
	}

	_, output, err := handler.LogContactInteraction(nil, nil, input)
	if err != nil {
		t.Fatalf("LogContactInteraction failed: %v", err)
	}

	if output.LastContactedAt == nil {
		t.Error("LastContactedAt was not set")
	}
}

func TestLogContactInteractionTypedInvalidDate(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create contact
	_, created, _ := handler.AddContact(nil, nil, AddContactInput{
		Name: "Invalid Date Contact",
	})

	// Log interaction with invalid date
	input := LogContactInteractionInput{
		ContactID:       created.ID,
		InteractionDate: "not-a-date",
	}

	_, _, err := handler.LogContactInteraction(nil, nil, input)
	if err == nil {
		t.Error("Expected error for invalid date format")
	}
}

func TestLogContactInteractionTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Missing contact_id
	input := LogContactInteractionInput{
		Note: "Test note",
	}

	_, _, err := handler.LogContactInteraction(nil, nil, input)
	if err == nil {
		t.Error("Expected error for missing contact_id")
	}
}

func TestDeleteContactTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Create contact
	_, created, _ := handler.AddContact(nil, nil, AddContactInput{
		Name: "To Delete",
	})

	// Delete contact
	input := DeleteContactInput{
		ID: created.ID,
	}

	_, output, err := handler.DeleteContact(nil, nil, input)
	if err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	if !output.Success {
		t.Error("Delete should succeed")
	}
}

func TestDeleteContactTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	// Missing ID
	input := DeleteContactInput{}

	_, _, err := handler.DeleteContact(nil, nil, input)
	if err == nil {
		t.Error("Expected error for missing ID")
	}
}

func TestDeleteContactTypedInvalidID(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewContactHandlers(client)

	input := DeleteContactInput{
		ID: "invalid-uuid",
	}

	_, _, err := handler.DeleteContact(nil, nil, input)
	if err == nil {
		t.Error("Expected error for invalid ID")
	}
}
