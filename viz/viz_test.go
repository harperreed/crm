// ABOUTME: Tests for the viz package graph generation
// ABOUTME: Validates DOT graph output for contacts, companies, and pipelines
package viz

import (
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestGenerateContactGraph(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp", Domain: "test.com", Industry: "Tech"}
	if err := db.CreateCompany(company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	contact1 := &repository.Contact{Name: "Alice", Email: "alice@test.com", CompanyID: &company.ID, CompanyName: company.Name}
	if err := db.CreateContact(contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &repository.Contact{Name: "Bob", Email: "bob@test.com", CompanyID: &company.ID, CompanyName: company.Name}
	if err := db.CreateContact(contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	rel := &repository.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague", Context: "work together"}
	if err := db.CreateRelationship(rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	generator := NewGraphGenerator(db)

	// Test with no specific contact (all contacts)
	graph, err := generator.GenerateContactGraph(nil)
	if err != nil {
		t.Fatalf("GenerateContactGraph failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}

	// Test with specific contact
	graph, err = generator.GenerateContactGraph(&contact1.ID)
	if err != nil {
		t.Fatalf("GenerateContactGraph with contactID failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}
}

func TestGenerateCompanyGraph(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp", Domain: "test.com", Industry: "Tech"}
	if err := db.CreateCompany(company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	contact1 := &repository.Contact{Name: "Alice", Email: "alice@test.com", CompanyID: &company.ID, CompanyName: company.Name}
	if err := db.CreateContact(contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &repository.Contact{Name: "Bob", Email: "bob@test.com", CompanyID: &company.ID, CompanyName: company.Name}
	if err := db.CreateContact(contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	rel := &repository.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague", Context: "work together"}
	if err := db.CreateRelationship(rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	generator := NewGraphGenerator(db)

	graph, err := generator.GenerateCompanyGraph(company.ID)
	if err != nil {
		t.Fatalf("GenerateCompanyGraph failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}
}

func TestGeneratePipelineGraph(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Test Corp", Domain: "test.com", Industry: "Tech"}
	if err := db.CreateCompany(company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	deal1 := &repository.Deal{CompanyID: company.ID, CompanyName: company.Name, Title: "Deal 1", Amount: 100000, Currency: "USD", Stage: "prospecting"}
	if err := db.CreateDeal(deal1); err != nil {
		t.Fatalf("Failed to create deal1: %v", err)
	}

	deal2 := &repository.Deal{CompanyID: company.ID, CompanyName: company.Name, Title: "Deal 2", Amount: 200000, Currency: "USD", Stage: "qualification"}
	if err := db.CreateDeal(deal2); err != nil {
		t.Fatalf("Failed to create deal2: %v", err)
	}

	generator := NewGraphGenerator(db)

	graph, err := generator.GeneratePipelineGraph()
	if err != nil {
		t.Fatalf("GeneratePipelineGraph failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}
}
