// ABOUTME: Deal MCP tool handlers
// ABOUTME: Implements create_deal, update_deal, and add_deal_note tools
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DealHandlers struct {
	client *charm.Client
}

func NewDealHandlers(client *charm.Client) *DealHandlers {
	return &DealHandlers{client: client}
}

type CreateDealInput struct {
	Title             string `json:"title" jsonschema:"Deal title (required)"`
	Amount            int64  `json:"amount,omitempty" jsonschema:"Deal amount in cents"`
	Currency          string `json:"currency,omitempty" jsonschema:"Currency code (default USD)"`
	Stage             string `json:"stage,omitempty" jsonschema:"Deal stage: prospecting, qualification, proposal, negotiation, closed_won, closed_lost"`
	CompanyName       string `json:"company_name" jsonschema:"Company name (required, will be created if not found)"`
	ContactName       string `json:"contact_name,omitempty" jsonschema:"Contact name (optional)"`
	ExpectedCloseDate string `json:"expected_close_date,omitempty" jsonschema:"Expected close date in ISO 8601 format"`
	InitialNote       string `json:"initial_note,omitempty" jsonschema:"Initial note for the deal"`
}

type DealOutput struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Amount            int64   `json:"amount,omitempty"`
	Currency          string  `json:"currency"`
	Stage             string  `json:"stage"`
	CompanyID         string  `json:"company_id"`
	ContactID         *string `json:"contact_id,omitempty"`
	ExpectedCloseDate *string `json:"expected_close_date,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	LastActivityAt    string  `json:"last_activity_at"`
}

func (h *DealHandlers) CreateDeal(_ context.Context, request *mcp.CallToolRequest, input CreateDealInput) (*mcp.CallToolResult, DealOutput, error) {
	if input.Title == "" {
		return nil, DealOutput{}, fmt.Errorf("title is required")
	}
	if input.CompanyName == "" {
		return nil, DealOutput{}, fmt.Errorf("company_name is required")
	}

	// Set defaults
	currency := input.Currency
	if currency == "" {
		currency = "USD"
	}

	stage := input.Stage
	if stage == "" {
		stage = charm.StageProspecting
	}

	// Validate stage
	if !isValidStage(stage) {
		return nil, DealOutput{}, fmt.Errorf("invalid stage: %s (valid: prospecting, qualification, proposal, negotiation, closed_won, closed_lost)", stage)
	}

	// Handle company lookup/creation (required)
	company, err := h.client.FindCompanyByName(input.CompanyName)
	if err != nil {
		return nil, DealOutput{}, fmt.Errorf("failed to lookup company: %w", err)
	}

	if company == nil {
		// Create new company
		company = &charm.Company{
			Name: input.CompanyName,
		}
		if err := h.client.CreateCompany(company); err != nil {
			return nil, DealOutput{}, fmt.Errorf("failed to create company: %w", err)
		}
	}

	deal := &charm.Deal{
		Title:       input.Title,
		Amount:      input.Amount,
		Currency:    currency,
		Stage:       stage,
		CompanyID:   company.ID,
		CompanyName: company.Name,
	}

	// Handle contact lookup if provided (optional)
	if input.ContactName != "" {
		filter := &charm.ContactFilter{
			Query: input.ContactName,
			Limit: 1,
		}
		contacts, err := h.client.ListContacts(filter)
		if err != nil {
			return nil, DealOutput{}, fmt.Errorf("failed to lookup contact: %w", err)
		}

		if len(contacts) > 0 {
			deal.ContactID = &contacts[0].ID
			deal.ContactName = contacts[0].Name
		}
	}

	// Parse expected close date if provided
	if input.ExpectedCloseDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, input.ExpectedCloseDate)
		if err != nil {
			return nil, DealOutput{}, fmt.Errorf("invalid expected_close_date format (use ISO 8601/RFC3339): %w", err)
		}
		deal.ExpectedCloseDate = &parsedTime
	}

	if err := h.client.CreateDeal(deal); err != nil {
		return nil, DealOutput{}, fmt.Errorf("failed to create deal: %w", err)
	}

	// Add initial note if provided
	if input.InitialNote != "" {
		note := &charm.DealNote{
			DealID:          deal.ID,
			DealTitle:       deal.Title,
			DealCompanyName: deal.CompanyName,
			Content:         input.InitialNote,
		}
		if err := h.client.CreateDealNote(note); err != nil {
			return nil, DealOutput{}, fmt.Errorf("failed to add initial note: %w", err)
		}

		// Reload deal to get updated last_activity_at
		deal, err = h.client.GetDeal(deal.ID)
		if err != nil {
			return nil, DealOutput{}, fmt.Errorf("failed to reload deal: %w", err)
		}
	}

	return nil, dealToOutput(deal), nil
}

type UpdateDealInput struct {
	ID                string `json:"id" jsonschema:"Deal ID (required)"`
	Title             string `json:"title,omitempty" jsonschema:"Updated deal title"`
	Amount            *int64 `json:"amount,omitempty" jsonschema:"Updated deal amount in cents"`
	Currency          string `json:"currency,omitempty" jsonschema:"Updated currency code"`
	Stage             string `json:"stage,omitempty" jsonschema:"Updated deal stage"`
	ExpectedCloseDate string `json:"expected_close_date,omitempty" jsonschema:"Updated expected close date in ISO 8601 format"`
}

func (h *DealHandlers) UpdateDeal(_ context.Context, request *mcp.CallToolRequest, input UpdateDealInput) (*mcp.CallToolResult, DealOutput, error) {
	if input.ID == "" {
		return nil, DealOutput{}, fmt.Errorf("id is required")
	}

	dealID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, DealOutput{}, fmt.Errorf("invalid id: %w", err)
	}

	deal, err := h.client.GetDeal(dealID)
	if err != nil {
		return nil, DealOutput{}, fmt.Errorf("failed to get deal: %w", err)
	}

	// Update fields if provided
	if input.Title != "" {
		deal.Title = input.Title
	}
	if input.Amount != nil {
		deal.Amount = *input.Amount
	}
	if input.Currency != "" {
		deal.Currency = input.Currency
	}
	if input.Stage != "" {
		if !isValidStage(input.Stage) {
			return nil, DealOutput{}, fmt.Errorf("invalid stage: %s (valid: prospecting, qualification, proposal, negotiation, closed_won, closed_lost)", input.Stage)
		}
		deal.Stage = input.Stage
	}
	if input.ExpectedCloseDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, input.ExpectedCloseDate)
		if err != nil {
			return nil, DealOutput{}, fmt.Errorf("invalid expected_close_date format (use ISO 8601/RFC3339): %w", err)
		}
		deal.ExpectedCloseDate = &parsedTime
	}

	if err := h.client.UpdateDeal(deal); err != nil {
		return nil, DealOutput{}, fmt.Errorf("failed to update deal: %w", err)
	}

	return nil, dealToOutput(deal), nil
}

type AddDealNoteInput struct {
	DealID  string `json:"deal_id" jsonschema:"Deal ID (required)"`
	Content string `json:"content" jsonschema:"Note content (required)"`
}

type DealNoteOutput struct {
	ID        string `json:"id"`
	DealID    string `json:"deal_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func (h *DealHandlers) AddDealNote(_ context.Context, request *mcp.CallToolRequest, input AddDealNoteInput) (*mcp.CallToolResult, DealNoteOutput, error) {
	if input.DealID == "" {
		return nil, DealNoteOutput{}, fmt.Errorf("deal_id is required")
	}
	if input.Content == "" {
		return nil, DealNoteOutput{}, fmt.Errorf("content is required")
	}

	dealID, err := uuid.Parse(input.DealID)
	if err != nil {
		return nil, DealNoteOutput{}, fmt.Errorf("invalid deal_id: %w", err)
	}

	// Verify deal exists and get denormalized fields
	deal, err := h.client.GetDeal(dealID)
	if err != nil {
		return nil, DealNoteOutput{}, fmt.Errorf("failed to get deal: %w", err)
	}

	note := &charm.DealNote{
		DealID:          dealID,
		DealTitle:       deal.Title,
		DealCompanyName: deal.CompanyName,
		Content:         input.Content,
	}

	if err := h.client.CreateDealNote(note); err != nil {
		return nil, DealNoteOutput{}, fmt.Errorf("failed to add note: %w", err)
	}

	return nil, dealNoteToOutput(note), nil
}

func dealToOutput(deal *charm.Deal) DealOutput {
	output := DealOutput{
		ID:             deal.ID.String(),
		Title:          deal.Title,
		Amount:         deal.Amount,
		Currency:       deal.Currency,
		Stage:          deal.Stage,
		CompanyID:      deal.CompanyID.String(),
		CreatedAt:      deal.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      deal.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastActivityAt: deal.LastActivityAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if deal.ContactID != nil {
		cid := deal.ContactID.String()
		output.ContactID = &cid
	}

	if deal.ExpectedCloseDate != nil {
		ecd := deal.ExpectedCloseDate.Format("2006-01-02T15:04:05Z07:00")
		output.ExpectedCloseDate = &ecd
	}

	return output
}

func dealNoteToOutput(note *charm.DealNote) DealNoteOutput {
	return DealNoteOutput{
		ID:        note.ID.String(),
		DealID:    note.DealID.String(),
		Content:   note.Content,
		CreatedAt: note.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type DeleteDealInput struct {
	ID string `json:"id" jsonschema:"Deal ID (required)"`
}

type DeleteDealOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *DealHandlers) DeleteDeal(_ context.Context, request *mcp.CallToolRequest, input DeleteDealInput) (*mcp.CallToolResult, DeleteDealOutput, error) {
	if input.ID == "" {
		return nil, DeleteDealOutput{}, fmt.Errorf("id is required")
	}

	dealID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, DeleteDealOutput{}, fmt.Errorf("invalid id: %w", err)
	}

	if err := h.client.DeleteDeal(dealID); err != nil {
		return nil, DeleteDealOutput{}, fmt.Errorf("failed to delete deal: %w", err)
	}

	return nil, DeleteDealOutput{
		Success: true,
		Message: fmt.Sprintf("Deal %s deleted successfully", dealID),
	}, nil
}

func isValidStage(stage string) bool {
	validStages := []string{
		charm.StageProspecting,
		charm.StageQualification,
		charm.StageProposal,
		charm.StageNegotiation,
		charm.StageClosedWon,
		charm.StageClosedLost,
	}

	for _, valid := range validStages {
		if stage == valid {
			return true
		}
	}
	return false
}

// Legacy map-based functions for tests.
func (h *DealHandlers) CreateDeal_Legacy(args map[string]interface{}) (interface{}, error) {
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required")
	}

	companyName, ok := args["company_name"].(string)
	if !ok || companyName == "" {
		return nil, fmt.Errorf("company_name is required")
	}

	// Set defaults
	currency := "USD"
	if c, ok := args["currency"].(string); ok && c != "" {
		currency = c
	}

	stage := charm.StageProspecting
	if s, ok := args["stage"].(string); ok && s != "" {
		stage = s
	}

	// Validate stage
	if !isValidStage(stage) {
		return nil, fmt.Errorf("invalid stage: %s (valid: prospecting, qualification, proposal, negotiation, closed_won, closed_lost)", stage)
	}

	// Handle company lookup/creation (required)
	company, err := h.client.FindCompanyByName(companyName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup company: %w", err)
	}

	if company == nil {
		// Create new company
		company = &charm.Company{
			Name: companyName,
		}
		if err := h.client.CreateCompany(company); err != nil {
			return nil, fmt.Errorf("failed to create company: %w", err)
		}
	}

	deal := &charm.Deal{
		Title:       title,
		Currency:    currency,
		Stage:       stage,
		CompanyID:   company.ID,
		CompanyName: company.Name,
	}

	// Handle amount
	if amt, ok := args["amount"].(float64); ok {
		deal.Amount = int64(amt)
	}

	// Handle contact lookup if provided (optional)
	if contactName, ok := args["contact_name"].(string); ok && contactName != "" {
		filter := &charm.ContactFilter{
			Query: contactName,
			Limit: 1,
		}
		contacts, err := h.client.ListContacts(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup contact: %w", err)
		}

		if len(contacts) > 0 {
			deal.ContactID = &contacts[0].ID
			deal.ContactName = contacts[0].Name
		}
	}

	// Parse expected close date if provided
	if expectedDateStr, ok := args["expected_close_date"].(string); ok && expectedDateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, expectedDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expected_close_date format (use ISO 8601/RFC3339): %w", err)
		}
		deal.ExpectedCloseDate = &parsedTime
	}

	if err := h.client.CreateDeal(deal); err != nil {
		return nil, fmt.Errorf("failed to create deal: %w", err)
	}

	// Add initial note if provided
	if initialNote, ok := args["initial_note"].(string); ok && initialNote != "" {
		note := &charm.DealNote{
			DealID:          deal.ID,
			DealTitle:       deal.Title,
			DealCompanyName: deal.CompanyName,
			Content:         initialNote,
		}
		if err := h.client.CreateDealNote(note); err != nil {
			return nil, fmt.Errorf("failed to add initial note: %w", err)
		}

		// Reload deal to get updated last_activity_at
		deal, err = h.client.GetDeal(deal.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to reload deal: %w", err)
		}
	}

	return dealToMap(deal), nil
}

func (h *DealHandlers) UpdateDeal_Legacy(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["id"].(string)
	if !ok || idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	dealID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	deal, err := h.client.GetDeal(dealID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	// Update fields if provided
	if title, ok := args["title"].(string); ok && title != "" {
		deal.Title = title
	}
	if amt, ok := args["amount"].(float64); ok {
		deal.Amount = int64(amt)
	}
	if currency, ok := args["currency"].(string); ok && currency != "" {
		deal.Currency = currency
	}
	if stage, ok := args["stage"].(string); ok && stage != "" {
		if !isValidStage(stage) {
			return nil, fmt.Errorf("invalid stage: %s (valid: prospecting, qualification, proposal, negotiation, closed_won, closed_lost)", stage)
		}
		deal.Stage = stage
	}
	if expectedDateStr, ok := args["expected_close_date"].(string); ok && expectedDateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, expectedDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expected_close_date format (use ISO 8601/RFC3339): %w", err)
		}
		deal.ExpectedCloseDate = &parsedTime
	}

	if err := h.client.UpdateDeal(deal); err != nil {
		return nil, fmt.Errorf("failed to update deal: %w", err)
	}

	return dealToMap(deal), nil
}

func (h *DealHandlers) AddDealNote_Legacy(args map[string]interface{}) (interface{}, error) {
	dealIDStr, ok := args["deal_id"].(string)
	if !ok || dealIDStr == "" {
		return nil, fmt.Errorf("deal_id is required")
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required")
	}

	dealID, err := uuid.Parse(dealIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid deal_id: %w", err)
	}

	// Verify deal exists and get denormalized fields
	deal, err := h.client.GetDeal(dealID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	note := &charm.DealNote{
		DealID:          dealID,
		DealTitle:       deal.Title,
		DealCompanyName: deal.CompanyName,
		Content:         content,
	}

	if err := h.client.CreateDealNote(note); err != nil {
		return nil, fmt.Errorf("failed to add note: %w", err)
	}

	// Update deal's LastActivityAt - this is important for follow-up tracking
	now := time.Now()
	deal.LastActivityAt = now
	if err := h.client.UpdateDeal(deal); err != nil {
		// Return error since timestamp updates are critical for follow-up tracking
		return nil, fmt.Errorf("note created but failed to update deal timestamp: %w", err)
	}

	// Update contact's LastContactedAt if deal has a contact
	if deal.ContactID != nil {
		contact, err := h.client.GetContact(*deal.ContactID)
		if err != nil {
			// Contact might not exist anymore - log but don't fail since note was created
			// This can happen if contact was deleted after deal was created
			_ = err // Contact not found is okay, skip timestamp update
		} else if contact != nil {
			contact.LastContactedAt = &now
			if err := h.client.UpdateContact(contact); err != nil {
				return nil, fmt.Errorf("note created but failed to update contact timestamp: %w", err)
			}
		}
	}

	return dealNoteToMap(note), nil
}

func dealToMap(deal *charm.Deal) map[string]interface{} {
	result := map[string]interface{}{
		"id":               deal.ID.String(),
		"title":            deal.Title,
		"amount":           deal.Amount,
		"currency":         deal.Currency,
		"stage":            deal.Stage,
		"company_id":       deal.CompanyID.String(),
		"created_at":       deal.CreatedAt,
		"updated_at":       deal.UpdatedAt,
		"last_activity_at": deal.LastActivityAt,
	}

	if deal.ContactID != nil {
		result["contact_id"] = deal.ContactID.String()
	}

	if deal.ExpectedCloseDate != nil {
		result["expected_close_date"] = *deal.ExpectedCloseDate
	}

	return result
}

func dealNoteToMap(note *charm.DealNote) map[string]interface{} {
	return map[string]interface{}{
		"id":         note.ID.String(),
		"deal_id":    note.DealID.String(),
		"content":    note.Content,
		"created_at": note.CreatedAt,
	}
}
