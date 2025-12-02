// ABOUTME: Office OS foundation types and interfaces
// ABOUTME: Defines BaseObject and core object operations
package officeos

import (
	"encoding/json"
	"time"
)

// ObjectKind represents the type of object in the Office OS.
type ObjectKind string

const (
	KindUser         ObjectKind = "user"
	KindRecord       ObjectKind = "record"
	KindTask         ObjectKind = "task"
	KindEvent        ObjectKind = "event"
	KindMessage      ObjectKind = "message"
	KindActivity     ObjectKind = "activity"
	KindNotification ObjectKind = "notification"
)

// ACLEntry represents an access control entry.
type ACLEntry struct {
	ActorID string `json:"actorId"`
	Role    string `json:"role"`
}

// BaseObject is the foundation for all Office OS objects.
type BaseObject struct {
	ID        string      `json:"id"`
	Kind      ObjectKind  `json:"kind"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	CreatedBy string      `json:"created_by"`
	ACL       []ACLEntry  `json:"acl"`
	Tags      []string    `json:"tags,omitempty"`
	Fields    interface{} `json:"fields"`
}

// FieldsAsJSON marshals the fields to JSON for storage.
func (b *BaseObject) FieldsAsJSON() (string, error) {
	bytes, err := json.Marshal(b.Fields)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ACLAsJSON marshals the ACL to JSON for storage.
func (b *BaseObject) ACLAsJSON() (string, error) {
	bytes, err := json.Marshal(b.ACL)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// TagsAsJSON marshals the tags to JSON for storage.
func (b *BaseObject) TagsAsJSON() (string, error) {
	if len(b.Tags) == 0 {
		return "[]", nil
	}
	bytes, err := json.Marshal(b.Tags)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ObjectStore defines the interface for storing and retrieving objects.
type ObjectStore interface {
	Create(obj *BaseObject) error
	Get(id string) (*BaseObject, error)
	Update(obj *BaseObject) error
	Delete(id string) error
	Query(kind ObjectKind, filters map[string]interface{}) ([]*BaseObject, error)
}

// ActivityHooks defines the interface for activity generation hooks.
type ActivityHooks interface {
	OnCreate(obj *BaseObject) error
	OnUpdate(oldObj, newObj *BaseObject) error
	OnDelete(obj *BaseObject) error
}
