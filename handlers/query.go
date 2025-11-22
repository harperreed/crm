// ABOUTME: Universal query tool handler
// ABOUTME: Implements flexible filtering across all CRM entity types
package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/db"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type QueryHandlers struct {
	db *sql.DB
}

func NewQueryHandlers(database *sql.DB) *QueryHandlers {
	return &QueryHandlers{db: database}
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

	// Query contacts using existing db function
	contacts, err := db.FindContacts(h.db, input.Query, companyID, input.Limit)
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find contacts: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(contacts))
	for i, c := range contacts {
		results[i] = contactToOutput(&c)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "contact",
		Results:    results,
		Count:      len(results),
	}, nil
}

func (h *QueryHandlers) queryCompanies(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Query companies using existing db function
	companies, err := db.FindCompanies(h.db, input.Query, input.Limit)
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find companies: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(companies))
	for i, c := range companies {
		results[i] = companyToOutput(&c)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "company",
		Results:    results,
		Count:      len(results),
	}, nil
}

func (h *QueryHandlers) queryDeals(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Extract filters
	var stage string
	var companyID *uuid.UUID
	var minAmount, maxAmount *int64

	if input.Filters != nil {
		// Extract stage filter
		if s, ok := input.Filters["stage"].(string); ok {
			stage = s
		}

		// Extract company_id filter
		if cid, ok := input.Filters["company_id"].(string); ok && cid != "" {
			id, err := uuid.Parse(cid)
			if err != nil {
				return nil, QueryCRMOutput{}, fmt.Errorf("invalid company_id: %w", err)
			}
			companyID = &id
		}

		// Extract min_amount filter
		if minAmountRaw, ok := input.Filters["min_amount"]; ok {
			if minAmountFloat, ok := minAmountRaw.(float64); ok {
				amt := int64(minAmountFloat)
				minAmount = &amt
			}
		}

		// Extract max_amount filter
		if maxAmountRaw, ok := input.Filters["max_amount"]; ok {
			if maxAmountFloat, ok := maxAmountRaw.(float64); ok {
				amt := int64(maxAmountFloat)
				maxAmount = &amt
			}
		}
	}

	// Query deals using existing db function
	deals, err := db.FindDeals(h.db, stage, companyID, input.Limit)
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find deals: %w", err)
	}

	// Filter by amount range in-memory (MVP approach)
	filteredDeals := make([]interface{}, 0)
	for _, d := range deals {
		// Check min/max amount filters
		if minAmount != nil && d.Amount < *minAmount {
			continue
		}
		if maxAmount != nil && d.Amount > *maxAmount {
			continue
		}
		filteredDeals = append(filteredDeals, dealToOutput(&d))
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "deal",
		Results:    filteredDeals,
		Count:      len(filteredDeals),
	}, nil
}

func (h *QueryHandlers) queryRelationships(input QueryCRMInput) (*mcp.CallToolResult, QueryCRMOutput, error) {
	// Extract filters
	var contactID *uuid.UUID
	var relationshipType string

	if input.Filters != nil {
		// Extract contact_id filter (required for relationships)
		if cid, ok := input.Filters["contact_id"].(string); ok && cid != "" {
			id, err := uuid.Parse(cid)
			if err != nil {
				return nil, QueryCRMOutput{}, fmt.Errorf("invalid contact_id: %w", err)
			}
			contactID = &id
		}

		// Extract relationship_type filter
		if rt, ok := input.Filters["relationship_type"].(string); ok {
			relationshipType = rt
		}
	}

	// contact_id is required for relationship queries
	if contactID == nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("contact_id filter is required for relationship queries")
	}

	// Query relationships using existing db function
	relationships, err := db.FindContactRelationships(h.db, *contactID, relationshipType)
	if err != nil {
		return nil, QueryCRMOutput{}, fmt.Errorf("failed to find relationships: %w", err)
	}

	// Convert to interface{} array
	results := make([]interface{}, len(relationships))
	for i, r := range relationships {
		results[i] = relationshipToOutput(&r)
	}

	return &mcp.CallToolResult{}, QueryCRMOutput{
		EntityType: "relationship",
		Results:    results,
		Count:      len(results),
	}, nil
}
