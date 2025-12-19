// ABOUTME: Query tool test suite
// ABOUTME: Tests universal query_crm tool with filtering across all entity types
package handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestQueryCRMContacts(t *testing.T) {
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	// Create test company and contacts
	company := &charm.Company{ID: uuid.New(), Name: "Test Corp"}
	if err := client.CreateCompany(company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	contact1 := &charm.Contact{
		ID:          uuid.New(),
		Name:        "Alice Smith",
		Email:       "alice@example.com",
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	}
	if err := client.CreateContact(contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &charm.Contact{
		ID:    uuid.New(),
		Name:  "Bob Jones",
		Email: "bob@example.com",
	}
	if err := client.CreateContact(contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	handlers := NewQueryHandlers(client)

	// Test: Query all contacts
	t.Run("QueryAllContacts", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "contact",
			Limit:      10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.EntityType != "contact" {
			t.Errorf("Expected entity_type 'contact', got %s", output.EntityType)
		}

		if output.Count != 2 {
			t.Errorf("Expected 2 contacts, got %d", output.Count)
		}

		if len(output.Results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(output.Results))
		}
	})

	// Test: Query contacts by name
	t.Run("QueryContactsByName", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "contact",
			Query:      "Alice",
			Limit:      10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 contact, got %d", output.Count)
		}
	})

	// Test: Query contacts by company_id
	t.Run("QueryContactsByCompanyID", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "contact",
			Filters: map[string]interface{}{
				"company_id": company.ID.String(),
			},
			Limit: 10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 contact with company_id, got %d", output.Count)
		}
	})
}

func TestQueryCRMCompanies(t *testing.T) {
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	// Create test companies
	company1 := &charm.Company{
		ID:     uuid.New(),
		Name:   "Alpha Corp",
		Domain: "alpha.com",
	}
	if err := client.CreateCompany(company1); err != nil {
		t.Fatalf("Failed to create company1: %v", err)
	}

	company2 := &charm.Company{
		ID:     uuid.New(),
		Name:   "Beta Inc",
		Domain: "beta.com",
	}
	if err := client.CreateCompany(company2); err != nil {
		t.Fatalf("Failed to create company2: %v", err)
	}

	handlers := NewQueryHandlers(client)

	// Test: Query all companies
	t.Run("QueryAllCompanies", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "company",
			Query:      "", // Empty query returns all
			Limit:      10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.EntityType != "company" {
			t.Errorf("Expected entity_type 'company', got %s", output.EntityType)
		}

		if output.Count != 2 {
			t.Errorf("Expected 2 companies, got %d", output.Count)
		}
	})

	// Test: Query companies by name
	t.Run("QueryCompaniesByName", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "company",
			Query:      "Alpha",
			Limit:      10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 company, got %d", output.Count)
		}
	})
}

func TestQueryCRMDeals(t *testing.T) {
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	// Create test company and deals
	company := &charm.Company{ID: uuid.New(), Name: "Deal Corp"}
	if err := client.CreateCompany(company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	deal1 := &charm.Deal{
		ID:          uuid.New(),
		Title:       "Big Deal",
		Amount:      100000,
		Currency:    "USD",
		Stage:       charm.StageProspecting,
		CompanyID:   company.ID,
		CompanyName: company.Name,
	}
	if err := client.CreateDeal(deal1); err != nil {
		t.Fatalf("Failed to create deal1: %v", err)
	}

	deal2 := &charm.Deal{
		ID:          uuid.New(),
		Title:       "Small Deal",
		Amount:      5000,
		Currency:    "USD",
		Stage:       charm.StageNegotiation,
		CompanyID:   company.ID,
		CompanyName: company.Name,
	}
	if err := client.CreateDeal(deal2); err != nil {
		t.Fatalf("Failed to create deal2: %v", err)
	}

	handlers := NewQueryHandlers(client)

	// Test: Query all deals
	t.Run("QueryAllDeals", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "deal",
			Limit:      10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.EntityType != "deal" {
			t.Errorf("Expected entity_type 'deal', got %s", output.EntityType)
		}

		if output.Count != 2 {
			t.Errorf("Expected 2 deals, got %d", output.Count)
		}
	})

	// Test: Query deals by stage
	t.Run("QueryDealsByStage", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "deal",
			Filters: map[string]interface{}{
				"stage": charm.StageProspecting,
			},
			Limit: 10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 deal with stage prospecting, got %d", output.Count)
		}
	})

	// Test: Query deals with min/max amount filtering
	t.Run("QueryDealsByAmountRange", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "deal",
			Filters: map[string]interface{}{
				"min_amount": float64(10000),
				"max_amount": float64(200000),
			},
			Limit: 10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 deal in amount range, got %d", output.Count)
		}
	})

	// Test: Query deals by company_id
	t.Run("QueryDealsByCompanyID", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "deal",
			Filters: map[string]interface{}{
				"company_id": company.ID.String(),
			},
			Limit: 10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 2 {
			t.Errorf("Expected 2 deals with company_id, got %d", output.Count)
		}
	})
}

func TestQueryCRMRelationships(t *testing.T) {
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	// Create test contacts
	contact1 := &charm.Contact{ID: uuid.New(), Name: "Alice"}
	if err := client.CreateContact(contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &charm.Contact{ID: uuid.New(), Name: "Bob"}
	if err := client.CreateContact(contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	// Create relationship
	rel := &charm.Relationship{
		ID:               uuid.New(),
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
		Context:          "Work together at XYZ",
	}
	if err := client.CreateRelationship(rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	handlers := NewQueryHandlers(client)

	// Test: Query relationships by contact_id
	t.Run("QueryRelationshipsByContactID", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "relationship",
			Filters: map[string]interface{}{
				"contact_id": contact1.ID.String(),
			},
			Limit: 10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.EntityType != "relationship" {
			t.Errorf("Expected entity_type 'relationship', got %s", output.EntityType)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 relationship, got %d", output.Count)
		}
	})

	// Test: Query relationships by type
	t.Run("QueryRelationshipsByType", func(t *testing.T) {
		input := QueryCRMInput{
			EntityType: "relationship",
			Filters: map[string]interface{}{
				"contact_id":        contact1.ID.String(),
				"relationship_type": "colleague",
			},
			Limit: 10,
		}

		_, output, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("QueryCRM failed: %v", err)
		}

		if output.Count != 1 {
			t.Errorf("Expected 1 colleague relationship, got %d", output.Count)
		}
	})
}

func TestQueryCRMInvalidEntityType(t *testing.T) {
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	handlers := NewQueryHandlers(client)

	input := QueryCRMInput{
		EntityType: "invalid_type",
		Limit:      10,
	}

	_, _, err := handlers.QueryCRM(context.Background(), &mcp.CallToolRequest{}, input)
	if err == nil {
		t.Fatal("Expected error for invalid entity_type, got nil")
	}

	expectedError := "invalid entity_type"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
