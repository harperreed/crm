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
