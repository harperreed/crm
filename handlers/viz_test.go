// ABOUTME: Tests for GraphViz visualization handlers
// ABOUTME: Validates graph generation for contacts, companies, and pipeline
package handlers

import (
	"context"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestNewVizHandlers(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewVizHandlers(db)
	if handlers == nil {
		t.Error("Expected non-nil VizHandlers")
	}
}

func TestGenerateGraphContacts(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	contact1 := &repository.Contact{Name: "Alice", Email: "alice@example.com"}
	_ = db.CreateContact(contact1)

	contact2 := &repository.Contact{Name: "Bob", Email: "bob@example.com"}
	_ = db.CreateContact(contact2)

	_ = db.CreateRelationship(&repository.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague"})

	handlers := NewVizHandlers(db)

	_, output, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type: "contacts",
	})
	if err != nil {
		t.Fatalf("GenerateGraph failed: %v", err)
	}

	if output.GraphType != "contacts" {
		t.Errorf("Expected graph type 'contacts', got %s", output.GraphType)
	}
	if output.DOTSource == "" {
		t.Error("Expected non-empty DOT source")
	}
}

func TestGenerateGraphContactsWithEntityID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	contact := &repository.Contact{Name: "Alice", Email: "alice@example.com"}
	_ = db.CreateContact(contact)

	handlers := NewVizHandlers(db)

	_, output, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type:     "contacts",
		EntityID: contact.ID.String(),
	})
	if err != nil {
		t.Fatalf("GenerateGraph failed: %v", err)
	}

	if output.GraphType != "contacts" {
		t.Errorf("Expected graph type 'contacts', got %s", output.GraphType)
	}
}

func TestGenerateGraphContactsInvalidEntityID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewVizHandlers(db)

	_, _, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type:     "contacts",
		EntityID: "invalid-uuid",
	})
	if err == nil {
		t.Error("Expected error for invalid entity ID")
	}
}

func TestGenerateGraphCompany(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	company := &repository.Company{Name: "Test Corp", Domain: "test.com"}
	_ = db.CreateCompany(company)

	_ = db.CreateContact(&repository.Contact{Name: "Alice", CompanyID: &company.ID, CompanyName: company.Name})
	_ = db.CreateContact(&repository.Contact{Name: "Bob", CompanyID: &company.ID, CompanyName: company.Name})

	handlers := NewVizHandlers(db)

	_, output, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type:     "company",
		EntityID: company.ID.String(),
	})
	if err != nil {
		t.Fatalf("GenerateGraph failed: %v", err)
	}

	if output.GraphType != "company" {
		t.Errorf("Expected graph type 'company', got %s", output.GraphType)
	}
	if output.DOTSource == "" {
		t.Error("Expected non-empty DOT source")
	}
}

func TestGenerateGraphCompanyMissingEntityID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewVizHandlers(db)

	_, _, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type: "company",
	})
	if err == nil {
		t.Error("Expected error for missing entity ID")
	}
}

func TestGenerateGraphCompanyInvalidEntityID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewVizHandlers(db)

	_, _, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type:     "company",
		EntityID: "invalid-uuid",
	})
	if err == nil {
		t.Error("Expected error for invalid entity ID")
	}
}

func TestGenerateGraphPipeline(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&repository.Deal{Title: "Deal 1", CompanyID: company.ID, CompanyName: company.Name, Stage: "prospecting", Amount: 100000, Currency: "USD"})
	_ = db.CreateDeal(&repository.Deal{Title: "Deal 2", CompanyID: company.ID, CompanyName: company.Name, Stage: "qualification", Amount: 200000, Currency: "USD"})

	handlers := NewVizHandlers(db)

	_, output, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type: "pipeline",
	})
	if err != nil {
		t.Fatalf("GenerateGraph failed: %v", err)
	}

	if output.GraphType != "pipeline" {
		t.Errorf("Expected graph type 'pipeline', got %s", output.GraphType)
	}
	if output.DOTSource == "" {
		t.Error("Expected non-empty DOT source")
	}
}

func TestGenerateGraphMissingType(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewVizHandlers(db)

	_, _, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{})
	if err == nil {
		t.Error("Expected error for missing type")
	}
}

func TestGenerateGraphUnknownType(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewVizHandlers(db)

	_, _, err := handlers.GenerateGraph(context.Background(), nil, GenerateGraphInput{
		Type: "unknown",
	})
	if err == nil {
		t.Error("Expected error for unknown type")
	}
}
