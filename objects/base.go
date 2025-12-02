// ABOUTME: Core Office OS object types and structures
// ABOUTME: Defines BaseObject and ACLEntry for all object kinds
package objects

import (
	"time"

	"github.com/google/uuid"
)

// BaseObject represents the core structure for all Office OS objects.
type BaseObject struct {
	ID        uuid.UUID              `json:"id"`
	Kind      string                 `json:"kind"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	CreatedBy uuid.UUID              `json:"created_by"`
	ACL       []ACLEntry             `json:"acl"`
	Tags      []string               `json:"tags,omitempty"`
	Fields    map[string]interface{} `json:"fields"`
}

// ACLEntry represents an access control entry.
type ACLEntry struct {
	ActorID uuid.UUID `json:"actorId"`
	Role    string    `json:"role"`
}

// Object kinds.
const (
	KindUser         = "user"
	KindRecord       = "record"
	KindTask         = "task"
	KindEvent        = "event"
	KindMessage      = "message"
	KindActivity     = "activity"
	KindNotification = "notification"
)

// ACL roles.
const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)
