// ABOUTME: Company model representing an organization in the CRM.
// ABOUTME: Provides a constructor and Touch method for timestamp management.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Company represents an organization tracked in the CRM.
type Company struct {
	ID        uuid.UUID
	Name      string         // required
	Domain    string         // optional
	Fields    map[string]any // flexible key-value pairs
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCompany creates a Company with the given name, generating a UUID
// and initializing Fields, Tags, and timestamps.
func NewCompany(name string) *Company {
	now := time.Now()
	return &Company{
		ID:        uuid.New(),
		Name:      name,
		Fields:    make(map[string]any),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Touch updates the UpdatedAt timestamp to the current time.
func (c *Company) Touch() {
	c.UpdatedAt = time.Now()
}
