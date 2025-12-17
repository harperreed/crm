// ABOUTME: Self-contained JSON payloads for vault sync operations
// ABOUTME: Includes denormalized data so entities can be reconstructed independently

package sync

// Entity name constants for vault sync.
const (
	EntityContact        = "contact"
	EntityCompany        = "company"
	EntityDeal           = "deal"
	EntityDealNote       = "deal_note"
	EntityRelationship   = "relationship"
	EntityInteractionLog = "interaction_log"
	EntityContactCadence = "contact_cadence"
	EntitySuggestion     = "suggestion"
)

// ContactPayload represents a contact with denormalized company name.
// Self-contained so it can be synced even if the company hasn't been synced yet.
type ContactPayload struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Email           string  `json:"email,omitempty"`
	Phone           string  `json:"phone,omitempty"`
	CompanyID       string  `json:"company_id,omitempty"`
	CompanyName     string  `json:"company_name,omitempty"`
	Notes           string  `json:"notes,omitempty"`
	LastContactedAt *string `json:"last_contacted_at,omitempty"` // RFC3339 timestamp
}

// CompanyPayload represents a company entity.
type CompanyPayload struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Domain   string `json:"domain,omitempty"`
	Industry string `json:"industry,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// DealPayload represents a deal with denormalized company and contact names.
// Self-contained so it can be synced independently of related entities.
type DealPayload struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Amount            int64   `json:"amount"` // cents
	Currency          string  `json:"currency"`
	Stage             string  `json:"stage"`
	CompanyID         string  `json:"company_id,omitempty"`
	CompanyName       string  `json:"company_name,omitempty"` // denormalized from CompanyID
	ContactID         string  `json:"contact_id,omitempty"`
	ContactName       string  `json:"contact_name,omitempty"`        // denormalized from ContactID
	ExpectedCloseDate *string `json:"expected_close_date,omitempty"` // RFC3339 timestamp
}

// DealNotePayload represents a note attached to a deal.
type DealNotePayload struct {
	ID              string `json:"id"`
	DealID          string `json:"deal_id"`
	DealTitle       string `json:"deal_title"` // denormalized from DealID
	DealCompanyID   string `json:"deal_company_id,omitempty"`
	DealCompanyName string `json:"deal_company_name,omitempty"`
	Content         string `json:"content"`
	CreatedAt       string `json:"created_at"` // RFC3339 timestamp
}

// RelationshipPayload represents a bidirectional relationship between contacts.
type RelationshipPayload struct {
	ID               string `json:"id"`
	ContactID1       string `json:"contact_id_1,omitempty"`
	ContactID2       string `json:"contact_id_2,omitempty"`
	Contact1Name     string `json:"contact1_name"` // denormalized from ContactID1
	Contact2Name     string `json:"contact2_name"` // denormalized from ContactID2
	RelationshipType string `json:"relationship_type"`
	Context          string `json:"context,omitempty"`
}

// InteractionLogPayload represents an interaction record.
type InteractionLogPayload struct {
	ID              string  `json:"id"`
	ContactID       string  `json:"contact_id,omitempty"`
	ContactName     string  `json:"contact_name"` // denormalized from ContactID
	InteractionType string  `json:"interaction_type"`
	InteractedAt    string  `json:"interacted_at"` // RFC3339 timestamp
	Sentiment       *string `json:"sentiment,omitempty"`
	Metadata        string  `json:"metadata,omitempty"` // JSON string
}

// ContactCadencePayload represents follow-up cadence settings for a contact.
type ContactCadencePayload struct {
	ID                   string `json:"id"`
	ContactName          string `json:"contact_name"` // denormalized from ContactID
	CadenceDays          int    `json:"cadence_days"`
	RelationshipStrength string `json:"relationship_strength"`
	PriorityScore        int    `json:"priority_score"`
}

// SuggestionPayload represents an AI suggestion.
type SuggestionPayload struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	Content       string  `json:"content"` // JSON string containing suggestion data
	Confidence    float64 `json:"confidence"`
	SourceService string  `json:"source_service"`
	Status        string  `json:"status"`
}
