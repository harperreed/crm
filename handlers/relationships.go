// ABOUTME: Relationship MCP tool handlers
// ABOUTME: Implements link_contacts, find_contact_relationships, and remove_relationship tools
package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RelationshipHandlers struct {
	client *charm.Client
}

func NewRelationshipHandlers(client *charm.Client) *RelationshipHandlers {
	return &RelationshipHandlers{client: client}
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

	// Get contact names for denormalization
	contact1, err := h.client.GetContact(contactID1)
	if err != nil {
		return nil, RelationshipOutput{}, fmt.Errorf("failed to get contact 1: %w", err)
	}

	contact2, err := h.client.GetContact(contactID2)
	if err != nil {
		return nil, RelationshipOutput{}, fmt.Errorf("failed to get contact 2: %w", err)
	}

	relationship := &charm.Relationship{
		ContactID1:       contactID1,
		ContactID2:       contactID2,
		Contact1Name:     contact1.Name,
		Contact2Name:     contact2.Name,
		RelationshipType: input.RelationshipType,
		Context:          input.Context,
	}

	if err := h.client.CreateRelationship(relationship); err != nil {
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

	relationships, err := h.client.ListRelationshipsForContact(contactID)
	if err != nil {
		return nil, FindContactRelationshipsOutput{}, fmt.Errorf("failed to find relationships: %w", err)
	}

	// Filter by relationship type if specified
	filtered := relationships
	if input.RelationshipType != "" {
		filtered = make([]*charm.Relationship, 0)
		for _, rel := range relationships {
			if rel.RelationshipType == input.RelationshipType {
				filtered = append(filtered, rel)
			}
		}
	}

	result := make([]RelationshipWithContactsOutput, len(filtered))
	for i, rel := range filtered {
		// Contact names are already denormalized in the relationship
		result[i] = RelationshipWithContactsOutput{
			ID: rel.ID.String(),
			Contact1: ContactBriefOutput{
				ID:   rel.ContactID1.String(),
				Name: rel.Contact1Name,
			},
			Contact2: ContactBriefOutput{
				ID:   rel.ContactID2.String(),
				Name: rel.Contact2Name,
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

	if err := h.client.DeleteRelationship(relationshipID); err != nil {
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

	// Get existing relationship
	rel, err := h.client.GetRelationship(relationshipID)
	if err != nil {
		return nil, UpdateRelationshipOutput{}, fmt.Errorf("failed to get relationship: %w", err)
	}

	// Update fields if provided
	if input.RelationshipType != "" {
		rel.RelationshipType = input.RelationshipType
	}
	if input.Context != "" {
		rel.Context = input.Context
	}

	if err := h.client.UpdateRelationship(rel); err != nil {
		return nil, UpdateRelationshipOutput{}, fmt.Errorf("failed to update relationship: %w", err)
	}

	return nil, UpdateRelationshipOutput{
		Success: true,
		Message: "Relationship updated successfully",
	}, nil
}

func relationshipToOutput(relationship *charm.Relationship) RelationshipOutput {
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

// Legacy map-based functions for tests.
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

	// Get contact names for denormalization
	contact1, err := h.client.GetContact(contactID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact 1: %w", err)
	}

	contact2, err := h.client.GetContact(contactID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact 2: %w", err)
	}

	relationship := &charm.Relationship{
		ContactID1:   contactID1,
		ContactID2:   contactID2,
		Contact1Name: contact1.Name,
		Contact2Name: contact2.Name,
	}

	if relationshipType, ok := args["relationship_type"].(string); ok {
		relationship.RelationshipType = relationshipType
	}

	if context, ok := args["context"].(string); ok {
		relationship.Context = context
	}

	if err := h.client.CreateRelationship(relationship); err != nil {
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

	relationships, err := h.client.ListRelationshipsForContact(contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to find relationships: %w", err)
	}

	// Filter by relationship type if specified
	relationshipType := ""
	if rt, ok := args["relationship_type"].(string); ok {
		relationshipType = rt
	}

	filtered := relationships
	if relationshipType != "" {
		filtered = make([]*charm.Relationship, 0)
		for _, rel := range relationships {
			if rel.RelationshipType == relationshipType {
				filtered = append(filtered, rel)
			}
		}
	}

	result := make([]map[string]interface{}, len(filtered))
	for i, rel := range filtered {
		// Contact names are already denormalized
		result[i] = map[string]interface{}{
			"id": rel.ID.String(),
			"contact_1": map[string]interface{}{
				"id":   rel.ContactID1.String(),
				"name": rel.Contact1Name,
			},
			"contact_2": map[string]interface{}{
				"id":   rel.ContactID2.String(),
				"name": rel.Contact2Name,
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

	if err := h.client.DeleteRelationship(relationshipID); err != nil {
		return nil, fmt.Errorf("failed to delete relationship: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Relationship deleted successfully",
	}, nil
}

func relationshipToMap(relationship *charm.Relationship) map[string]interface{} {
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
