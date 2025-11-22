// ABOUTME: Tests for company MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"database/sql"
	"testing"

	"github.com/harperreed/pagen/db"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	if err := db.InitSchema(database); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return database
}

func TestAddCompanyHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewCompanyHandlers(database)

	input := map[string]interface{}{
		"name":     "Test Corp",
		"domain":   "test.com",
		"industry": "Tech",
		"notes":    "Test company",
	}

	result, err := handler.AddCompany_Legacy(input)
	if err != nil {
		t.Fatalf("AddCompany failed: %v", err)
	}

	// Verify result structure
	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["name"] != "Test Corp" {
		t.Errorf("Expected name 'Test Corp', got %v", data["name"])
	}

	if data["id"] == nil {
		t.Error("ID was not set")
	}
}

func TestAddCompanyValidation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewCompanyHandlers(database)

	// Missing required name
	input := map[string]interface{}{
		"domain": "test.com",
	}

	_, err := handler.AddCompany_Legacy(input)
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestFindCompaniesHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewCompanyHandlers(database)

	// Add test companies
	_, _ = handler.AddCompany_Legacy(map[string]interface{}{"name": "Acme Corp"})
	_, _ = handler.AddCompany_Legacy(map[string]interface{}{"name": "Beta Inc"})

	input := map[string]interface{}{
		"query": "corp",
		"limit": float64(10),
	}

	result, err := handler.FindCompanies_Legacy(input)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	companies, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(companies) == 0 {
		t.Error("Expected to find companies")
	}
}
