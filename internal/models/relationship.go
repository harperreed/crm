// ABOUTME: Relationship model representing a connection between two entities in the CRM.
// ABOUTME: Links a source entity to a target entity with a type and optional context.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Relationship represents a directed link between two CRM entities.
type Relationship struct {
	ID        uuid.UUID
	SourceID  uuid.UUID
	TargetID  uuid.UUID
	Type      string
	Context   string
	CreatedAt time.Time
}

// NewRelationship creates a Relationship linking sourceID to targetID
// with the given type and context, generating a UUID and setting CreatedAt.
func NewRelationship(sourceID, targetID uuid.UUID, relType, context string) *Relationship {
	return &Relationship{
		ID:        uuid.New(),
		SourceID:  sourceID,
		TargetID:  targetID,
		Type:      relType,
		Context:   context,
		CreatedAt: time.Now(),
	}
}
