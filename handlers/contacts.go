// ABOUTME: Contact MCP tool handlers
// ABOUTME: Implements add_contact, find_contacts, update_contact, and log_contact_interaction tools
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ContactHandlers struct {
	db *sql.DB
}

func NewContactHandlers(database *sql.DB) *ContactHandlers {
	return &ContactHandlers{db: database}
}

type AddContactInput struct {
	Name        string `json:"name" jsonschema:"Contact name (required)"`
	Email       string `json:"email,omitempty" jsonschema:"Contact email address"`
	Phone       string `json:"phone,omitempty" jsonschema:"Contact phone number"`
	CompanyName string `json:"company_name,omitempty" jsonschema:"Company name (will be looked up or created)"`
	Notes       string `json:"notes,omitempty" jsonschema:"Additional notes about the contact"`
}

type ContactOutput struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Email           string  `json:"email,omitempty"`
	Phone           string  `json:"phone,omitempty"`
	CompanyID       *string `json:"company_id,omitempty"`
	Notes           string  `json:"notes,omitempty"`
	LastContactedAt *string `json:"last_contacted_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func (h *ContactHandlers) AddContact(_ context.Context, request *mcp.CallToolRequest, input AddContactInput) (*mcp.CallToolResult, ContactOutput, error) {
	if input.Name == "" {
		return nil, ContactOutput{}, fmt.Errorf("name is required")
	}

	contact := &models.Contact{
		Name:  input.Name,
		Email: input.Email,
		Phone: input.Phone,
		Notes: input.Notes,
	}

	// Handle company lookup/creation if company_name provided
	if input.CompanyName != "" {
		company, err := db.FindCompanyByName(h.db, input.CompanyName)
		if err != nil {
			return nil, ContactOutput{}, fmt.Errorf("failed to lookup company: %w", err)
		}

		if company == nil {
			// Create new company
			company = &models.Company{
				Name: input.CompanyName,
			}
			if err := db.CreateCompany(h.db, company); err != nil {
				return nil, ContactOutput{}, fmt.Errorf("failed to create company: %w", err)
			}
		}

		contact.CompanyID = &company.ID
	}

	if err := db.CreateContact(h.db, contact); err != nil {
		return nil, ContactOutput{}, fmt.Errorf("failed to create contact: %w", err)
	}

	return nil, contactToOutput(contact), nil
}

type FindContactsInput struct {
	Query     string `json:"query,omitempty" jsonschema:"Search query (searches name and email)"`
	CompanyID string `json:"company_id,omitempty" jsonschema:"Filter by company ID"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of results (default 10)"`
}

type FindContactsOutput struct {
	Contacts []ContactOutput `json:"contacts"`
}

func (h *ContactHandlers) FindContacts(_ context.Context, request *mcp.CallToolRequest, input FindContactsInput) (*mcp.CallToolResult, FindContactsOutput, error) {
	query := input.Query
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	var companyID *uuid.UUID
	if input.CompanyID != "" {
		cid, err := uuid.Parse(input.CompanyID)
		if err != nil {
			return nil, FindContactsOutput{}, fmt.Errorf("invalid company_id: %w", err)
		}
		companyID = &cid
	}

	contacts, err := db.FindContacts(h.db, query, companyID, limit)
	if err != nil {
		return nil, FindContactsOutput{}, fmt.Errorf("failed to find contacts: %w", err)
	}

	result := make([]ContactOutput, len(contacts))
	for i, contact := range contacts {
		result[i] = contactToOutput(&contact)
	}

	return nil, FindContactsOutput{Contacts: result}, nil
}

type UpdateContactInput struct {
	ID    string `json:"id" jsonschema:"Contact ID (required)"`
	Name  string `json:"name,omitempty" jsonschema:"Updated contact name"`
	Email string `json:"email,omitempty" jsonschema:"Updated email address"`
	Phone string `json:"phone,omitempty" jsonschema:"Updated phone number"`
	Notes string `json:"notes,omitempty" jsonschema:"Updated notes"`
}

func (h *ContactHandlers) UpdateContact(_ context.Context, request *mcp.CallToolRequest, input UpdateContactInput) (*mcp.CallToolResult, ContactOutput, error) {
	if input.ID == "" {
		return nil, ContactOutput{}, fmt.Errorf("id is required")
	}

	contactID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, ContactOutput{}, fmt.Errorf("invalid id: %w", err)
	}

	contact, err := db.GetContact(h.db, contactID)
	if err != nil {
		return nil, ContactOutput{}, fmt.Errorf("failed to get contact: %w", err)
	}
	if contact == nil {
		return nil, ContactOutput{}, fmt.Errorf("contact not found")
	}

	// Update fields if provided
	if input.Name != "" {
		contact.Name = input.Name
	}
	if input.Email != "" {
		contact.Email = input.Email
	}
	if input.Phone != "" {
		contact.Phone = input.Phone
	}
	if input.Notes != "" {
		contact.Notes = input.Notes
	}

	if err := db.UpdateContact(h.db, contact); err != nil {
		return nil, ContactOutput{}, fmt.Errorf("failed to update contact: %w", err)
	}

	return nil, contactToOutput(contact), nil
}

type LogContactInteractionInput struct {
	ContactID       string `json:"contact_id" jsonschema:"Contact ID (required)"`
	Note            string `json:"note,omitempty" jsonschema:"Note about the interaction"`
	InteractionDate string `json:"interaction_date,omitempty" jsonschema:"Date of interaction (ISO 8601 format, defaults to now)"`
}

func (h *ContactHandlers) LogContactInteraction(_ context.Context, request *mcp.CallToolRequest, input LogContactInteractionInput) (*mcp.CallToolResult, ContactOutput, error) {
	if input.ContactID == "" {
		return nil, ContactOutput{}, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(input.ContactID)
	if err != nil {
		return nil, ContactOutput{}, fmt.Errorf("invalid contact_id: %w", err)
	}

	contact, err := db.GetContact(h.db, contactID)
	if err != nil {
		return nil, ContactOutput{}, fmt.Errorf("failed to get contact: %w", err)
	}
	if contact == nil {
		return nil, ContactOutput{}, fmt.Errorf("contact not found")
	}

	// Parse interaction date or use current time
	interactionTime := time.Now()
	if input.InteractionDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, input.InteractionDate)
		if err != nil {
			return nil, ContactOutput{}, fmt.Errorf("invalid interaction_date format (use ISO 8601/RFC3339): %w", err)
		}
		interactionTime = parsedTime
	}

	// Update last_contacted_at
	if err := db.UpdateContactLastContacted(h.db, contactID, interactionTime); err != nil {
		return nil, ContactOutput{}, fmt.Errorf("failed to update last contacted: %w", err)
	}

	// Append note if provided
	if input.Note != "" {
		contact, err = db.GetContact(h.db, contactID)
		if err != nil {
			return nil, ContactOutput{}, fmt.Errorf("failed to get contact: %w", err)
		}

		timestamp := interactionTime.Format("2006-01-02 15:04:05")
		noteEntry := fmt.Sprintf("[%s] %s", timestamp, input.Note)
		if contact.Notes != "" {
			contact.Notes = contact.Notes + "\n" + noteEntry
		} else {
			contact.Notes = noteEntry
		}

		if err := db.UpdateContact(h.db, contact); err != nil {
			return nil, ContactOutput{}, fmt.Errorf("failed to update notes: %w", err)
		}
	}

	// Reload contact to get updated values
	contact, err = db.GetContact(h.db, contactID)
	if err != nil {
		return nil, ContactOutput{}, fmt.Errorf("failed to reload contact: %w", err)
	}

	return nil, contactToOutput(contact), nil
}

func contactToOutput(contact *models.Contact) ContactOutput {
	output := ContactOutput{
		ID:        contact.ID.String(),
		Name:      contact.Name,
		Email:     contact.Email,
		Phone:     contact.Phone,
		Notes:     contact.Notes,
		CreatedAt: contact.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: contact.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if contact.CompanyID != nil {
		cid := contact.CompanyID.String()
		output.CompanyID = &cid
	}

	if contact.LastContactedAt != nil {
		lca := contact.LastContactedAt.Format("2006-01-02T15:04:05Z07:00")
		output.LastContactedAt = &lca
	}

	return output
}

// Legacy map-based functions for tests
func (h *ContactHandlers) AddContact_Legacy(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	contact := &models.Contact{
		Name: name,
	}

	if email, ok := args["email"].(string); ok {
		contact.Email = email
	}

	if phone, ok := args["phone"].(string); ok {
		contact.Phone = phone
	}

	if notes, ok := args["notes"].(string); ok {
		contact.Notes = notes
	}

	// Handle company lookup/creation if company_name provided
	if companyName, ok := args["company_name"].(string); ok && companyName != "" {
		company, err := db.FindCompanyByName(h.db, companyName)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup company: %w", err)
		}

		if company == nil {
			// Create new company
			company = &models.Company{
				Name: companyName,
			}
			if err := db.CreateCompany(h.db, company); err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}
		}

		contact.CompanyID = &company.ID
	}

	if err := db.CreateContact(h.db, contact); err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	return contactToMap(contact), nil
}

func (h *ContactHandlers) FindContacts_Legacy(args map[string]interface{}) (interface{}, error) {
	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var companyID *uuid.UUID
	if cid, ok := args["company_id"].(string); ok && cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			return nil, fmt.Errorf("invalid company_id: %w", err)
		}
		companyID = &id
	}

	contacts, err := db.FindContacts(h.db, query, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find contacts: %w", err)
	}

	result := make([]map[string]interface{}, len(contacts))
	for i, contact := range contacts {
		result[i] = contactToMap(&contact)
	}

	return result, nil
}

func (h *ContactHandlers) UpdateContact_Legacy(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["id"].(string)
	if !ok || idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	contactID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	contact, err := db.GetContact(h.db, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	if contact == nil {
		return nil, fmt.Errorf("contact not found")
	}

	// Update fields if provided
	if name, ok := args["name"].(string); ok && name != "" {
		contact.Name = name
	}
	if email, ok := args["email"].(string); ok {
		contact.Email = email
	}
	if phone, ok := args["phone"].(string); ok {
		contact.Phone = phone
	}
	if notes, ok := args["notes"].(string); ok {
		contact.Notes = notes
	}

	if err := db.UpdateContact(h.db, contact); err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	return contactToMap(contact), nil
}

func (h *ContactHandlers) LogContactInteraction_Legacy(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["contact_id"].(string)
	if !ok || idStr == "" {
		return nil, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id: %w", err)
	}

	contact, err := db.GetContact(h.db, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	if contact == nil {
		return nil, fmt.Errorf("contact not found")
	}

	// Parse interaction date or use current time
	interactionTime := time.Now()
	if dateStr, ok := args["interaction_date"].(string); ok && dateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid interaction_date format (use ISO 8601/RFC3339): %w", err)
		}
		interactionTime = parsedTime
	}

	// Update last_contacted_at
	if err := db.UpdateContactLastContacted(h.db, contactID, interactionTime); err != nil {
		return nil, fmt.Errorf("failed to update last contacted: %w", err)
	}

	// Append note if provided
	if note, ok := args["note"].(string); ok && note != "" {
		contact, err = db.GetContact(h.db, contactID)
		if err != nil {
			return nil, fmt.Errorf("failed to get contact: %w", err)
		}

		timestamp := interactionTime.Format("2006-01-02 15:04:05")
		noteEntry := fmt.Sprintf("[%s] %s", timestamp, note)
		if contact.Notes != "" {
			contact.Notes = contact.Notes + "\n" + noteEntry
		} else {
			contact.Notes = noteEntry
		}

		if err := db.UpdateContact(h.db, contact); err != nil {
			return nil, fmt.Errorf("failed to update notes: %w", err)
		}
	}

	// Reload contact to get updated values
	contact, err = db.GetContact(h.db, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload contact: %w", err)
	}

	return contactToMap(contact), nil
}

func contactToMap(contact *models.Contact) map[string]interface{} {
	result := map[string]interface{}{
		"id":         contact.ID.String(),
		"name":       contact.Name,
		"email":      contact.Email,
		"phone":      contact.Phone,
		"notes":      contact.Notes,
		"created_at": contact.CreatedAt,
		"updated_at": contact.UpdatedAt,
	}

	if contact.CompanyID != nil {
		result["company_id"] = contact.CompanyID.String()
	}

	if contact.LastContactedAt != nil {
		result["last_contacted_at"] = *contact.LastContactedAt
	}

	return result
}
