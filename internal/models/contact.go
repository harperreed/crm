// ABOUTME: Contact model representing a person in the CRM.
// ABOUTME: Provides a constructor and Touch method for timestamp management.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Contact represents a person tracked in the CRM.
type Contact struct {
	ID        uuid.UUID
	Name      string         // required
	Email     string         // optional
	Phone     string         // optional
	Fields    map[string]any // flexible key-value pairs
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewContact creates a Contact with the given name, generating a UUID
// and initializing Fields, Tags, and timestamps.
func NewContact(name string) *Contact {
	now := time.Now()
	return &Contact{
		ID:        uuid.New(),
		Name:      name,
		Fields:    make(map[string]any),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Touch updates the UpdatedAt timestamp to the current time.
func (c *Contact) Touch() {
	c.UpdatedAt = time.Now()
}
