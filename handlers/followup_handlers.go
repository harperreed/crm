// ABOUTME: MCP handlers for follow-up operations
// ABOUTME: Provides follow-up list, interaction logging, and cadence management to Claude
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type FollowupHandlers struct {
	db *repository.DB
}

func NewFollowupHandlers(db *repository.DB) *FollowupHandlers {
	return &FollowupHandlers{db: db}
}

type GetFollowupListInput struct {
	Limit       *int     `json:"limit,omitempty" jsonschema:"Maximum number of contacts to return (default 10)"`
	OverdueOnly *bool    `json:"overdue_only,omitempty" jsonschema:"Only show overdue contacts"`
	MinPriority *float64 `json:"min_priority,omitempty" jsonschema:"Minimum priority score"`
}

type GetFollowupListOutput struct {
	Followups []*repository.FollowupContact `json:"followups"`
	Count     int                           `json:"count"`
}

func (h *FollowupHandlers) GetFollowupList(_ context.Context, _ *mcp.CallToolRequest, input GetFollowupListInput) (*mcp.CallToolResult, GetFollowupListOutput, error) {
	limit := 10
	if input.Limit != nil {
		limit = *input.Limit
	}

	overdueOnly := false
	if input.OverdueOnly != nil {
		overdueOnly = *input.OverdueOnly
	}

	minPriority := 0.0
	if input.MinPriority != nil {
		minPriority = *input.MinPriority
	}

	allFollowups, err := h.db.GetFollowupList(limit)
	if err != nil {
		return nil, GetFollowupListOutput{}, fmt.Errorf("failed to get followup list: %w", err)
	}

	// Apply filters in memory
	var followups []*repository.FollowupContact
	for _, f := range allFollowups {
		if overdueOnly && f.PriorityScore <= 0 {
			continue
		}
		if minPriority > 0 && f.PriorityScore < minPriority {
			continue
		}
		followups = append(followups, f)
	}

	output := GetFollowupListOutput{
		Followups: followups,
		Count:     len(followups),
	}

	return nil, output, nil
}

type LogInteractionInput struct {
	ContactID       string  `json:"contact_id" jsonschema:"Contact ID or name (required)"`
	InteractionType string  `json:"interaction_type" jsonschema:"Type of interaction: meeting, call, email, message, or event (required)"`
	Notes           *string `json:"notes,omitempty" jsonschema:"Notes about the interaction"`
	Sentiment       *string `json:"sentiment,omitempty" jsonschema:"Sentiment: positive, neutral, or negative"`
}

type LogInteractionOutput struct {
	Success         bool    `json:"success"`
	Message         string  `json:"message"`
	InteractionID   string  `json:"interaction_id"`
	UpdatedPriority float64 `json:"updated_priority"`
}

func (h *FollowupHandlers) LogInteraction(_ context.Context, _ *mcp.CallToolRequest, input LogInteractionInput) (*mcp.CallToolResult, LogInteractionOutput, error) {
	// Resolve contact ID
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(input.ContactID)
	if err == nil {
		contactID = parsedID
	} else {
		contacts, err := h.db.ListContacts(&repository.ContactFilter{
			Query: input.ContactID,
			Limit: 10,
		})
		if err != nil {
			return nil, LogInteractionOutput{}, fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return nil, LogInteractionOutput{}, fmt.Errorf("no contact found matching: %s", input.ContactID)
		}
		contactID = contacts[0].ID
	}

	// Get contact name for denormalization
	contact, err := h.db.GetContact(contactID)
	if err != nil {
		return nil, LogInteractionOutput{}, fmt.Errorf("failed to get contact: %w", err)
	}

	interaction := &repository.InteractionLog{
		ID:              uuid.New(),
		ContactID:       contactID,
		ContactName:     contact.Name,
		InteractionType: input.InteractionType,
		Timestamp:       time.Now(),
		Sentiment:       input.Sentiment,
	}

	if input.Notes != nil {
		interaction.Notes = *input.Notes
	}

	err = h.db.CreateInteractionLog(interaction)
	if err != nil {
		return nil, LogInteractionOutput{}, fmt.Errorf("failed to log interaction: %w", err)
	}

	// Update cadence after interaction
	err = h.db.UpdateCadenceAfterInteraction(contactID, interaction.Timestamp)
	if err != nil {
		// Log but don't fail - the interaction was logged successfully
		fmt.Printf("Warning: failed to update cadence: %v\n", err)
	}

	// Get updated priority
	cadence, _ := h.db.GetContactCadence(contactID)
	priority := 0.0
	if cadence != nil {
		priority = cadence.PriorityScore
	}

	output := LogInteractionOutput{
		Success:         true,
		Message:         fmt.Sprintf("Logged %s interaction", input.InteractionType),
		InteractionID:   interaction.ID.String(),
		UpdatedPriority: priority,
	}

	return nil, output, nil
}

type SetCadenceInput struct {
	ContactID string `json:"contact_id" jsonschema:"Contact ID or name (required)"`
	Days      int    `json:"days" jsonschema:"Cadence in days (required)"`
	Strength  string `json:"strength" jsonschema:"Relationship strength: weak, medium, or strong (required)"`
}

type SetCadenceOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *FollowupHandlers) SetCadence(_ context.Context, _ *mcp.CallToolRequest, input SetCadenceInput) (*mcp.CallToolResult, SetCadenceOutput, error) {
	// Resolve contact ID
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(input.ContactID)
	if err == nil {
		contactID = parsedID
	} else {
		contacts, err := h.db.ListContacts(&repository.ContactFilter{
			Query: input.ContactID,
			Limit: 10,
		})
		if err != nil {
			return nil, SetCadenceOutput{}, fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return nil, SetCadenceOutput{}, fmt.Errorf("no contact found matching: %s", input.ContactID)
		}
		contactID = contacts[0].ID
	}

	// Get contact name for denormalization
	contact, err := h.db.GetContact(contactID)
	if err != nil {
		return nil, SetCadenceOutput{}, fmt.Errorf("failed to get contact: %w", err)
	}

	// Create or update cadence
	cadence := &repository.ContactCadence{
		ContactID:            contactID,
		ContactName:          contact.Name,
		CadenceDays:          input.Days,
		RelationshipStrength: input.Strength,
		PriorityScore:        0, // Will be recalculated
	}
	err = h.db.SaveContactCadence(cadence)
	if err != nil {
		return nil, SetCadenceOutput{}, fmt.Errorf("failed to set cadence: %w", err)
	}

	output := SetCadenceOutput{
		Success: true,
		Message: fmt.Sprintf("Set cadence to %d days (%s strength)", input.Days, input.Strength),
	}

	return nil, output, nil
}
