// ABOUTME: Tests for MCP resource handlers
// ABOUTME: Validates resource reading for contacts, companies, deals, and pipeline
package handlers

import (
	"context"
	"testing"

	"github.com/harperreed/pagen/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewResourceHandlers(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewResourceHandlers(db)
	if handlers == nil {
		t.Error("Expected non-nil ResourceHandlers")
	}
}

func TestReadAllContacts(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create test data
	_ = db.CreateContact(&repository.Contact{Name: "Alice", Email: "alice@example.com"})
	_ = db.CreateContact(&repository.Contact{Name: "Bob", Email: "bob@example.com"})

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://contacts",
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}

	if result.Contents[0].URI != "crm://contacts" {
		t.Errorf("Expected URI 'crm://contacts', got %s", result.Contents[0].URI)
	}
}

func TestReadSingleContact(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	contact := &repository.Contact{Name: "Alice", Email: "alice@example.com"}
	_ = db.CreateContact(contact)

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://contacts/" + contact.ID.String(),
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}
}

func TestReadContactInvalidID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://contacts/invalid-uuid",
		},
	}

	_, err := handlers.ReadResource(context.Background(), request)
	if err == nil {
		t.Error("Expected error for invalid contact ID")
	}
}

func TestReadAllCompanies(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	_ = db.CreateCompany(&repository.Company{Name: "Acme Corp", Domain: "acme.com"})
	_ = db.CreateCompany(&repository.Company{Name: "Beta Inc", Domain: "beta.io"})

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://companies",
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}
}

func TestReadSingleCompany(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	company := &repository.Company{Name: "Test Corp", Domain: "test.com"}
	_ = db.CreateCompany(company)

	// Add contact to company
	_ = db.CreateContact(&repository.Contact{Name: "Alice", CompanyID: &company.ID, CompanyName: company.Name})

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://companies/" + company.ID.String(),
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}
}

func TestReadCompanyInvalidID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://companies/invalid-uuid",
		},
	}

	_, err := handlers.ReadResource(context.Background(), request)
	if err == nil {
		t.Error("Expected error for invalid company ID")
	}
}

func TestReadAllDeals(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&repository.Deal{Title: "Deal 1", CompanyID: company.ID, CompanyName: company.Name, Amount: 100000, Currency: "USD", Stage: "prospecting"})
	_ = db.CreateDeal(&repository.Deal{Title: "Deal 2", CompanyID: company.ID, CompanyName: company.Name, Amount: 200000, Currency: "USD", Stage: "qualification"})

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://deals",
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}
}

func TestReadSingleDeal(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	deal := &repository.Deal{Title: "Big Deal", CompanyID: company.ID, CompanyName: company.Name, Amount: 100000, Currency: "USD", Stage: "prospecting"}
	_ = db.CreateDeal(deal)

	// Add deal note
	_ = db.CreateDealNote(&repository.DealNote{DealID: deal.ID, DealTitle: deal.Title, Content: "Test note"})

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://deals/" + deal.ID.String(),
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}
}

func TestReadDealInvalidID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://deals/invalid-uuid",
		},
	}

	_, err := handlers.ReadResource(context.Background(), request)
	if err == nil {
		t.Error("Expected error for invalid deal ID")
	}
}

func TestReadPipeline(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&repository.Deal{Title: "Deal 1", CompanyID: company.ID, CompanyName: company.Name, Amount: 100000, Currency: "USD", Stage: "prospecting"})
	_ = db.CreateDeal(&repository.Deal{Title: "Deal 2", CompanyID: company.ID, CompanyName: company.Name, Amount: 200000, Currency: "USD", Stage: "prospecting"})
	_ = db.CreateDeal(&repository.Deal{Title: "Deal 3", CompanyID: company.ID, CompanyName: company.Name, Amount: 300000, Currency: "USD", Stage: "qualification"})

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://pipeline",
		},
	}

	result, err := handlers.ReadResource(context.Background(), request)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("Expected non-empty result")
	}

	if result.Contents[0].URI != "crm://pipeline" {
		t.Errorf("Expected URI 'crm://pipeline', got %s", result.Contents[0].URI)
	}
}

func TestReadResourceInvalidScheme(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "http://contacts",
		},
	}

	_, err := handlers.ReadResource(context.Background(), request)
	if err == nil {
		t.Error("Expected error for invalid URI scheme")
	}
}

func TestReadResourceUnknownResource(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewResourceHandlers(db)

	request := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "crm://unknown",
		},
	}

	_, err := handlers.ReadResource(context.Background(), request)
	if err == nil {
		t.Error("Expected error for unknown resource")
	}
}
