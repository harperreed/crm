// ABOUTME: Universal query tool handler
// ABOUTME: Implements flexible filtering across all CRM entity types
package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type QueryHandlers struct {
	db *repository.DB
}

func NewQueryHandlers(db *repository.DB) *QueryHandlers {
	return &QueryHandlers{db: db}
}

type QueryCRMInput struct {
	EntityType string                 `json:"entity_type" jsonschema:"Type of entity to query (contact, company, deal, relationship)"`
	Query      string                 `json:"query,omitempty" jsonschema:"Search query (for name/email/domain)"`
	Filters    map[string]interface{} `json:"filters,omitempty" jsonschema:"Additional filters as key-value pairs"`
	Limit      int                    `json:"limit,omitempty" jsonschema:"Maximum results to return (default 10)"`
}

type QueryCRMOutput struct {
	EntityType string        `json:"entity_type"`
	Results    []interface{} `json:"results"`
	Count      int           `json:"count"`
}

func (h *QueryHandlers) QueryCRM(ctx context.Context, req *mcp.CallToolRequest, input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Set default limit
	if input.Limit == 0 {
		input.Limit = 10
	}

	switch input.EntityType {
	case "contact":
		return h.queryContacts(input)
	case "company":
		return h.queryCompanies(input)
	case "deal":
		return h.queryDeals(input)
	case "relationship":
		return h.queryRelationships(input)
	default:
		return nil, QueryCRMOutput{}, fmt.Errorf("invalid entity_type: %s (valid: contact, company, deal, relationship)", input.EntityType)
	}
}

func (h *QueryHandlers) queryContacts(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Extract company_id filter if present
	var companyID *uuid.UUID
	if input.Filters != nil {
		if cid, ok := input.Filters["company_id"].(string); ok && cid != "" {
			id, err := uuid.Parse(cid)
			if err != nil {
				return nil, QueryCRMOutput{}, fmt.Errorf("invalid company_id: %w", err)
			}
			companyID = &id
		}
	}

	// Query contacts using charm client
	contacts, err := h.db.ListContacts(&repository.ContactFilter{
		Query:     input.Query,
		CompanyID: companyID,
		Limit:     input.Limit,
	})
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find contacts: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(contacts))
	for i, c := range contacts {
		results[i] = contactToOutput(c)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "contact",
		Results:    results,
		Count:      len(results),
	}, nil
}

func (h *QueryHandlers) queryCompanies(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Query companies using charm client
	companies, err := h.db.ListCompanies(&repository.CompanyFilter{
		Query: input.Query,
		Limit: input.Limit,
	})
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find companies: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(companies))
	for i, c := range companies {
		results[i] = companyToOutput(c)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "company",
		Results:    results,
		Count:      len(results),
	}, nil
}

func (h *QueryHandlers) queryDeals(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Build filter from input
	filter := &repository.DealFilter{
		Limit: input.Limit,
	}

	if input.Filters != nil {
		// Extract stage filter
		if s, ok := input.Filters["stage"].(string); ok {
			filter.Stage = s
		}

		// Extract company_id filter
		if cid, ok := input.Filters["company_id"].(string); ok && cid != "" {
			id, err := uuid.Parse(cid)
			if err != nil {
				return nil, QueryCRMOutput{}, fmt.Errorf("invalid company_id: %w", err)
			}
			filter.CompanyID = &id
		}

		// Extract min_amount filter
		if minAmountRaw, ok := input.Filters["min_amount"]; ok {
			if minAmountFloat, ok := minAmountRaw.(float64); ok {
				filter.MinAmount = int64(minAmountFloat)
			}
		}

		// Extract max_amount filter
		if maxAmountRaw, ok := input.Filters["max_amount"]; ok {
			if maxAmountFloat, ok := maxAmountRaw.(float64); ok {
				filter.MaxAmount = int64(maxAmountFloat)
			}
		}
	}

	// Query deals using charm client
	deals, err := h.db.ListDeals(filter)
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find deals: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(deals))
	for i, d := range deals {
		results[i] = dealToOutput(d)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "deal",
		Results:    results,
		Count:      len(results),
	}, nil
}

func (h *QueryHandlers) queryRelationships(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Build filter from input
	filter := &repository.RelationshipFilter{
		Limit: input.Limit,
	}

	if input.Filters != nil {
		// Extract contact_id filter (required for relationships)
		if cid, ok := input.Filters["contact_id"].(string); ok && cid != "" {
			id, err := uuid.Parse(cid)
			if err != nil {
				return nil, QueryCRMOutput{}, fmt.Errorf("invalid contact_id: %w", err)
			}
			filter.ContactID = &id
		}

		// Extract relationship_type filter
		if rt, ok := input.Filters["relationship_type"].(string); ok {
			filter.RelationshipType = rt
		}
	}

	// contact_id is required for relationship queries
	if filter.ContactID == nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("contact_id filter is required for relationship queries")
	}

	// Query relationships using charm client
	relationships, err := h.db.ListRelationships(filter)
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find relationships: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(relationships))
	for i, r := range relationships {
		results[i] = relationshipToOutput(r)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "relationship",
		Results:    results,
		Count:      len(results),
	}, nil
}
