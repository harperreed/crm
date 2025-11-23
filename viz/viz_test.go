package viz

import (
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	tmpDb := "/tmp/test_viz_" + uuid.New().String() + ".db"
	database, err := sql.Open("sqlite3", tmpDb)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.InitSchema(database); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.Remove(tmpDb)
	}

	return database, cleanup
}

func TestGenerateContactGraph(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test data
	company := &models.Company{Name: "Test Corp", Domain: "test.com", Industry: "Tech"}
	if err := db.CreateCompany(database, company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	contact1 := &models.Contact{Name: "Alice", Email: "alice@test.com", CompanyID: &company.ID}
	if err := db.CreateContact(database, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob", Email: "bob@test.com", CompanyID: &company.ID}
	if err := db.CreateContact(database, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	rel := &models.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague", Context: "work together"}
	if err := db.CreateRelationship(database, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	generator := NewGraphGenerator(database)

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
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test data
	company := &models.Company{Name: "Test Corp", Domain: "test.com", Industry: "Tech"}
	if err := db.CreateCompany(database, company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	contact1 := &models.Contact{Name: "Alice", Email: "alice@test.com", CompanyID: &company.ID}
	if err := db.CreateContact(database, contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob", Email: "bob@test.com", CompanyID: &company.ID}
	if err := db.CreateContact(database, contact2); err != nil {
		t.Fatalf("Failed to create contact2: %v", err)
	}

	rel := &models.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague", Context: "work together"}
	if err := db.CreateRelationship(database, rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	generator := NewGraphGenerator(database)

	graph, err := generator.GenerateCompanyGraph(company.ID)
	if err != nil {
		t.Fatalf("GenerateCompanyGraph failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}
}

func TestGeneratePipelineGraph(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test data
	company := &models.Company{Name: "Test Corp", Domain: "test.com", Industry: "Tech"}
	if err := db.CreateCompany(database, company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	deal1 := &models.Deal{CompanyID: company.ID, Title: "Deal 1", Amount: 100000, Currency: "USD", Stage: "prospecting"}
	if err := db.CreateDeal(database, deal1); err != nil {
		t.Fatalf("Failed to create deal1: %v", err)
	}

	deal2 := &models.Deal{CompanyID: company.ID, Title: "Deal 2", Amount: 200000, Currency: "USD", Stage: "qualification"}
	if err := db.CreateDeal(database, deal2); err != nil {
		t.Fatalf("Failed to create deal2: %v", err)
	}

	generator := NewGraphGenerator(database)

	graph, err := generator.GeneratePipelineGraph()
	if err != nil {
		t.Fatalf("GeneratePipelineGraph failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}
}
