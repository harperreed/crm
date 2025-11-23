// ABOUTME: Relationship MCP tool handlers
// ABOUTME: Implements link_contacts, find_contact_relationships, and remove_relationship tools
package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RelationshipHandlers struct {
	db *sql.DB
}

func NewRelationshipHandlers(database *sql.DB) *RelationshipHandlers {
	return &RelationshipHandlers{db: database}
}

type LinkContactsInput struct {
	ContactID1       string `json:"contact_id_1" jsonschema:"First contact ID (required)"`
	ContactID2       string `json:"contact_id_2" jsonschema:"Second contact ID (required)"`
	RelationshipType string `json:"relationship_type,omitempty" jsonschema:"Type of relationship (e.g., colleague, friend, saw_together)"`
	Context          string `json:"context,omitempty" jsonschema:"Description of how they're connected"`
}

type RelationshipOutput struct {
	ID               string `json:"id"`
	ContactID1       string `json:"contact_id_1"`
	ContactID2       string `json:"contact_id_2"`
	RelationshipType string `json:"relationship_type,omitempty"`
	Context          string `json:"context,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func (h *RelationshipHandlers) LinkContacts(_ context.Context, request *mcp.CallToolRequest, input LinkContactsInput) (*mcp.CallToolResult, RelationshipOutput, error) {
	if input.ContactID1 == "" {
		return nil, RelationshipOutput{}, fmt.Errorf("contact_id_1 is required")
	}

	if input.ContactID2 == "" {
		return nil, RelationshipOutput{}, fmt.Errorf("contact_id_2 is required")
	}

	contactID1, err := uuid.Parse(input.ContactID1)
	if err != nil {
		return nil, RelationshipOutput{}, fmt.Errorf("invalid contact_id_1: %w", err)
	}

	contactID2, err := uuid.Parse(input.ContactID2)
	if err != nil {
		return nil, RelationshipOutput{}, fmt.Errorf("invalid contact_id_2: %w", err)
	}

	relationship := &models.Relationship{
		ContactID1:       contactID1,
		ContactID2:       contactID2,
		RelationshipType: input.RelationshipType,
		Context:          input.Context,
	}

	if err := db.CreateRelationship(h.db, relationship); err != nil {
		return nil, RelationshipOutput{}, fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil, relationshipToOutput(relationship), nil
}

type FindContactRelationshipsInput struct {
	ContactID        string `json:"contact_id" jsonschema:"Contact ID (required)"`
	RelationshipType string `json:"relationship_type,omitempty" jsonschema:"Filter by relationship type"`
}

type ContactBriefOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RelationshipWithContactsOutput struct {
	ID               string             `json:"id"`
	Contact1         ContactBriefOutput `json:"contact_1"`
	Contact2         ContactBriefOutput `json:"contact_2"`
	RelationshipType string             `json:"relationship_type,omitempty"`
	Context          string             `json:"context,omitempty"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
}

type FindContactRelationshipsOutput struct {
	Relationships []RelationshipWithContactsOutput `json:"relationships"`
}

func (h *RelationshipHandlers) FindContactRelationships(_ context.Context, request *mcp.CallToolRequest, input FindContactRelationshipsInput) (*mcp.CallToolResult, FindContactRelationshipsOutput, error) {
	if input.ContactID == "" {
		return nil, FindContactRelationshipsOutput{}, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(input.ContactID)
	if err != nil {
		return nil, FindContactRelationshipsOutput{}, fmt.Errorf("invalid contact_id: %w", err)
	}

	relationships, err := db.FindContactRelationships(h.db, contactID, input.RelationshipType)
	if err != nil {
		return nil, FindContactRelationshipsOutput{}, fmt.Errorf("failed to find relationships: %w", err)
	}

	result := make([]RelationshipWithContactsOutput, len(relationships))
	for i, rel := range relationships {
		// Get contact details for both contacts
		contact1, err := db.GetContact(h.db, rel.ContactID1)
		if err != nil {
			return nil, FindContactRelationshipsOutput{}, fmt.Errorf("failed to get contact 1: %w", err)
		}

		contact2, err := db.GetContact(h.db, rel.ContactID2)
		if err != nil {
			return nil, FindContactRelationshipsOutput{}, fmt.Errorf("failed to get contact 2: %w", err)
		}

		result[i] = RelationshipWithContactsOutput{
			ID: rel.ID.String(),
			Contact1: ContactBriefOutput{
				ID:   rel.ContactID1.String(),
				Name: contact1.Name,
			},
			Contact2: ContactBriefOutput{
				ID:   rel.ContactID2.String(),
				Name: contact2.Name,
			},
			RelationshipType: rel.RelationshipType,
			Context:          rel.Context,
			CreatedAt:        rel.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        rel.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return nil, FindContactRelationshipsOutput{Relationships: result}, nil
}

type RemoveRelationshipInput struct {
	RelationshipID string `json:"relationship_id" jsonschema:"Relationship ID (required)"`
}

type RemoveRelationshipOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *RelationshipHandlers) RemoveRelationship(_ context.Context, request *mcp.CallToolRequest, input RemoveRelationshipInput) (*mcp.CallToolResult, RemoveRelationshipOutput, error) {
	if input.RelationshipID == "" {
		return nil, RemoveRelationshipOutput{}, fmt.Errorf("relationship_id is required")
	}

	relationshipID, err := uuid.Parse(input.RelationshipID)
	if err != nil {
		return nil, RemoveRelationshipOutput{}, fmt.Errorf("invalid relationship_id: %w", err)
	}

	if err := db.DeleteRelationship(h.db, relationshipID); err != nil {
		return nil, RemoveRelationshipOutput{}, fmt.Errorf("failed to delete relationship: %w", err)
	}

	return nil, RemoveRelationshipOutput{
		Success: true,
		Message: "Relationship deleted successfully",
	}, nil
}

type UpdateRelationshipInput struct {
	RelationshipID   string `json:"relationship_id" jsonschema:"Relationship ID (required)"`
	RelationshipType string `json:"relationship_type,omitempty" jsonschema:"Updated relationship type"`
	Context          string `json:"context,omitempty" jsonschema:"Updated relationship context"`
}

type UpdateRelationshipOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *RelationshipHandlers) UpdateRelationship(_ context.Context, request *mcp.CallToolRequest, input UpdateRelationshipInput) (*mcp.CallToolResult, UpdateRelationshipOutput, error) {
	if input.RelationshipID == "" {
		return nil, UpdateRelationshipOutput{}, fmt.Errorf("relationship_id is required")
	}

	relationshipID, err := uuid.Parse(input.RelationshipID)
	if err != nil {
		return nil, UpdateRelationshipOutput{}, fmt.Errorf("invalid relationship_id: %w", err)
	}

	if err := db.UpdateRelationship(h.db, relationshipID, input.RelationshipType, input.Context); err != nil {
		return nil, UpdateRelationshipOutput{}, fmt.Errorf("failed to update relationship: %w", err)
	}

	return nil, UpdateRelationshipOutput{
		Success: true,
		Message: "Relationship updated successfully",
	}, nil
}

func relationshipToOutput(relationship *models.Relationship) RelationshipOutput {
	return RelationshipOutput{
		ID:               relationship.ID.String(),
		ContactID1:       relationship.ContactID1.String(),
		ContactID2:       relationship.ContactID2.String(),
		RelationshipType: relationship.RelationshipType,
		Context:          relationship.Context,
		CreatedAt:        relationship.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        relationship.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Legacy map-based functions for tests
func (h *RelationshipHandlers) LinkContacts_Legacy(args map[string]interface{}) (interface{}, error) {
	contactID1Str, ok := args["contact_id_1"].(string)
	if !ok || contactID1Str == "" {
		return nil, fmt.Errorf("contact_id_1 is required")
	}

	contactID2Str, ok := args["contact_id_2"].(string)
	if !ok || contactID2Str == "" {
		return nil, fmt.Errorf("contact_id_2 is required")
	}

	contactID1, err := uuid.Parse(contactID1Str)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id_1: %w", err)
	}

	contactID2, err := uuid.Parse(contactID2Str)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id_2: %w", err)
	}

	relationship := &models.Relationship{
		ContactID1: contactID1,
		ContactID2: contactID2,
	}

	if relationshipType, ok := args["relationship_type"].(string); ok {
		relationship.RelationshipType = relationshipType
	}

	if context, ok := args["context"].(string); ok {
		relationship.Context = context
	}

	if err := db.CreateRelationship(h.db, relationship); err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	return relationshipToMap(relationship), nil
}

func (h *RelationshipHandlers) FindContactRelationships_Legacy(args map[string]interface{}) (interface{}, error) {
	contactIDStr, ok := args["contact_id"].(string)
	if !ok || contactIDStr == "" {
		return nil, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id: %w", err)
	}

	relationshipType := ""
	if rt, ok := args["relationship_type"].(string); ok {
		relationshipType = rt
	}

	relationships, err := db.FindContactRelationships(h.db, contactID, relationshipType)
	if err != nil {
		return nil, fmt.Errorf("failed to find relationships: %w", err)
	}

	result := make([]map[string]interface{}, len(relationships))
	for i, rel := range relationships {
		// Get contact details for both contacts
		contact1, err := db.GetContact(h.db, rel.ContactID1)
		if err != nil {
			return nil, fmt.Errorf("failed to get contact 1: %w", err)
		}

		contact2, err := db.GetContact(h.db, rel.ContactID2)
		if err != nil {
			return nil, fmt.Errorf("failed to get contact 2: %w", err)
		}

		result[i] = map[string]interface{}{
			"id": rel.ID.String(),
			"contact_1": map[string]interface{}{
				"id":   rel.ContactID1.String(),
				"name": contact1.Name,
			},
			"contact_2": map[string]interface{}{
				"id":   rel.ContactID2.String(),
				"name": contact2.Name,
			},
			"relationship_type": rel.RelationshipType,
			"context":           rel.Context,
			"created_at":        rel.CreatedAt,
			"updated_at":        rel.UpdatedAt,
		}
	}

	return result, nil
}

func (h *RelationshipHandlers) RemoveRelationship_Legacy(args map[string]interface{}) (interface{}, error) {
	relationshipIDStr, ok := args["relationship_id"].(string)
	if !ok || relationshipIDStr == "" {
		return nil, fmt.Errorf("relationship_id is required")
	}

	relationshipID, err := uuid.Parse(relationshipIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid relationship_id: %w", err)
	}

	if err := db.DeleteRelationship(h.db, relationshipID); err != nil {
		return nil, fmt.Errorf("failed to delete relationship: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Relationship deleted successfully",
	}, nil
}

func relationshipToMap(relationship *models.Relationship) map[string]interface{} {
	return map[string]interface{}{
		"id":                relationship.ID.String(),
		"contact_id_1":      relationship.ContactID1.String(),
		"contact_id_2":      relationship.ContactID2.String(),
		"relationship_type": relationship.RelationshipType,
		"context":           relationship.Context,
		"created_at":        relationship.CreatedAt,
		"updated_at":        relationship.UpdatedAt,
	}
}
