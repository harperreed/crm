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

func TestGenerateCompleteGraph(t *testing.T) {
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

	rel := &repository.Relationship{ContactID1: contact1.ID, ContactID2: contact2.ID, RelationshipType: "colleague"}
	if err := db.CreateRelationship(rel); err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	deal := &repository.Deal{CompanyID: company.ID, CompanyName: company.Name, ContactID: &contact1.ID, Title: "Big Deal", Amount: 100000, Currency: "USD", Stage: "prospecting"}
	if err := db.CreateDeal(deal); err != nil {
		t.Fatalf("Failed to create deal: %v", err)
	}

	generator := NewGraphGenerator(db)

	graph, err := generator.GenerateCompleteGraph()
	if err != nil {
		t.Fatalf("GenerateCompleteGraph failed: %v", err)
	}
	if len(graph) < 10 {
		t.Errorf("Expected non-empty graph, got %d bytes", len(graph))
	}
}

func TestGenerateDashboardStats(t *testing.T) {
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

	contact := &repository.Contact{Name: "Alice", Email: "alice@test.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("Failed to create contact: %v", err)
	}

	deal := &repository.Deal{CompanyID: company.ID, CompanyName: company.Name, Title: "Deal 1", Amount: 100000, Currency: "USD", Stage: "prospecting"}
	if err := db.CreateDeal(deal); err != nil {
		t.Fatalf("Failed to create deal: %v", err)
	}

	stats, err := GenerateDashboardStats(db)
	if err != nil {
		t.Fatalf("GenerateDashboardStats failed: %v", err)
	}

	if stats.TotalContacts != 1 {
		t.Errorf("Expected 1 contact, got %d", stats.TotalContacts)
	}
	if stats.TotalCompanies != 1 {
		t.Errorf("Expected 1 company, got %d", stats.TotalCompanies)
	}
	if stats.TotalDeals != 1 {
		t.Errorf("Expected 1 deal, got %d", stats.TotalDeals)
	}

	if pstats, exists := stats.PipelineByStage["prospecting"]; exists {
		if pstats.Count != 1 {
			t.Errorf("Expected 1 deal in prospecting, got %d", pstats.Count)
		}
		if pstats.Amount != 100000 {
			t.Errorf("Expected amount 100000 in prospecting, got %d", pstats.Amount)
		}
	} else {
		t.Error("Expected prospecting stage to exist")
	}
}

func TestGenerateDashboardStatsWithStaleContacts(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Contact never contacted (nil LastContactedAt)
	contact1 := &repository.Contact{Name: "Never Contacted"}
	if err := db.CreateContact(contact1); err != nil {
		t.Fatalf("Failed to create contact1: %v", err)
	}

	stats, err := GenerateDashboardStats(db)
	if err != nil {
		t.Fatalf("GenerateDashboardStats failed: %v", err)
	}

	if len(stats.StaleContacts) != 1 {
		t.Errorf("Expected 1 stale contact, got %d", len(stats.StaleContacts))
	}
	if len(stats.StaleContacts) > 0 && stats.StaleContacts[0].DaysSince != -1 {
		t.Errorf("Expected DaysSince -1 for never contacted, got %d", stats.StaleContacts[0].DaysSince)
	}
}

func TestRenderDashboard(t *testing.T) {
	stats := &DashboardStats{
		TotalContacts:  5,
		TotalCompanies: 3,
		TotalDeals:     10,
		PipelineByStage: map[string]PipelineStageStats{
			"prospecting":   {Stage: "prospecting", Count: 4, Amount: 400000},
			"qualification": {Stage: "qualification", Count: 3, Amount: 300000},
			"proposal":      {Stage: "proposal", Count: 2, Amount: 200000},
			"closed_won":    {Stage: "closed_won", Count: 1, Amount: 100000},
		},
		StaleContacts: []StaleContact{
			{Name: "Old Contact", DaysSince: 45},
		},
		StaleDeals: []StaleDeal{
			{Title: "Stale Deal", DaysSince: 20},
		},
	}

	output := RenderDashboard(stats)

	if output == "" {
		t.Error("Expected non-empty dashboard output")
	}

	// Check for key sections
	if !containsString(output, "PIPELINE OVERVIEW") {
		t.Error("Expected PIPELINE OVERVIEW section")
	}
	if !containsString(output, "STATS") {
		t.Error("Expected STATS section")
	}
	if !containsString(output, "NEEDS ATTENTION") {
		t.Error("Expected NEEDS ATTENTION section")
	}
}

func TestRenderDashboardNoStaleItems(t *testing.T) {
	stats := &DashboardStats{
		TotalContacts:   2,
		TotalCompanies:  1,
		TotalDeals:      3,
		PipelineByStage: map[string]PipelineStageStats{},
		StaleContacts:   []StaleContact{},
		StaleDeals:      []StaleDeal{},
	}

	output := RenderDashboard(stats)

	if output == "" {
		t.Error("Expected non-empty dashboard output")
	}

	// NEEDS ATTENTION should not appear when no stale items
	if containsString(output, "NEEDS ATTENTION") {
		t.Error("Should not show NEEDS ATTENTION when no stale items")
	}
}

func TestGeneratePipelineGraphEmptyDB(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	generator := NewGraphGenerator(db)

	graph, err := generator.GeneratePipelineGraph()
	if err != nil {
		t.Fatalf("GeneratePipelineGraph failed on empty DB: %v", err)
	}
	if len(graph) == 0 {
		t.Error("Expected some graph output even for empty DB")
	}
}

func TestGenerateContactGraphEmpty(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	generator := NewGraphGenerator(db)

	// Test with nil contactID (all contacts)
	graph, err := generator.GenerateContactGraph(nil)
	if err != nil {
		t.Fatalf("GenerateContactGraph failed: %v", err)
	}
	if len(graph) == 0 {
		t.Error("Expected some graph output")
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}

func TestGenerateCompleteGraphEmpty(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	generator := NewGraphGenerator(db)

	graph, err := generator.GenerateCompleteGraph()
	if err != nil {
		t.Fatalf("GenerateCompleteGraph failed on empty DB: %v", err)
	}
	if len(graph) == 0 {
		t.Error("Expected some graph output even for empty DB")
	}
}
