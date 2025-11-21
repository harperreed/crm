// ABOUTME: Company MCP tool handlers
// ABOUTME: Implements add_company and find_companies tools
package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CompanyHandlers struct {
	db *sql.DB
}

func NewCompanyHandlers(database *sql.DB) *CompanyHandlers {
	return &CompanyHandlers{db: database}
}

type AddCompanyInput struct {
	Name     string `json:"name" jsonschema:"Company name (required)"`
	Domain   string `json:"domain,omitempty" jsonschema:"Company domain (e.g., acme.com)"`
	Industry string `json:"industry,omitempty" jsonschema:"Industry or sector"`
	Notes    string `json:"notes,omitempty" jsonschema:"Additional notes about the company"`
}

type CompanyOutput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Domain    string `json:"domain,omitempty"`
	Industry  string `json:"industry,omitempty"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *CompanyHandlers) AddCompany(_ context.Context, request *mcp.CallToolRequest, input AddCompanyInput) (*mcp.CallToolResult, CompanyOutput, error) {
	if input.Name == "" {
		return nil, CompanyOutput{}, fmt.Errorf("name is required")
	}

	company := &models.Company{
		Name:     input.Name,
		Domain:   input.Domain,
		Industry: input.Industry,
		Notes:    input.Notes,
	}

	if err := db.CreateCompany(h.db, company); err != nil {
		return nil, CompanyOutput{}, fmt.Errorf("failed to create company: %w", err)
	}

	return nil, companyToOutput(company), nil
}

type FindCompaniesInput struct {
	Query string `json:"query,omitempty" jsonschema:"Search query (searches name and domain)"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results (default 10)"`
}

type FindCompaniesOutput struct {
	Companies []CompanyOutput `json:"companies"`
}

func (h *CompanyHandlers) FindCompanies(_ context.Context, request *mcp.CallToolRequest, input FindCompaniesInput) (*mcp.CallToolResult, FindCompaniesOutput, error) {
	query := input.Query
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	companies, err := db.FindCompanies(h.db, query, limit)
	if err != nil {
		return nil, FindCompaniesOutput{}, fmt.Errorf("failed to find companies: %w", err)
	}

	result := make([]CompanyOutput, len(companies))
	for i, company := range companies {
		result[i] = companyToOutput(&company)
	}

	return nil, FindCompaniesOutput{Companies: result}, nil
}

func companyToOutput(company *models.Company) CompanyOutput {
	return CompanyOutput{
		ID:        company.ID.String(),
		Name:      company.Name,
		Domain:    company.Domain,
		Industry:  company.Industry,
		Notes:     company.Notes,
		CreatedAt: company.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: company.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Legacy map-based functions for tests
func (h *CompanyHandlers) AddCompany_Legacy(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	company := &models.Company{
		Name: name,
	}

	if domain, ok := args["domain"].(string); ok {
		company.Domain = domain
	}

	if industry, ok := args["industry"].(string); ok {
		company.Industry = industry
	}

	if notes, ok := args["notes"].(string); ok {
		company.Notes = notes
	}

	if err := db.CreateCompany(h.db, company); err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return companyToMap(company), nil
}

func (h *CompanyHandlers) FindCompanies_Legacy(args map[string]interface{}) (interface{}, error) {
	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	companies, err := db.FindCompanies(h.db, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find companies: %w", err)
	}

	result := make([]map[string]interface{}, len(companies))
	for i, company := range companies {
		result[i] = companyToMap(&company)
	}

	return result, nil
}

func companyToMap(company *models.Company) map[string]interface{} {
	return map[string]interface{}{
		"id":         company.ID.String(),
		"name":       company.Name,
		"domain":     company.Domain,
		"industry":   company.Industry,
		"notes":      company.Notes,
		"created_at": company.CreatedAt,
		"updated_at": company.UpdatedAt,
	}
}
