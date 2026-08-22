// ABOUTME: MCP tool handlers for CRM CRUD and history operations.
// ABOUTME: Defines 14 tools with JSON schema input validation and result helpers.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harperreed/crm/v2/internal/history"
	"github.com/harperreed/crm/v2/internal/models"
	"github.com/harperreed/crm/v2/internal/storage"
)

// registerTools adds all 14 CRM tools to the MCP server.
func (s *Server) registerTools() {
	s.server.AddTool(addContactTool(), s.handleAddContact)
	s.server.AddTool(listContactsTool(), s.handleListContacts)
	s.server.AddTool(getContactTool(), s.handleGetContact)
	s.server.AddTool(updateContactTool(), s.handleUpdateContact)
	s.server.AddTool(deleteContactTool(), s.handleDeleteContact)
	s.server.AddTool(addCompanyTool(), s.handleAddCompany)
	s.server.AddTool(listCompaniesTool(), s.handleListCompanies)
	s.server.AddTool(getCompanyTool(), s.handleGetCompany)
	s.server.AddTool(updateCompanyTool(), s.handleUpdateCompany)
	s.server.AddTool(deleteCompanyTool(), s.handleDeleteCompany)
	s.server.AddTool(linkTool(), s.handleLink)
	s.server.AddTool(unlinkTool(), s.handleUnlink)
	s.server.AddTool(listHistoryTool(), s.handleListHistory)
	s.server.AddTool(getHistoryEventTool(), s.handleGetHistoryEvent)
}

// --- result helpers ---

func errResult(msg string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}, nil
}

func textResult(msg string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, nil
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(fmt.Sprintf("marshal result: %v", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil
}

// resolveContact looks up a contact by full UUID or prefix string.
func (s *Server) resolveContact(idStr string) (*models.Contact, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return s.store.GetContact(id)
	}
	return s.store.GetContactByPrefix(idStr)
}

// resolveCompany looks up a company by full UUID or prefix string.
func (s *Server) resolveCompany(idStr string) (*models.Company, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return s.store.GetCompany(id)
	}
	return s.store.GetCompanyByPrefix(idStr)
}

// --- tool definitions ---

func addContactTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "add_contact",
		Description: "Add a new contact to the CRM",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name":   {"type": "string", "description": "Contact name (required)"},
				"email":  {"type": "string", "description": "Email address"},
				"phone":  {"type": "string", "description": "Phone number"},
				"fields": {"type": "object", "description": "Additional key-value fields"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Tags for categorization"}
			},
			"required": ["name"]
		}`),
	}
}

func listContactsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_contacts",
		Description: "List contacts with optional filtering",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"tag":    {"type": "string", "description": "Filter by tag"},
				"search": {"type": "string", "description": "Full-text search query"},
				"limit":  {"type": "integer", "description": "Maximum results (default 20)"}
			}
		}`),
	}
}

func getContactTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_contact",
		Description: "Get a contact by ID (full UUID or prefix)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Contact UUID or prefix (min 6 chars)"}
			},
			"required": ["id"]
		}`),
	}
}

func updateContactTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_contact",
		Description: "Update an existing contact's fields",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id":     {"type": "string", "description": "Contact UUID or prefix"},
				"name":   {"type": "string", "description": "New name"},
				"email":  {"type": "string", "description": "New email"},
				"phone":  {"type": "string", "description": "New phone"},
				"fields": {"type": "object", "description": "Fields to merge (keys are added/overwritten)"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Replacement tags"}
			},
			"required": ["id"]
		}`),
	}
}

func deleteContactTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_contact",
		Description: "Delete a contact by ID",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Contact UUID or prefix"}
			},
			"required": ["id"]
		}`),
	}
}

func addCompanyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "add_company",
		Description: "Add a new company to the CRM",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name":   {"type": "string", "description": "Company name (required)"},
				"domain": {"type": "string", "description": "Company domain/website"},
				"fields": {"type": "object", "description": "Additional key-value fields"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Tags for categorization"}
			},
			"required": ["name"]
		}`),
	}
}

func listCompaniesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_companies",
		Description: "List companies with optional filtering",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"tag":    {"type": "string", "description": "Filter by tag"},
				"search": {"type": "string", "description": "Full-text search query"},
				"limit":  {"type": "integer", "description": "Maximum results (default 20)"}
			}
		}`),
	}
}

func getCompanyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_company",
		Description: "Get a company by ID (full UUID or prefix)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Company UUID or prefix (min 6 chars)"}
			},
			"required": ["id"]
		}`),
	}
}

func updateCompanyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_company",
		Description: "Update an existing company's fields",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id":     {"type": "string", "description": "Company UUID or prefix"},
				"name":   {"type": "string", "description": "New name"},
				"domain": {"type": "string", "description": "New domain"},
				"fields": {"type": "object", "description": "Fields to merge (keys are added/overwritten)"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Replacement tags"}
			},
			"required": ["id"]
		}`),
	}
}

func deleteCompanyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_company",
		Description: "Delete a company by ID",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Company UUID or prefix"}
			},
			"required": ["id"]
		}`),
	}
}

func linkTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "link",
		Description: "Create a relationship between two entities",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"source_id": {"type": "string", "description": "Source entity UUID"},
				"target_id": {"type": "string", "description": "Target entity UUID"},
				"type":      {"type": "string", "description": "Relationship type (e.g. works_at, knows)"},
				"context":   {"type": "string", "description": "Optional context for the relationship"}
			},
			"required": ["source_id", "target_id", "type"]
		}`),
	}
}

func unlinkTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "unlink",
		Description: "Delete a relationship by ID",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Relationship UUID"}
			},
			"required": ["id"]
		}`),
	}
}

// --- tool handlers ---

func (s *Server) handleAddContact(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Name   string         `json:"name"`
		Email  string         `json:"email"`
		Phone  string         `json:"phone"`
		Fields map[string]any `json:"fields"`
		Tags   []string       `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.Name == "" {
		return errResult("name is required")
	}

	contact := models.NewContact(params.Name)
	contact.Email = params.Email
	contact.Phone = params.Phone
	if params.Fields != nil {
		contact.Fields = params.Fields
	}
	if params.Tags != nil {
		contact.Tags = params.Tags
	}

	if err := s.store.CreateContact(contact); err != nil {
		return errResult(fmt.Sprintf("create contact: %v", err))
	}
	return jsonResult(contact)
}

func (s *Server) handleListContacts(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Tag    *string `json:"tag"`
		Search string  `json:"search"`
		Limit  int     `json:"limit"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	contacts, err := s.store.ListContacts(&storage.ContactFilter{
		Tag:    params.Tag,
		Search: params.Search,
		Limit:  limit,
	})
	if err != nil {
		return errResult(fmt.Sprintf("list contacts: %v", err))
	}
	return jsonResult(contacts)
}

func (s *Server) handleGetContact(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	contact, err := s.resolveContact(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("get contact: %v", err))
	}
	return jsonResult(contact)
}

type updateContactParams struct {
	ID     string          `json:"id"`
	Name   *string         `json:"name"`
	Email  *string         `json:"email"`
	Phone  *string         `json:"phone"`
	Fields map[string]any  `json:"fields"`
	Tags   json.RawMessage `json:"tags"`
}

func (s *Server) handleUpdateContact(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params updateContactParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	contact, err := s.resolveContact(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("get contact: %v", err))
	}

	before, err := history.SnapshotContact(contact)
	if err != nil {
		return errResult(fmt.Sprintf("snapshot contact: %v", err))
	}
	candidate := contactUpdateCandidate(contact)
	if err := applyContactUpdate(&candidate, params); err != nil {
		return errResult(err.Error())
	}

	after, err := history.SnapshotContact(&candidate)
	if err != nil {
		return errResult(fmt.Sprintf("snapshot updated contact: %v", err))
	}
	changed, err := history.ChangedFields(history.EntityContact, before, after)
	if err != nil {
		return errResult(fmt.Sprintf("compare contact snapshots: %v", err))
	}
	if len(changed) == 0 {
		return jsonResult(contact)
	}

	candidate.Touch()
	if err := s.store.UpdateContact(&candidate); err != nil {
		return errResult(fmt.Sprintf("update contact: %v", err))
	}
	return jsonResult(&candidate)
}

func (s *Server) handleDeleteContact(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	contact, err := s.resolveContact(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("get contact: %v", err))
	}

	if err := s.store.DeleteContact(contact.ID); err != nil {
		return errResult(fmt.Sprintf("delete contact: %v", err))
	}
	return textResult(fmt.Sprintf("deleted contact %s (%s)", contact.Name, contact.ID))
}

func (s *Server) handleAddCompany(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Name   string         `json:"name"`
		Domain string         `json:"domain"`
		Fields map[string]any `json:"fields"`
		Tags   []string       `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.Name == "" {
		return errResult("name is required")
	}

	company := models.NewCompany(params.Name)
	company.Domain = params.Domain
	if params.Fields != nil {
		company.Fields = params.Fields
	}
	if params.Tags != nil {
		company.Tags = params.Tags
	}

	if err := s.store.CreateCompany(company); err != nil {
		return errResult(fmt.Sprintf("create company: %v", err))
	}
	return jsonResult(company)
}

func (s *Server) handleListCompanies(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Tag    *string `json:"tag"`
		Search string  `json:"search"`
		Limit  int     `json:"limit"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	companies, err := s.store.ListCompanies(&storage.CompanyFilter{
		Tag:    params.Tag,
		Search: params.Search,
		Limit:  limit,
	})
	if err != nil {
		return errResult(fmt.Sprintf("list companies: %v", err))
	}
	return jsonResult(companies)
}

func (s *Server) handleGetCompany(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	company, err := s.resolveCompany(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("get company: %v", err))
	}
	return jsonResult(company)
}

type updateCompanyParams struct {
	ID     string          `json:"id"`
	Name   *string         `json:"name"`
	Domain *string         `json:"domain"`
	Fields map[string]any  `json:"fields"`
	Tags   json.RawMessage `json:"tags"`
}

func (s *Server) handleUpdateCompany(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params updateCompanyParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	company, err := s.resolveCompany(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("get company: %v", err))
	}

	before, err := history.SnapshotCompany(company)
	if err != nil {
		return errResult(fmt.Sprintf("snapshot company: %v", err))
	}
	candidate := companyUpdateCandidate(company)
	if err := applyCompanyUpdate(&candidate, params); err != nil {
		return errResult(err.Error())
	}

	after, err := history.SnapshotCompany(&candidate)
	if err != nil {
		return errResult(fmt.Sprintf("snapshot updated company: %v", err))
	}
	changed, err := history.ChangedFields(history.EntityCompany, before, after)
	if err != nil {
		return errResult(fmt.Sprintf("compare company snapshots: %v", err))
	}
	if len(changed) == 0 {
		return jsonResult(company)
	}

	candidate.Touch()
	if err := s.store.UpdateCompany(&candidate); err != nil {
		return errResult(fmt.Sprintf("update company: %v", err))
	}
	return jsonResult(&candidate)
}

func contactUpdateCandidate(contact *models.Contact) models.Contact {
	candidate := *contact
	candidate.Fields = make(map[string]any, len(contact.Fields))
	for key, value := range contact.Fields {
		candidate.Fields[key] = value
	}
	candidate.Tags = append([]string{}, contact.Tags...)
	return candidate
}

func applyContactUpdate(candidate *models.Contact, params updateContactParams) error {
	if params.Name != nil {
		candidate.Name = *params.Name
	}
	if params.Email != nil {
		candidate.Email = *params.Email
	}
	if params.Phone != nil {
		candidate.Phone = *params.Phone
	}
	for key, value := range params.Fields {
		candidate.Fields[key] = value
	}
	if params.Tags == nil {
		return nil
	}
	var tags []string
	if err := json.Unmarshal(params.Tags, &tags); err != nil {
		return fmt.Errorf("invalid tags: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}
	candidate.Tags = tags
	return nil
}

func companyUpdateCandidate(company *models.Company) models.Company {
	candidate := *company
	candidate.Fields = make(map[string]any, len(company.Fields))
	for key, value := range company.Fields {
		candidate.Fields[key] = value
	}
	candidate.Tags = append([]string{}, company.Tags...)
	return candidate
}

func applyCompanyUpdate(candidate *models.Company, params updateCompanyParams) error {
	if params.Name != nil {
		candidate.Name = *params.Name
	}
	if params.Domain != nil {
		candidate.Domain = *params.Domain
	}
	for key, value := range params.Fields {
		candidate.Fields[key] = value
	}
	if params.Tags == nil {
		return nil
	}
	var tags []string
	if err := json.Unmarshal(params.Tags, &tags); err != nil {
		return fmt.Errorf("invalid tags: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}
	candidate.Tags = tags
	return nil
}

func (s *Server) handleDeleteCompany(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	company, err := s.resolveCompany(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("get company: %v", err))
	}

	if err := s.store.DeleteCompany(company.ID); err != nil {
		return errResult(fmt.Sprintf("delete company: %v", err))
	}
	return textResult(fmt.Sprintf("deleted company %s (%s)", company.Name, company.ID))
}

func (s *Server) handleLink(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		SourceID string `json:"source_id"`
		TargetID string `json:"target_id"`
		Type     string `json:"type"`
		Context  string `json:"context"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.SourceID == "" || params.TargetID == "" || params.Type == "" {
		return errResult("source_id, target_id, and type are all required")
	}

	sourceID, err := uuid.Parse(params.SourceID)
	if err != nil {
		return errResult(fmt.Sprintf("invalid source_id: %v", err))
	}
	targetID, err := uuid.Parse(params.TargetID)
	if err != nil {
		return errResult(fmt.Sprintf("invalid target_id: %v", err))
	}

	rel := models.NewRelationship(sourceID, targetID, params.Type, params.Context)
	if err := s.store.CreateRelationship(rel); err != nil {
		return errResult(fmt.Sprintf("create relationship: %v", err))
	}
	return jsonResult(rel)
}

func (s *Server) handleUnlink(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if params.ID == "" {
		return errResult("id is required")
	}

	id, err := uuid.Parse(params.ID)
	if err != nil {
		return errResult(fmt.Sprintf("invalid id: %v", err))
	}

	if err := s.store.DeleteRelationship(id); err != nil {
		return errResult(fmt.Sprintf("delete relationship: %v", err))
	}
	return textResult(fmt.Sprintf("deleted relationship %s", id))
}
