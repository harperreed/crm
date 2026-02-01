// ABOUTME: Tests for MCP prompt handlers
// ABOUTME: Validates prompt generation for contacts, deals, relationships, and company overviews
package handlers

import (
	"context"
	"testing"

	"github.com/harperreed/pagen/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetPromptContactSummary(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	contact := &repository.Contact{Name: "John Doe", Email: "john@example.com", CompanyID: &company.ID, CompanyName: company.Name, Notes: "Test notes"}
	_ = db.CreateContact(contact)

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "contact-summary",
			Arguments: map[string]string{"contact_id": contact.ID.String()},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Messages) == 0 {
		t.Error("Expected at least one message")
	}
}

func TestGetPromptContactSummaryMissingID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "contact-summary",
			Arguments: map[string]string{},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for missing contact_id")
	}
}

func TestGetPromptContactSummaryInvalidID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "contact-summary",
			Arguments: map[string]string{"contact_id": "invalid-uuid"},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for invalid contact_id")
	}
}

func TestGetPromptDealAnalysis(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&repository.Deal{CompanyID: company.ID, CompanyName: company.Name, Title: "Deal 1", Amount: 100000, Currency: "USD", Stage: "prospecting"})
	_ = db.CreateDeal(&repository.Deal{CompanyID: company.ID, CompanyName: company.Name, Title: "Deal 2", Amount: 200000, Currency: "USD", Stage: "qualification"})

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "deal-analysis",
			Arguments: map[string]string{},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Description != "Deal pipeline analysis" {
		t.Errorf("Expected description 'Deal pipeline analysis', got %s", result.Description)
	}
}

func TestGetPromptRelationshipMapContact(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	contact1 := &repository.Contact{Name: "Alice"}
	_ = db.CreateContact(contact1)

	contact2 := &repository.Contact{Name: "Bob"}
	_ = db.CreateContact(contact2)

	_ = db.CreateRelationship(&repository.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague"})

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "relationship-map",
			Arguments: map[string]string{"entity_type": "contact", "entity_id": contact1.ID.String()},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetPromptRelationshipMapCompany(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateContact(&repository.Contact{Name: "Alice", Email: "alice@test.com", CompanyID: &company.ID, CompanyName: company.Name})
	_ = db.CreateContact(&repository.Contact{Name: "Bob", Email: "bob@test.com", CompanyID: &company.ID, CompanyName: company.Name})

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "relationship-map",
			Arguments: map[string]string{"entity_type": "company", "entity_id": company.ID.String()},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetPromptRelationshipMapMissingEntityType(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "relationship-map",
			Arguments: map[string]string{"entity_id": "00000000-0000-0000-0000-000000000000"},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for missing entity_type")
	}
}

func TestGetPromptRelationshipMapMissingEntityID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "relationship-map",
			Arguments: map[string]string{"entity_type": "contact"},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for missing entity_id")
	}
}

func TestGetPromptRelationshipMapInvalidEntityType(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "relationship-map",
			Arguments: map[string]string{"entity_type": "invalid", "entity_id": "00000000-0000-0000-0000-000000000000"},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for invalid entity_type")
	}
}

func TestGetPromptFollowUpSuggestions(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contacts with different last_contacted_at values
	_ = db.CreateContact(&repository.Contact{Name: "Never Contacted"})
	_ = db.CreateContact(&repository.Contact{Name: "Recent Contact"})

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "follow-up-suggestions",
			Arguments: map[string]string{},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetPromptFollowUpSuggestionsWithDaysParam(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	_ = db.CreateContact(&repository.Contact{Name: "Test Contact"})

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "follow-up-suggestions",
			Arguments: map[string]string{"days_since_contact": "14"},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetPromptCompanyOverview(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp", Domain: "test.com", Industry: "Technology", Notes: "Important company"}
	_ = db.CreateCompany(company)

	_ = db.CreateContact(&repository.Contact{Name: "Alice", Email: "alice@test.com", CompanyID: &company.ID, CompanyName: company.Name})
	_ = db.CreateDeal(&repository.Deal{CompanyID: company.ID, CompanyName: company.Name, Title: "Big Deal", Amount: 100000, Currency: "USD", Stage: "prospecting"})

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "company-overview",
			Arguments: map[string]string{"company_id": company.ID.String()},
		},
	}

	result, err := handlers.GetPrompt(context.Background(), request)
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetPromptCompanyOverviewMissingID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "company-overview",
			Arguments: map[string]string{},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for missing company_id")
	}
}

func TestGetPromptUnknownPrompt(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewPromptHandlers(db)

	request := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "unknown-prompt",
			Arguments: map[string]string{},
		},
	}

	_, err := handlers.GetPrompt(context.Background(), request)
	if err == nil {
		t.Error("Expected error for unknown prompt")
	}
}
