// ABOUTME: Data models for CRM entities
// ABOUTME: Defines Contact, Company, Deal, and DealNote structs
package models

import (
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	CompanyID       *uuid.UUID `json:"company_id,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	LastContactedAt *time.Time `json:"last_contacted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Company struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain,omitempty"`
	Industry  string    `json:"industry,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Deal struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	Amount            int64      `json:"amount,omitempty"` // in cents
	Currency          string     `json:"currency"`
	Stage             string     `json:"stage"`
	CompanyID         uuid.UUID  `json:"company_id"`
	ContactID         *uuid.UUID `json:"contact_id,omitempty"`
	ExpectedCloseDate *time.Time `json:"expected_close_date,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
}

type DealNote struct {
	ID        uuid.UUID `json:"id"`
	DealID    uuid.UUID `json:"deal_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Relationship struct {
	ID               uuid.UUID `json:"id"`
	ContactID1       uuid.UUID `json:"contact_id_1"`
	ContactID2       uuid.UUID `json:"contact_id_2"`
	RelationshipType string    `json:"relationship_type,omitempty"`
	Context          string    `json:"context,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const (
	StageProspecting   = "prospecting"
	StageQualification = "qualification"
	StageProposal      = "proposal"
	StageNegotiation   = "negotiation"
	StageClosedWon     = "closed_won"
	StageClosedLost    = "closed_lost"
)

// RelationshipStrength constants
const (
	StrengthWeak   = "weak"
	StrengthMedium = "medium"
	StrengthStrong = "strong"
)

// InteractionType constants
const (
	InteractionMeeting = "meeting"
	InteractionCall    = "call"
	InteractionEmail   = "email"
	InteractionMessage = "message"
	InteractionEvent   = "event"
)

// Sentiment constants
const (
	SentimentPositive = "positive"
	SentimentNeutral  = "neutral"
	SentimentNegative = "negative"
)

type ContactCadence struct {
	ContactID            uuid.UUID  `json:"contact_id"`
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	LastInteractionDate  *time.Time `json:"last_interaction_date,omitempty"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}

// ComputePriorityScore calculates the priority score for a contact
// Based on days overdue and relationship strength
func (c *ContactCadence) ComputePriorityScore() float64 {
	if c.LastInteractionDate == nil {
		return 0.0
	}

	daysSinceContact := int(time.Since(*c.LastInteractionDate).Hours() / 24)
	daysOverdue := daysSinceContact - c.CadenceDays

	if daysOverdue <= 0 {
		return 0.0
	}

	baseScore := float64(daysOverdue * 2)

	// Apply relationship multiplier
	multiplier := 1.0
	switch c.RelationshipStrength {
	case StrengthStrong:
		multiplier = 2.0
	case StrengthMedium:
		multiplier = 1.5
	case StrengthWeak:
		multiplier = 1.0
	}

	return baseScore * multiplier
}

// UpdateNextFollowup sets the next followup date based on last interaction and cadence
func (c *ContactCadence) UpdateNextFollowup() {
	if c.LastInteractionDate != nil {
		next := c.LastInteractionDate.AddDate(0, 0, c.CadenceDays)
		c.NextFollowupDate = &next
	}
}

type InteractionLog struct {
	ID              uuid.UUID `json:"id"`
	ContactID       uuid.UUID `json:"contact_id"`
	InteractionType string    `json:"interaction_type"`
	Timestamp       time.Time `json:"timestamp"`
	Notes           string    `json:"notes,omitempty"`
	Sentiment       *string   `json:"sentiment,omitempty"`
}

// FollowupContact combines Contact with cadence info for follow-up views
type FollowupContact struct {
	Contact
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	DaysSinceContact     int        `json:"days_since_contact"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}
