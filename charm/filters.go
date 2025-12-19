// ABOUTME: Client-side filter structures for KV queries
// ABOUTME: Since KV has no SQL, we filter in memory after prefix scans

package charm

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ContactFilter defines criteria for filtering contacts.
type ContactFilter struct {
	Query     string     // Full-text search in name, email, notes
	CompanyID *uuid.UUID // Filter by company
	Limit     int        // Max results (0 = unlimited)
}

// Matches returns true if the contact matches the filter.
func (f *ContactFilter) Matches(c *Contact) bool {
	if f == nil {
		return true
	}

	// Filter by company
	if f.CompanyID != nil {
		if c.CompanyID == nil || *c.CompanyID != *f.CompanyID {
			return false
		}
	}

	// Filter by query string
	if f.Query != "" {
		q := strings.ToLower(f.Query)
		if !strings.Contains(strings.ToLower(c.Name), q) &&
			!strings.Contains(strings.ToLower(c.Email), q) &&
			!strings.Contains(strings.ToLower(c.Notes), q) &&
			!strings.Contains(strings.ToLower(c.CompanyName), q) {
			return false
		}
	}

	return true
}

// CompanyFilter defines criteria for filtering companies.
type CompanyFilter struct {
	Query    string // Full-text search in name, domain, industry, notes
	Industry string // Filter by industry
	Limit    int    // Max results (0 = unlimited)
}

// Matches returns true if the company matches the filter.
func (f *CompanyFilter) Matches(c *Company) bool {
	if f == nil {
		return true
	}

	// Filter by industry
	if f.Industry != "" && !strings.EqualFold(c.Industry, f.Industry) {
		return false
	}

	// Filter by query string
	if f.Query != "" {
		q := strings.ToLower(f.Query)
		if !strings.Contains(strings.ToLower(c.Name), q) &&
			!strings.Contains(strings.ToLower(c.Domain), q) &&
			!strings.Contains(strings.ToLower(c.Industry), q) &&
			!strings.Contains(strings.ToLower(c.Notes), q) {
			return false
		}
	}

	return true
}

// DealFilter defines criteria for filtering deals.
type DealFilter struct {
	Query     string     // Full-text search in title, company name
	Stage     string     // Filter by deal stage
	CompanyID *uuid.UUID // Filter by company
	ContactID *uuid.UUID // Filter by contact
	MinAmount int64      // Minimum amount in cents
	MaxAmount int64      // Maximum amount in cents (0 = unlimited)
	Limit     int        // Max results (0 = unlimited)
}

// Matches returns true if the deal matches the filter.
func (f *DealFilter) Matches(d *Deal) bool {
	if f == nil {
		return true
	}

	// Filter by query string
	if f.Query != "" {
		q := strings.ToLower(f.Query)
		if !strings.Contains(strings.ToLower(d.Title), q) &&
			!strings.Contains(strings.ToLower(d.CompanyName), q) {
			return false
		}
	}

	// Filter by stage
	if f.Stage != "" && d.Stage != f.Stage {
		return false
	}

	// Filter by company
	if f.CompanyID != nil && d.CompanyID != *f.CompanyID {
		return false
	}

	// Filter by contact
	if f.ContactID != nil {
		if d.ContactID == nil || *d.ContactID != *f.ContactID {
			return false
		}
	}

	// Filter by amount range
	if f.MinAmount > 0 && d.Amount < f.MinAmount {
		return false
	}
	if f.MaxAmount > 0 && d.Amount > f.MaxAmount {
		return false
	}

	return true
}

// InteractionFilter defines criteria for filtering interaction logs.
type InteractionFilter struct {
	ContactID       *uuid.UUID // Filter by contact
	InteractionType string     // Filter by type (meeting, call, email, etc.)
	Since           *time.Time // Only interactions after this time
	Before          *time.Time // Only interactions before this time
	Sentiment       string     // Filter by sentiment
	Limit           int        // Max results (0 = unlimited)
}

// Matches returns true if the interaction matches the filter.
func (f *InteractionFilter) Matches(i *InteractionLog) bool {
	if f == nil {
		return true
	}

	// Filter by contact
	if f.ContactID != nil && i.ContactID != *f.ContactID {
		return false
	}

	// Filter by type
	if f.InteractionType != "" && i.InteractionType != f.InteractionType {
		return false
	}

	// Filter by time range
	if f.Since != nil && i.Timestamp.Before(*f.Since) {
		return false
	}
	if f.Before != nil && i.Timestamp.After(*f.Before) {
		return false
	}

	// Filter by sentiment
	if f.Sentiment != "" {
		if i.Sentiment == nil || *i.Sentiment != f.Sentiment {
			return false
		}
	}

	return true
}

// SuggestionFilter defines criteria for filtering suggestions.
type SuggestionFilter struct {
	Type          string  // Filter by suggestion type
	Status        string  // Filter by status
	MinConfidence float64 // Minimum confidence score
	Limit         int     // Max results (0 = unlimited)
}

// Matches returns true if the suggestion matches the filter.
func (f *SuggestionFilter) Matches(s *Suggestion) bool {
	if f == nil {
		return true
	}

	// Filter by type
	if f.Type != "" && s.Type != f.Type {
		return false
	}

	// Filter by status
	if f.Status != "" && s.Status != f.Status {
		return false
	}

	// Filter by confidence
	if f.MinConfidence > 0 && s.Confidence < f.MinConfidence {
		return false
	}

	return true
}

// FollowupFilter defines criteria for filtering contacts needing follow-up.
type FollowupFilter struct {
	OverdueOnly bool    // Only contacts past their next followup date
	MinPriority float64 // Minimum priority score
	Limit       int     // Max results (0 = unlimited)
}

// RelationshipFilter defines criteria for filtering relationships.
type RelationshipFilter struct {
	ContactID        *uuid.UUID // Filter by either contact in the relationship
	RelationshipType string     // Filter by relationship type
	Limit            int        // Max results (0 = unlimited)
}

// Matches returns true if the relationship matches the filter.
func (f *RelationshipFilter) Matches(r *Relationship) bool {
	if f == nil {
		return true
	}

	// Filter by contact (either side of the relationship)
	if f.ContactID != nil {
		if r.ContactID1 != *f.ContactID && r.ContactID2 != *f.ContactID {
			return false
		}
	}

	// Filter by relationship type
	if f.RelationshipType != "" && r.RelationshipType != f.RelationshipType {
		return false
	}

	return true
}
