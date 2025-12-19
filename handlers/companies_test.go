// ABOUTME: Tests for company MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"testing"

	"github.com/harperreed/pagen/charm"
)

func TestAddCompanyHandler(t *testing.T) {
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	handler := NewCompanyHandlers(client)

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
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	handler := NewCompanyHandlers(client)

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
	client, cleanup := charm.NewTestClient(t)
	defer cleanup()

	handler := NewCompanyHandlers(client)

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
