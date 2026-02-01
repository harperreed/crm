// ABOUTME: MCP resource handlers for exposing CRM data
// ABOUTME: Provides read-only access to contacts, companies, and deals via URI
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ResourceHandlers struct {
	db *repository.DB
}

func NewResourceHandlers(db *repository.DB) *ResourceHandlers {
	return &ResourceHandlers{db: db}
}

// ReadResource handles resource read requests.
func (h *ResourceHandlers) ReadResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := request.Params.URI
	// Parse the URI
	if !strings.HasPrefix(uri, "crm://") {
		return nil, fmt.Errorf("invalid URI scheme: expected crm://")
	}

	path := strings.TrimPrefix(uri, "crm://")
	parts := strings.Split(path, "/")

	switch parts[0] {
	case "contacts":
		if len(parts) == 1 {
			return h.readAllContacts()
		}
		return h.readContact(parts[1])

	case "companies":
		if len(parts) == 1 {
			return h.readAllCompanies()
		}
		return h.readCompany(parts[1])

	case "deals":
		if len(parts) == 1 {
			return h.readAllDeals()
		}
		return h.readDeal(parts[1])

	case "pipeline":
		return h.readPipeline()

	default:
		return nil, fmt.Errorf("unknown resource: %s", parts[0])
	}
}

func (h *ResourceHandlers) readAllContacts() (*mcp.ReadResourceResult, error) {
	contacts, err := h.db.ListContacts(&repository.ContactFilter{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}

	data, err := json.MarshalIndent(contacts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contacts: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      "crm://contacts",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}

func (h *ResourceHandlers) readContact(idStr string) (*mcp.ReadResourceResult, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contact ID: %w", err)
	}

	contact, err := h.db.GetContact(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contact: %w", err)
	}

	data, err := json.MarshalIndent(contact, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contact: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      fmt.Sprintf("crm://contacts/%s", idStr),
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}

func (h *ResourceHandlers) readAllCompanies() (*mcp.ReadResourceResult, error) {
	companies, err := h.db.ListCompanies(&repository.CompanyFilter{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch companies: %w", err)
	}

	data, err := json.MarshalIndent(companies, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal companies: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      "crm://companies",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}

func (h *ResourceHandlers) readCompany(idStr string) (*mcp.ReadResourceResult, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	company, err := h.db.GetCompany(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch company: %w", err)
	}

	// Include associated contacts
	contacts, err := h.db.ListContacts(&repository.ContactFilter{CompanyID: &id, Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch company contacts: %w", err)
	}

	companyData := struct {
		*repository.Company
		Contacts []*repository.Contact `json:"contacts"`
	}{
		Company:  company,
		Contacts: contacts,
	}

	data, err := json.MarshalIndent(companyData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal company: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      fmt.Sprintf("crm://companies/%s", idStr),
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}

func (h *ResourceHandlers) readAllDeals() (*mcp.ReadResourceResult, error) {
	deals, err := h.db.ListDeals(&repository.DealFilter{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deals: %w", err)
	}

	data, err := json.MarshalIndent(deals, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deals: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      "crm://deals",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}

func (h *ResourceHandlers) readDeal(idStr string) (*mcp.ReadResourceResult, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid deal ID: %w", err)
	}

	deal, err := h.db.GetDeal(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deal: %w", err)
	}

	// Include deal notes/history
	notes, err := h.db.ListDealNotes(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deal notes: %w", err)
	}

	dealData := struct {
		*repository.Deal
		Notes []*repository.DealNote `json:"notes"`
	}{
		Deal:  deal,
		Notes: notes,
	}

	data, err := json.MarshalIndent(dealData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deal: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      fmt.Sprintf("crm://deals/%s", idStr),
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}

func (h *ResourceHandlers) readPipeline() (*mcp.ReadResourceResult, error) {
	allDeals, err := h.db.ListDeals(&repository.DealFilter{Limit: 10000})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deals: %w", err)
	}

	// Group by stage and calculate totals
	pipeline := make(map[string]struct {
		Count  int   `json:"count"`
		Amount int64 `json:"total_amount"`
	})

	for _, deal := range allDeals {
		stage := deal.Stage
		if stage == "" {
			stage = "unknown"
		}
		p := pipeline[stage]
		p.Count++
		p.Amount += deal.Amount
		pipeline[stage] = p
	}

	data, err := json.MarshalIndent(pipeline, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
		{
			URI:      "crm://pipeline",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}}, nil
}
