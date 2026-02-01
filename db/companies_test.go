// ABOUTME: Tests for company database operations
// ABOUTME: Covers CRUD operations and company lookups
package db

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	if err := InitSchema(db); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return db
}

func TestCreateCompany(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	company := &models.Company{
		Name:     "Acme Corp",
		Domain:   "acme.com",
		Industry: "Technology",
		Notes:    "Test company",
	}

	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	if company.ID == uuid.Nil {
		t.Error("Company ID was not set")
	}

	if company.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}

	if company.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}
}

func TestGetCompany(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{Name: "Test Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Get company
	found, err := GetCompany(db, company.ID)
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}

	if found == nil {
		t.Fatal("Company not found")
	}

	if found.Name != company.Name {
		t.Errorf("Expected name %s, got %s", company.Name, found.Name)
	}
}

func TestFindCompanies(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create test companies
	companies := []*models.Company{
		{Name: "Acme Corp", Domain: "acme.com"},
		{Name: "Beta Inc", Domain: "beta.com"},
		{Name: "Acme Industries", Domain: "acme-ind.com"},
	}

	for _, c := range companies {
		if err := CreateCompany(db, c); err != nil {
			t.Fatalf("CreateCompany failed: %v", err)
		}
	}

	// Search for "acme"
	results, err := FindCompanies(db, "acme", 10)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestFindCompanyByName(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	company := &models.Company{Name: "Unique Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	found, err := FindCompanyByName(db, "unique corp")
	if err != nil {
		t.Fatalf("FindCompanyByName failed: %v", err)
	}

	if found == nil {
		t.Fatal("Company not found")
	}

	if found.ID != company.ID {
		t.Error("Found wrong company")
	}
}

func TestUpdateCompany(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{
		Name:     "Original Corp",
		Domain:   "original.com",
		Industry: "Tech",
		Notes:    "Original notes",
	}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Update company
	updates := &models.Company{
		Name:     "Updated Corp",
		Domain:   "updated.com",
		Industry: "Finance",
		Notes:    "Updated notes",
	}
	if err := UpdateCompany(db, company.ID, updates); err != nil {
		t.Fatalf("UpdateCompany failed: %v", err)
	}

	// Verify updates
	found, err := GetCompany(db, company.ID)
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}
	if found.Name != "Updated Corp" {
		t.Errorf("Expected name 'Updated Corp', got %s", found.Name)
	}
	if found.Domain != "updated.com" {
		t.Errorf("Expected domain 'updated.com', got %s", found.Domain)
	}
	if found.Industry != "Finance" {
		t.Errorf("Expected industry 'Finance', got %s", found.Industry)
	}
	if found.Notes != "Updated notes" {
		t.Errorf("Expected notes 'Updated notes', got %s", found.Notes)
	}

	// Test update non-existent company
	err = UpdateCompany(db, uuid.New(), updates)
	if err == nil {
		t.Error("Expected error when updating non-existent company")
	}
}

func TestDeleteCompany(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company with no dependencies
	company := &models.Company{Name: "To Delete Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Delete company
	if err := DeleteCompany(db, company.ID); err != nil {
		t.Fatalf("DeleteCompany failed: %v", err)
	}

	// Verify company is deleted
	found, err := GetCompany(db, company.ID)
	if err != nil {
		t.Fatalf("GetCompany after delete failed: %v", err)
	}
	if found != nil {
		t.Error("Company should be deleted")
	}

	// Test deleting non-existent company
	err = DeleteCompany(db, uuid.New())
	if err == nil {
		t.Error("Expected error when deleting non-existent company")
	}
}

func TestDeleteCompanyWithDeals(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{Name: "Company With Deals"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create deal for this company
	deal := &models.Deal{
		Title:     "Important Deal",
		CompanyID: company.ID,
		Stage:     "prospecting",
		Currency:  "USD",
	}
	if err := CreateDeal(db, deal); err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	// Try to delete company - should fail
	err := DeleteCompany(db, company.ID)
	if err == nil {
		t.Error("Expected error when deleting company with active deals")
	}
	if err != nil && !containsStr(err.Error(), "active deals") {
		t.Errorf("Expected error about active deals, got: %v", err)
	}
}

func TestDeleteCompanyWithContacts(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create company
	company := &models.Company{Name: "Company With Contacts"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contact at this company
	contact := &models.Contact{
		Name:      "John Doe",
		Email:     "john@company.com",
		CompanyID: &company.ID,
	}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Delete company - should succeed but remove company_id from contacts
	if err := DeleteCompany(db, company.ID); err != nil {
		t.Fatalf("DeleteCompany failed: %v", err)
	}

	// Verify contact still exists but without company_id
	foundContact, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if foundContact == nil {
		t.Fatal("Contact should still exist")
	}
	if foundContact.CompanyID != nil {
		t.Error("Contact company_id should be nil after company deletion")
	}
}

func TestGetCompanyNonexistent(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Try to get non-existent company
	found, err := GetCompany(db, uuid.New())
	if err != nil {
		t.Fatalf("GetCompany should not error for non-existent ID: %v", err)
	}
	if found != nil {
		t.Error("Expected nil for non-existent company")
	}
}

func TestFindCompanyByNameNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Try to find non-existent company by name
	found, err := FindCompanyByName(db, "nonexistent company")
	if err != nil {
		t.Fatalf("FindCompanyByName should not error for non-existent name: %v", err)
	}
	if found != nil {
		t.Error("Expected nil for non-existent company name")
	}
}

func TestFindCompaniesWithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create multiple companies
	for i := 0; i < 5; i++ {
		company := &models.Company{Name: "Test Company"}
		if err := CreateCompany(db, company); err != nil {
			t.Fatalf("CreateCompany failed: %v", err)
		}
	}

	// Test with limit
	results, err := FindCompanies(db, "Test", 2)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(results))
	}

	// Test with zero limit (defaults to 10)
	results, err = FindCompanies(db, "Test", 0)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("Expected 5 results with default limit, got %d", len(results))
	}

	// Test empty query returns all
	results, err = FindCompanies(db, "", 10)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("Expected 5 results with empty query, got %d", len(results))
	}
}

// Helper function for string contains check.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsStrHelper(s, substr))
}

func containsStrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
