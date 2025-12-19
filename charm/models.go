// ABOUTME: Data models for Charm KV storage with denormalized relationships
// ABOUTME: Based on existing sync payload patterns for self-contained entities

package charm

import (
	"time"

	"github.com/google/uuid"
)

// Contact represents a contact stored in KV
// CompanyName is denormalized so contacts can be displayed without looking up companies.
type Contact struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	CompanyID       *uuid.UUID `json:"company_id,omitempty"`
	CompanyName     string     `json:"company_name,omitempty"` // denormalized
	Notes           string     `json:"notes,omitempty"`
	LastContactedAt *time.Time `json:"last_contacted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Company represents a company stored in KV.
type Company struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain,omitempty"`
	Industry  string    `json:"industry,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Deal represents a deal stored in KV
// CompanyName and ContactName are denormalized for display without lookups.
type Deal struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	Amount            int64      `json:"amount,omitempty"` // in cents
	Currency          string     `json:"currency"`
	Stage             string     `json:"stage"`
	CompanyID         uuid.UUID  `json:"company_id"`
	CompanyName       string     `json:"company_name,omitempty"` // denormalized
	ContactID         *uuid.UUID `json:"contact_id,omitempty"`
	ContactName       string     `json:"contact_name,omitempty"` // denormalized
	ExpectedCloseDate *time.Time `json:"expected_close_date,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
}

// DealNote represents a note attached to a deal
// DealTitle and DealCompanyName are denormalized for context.
type DealNote struct {
	ID              uuid.UUID `json:"id"`
	DealID          uuid.UUID `json:"deal_id"`
	DealTitle       string    `json:"deal_title,omitempty"`        // denormalized
	DealCompanyName string    `json:"deal_company_name,omitempty"` // denormalized
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
}

// Relationship represents a bidirectional relationship between contacts
// Contact names are denormalized for display.
type Relationship struct {
	ID               uuid.UUID `json:"id"`
	ContactID1       uuid.UUID `json:"contact_id_1"`
	ContactID2       uuid.UUID `json:"contact_id_2"`
	Contact1Name     string    `json:"contact1_name,omitempty"` // denormalized
	Contact2Name     string    `json:"contact2_name,omitempty"` // denormalized
	RelationshipType string    `json:"relationship_type,omitempty"`
	Context          string    `json:"context,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// InteractionLog records an interaction with a contact.
type InteractionLog struct {
	ID              uuid.UUID `json:"id"`
	ContactID       uuid.UUID `json:"contact_id"`
	ContactName     string    `json:"contact_name,omitempty"` // denormalized
	InteractionType string    `json:"interaction_type"`
	Timestamp       time.Time `json:"timestamp"`
	Notes           string    `json:"notes,omitempty"`
	Sentiment       *string   `json:"sentiment,omitempty"`
	Metadata        string    `json:"metadata,omitempty"`
}

// ContactCadence tracks follow-up settings for a contact.
type ContactCadence struct {
	ContactID            uuid.UUID  `json:"contact_id"`
	ContactName          string     `json:"contact_name,omitempty"` // denormalized
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	LastInteractionDate  *time.Time `json:"last_interaction_date,omitempty"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}

// FollowupContact combines Contact with cadence info for follow-up views.
type FollowupContact struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Email                string     `json:"email,omitempty"`
	Phone                string     `json:"phone,omitempty"`
	CompanyID            *uuid.UUID `json:"company_id,omitempty"`
	CompanyName          string     `json:"company_name,omitempty"`
	Notes                string     `json:"notes,omitempty"`
	LastContactedAt      *time.Time `json:"last_contacted_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	DaysSinceContact     int        `json:"days_since_contact"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}

// Suggestion represents an AI-generated suggestion.
type Suggestion struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Confidence    float64    `json:"confidence"`
	SourceService string     `json:"source_service"`
	SourceID      string     `json:"source_id,omitempty"`
	SourceData    string     `json:"source_data,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
}

// SyncState tracks sync status for external services (Google, etc.)
type SyncState struct {
	Service       string     `json:"service"`
	LastSyncTime  *time.Time `json:"last_sync_time,omitempty"`
	LastSyncToken string     `json:"last_sync_token,omitempty"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SyncLog records imported entities from external services.
type SyncLog struct {
	ID            uuid.UUID `json:"id"`
	SourceService string    `json:"source_service"`
	SourceID      string    `json:"source_id"`
	EntityType    string    `json:"entity_type"`
	EntityID      uuid.UUID `json:"entity_id"`
	ImportedAt    time.Time `json:"imported_at"`
	Metadata      string    `json:"metadata,omitempty"`
}

// Stage constants for deals.
const (
	StageProspecting   = "prospecting"
	StageQualification = "qualification"
	StageProposal      = "proposal"
	StageNegotiation   = "negotiation"
	StageClosedWon     = "closed_won"
	StageClosedLost    = "closed_lost"
)

// RelationshipStrength constants.
const (
	StrengthWeak   = "weak"
	StrengthMedium = "medium"
	StrengthStrong = "strong"
)

// InteractionType constants.
const (
	InteractionMeeting = "meeting"
	InteractionCall    = "call"
	InteractionEmail   = "email"
	InteractionMessage = "message"
	InteractionEvent   = "event"
)

// Sentiment constants.
const (
	SentimentPositive = "positive"
	SentimentNeutral  = "neutral"
	SentimentNegative = "negative"
)

// SyncStatus constants.
const (
	SyncStatusIdle    = "idle"
	SyncStatusSyncing = "syncing"
	SyncStatusError   = "error"
)

// SuggestionType constants.
const (
	SuggestionTypeDeal         = "deal"
	SuggestionTypeRelationship = "relationship"
	SuggestionTypeCompany      = "company"
)

// SuggestionStatus constants.
const (
	SuggestionStatusPending  = "pending"
	SuggestionStatusAccepted = "accepted"
	SuggestionStatusRejected = "rejected"
)
