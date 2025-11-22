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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
