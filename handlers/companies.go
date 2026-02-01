// ABOUTME: Company MCP tool handlers
// ABOUTME: Implements add_company and find_companies tools
package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CompanyHandlers struct {
	db *repository.DB
}

func NewCompanyHandlers(db *repository.DB) *CompanyHandlers {
	return &CompanyHandlers{db: db}
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

	company := &repository.Company{
		Name:     input.Name,
		Domain:   input.Domain,
		Industry: input.Industry,
		Notes:    input.Notes,
	}

	if err := h.db.CreateCompany(company); err != nil {
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
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	filter := &repository.CompanyFilter{
		Query: input.Query,
		Limit: limit,
	}

	companies, err := h.db.ListCompanies(filter)
	if err != nil {
		return nil, FindCompaniesOutput{}, fmt.Errorf("failed to find companies: %w", err)
	}

	result := make([]CompanyOutput, len(companies))
	for i, company := range companies {
		result[i] = companyToOutput(company)
	}

	return nil, FindCompaniesOutput{Companies: result}, nil
}

func companyToOutput(company *repository.Company) CompanyOutput {
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

// Legacy map-based functions for tests.
func (h *CompanyHandlers) AddCompany_Legacy(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	company := &repository.Company{
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

	if err := h.db.CreateCompany(company); err != nil {
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

	filter := &repository.CompanyFilter{
		Query: query,
		Limit: limit,
	}

	companies, err := h.db.ListCompanies(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find companies: %w", err)
	}

	result := make([]map[string]interface{}, len(companies))
	for i, company := range companies {
		result[i] = companyToMap(company)
	}

	return result, nil
}

func companyToMap(company *repository.Company) map[string]interface{} {
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

type UpdateCompanyInput struct {
	CompanyID string `json:"company_id" jsonschema:"UUID of the company to update"`
	Name      string `json:"name,omitempty" jsonschema:"Updated company name"`
	Domain    string `json:"domain,omitempty" jsonschema:"Updated domain"`
	Industry  string `json:"industry,omitempty" jsonschema:"Updated industry"`
	Notes     string `json:"notes,omitempty" jsonschema:"Updated notes"`
}

func (h *CompanyHandlers) UpdateCompany(_ context.Context, request *mcp.CallToolRequest, input UpdateCompanyInput) (*mcp.CallToolResult, CompanyOutput, error) {
	if input.CompanyID == "" {
		return nil, CompanyOutput{}, fmt.Errorf("company_id is required")
	}

	companyID, err := uuid.Parse(input.CompanyID)
	if err != nil {
		return nil, CompanyOutput{}, fmt.Errorf("invalid company_id: %w", err)
	}

	// Get existing company
	company, err := h.db.GetCompany(companyID)
	if err != nil {
		return nil, CompanyOutput{}, fmt.Errorf("company not found: %w", err)
	}

	// Apply updates
	if input.Name != "" {
		company.Name = input.Name
	}
	if input.Domain != "" {
		company.Domain = input.Domain
	}
	if input.Industry != "" {
		company.Industry = input.Industry
	}
	if input.Notes != "" {
		company.Notes = input.Notes
	}

	err = h.db.UpdateCompany(company)
	if err != nil {
		return nil, CompanyOutput{}, fmt.Errorf("failed to update company: %w", err)
	}

	return nil, companyToOutput(company), nil
}

type DeleteCompanyInput struct {
	CompanyID string `json:"company_id" jsonschema:"UUID of the company to delete"`
}

type DeleteCompanyOutput struct {
	Message string `json:"message"`
}

func (h *CompanyHandlers) DeleteCompany(_ context.Context, request *mcp.CallToolRequest, input DeleteCompanyInput) (*mcp.CallToolResult, DeleteCompanyOutput, error) {
	if input.CompanyID == "" {
		return nil, DeleteCompanyOutput{}, fmt.Errorf("company_id is required")
	}

	companyID, err := uuid.Parse(input.CompanyID)
	if err != nil {
		return nil, DeleteCompanyOutput{}, fmt.Errorf("invalid company_id: %w", err)
	}

	err = h.db.DeleteCompany(companyID)
	if err != nil {
		return nil, DeleteCompanyOutput{}, fmt.Errorf("failed to delete company: %w", err)
	}

	return nil, DeleteCompanyOutput{
		Message: fmt.Sprintf("Deleted company: %s", companyID),
	}, nil
}
