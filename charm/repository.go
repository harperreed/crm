// ABOUTME: CRUD operations for all entity types using Charm KV
// ABOUTME: Client-side filtering replaces SQL WHERE clauses

package charm

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
)

// ============================================================================
// Contact Operations
// ============================================================================

// CreateContact creates a new contact.
func (c *Client) CreateContact(contact *Contact) error {
	if contact.ID == uuid.Nil {
		contact.ID = uuid.New()
	}
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	data, err := json.Marshal(contact)
	if err != nil {
		return fmt.Errorf("failed to marshal contact: %w", err)
	}

	return c.Set(ContactKey(contact.ID.String()), data)
}

// GetContact retrieves a contact by ID.
func (c *Client) GetContact(id uuid.UUID) (*Contact, error) {
	data, err := c.Get(ContactKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("contact not found: %s", id)
	}

	var contact Contact
	if err := json.Unmarshal(data, &contact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contact: %w", err)
	}
	return &contact, nil
}

// UpdateContact updates an existing contact.
func (c *Client) UpdateContact(contact *Contact) error {
	contact.UpdatedAt = time.Now()

	data, err := json.Marshal(contact)
	if err != nil {
		return fmt.Errorf("failed to marshal contact: %w", err)
	}

	return c.Set(ContactKey(contact.ID.String()), data)
}

// DeleteContact removes a contact by ID.
func (c *Client) DeleteContact(id uuid.UUID) error {
	return c.Delete(ContactKey(id.String()))
}

// ListContacts returns all contacts matching the filter.
func (c *Client) ListContacts(filter *ContactFilter) ([]*Contact, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixContact))
	if err != nil {
		return nil, err
	}

	var contacts []*Contact
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var contact Contact
		if err := json.Unmarshal(data, &contact); err != nil {
			continue
		}

		if filter.Matches(&contact) {
			contacts = append(contacts, &contact)
		}
	}

	// Sort by name
	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].Name < contacts[j].Name
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(contacts) > filter.Limit {
		contacts = contacts[:filter.Limit]
	}

	return contacts, nil
}

// FindContactByName finds a contact by exact name match.
func (c *Client) FindContactByName(name string) (*Contact, error) {
	contacts, err := c.ListContacts(&ContactFilter{Query: name, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(contacts) == 0 {
		return nil, nil
	}
	// Verify exact match
	if contacts[0].Name == name {
		return contacts[0], nil
	}
	return nil, nil
}

// ============================================================================
// Company Operations
// ============================================================================

// CreateCompany creates a new company.
func (c *Client) CreateCompany(company *Company) error {
	if company.ID == uuid.Nil {
		company.ID = uuid.New()
	}
	now := time.Now()
	company.CreatedAt = now
	company.UpdatedAt = now

	data, err := json.Marshal(company)
	if err != nil {
		return fmt.Errorf("failed to marshal company: %w", err)
	}

	return c.Set(CompanyKey(company.ID.String()), data)
}

// GetCompany retrieves a company by ID.
func (c *Client) GetCompany(id uuid.UUID) (*Company, error) {
	data, err := c.Get(CompanyKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("company not found: %s", id)
	}

	var company Company
	if err := json.Unmarshal(data, &company); err != nil {
		return nil, fmt.Errorf("failed to unmarshal company: %w", err)
	}
	return &company, nil
}

// UpdateCompany updates an existing company.
func (c *Client) UpdateCompany(company *Company) error {
	company.UpdatedAt = time.Now()

	data, err := json.Marshal(company)
	if err != nil {
		return fmt.Errorf("failed to marshal company: %w", err)
	}

	return c.Set(CompanyKey(company.ID.String()), data)
}

// DeleteCompany removes a company by ID.
func (c *Client) DeleteCompany(id uuid.UUID) error {
	return c.Delete(CompanyKey(id.String()))
}

// ListCompanies returns all companies matching the filter.
func (c *Client) ListCompanies(filter *CompanyFilter) ([]*Company, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixCompany))
	if err != nil {
		return nil, err
	}

	var companies []*Company
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var company Company
		if err := json.Unmarshal(data, &company); err != nil {
			continue
		}

		if filter.Matches(&company) {
			companies = append(companies, &company)
		}
	}

	// Sort by name
	sort.Slice(companies, func(i, j int) bool {
		return companies[i].Name < companies[j].Name
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(companies) > filter.Limit {
		companies = companies[:filter.Limit]
	}

	return companies, nil
}

// FindCompanyByName finds a company by exact name match.
func (c *Client) FindCompanyByName(name string) (*Company, error) {
	companies, err := c.ListCompanies(&CompanyFilter{Query: name, Limit: 10})
	if err != nil {
		return nil, err
	}
	for _, company := range companies {
		if company.Name == name {
			return company, nil
		}
	}
	return nil, nil
}

// ============================================================================
// Deal Operations
// ============================================================================

// CreateDeal creates a new deal.
func (c *Client) CreateDeal(deal *Deal) error {
	if deal.ID == uuid.Nil {
		deal.ID = uuid.New()
	}
	now := time.Now()
	deal.CreatedAt = now
	deal.UpdatedAt = now
	deal.LastActivityAt = now

	data, err := json.Marshal(deal)
	if err != nil {
		return fmt.Errorf("failed to marshal deal: %w", err)
	}

	return c.Set(DealKey(deal.ID.String()), data)
}

// GetDeal retrieves a deal by ID.
func (c *Client) GetDeal(id uuid.UUID) (*Deal, error) {
	data, err := c.Get(DealKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("deal not found: %s", id)
	}

	var deal Deal
	if err := json.Unmarshal(data, &deal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deal: %w", err)
	}
	return &deal, nil
}

// UpdateDeal updates an existing deal.
func (c *Client) UpdateDeal(deal *Deal) error {
	deal.UpdatedAt = time.Now()
	deal.LastActivityAt = time.Now()

	data, err := json.Marshal(deal)
	if err != nil {
		return fmt.Errorf("failed to marshal deal: %w", err)
	}

	return c.Set(DealKey(deal.ID.String()), data)
}

// DeleteDeal removes a deal by ID.
func (c *Client) DeleteDeal(id uuid.UUID) error {
	return c.Delete(DealKey(id.String()))
}

// ListDeals returns all deals matching the filter.
func (c *Client) ListDeals(filter *DealFilter) ([]*Deal, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixDeal))
	if err != nil {
		return nil, err
	}

	var deals []*Deal
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var deal Deal
		if err := json.Unmarshal(data, &deal); err != nil {
			continue
		}

		if filter.Matches(&deal) {
			deals = append(deals, &deal)
		}
	}

	// Sort by last activity (most recent first)
	sort.Slice(deals, func(i, j int) bool {
		return deals[i].LastActivityAt.After(deals[j].LastActivityAt)
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(deals) > filter.Limit {
		deals = deals[:filter.Limit]
	}

	return deals, nil
}

// ============================================================================
// DealNote Operations
// ============================================================================

// CreateDealNote creates a new deal note.
func (c *Client) CreateDealNote(note *DealNote) error {
	if note.ID == uuid.Nil {
		note.ID = uuid.New()
	}
	note.CreatedAt = time.Now()

	data, err := json.Marshal(note)
	if err != nil {
		return fmt.Errorf("failed to marshal deal note: %w", err)
	}

	return c.Set(DealNoteKey(note.ID.String()), data)
}

// GetDealNote retrieves a deal note by ID.
func (c *Client) GetDealNote(id uuid.UUID) (*DealNote, error) {
	data, err := c.Get(DealNoteKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("deal note not found: %s", id)
	}

	var note DealNote
	if err := json.Unmarshal(data, &note); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deal note: %w", err)
	}
	return &note, nil
}

// DeleteDealNote removes a deal note by ID.
func (c *Client) DeleteDealNote(id uuid.UUID) error {
	return c.Delete(DealNoteKey(id.String()))
}

// ListDealNotes returns all notes for a deal.
func (c *Client) ListDealNotes(dealID uuid.UUID) ([]*DealNote, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixDealNote))
	if err != nil {
		return nil, err
	}

	var notes []*DealNote
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var note DealNote
		if err := json.Unmarshal(data, &note); err != nil {
			continue
		}

		if note.DealID == dealID {
			notes = append(notes, &note)
		}
	}

	// Sort by created time (oldest first)
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].CreatedAt.Before(notes[j].CreatedAt)
	})

	return notes, nil
}

// ============================================================================
// Relationship Operations
// ============================================================================

// CreateRelationship creates a new relationship between contacts.
func (c *Client) CreateRelationship(rel *Relationship) error {
	if rel.ID == uuid.Nil {
		rel.ID = uuid.New()
	}
	now := time.Now()
	rel.CreatedAt = now
	rel.UpdatedAt = now

	data, err := json.Marshal(rel)
	if err != nil {
		return fmt.Errorf("failed to marshal relationship: %w", err)
	}

	return c.Set(RelationshipKey(rel.ID.String()), data)
}

// GetRelationship retrieves a relationship by ID.
func (c *Client) GetRelationship(id uuid.UUID) (*Relationship, error) {
	data, err := c.Get(RelationshipKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("relationship not found: %s", id)
	}

	var rel Relationship
	if err := json.Unmarshal(data, &rel); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relationship: %w", err)
	}
	return &rel, nil
}

// UpdateRelationship updates an existing relationship.
func (c *Client) UpdateRelationship(rel *Relationship) error {
	rel.UpdatedAt = time.Now()

	data, err := json.Marshal(rel)
	if err != nil {
		return fmt.Errorf("failed to marshal relationship: %w", err)
	}

	return c.Set(RelationshipKey(rel.ID.String()), data)
}

// DeleteRelationship removes a relationship by ID.
func (c *Client) DeleteRelationship(id uuid.UUID) error {
	return c.Delete(RelationshipKey(id.String()))
}

// ListRelationshipsForContact returns all relationships involving a contact.
func (c *Client) ListRelationshipsForContact(contactID uuid.UUID) ([]*Relationship, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixRelationship))
	if err != nil {
		return nil, err
	}

	var rels []*Relationship
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var rel Relationship
		if err := json.Unmarshal(data, &rel); err != nil {
			continue
		}

		if rel.ContactID1 == contactID || rel.ContactID2 == contactID {
			rels = append(rels, &rel)
		}
	}

	return rels, nil
}

// GetRelationshipBetween finds a relationship between two specific contacts.
func (c *Client) GetRelationshipBetween(contactID1, contactID2 uuid.UUID) (*Relationship, error) {
	rels, err := c.ListRelationshipsForContact(contactID1)
	if err != nil {
		return nil, err
	}

	for _, rel := range rels {
		if (rel.ContactID1 == contactID1 && rel.ContactID2 == contactID2) ||
			(rel.ContactID1 == contactID2 && rel.ContactID2 == contactID1) {
			return rel, nil
		}
	}
	return nil, nil
}

// ListRelationships returns relationships matching the filter.
func (c *Client) ListRelationships(filter *RelationshipFilter) ([]*Relationship, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixRelationship))
	if err != nil {
		return nil, err
	}

	var rels []*Relationship
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var rel Relationship
		if err := json.Unmarshal(data, &rel); err != nil {
			continue
		}

		if filter != nil && !filter.Matches(&rel) {
			continue
		}

		rels = append(rels, &rel)

		// Apply limit
		if filter != nil && filter.Limit > 0 && len(rels) >= filter.Limit {
			break
		}
	}

	return rels, nil
}

// ============================================================================
// InteractionLog Operations
// ============================================================================

// CreateInteractionLog creates a new interaction log entry.
func (c *Client) CreateInteractionLog(log *InteractionLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal interaction log: %w", err)
	}

	return c.Set(InteractionLogKey(log.ID.String()), data)
}

// GetInteractionLog retrieves an interaction log entry by ID.
func (c *Client) GetInteractionLog(id uuid.UUID) (*InteractionLog, error) {
	data, err := c.Get(InteractionLogKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("interaction log not found: %s", id)
	}

	var log InteractionLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("failed to unmarshal interaction log: %w", err)
	}
	return &log, nil
}

// DeleteInteractionLog removes an interaction log entry by ID.
func (c *Client) DeleteInteractionLog(id uuid.UUID) error {
	return c.Delete(InteractionLogKey(id.String()))
}

// ListInteractionLogs returns interactions matching the filter.
func (c *Client) ListInteractionLogs(filter *InteractionFilter) ([]*InteractionLog, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixInteractionLog))
	if err != nil {
		return nil, err
	}

	var logs []*InteractionLog
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var log InteractionLog
		if err := json.Unmarshal(data, &log); err != nil {
			continue
		}

		if filter.Matches(&log) {
			logs = append(logs, &log)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(logs) > filter.Limit {
		logs = logs[:filter.Limit]
	}

	return logs, nil
}

// ============================================================================
// ContactCadence Operations
// ============================================================================

// SaveContactCadence saves or updates a contact cadence.
func (c *Client) SaveContactCadence(cadence *ContactCadence) error {
	data, err := json.Marshal(cadence)
	if err != nil {
		return fmt.Errorf("failed to marshal contact cadence: %w", err)
	}

	return c.Set(ContactCadenceKey(cadence.ContactID.String()), data)
}

// GetContactCadence retrieves a contact cadence by contact ID.
// Returns (nil, nil) if no cadence exists for the contact.
func (c *Client) GetContactCadence(contactID uuid.UUID) (*ContactCadence, error) {
	data, err := c.Get(ContactCadenceKey(contactID.String()))
	if err != nil {
		// Handle key not found - return nil, nil to indicate no cadence exists
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil
		}
		// Also handle wrapped errors and string-based errors from test client
		if strings.Contains(err.Error(), "Key not found") {
			return nil, nil
		}
		return nil, err
	}
	if data == nil {
		return nil, nil // No cadence set for this contact
	}

	var cadence ContactCadence
	if err := json.Unmarshal(data, &cadence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contact cadence: %w", err)
	}
	return &cadence, nil
}

// DeleteContactCadence removes a contact cadence.
func (c *Client) DeleteContactCadence(contactID uuid.UUID) error {
	return c.Delete(ContactCadenceKey(contactID.String()))
}

// ListContactCadences returns all contact cadences.
func (c *Client) ListContactCadences() ([]*ContactCadence, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixContactCadence))
	if err != nil {
		return nil, err
	}

	var cadences []*ContactCadence
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var cadence ContactCadence
		if err := json.Unmarshal(data, &cadence); err != nil {
			continue
		}

		cadences = append(cadences, &cadence)
	}

	// Sort by priority score (highest first)
	sort.Slice(cadences, func(i, j int) bool {
		return cadences[i].PriorityScore > cadences[j].PriorityScore
	})

	return cadences, nil
}

// GetFollowupList returns contacts needing follow-up, sorted by priority
// This combines cadence data with contact information similar to the SQL version.
func (c *Client) GetFollowupList(limit int) ([]*FollowupContact, error) {
	// Get all cadences sorted by priority
	cadences, err := c.ListContactCadences()
	if err != nil {
		return nil, err
	}

	var followups []*FollowupContact
	for _, cadence := range cadences {
		// Only include contacts with priority > 0
		if cadence.PriorityScore <= 0 {
			continue
		}

		// Get the contact details
		contact, err := c.GetContact(cadence.ContactID)
		if err != nil {
			continue // Skip if contact not found
		}

		// Calculate days since contact
		daysSince := 0
		if cadence.LastInteractionDate != nil {
			daysSince = int(time.Since(*cadence.LastInteractionDate).Hours() / 24)
		}

		// Build followup contact
		followup := &FollowupContact{
			ID:                   contact.ID,
			Name:                 contact.Name,
			Email:                contact.Email,
			Phone:                contact.Phone,
			CompanyID:            contact.CompanyID,
			CompanyName:          contact.CompanyName,
			Notes:                contact.Notes,
			LastContactedAt:      contact.LastContactedAt,
			CreatedAt:            contact.CreatedAt,
			UpdatedAt:            contact.UpdatedAt,
			CadenceDays:          cadence.CadenceDays,
			RelationshipStrength: cadence.RelationshipStrength,
			PriorityScore:        cadence.PriorityScore,
			DaysSinceContact:     daysSince,
			NextFollowupDate:     cadence.NextFollowupDate,
		}

		followups = append(followups, followup)

		// Apply limit
		if limit > 0 && len(followups) >= limit {
			break
		}
	}

	return followups, nil
}

// UpdateCadenceAfterInteraction updates cadence when interaction is logged.
func (c *Client) UpdateCadenceAfterInteraction(contactID uuid.UUID, timestamp time.Time) error {
	// Get or create cadence
	cadence, err := c.GetContactCadence(contactID)
	if err != nil {
		return err
	}

	if cadence == nil {
		// Create default cadence
		cadence = &ContactCadence{
			ContactID:            contactID,
			CadenceDays:          30,
			RelationshipStrength: StrengthMedium,
		}
	}

	// Update timestamps
	cadence.LastInteractionDate = &timestamp
	next := timestamp.AddDate(0, 0, cadence.CadenceDays)
	cadence.NextFollowupDate = &next

	// Compute priority score
	daysSinceContact := int(time.Since(*cadence.LastInteractionDate).Hours() / 24)
	daysOverdue := daysSinceContact - cadence.CadenceDays

	if daysOverdue <= 0 {
		cadence.PriorityScore = 0.0
	} else {
		baseScore := float64(daysOverdue * 2)
		multiplier := 1.0
		switch cadence.RelationshipStrength {
		case StrengthStrong:
			multiplier = 2.0
		case StrengthMedium:
			multiplier = 1.5
		case StrengthWeak:
			multiplier = 1.0
		}
		cadence.PriorityScore = baseScore * multiplier
	}

	return c.SaveContactCadence(cadence)
}

// ============================================================================
// Suggestion Operations
// ============================================================================

// CreateSuggestion creates a new suggestion.
func (c *Client) CreateSuggestion(suggestion *Suggestion) error {
	if suggestion.ID == uuid.Nil {
		suggestion.ID = uuid.New()
	}
	suggestion.CreatedAt = time.Now()

	data, err := json.Marshal(suggestion)
	if err != nil {
		return fmt.Errorf("failed to marshal suggestion: %w", err)
	}

	return c.Set(SuggestionKey(suggestion.ID.String()), data)
}

// GetSuggestion retrieves a suggestion by ID.
func (c *Client) GetSuggestion(id uuid.UUID) (*Suggestion, error) {
	data, err := c.Get(SuggestionKey(id.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("suggestion not found: %s", id)
	}

	var suggestion Suggestion
	if err := json.Unmarshal(data, &suggestion); err != nil {
		return nil, fmt.Errorf("failed to unmarshal suggestion: %w", err)
	}
	return &suggestion, nil
}

// UpdateSuggestion updates an existing suggestion.
func (c *Client) UpdateSuggestion(suggestion *Suggestion) error {
	data, err := json.Marshal(suggestion)
	if err != nil {
		return fmt.Errorf("failed to marshal suggestion: %w", err)
	}

	return c.Set(SuggestionKey(suggestion.ID.String()), data)
}

// DeleteSuggestion removes a suggestion by ID.
func (c *Client) DeleteSuggestion(id uuid.UUID) error {
	return c.Delete(SuggestionKey(id.String()))
}

// ListSuggestions returns suggestions matching the filter.
func (c *Client) ListSuggestions(filter *SuggestionFilter) ([]*Suggestion, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixSuggestion))
	if err != nil {
		return nil, err
	}

	var suggestions []*Suggestion
	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var suggestion Suggestion
		if err := json.Unmarshal(data, &suggestion); err != nil {
			continue
		}

		if filter.Matches(&suggestion) {
			suggestions = append(suggestions, &suggestion)
		}
	}

	// Sort by confidence (highest first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(suggestions) > filter.Limit {
		suggestions = suggestions[:filter.Limit]
	}

	return suggestions, nil
}

// ============================================================================
// SyncState Operations
// ============================================================================

// SaveSyncState saves sync state for a service.
func (c *Client) SaveSyncState(state *SyncState) error {
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal sync state: %w", err)
	}

	return c.Set(SyncStateKey(state.Service), data)
}

// GetSyncState retrieves sync state for a service.
// Returns (nil, nil) if no state exists for the service.
func (c *Client) GetSyncState(service string) (*SyncState, error) {
	data, err := c.Get(SyncStateKey(service))
	if err != nil {
		// Handle key not found - return nil, nil to indicate no state exists
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil
		}
		// Also handle wrapped errors and string-based errors from test client
		if strings.Contains(err.Error(), "Key not found") {
			return nil, nil
		}
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync state: %w", err)
	}
	return &state, nil
}

// ============================================================================
// SyncLog Operations
// ============================================================================

// CreateSyncLog creates a sync log entry.
func (c *Client) CreateSyncLog(log *SyncLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.ImportedAt = time.Now()

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal sync log: %w", err)
	}

	return c.Set(SyncLogKey(log.ID.String()), data)
}

// FindSyncLogBySource finds a sync log by source service and ID.
func (c *Client) FindSyncLogBySource(service, sourceID string) (*SyncLog, error) {
	keys, err := c.KeysWithPrefix([]byte(PrefixSyncLog))
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		data, err := c.Get(key)
		if err != nil {
			continue
		}

		var log SyncLog
		if err := json.Unmarshal(data, &log); err != nil {
			continue
		}

		if log.SourceService == service && log.SourceID == sourceID {
			return &log, nil
		}
	}
	return nil, nil
}
