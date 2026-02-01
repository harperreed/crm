// ABOUTME: Tests for company MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestAddCompanyHandler(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

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
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

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

// Tests for typed handlers (non-legacy).
func TestAddCompanyTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	input := AddCompanyInput{
		Name:     "Typed Corp",
		Domain:   "typed.com",
		Industry: "Technology",
		Notes:    "Typed test company",
	}

	_, output, err := handler.AddCompany(nil, nil, input)
	if err != nil {
		t.Fatalf("AddCompany failed: %v", err)
	}

	if output.Name != "Typed Corp" {
		t.Errorf("Expected name 'Typed Corp', got %v", output.Name)
	}
	if output.Domain != "typed.com" {
		t.Errorf("Expected domain 'typed.com', got %v", output.Domain)
	}
	if output.ID == "" {
		t.Error("ID was not set")
	}
}

func TestAddCompanyTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Missing name
	input := AddCompanyInput{
		Domain: "noname.com",
	}

	_, _, err := handler.AddCompany(nil, nil, input)
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

func TestFindCompaniesTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Add test companies
	_, _, _ = handler.AddCompany(nil, nil, AddCompanyInput{Name: "Typed Alpha"})
	_, _, _ = handler.AddCompany(nil, nil, AddCompanyInput{Name: "Typed Beta"})

	input := FindCompaniesInput{
		Query: "typed",
		Limit: 10,
	}

	_, output, err := handler.FindCompanies(nil, nil, input)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	if len(output.Companies) != 2 {
		t.Errorf("Expected 2 companies, got %d", len(output.Companies))
	}
}

func TestFindCompaniesTypedDefaultLimit(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Add test company
	_, _, _ = handler.AddCompany(nil, nil, AddCompanyInput{Name: "Limit Test Corp"})

	// Zero limit should default to 10
	input := FindCompaniesInput{
		Query: "limit",
		Limit: 0,
	}

	_, output, err := handler.FindCompanies(nil, nil, input)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	if len(output.Companies) != 1 {
		t.Errorf("Expected 1 company, got %d", len(output.Companies))
	}
}

func TestUpdateCompanyTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Create company
	_, created, _ := handler.AddCompany(nil, nil, AddCompanyInput{
		Name:   "Original Corp",
		Domain: "original.com",
	})

	// Update company
	input := UpdateCompanyInput{
		CompanyID: created.ID,
		Name:      "Updated Corp",
		Domain:    "updated.com",
		Industry:  "Finance",
		Notes:     "Updated notes",
	}

	_, output, err := handler.UpdateCompany(nil, nil, input)
	if err != nil {
		t.Fatalf("UpdateCompany failed: %v", err)
	}

	if output.Name != "Updated Corp" {
		t.Errorf("Expected name 'Updated Corp', got %v", output.Name)
	}
	if output.Domain != "updated.com" {
		t.Errorf("Expected domain 'updated.com', got %v", output.Domain)
	}
}

func TestUpdateCompanyTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Missing company_id
	input := UpdateCompanyInput{
		Name: "Test",
	}

	_, _, err := handler.UpdateCompany(nil, nil, input)
	if err == nil {
		t.Error("Expected error for missing company_id")
	}
}

func TestUpdateCompanyTypedInvalidID(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	input := UpdateCompanyInput{
		CompanyID: "invalid-uuid",
		Name:      "Test",
	}

	_, _, err := handler.UpdateCompany(nil, nil, input)
	if err == nil {
		t.Error("Expected error for invalid company_id")
	}
}

func TestDeleteCompanyTyped(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Create company
	_, created, _ := handler.AddCompany(nil, nil, AddCompanyInput{
		Name: "To Delete Corp",
	})

	// Delete company
	input := DeleteCompanyInput{
		CompanyID: created.ID,
	}

	_, output, err := handler.DeleteCompany(nil, nil, input)
	if err != nil {
		t.Fatalf("DeleteCompany failed: %v", err)
	}

	if output.Message == "" {
		t.Error("Expected success message")
	}
}

func TestDeleteCompanyTypedValidation(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	// Missing company_id
	input := DeleteCompanyInput{}

	_, _, err := handler.DeleteCompany(nil, nil, input)
	if err == nil {
		t.Error("Expected error for missing company_id")
	}
}

func TestDeleteCompanyTypedInvalidID(t *testing.T) {
	client := func() *repository.DB { db, cleanup, _ := repository.NewTestDB(); t.Cleanup(cleanup); return db }()

	handler := NewCompanyHandlers(client)

	input := DeleteCompanyInput{
		CompanyID: "invalid-uuid",
	}

	_, _, err := handler.DeleteCompany(nil, nil, input)
	if err == nil {
		t.Error("Expected error for invalid company_id")
	}
}
