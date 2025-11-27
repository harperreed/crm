// ABOUTME: Query tool test suite
// ABOUTME: Tests universal query_crm tool with filtering across all entity types
package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupQueryTestDB(t *testing.T) (*sql.DB, func()) {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	if err := db.InitSchema(database); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	cleanup := func() {
		_ = database.Close()
	}

	return database, cleanup
}

func TestQueryCRMContacts(t *testing.T) {
	database, cleanup := setupQueryTestDB(t)
	defer cleanup()

	// Create test company and contacts
	company := &models.Company{Name: "Test Corp"}
	if err := db.CreateCompany(database, company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	contact1 := &models.Contact{
		Name:      "Alice Smith",
		Email:     "alice@example.com",
		CompanyID: &company.ID,
	}
	if err := db.CreateContact(database, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{
		Name:  "Bob Jones",
		Email: "bob@example.com",
	}
	if err := db.CreateContact(database, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	handlers := NewQueryHandlers(database)

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
	database, cleanup := setupQueryTestDB(t)
	defer cleanup()

	// Create test companies
	company1 := &models.Company{
		Name:   "Alpha Corp",
		Domain: "alpha.com",
	}
	if err := db.CreateCompany(database, company1); err != nil {
		t.Fatalf("Failed to create company1: %v", err)
	}

	company2 := &models.Company{
		Name:   "Beta Inc",
		Domain: "beta.com",
	}
	if err := db.CreateCompany(database, company2); err != nil {
		t.Fatalf("Failed to create company2: %v", err)
	}

	handlers := NewQueryHandlers(database)

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
	database, cleanup := setupQueryTestDB(t)
	defer cleanup()

	// Create test company and deals
	company := &models.Company{Name: "Deal Corp"}
	if err := db.CreateCompany(database, company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	deal1 := &models.Deal{
		Title:     "Big Deal",
		Amount:    100000,
		Currency:  "USD",
		Stage:     models.StageProspecting,
		CompanyID: company.ID,
	}
	if err := db.CreateDeal(database, deal1); err != nil {
		t.Fatalf("Failed to create deal1: %v", err)
	}

	deal2 := &models.Deal{
		Title:     "Small Deal",
		Amount:    5000,
		Currency:  "USD",
		Stage:     models.StageNegotiation,
		CompanyID: company.ID,
	}
	if err := db.CreateDeal(database, deal2); err != nil {
		t.Fatalf("Failed to create deal2: %v", err)
	}

	handlers := NewQueryHandlers(database)

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
				"stage": models.StageProspecting,
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
	database, cleanup := setupQueryTestDB(t)
	defer cleanup()

	// Create test contacts
	contact1 := &models.Contact{Name: "Alice"}
	if err := db.CreateContact(database, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob"}
	if err := db.CreateContact(database, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	// Create relationship
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
		Context:          "Work together at XYZ",
	}
	if err := db.CreateRelationship(database, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	handlers := NewQueryHandlers(database)

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
	database, cleanup := setupQueryTestDB(t)
	defer cleanup()

	handlers := NewQueryHandlers(database)

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

// Helper function to check if string contains substring
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
