// ABOUTME: Vault sync implementation for bidirectional CRM entity synchronization
// ABOUTME: Queues local changes, applies remote changes, and handles encrypted sync with vault server

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

// AppID is the namespace UUID for pagen sync data.
const AppID = "e9240d3f-967d-485e-8c63-e0adf7eecca0"

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// VaultSyncer manages bidirectional synchronization with vault server.
type VaultSyncer struct {
	config      *VaultConfig
	store       *vault.Store
	keys        vault.Keys
	client      *vault.Client
	vaultSyncer *vault.Syncer
	appDB       *sql.DB
}

// NewVaultSyncer creates a new vault syncer instance.
func NewVaultSyncer(cfg *VaultConfig, appDB *sql.DB) (*VaultSyncer, error) {
	if !cfg.IsConfigured() {
		return nil, fmt.Errorf("vault config is not properly configured")
	}

	// Parse the derived key (hex-encoded seed)
	seed, err := vault.ParseSeedPhrase(cfg.DerivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse seed phrase: %w", err)
	}

	// Derive encryption keys
	keys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	if err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}

	// Open vault store
	store, err := vault.OpenStore(cfg.VaultDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open vault store: %w", err)
	}

	// Create sync client
	client := vault.NewClient(vault.SyncConfig{
		AppID:        AppID,
		BaseURL:      cfg.Server,
		DeviceID:     cfg.DeviceID,
		AuthToken:    cfg.Token,
		RefreshToken: cfg.RefreshToken,
		TokenExpires: parseTime(cfg.TokenExpires),
		OnTokenRefresh: func(token, refreshToken string, expires time.Time) {
			cfg.Token = token
			cfg.RefreshToken = refreshToken
			cfg.TokenExpires = expires.Format(time.RFC3339)
			if err := SaveVaultConfig(cfg); err != nil {
				log.Printf("warning: failed to persist refreshed vault token: %v", err)
			}
		},
	})

	return &VaultSyncer{
		config:      cfg,
		store:       store,
		keys:        keys,
		client:      client,
		vaultSyncer: vault.NewSyncer(store, client, keys, cfg.UserID),
		appDB:       appDB,
	}, nil
}

// Close closes the vault store.
func (s *VaultSyncer) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// Sync performs a full bidirectional sync without events.
func (s *VaultSyncer) Sync(ctx context.Context) error {
	if !s.canSync() {
		return errors.New("vault sync not configured")
	}
	return vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange)
}

// SyncWithEvents performs a full bidirectional sync with event callbacks.
func (s *VaultSyncer) SyncWithEvents(ctx context.Context, events vault.SyncEvents) error {
	if !s.canSync() {
		return errors.New("vault sync not configured")
	}
	return vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange, &events)
}

// PendingCount returns the number of pending changes in the outbox.
func (s *VaultSyncer) PendingCount(ctx context.Context) (int, error) {
	return s.store.PendingCount(ctx)
}

// PendingChanges returns the list of pending outbox items.
func (s *VaultSyncer) PendingChanges(ctx context.Context) ([]vault.OutboxItem, error) {
	return s.store.DequeueBatch(ctx, 1000)
}

// LastSyncedSeq returns the last synced sequence number.
func (s *VaultSyncer) LastSyncedSeq(ctx context.Context) (int64, error) {
	seqStr, err := s.store.GetState(ctx, "last_synced_seq", "0")
	if err != nil {
		return 0, err
	}
	var seq int64
	if _, err := fmt.Sscanf(seqStr, "%d", &seq); err != nil {
		return 0, err
	}
	return seq, nil
}

func (s *VaultSyncer) canSync() bool {
	return s.client != nil && s.config.UserID != "" && s.config.Token != ""
}

// QueueContactChange queues a contact change for sync.
func (s *VaultSyncer) QueueContactChange(ctx context.Context, contact *models.Contact, companyName string, op vault.Op) error {
	var lastContactedAt *string
	if contact.LastContactedAt != nil {
		ts := contact.LastContactedAt.Format(time.RFC3339)
		lastContactedAt = &ts
	}

	var companyID string
	if contact.CompanyID != nil {
		companyID = contact.CompanyID.String()
	}

	payload := ContactPayload{
		ID:              contact.ID.String(),
		Name:            contact.Name,
		Email:           contact.Email,
		Phone:           contact.Phone,
		CompanyID:       companyID,
		CompanyName:     companyName,
		Notes:           contact.Notes,
		LastContactedAt: lastContactedAt,
	}
	return s.queueChange(ctx, EntityContact, contact.ID.String(), op, payload)
}

// QueueCompanyChange queues a company change for sync.
func (s *VaultSyncer) QueueCompanyChange(ctx context.Context, company *models.Company, op vault.Op) error {
	payload := CompanyPayload{
		ID:       company.ID.String(),
		Name:     company.Name,
		Domain:   company.Domain,
		Industry: company.Industry,
		Notes:    company.Notes,
	}
	return s.queueChange(ctx, EntityCompany, company.ID.String(), op, payload)
}

// QueueDealChange queues a deal change for sync.
func (s *VaultSyncer) QueueDealChange(ctx context.Context, deal *models.Deal, companyName, contactName string, op vault.Op) error {
	var expectedCloseDate *string
	if deal.ExpectedCloseDate != nil {
		ts := deal.ExpectedCloseDate.Format(time.RFC3339)
		expectedCloseDate = &ts
	}

	var companyID string
	if deal.CompanyID != uuid.Nil {
		companyID = deal.CompanyID.String()
	}

	var contactID string
	if deal.ContactID != nil {
		contactID = deal.ContactID.String()
	}

	payload := DealPayload{
		ID:                deal.ID.String(),
		Title:             deal.Title,
		Amount:            deal.Amount,
		Currency:          deal.Currency,
		Stage:             deal.Stage,
		CompanyID:         companyID,
		CompanyName:       companyName,
		ContactID:         contactID,
		ContactName:       contactName,
		ExpectedCloseDate: expectedCloseDate,
	}
	return s.queueChange(ctx, EntityDeal, deal.ID.String(), op, payload)
}

// QueueDealNoteChange queues a deal note change for sync.
func (s *VaultSyncer) QueueDealNoteChange(ctx context.Context, note *models.DealNote, dealTitle string, op vault.Op) error {
	var companyID, companyName string
	if deal, err := db.GetDeal(s.appDB, note.DealID); err == nil && deal != nil {
		companyID = deal.CompanyID.String()
		if dealTitle == "" {
			dealTitle = deal.Title
		}
		if company, err := db.GetCompany(s.appDB, deal.CompanyID); err == nil && company != nil {
			companyName = company.Name
		}
	}
	payload := DealNotePayload{
		ID:              note.ID.String(),
		DealID:          note.DealID.String(),
		DealTitle:       dealTitle,
		DealCompanyID:   companyID,
		DealCompanyName: companyName,
		Content:         note.Content,
		CreatedAt:       note.CreatedAt.Format(time.RFC3339),
	}
	return s.queueChange(ctx, EntityDealNote, note.ID.String(), op, payload)
}

// QueueRelationshipChange queues a relationship change for sync.
func (s *VaultSyncer) QueueRelationshipChange(ctx context.Context, rel *models.Relationship, contact1Name, contact2Name string, op vault.Op) error {
	payload := RelationshipPayload{
		ID:               rel.ID.String(),
		ContactID1:       rel.ContactID1.String(),
		ContactID2:       rel.ContactID2.String(),
		Contact1Name:     contact1Name,
		Contact2Name:     contact2Name,
		RelationshipType: rel.RelationshipType,
		Context:          rel.Context,
	}
	return s.queueChange(ctx, EntityRelationship, rel.ID.String(), op, payload)
}

// QueueInteractionLogChange queues an interaction log change for sync.
func (s *VaultSyncer) QueueInteractionLogChange(ctx context.Context, interaction *models.InteractionLog, contactName string, op vault.Op) error {
	payload := InteractionLogPayload{
		ID:              interaction.ID.String(),
		ContactID:       interaction.ContactID.String(),
		ContactName:     contactName,
		InteractionType: interaction.InteractionType,
		InteractedAt:    interaction.Timestamp.Format(time.RFC3339),
		Sentiment:       interaction.Sentiment,
		Metadata:        interaction.Metadata,
	}
	return s.queueChange(ctx, EntityInteractionLog, interaction.ID.String(), op, payload)
}

// QueueContactCadenceChange queues a contact cadence change for sync.
func (s *VaultSyncer) QueueContactCadenceChange(ctx context.Context, cadence *models.ContactCadence, contactName string, op vault.Op) error {
	payload := ContactCadencePayload{
		ID:                   cadence.ContactID.String(),
		ContactName:          contactName,
		CadenceDays:          cadence.CadenceDays,
		RelationshipStrength: cadence.RelationshipStrength,
		PriorityScore:        int(cadence.PriorityScore),
	}
	return s.queueChange(ctx, EntityContactCadence, cadence.ContactID.String(), op, payload)
}

// QueueSuggestionChange queues a suggestion change for sync.
func (s *VaultSyncer) QueueSuggestionChange(ctx context.Context, suggestion *models.Suggestion, op vault.Op) error {
	payload := SuggestionPayload{
		ID:            suggestion.ID.String(),
		Type:          suggestion.Type,
		Content:       suggestion.SourceData,
		Confidence:    suggestion.Confidence,
		SourceService: suggestion.SourceService,
		Status:        suggestion.Status,
	}
	return s.queueChange(ctx, EntitySuggestion, suggestion.ID.String(), op, payload)
}

// queueChange is the private helper that handles encryption and enqueueing.
func (s *VaultSyncer) queueChange(ctx context.Context, entity, entityID string, op vault.Op, payload interface{}) error {
	if s.vaultSyncer == nil {
		return errors.New("vault sync not configured")
	}

	if _, err := s.vaultSyncer.QueueChange(ctx, entity, entityID, op, payload); err != nil {
		return fmt.Errorf("failed to queue change: %w", err)
	}

	if s.config.AutoSync && s.canSync() {
		return s.Sync(ctx)
	}

	return nil
}

// applyChange is the callback function that applies incoming changes to the local database.
func (s *VaultSyncer) applyChange(ctx context.Context, c vault.Change) error {
	switch c.Entity {
	case EntityContact:
		return s.applyContactChange(ctx, c)
	case EntityCompany:
		return s.applyCompanyChange(ctx, c)
	case EntityDeal:
		return s.applyDealChange(ctx, c)
	case EntityDealNote:
		return s.applyDealNoteChange(ctx, c)
	case EntityRelationship:
		return s.applyRelationshipChange(ctx, c)
	case EntityInteractionLog:
		return s.applyInteractionLogChange(ctx, c)
	case EntityContactCadence:
		return s.applyContactCadenceChange(ctx, c)
	case EntitySuggestion:
		return s.applySuggestionChange(ctx, c)
	default:
		// Skip unknown entities for forward compatibility
		return nil
	}
}

// applyContactChange applies a contact change from vault.
func (s *VaultSyncer) applyContactChange(ctx context.Context, c vault.Change) error {
	var payload ContactPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal contact payload: %w", err)
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("failed to parse contact ID: %w", err)
	}

	switch c.Op {
	case vault.OpUpsert:
		var companyUUID *uuid.UUID
		if payload.CompanyID != "" {
			if parsed, err := uuid.Parse(payload.CompanyID); err == nil {
				if company, err := s.ensureCompany(ctx, parsed, payload.CompanyName); err == nil && company != nil {
					companyUUID = &company.ID
				} else if err != nil {
					return fmt.Errorf("ensure company: %w", err)
				}
			}
		} else if payload.CompanyName != "" {
			company, err := db.FindCompanyByName(s.appDB, payload.CompanyName)
			if err != nil {
				return fmt.Errorf("failed to find company: %w", err)
			}
			if company != nil {
				companyUUID = &company.ID
			}
		}

		contact := &models.Contact{
			ID:        id,
			Name:      payload.Name,
			Email:     payload.Email,
			Phone:     payload.Phone,
			Notes:     payload.Notes,
			CompanyID: companyUUID,
		}

		if payload.LastContactedAt != nil {
			if t, err := time.Parse(time.RFC3339, *payload.LastContactedAt); err == nil {
				contact.LastContactedAt = &t
			}
		}

		existing, err := db.GetContact(s.appDB, id)
		if err != nil {
			return fmt.Errorf("failed to check existing contact: %w", err)
		}

		if existing == nil {
			if err := db.CreateContact(s.appDB, contact); err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}
		} else {
			if err := db.UpdateContact(s.appDB, id, contact); err != nil {
				return fmt.Errorf("failed to update contact: %w", err)
			}
		}

		if contact.LastContactedAt != nil {
			if err := db.UpdateContactLastContacted(s.appDB, contact.ID, *contact.LastContactedAt); err != nil {
				return fmt.Errorf("update last contacted: %w", err)
			}
		}

	case vault.OpDelete:
		if err := db.DeleteContact(s.appDB, id); err != nil {
			return fmt.Errorf("failed to delete contact: %w", err)
		}
	}

	return nil
}

// applyCompanyChange applies a company change from vault.
func (s *VaultSyncer) applyCompanyChange(ctx context.Context, c vault.Change) error {
	var payload CompanyPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal company payload: %w", err)
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("failed to parse company ID: %w", err)
	}

	switch c.Op {
	case vault.OpUpsert:
		company := &models.Company{
			ID:       id,
			Name:     payload.Name,
			Domain:   payload.Domain,
			Industry: payload.Industry,
			Notes:    payload.Notes,
		}

		// Check if company exists
		existing, err := db.GetCompany(s.appDB, id)
		if err != nil {
			return fmt.Errorf("failed to check existing company: %w", err)
		}

		if existing == nil {
			if err := db.CreateCompany(s.appDB, company); err != nil {
				return fmt.Errorf("failed to create company: %w", err)
			}
		} else {
			if err := db.UpdateCompany(s.appDB, id, company); err != nil {
				return fmt.Errorf("failed to update company: %w", err)
			}
		}

	case vault.OpDelete:
		if err := db.DeleteCompany(s.appDB, id); err != nil {
			return fmt.Errorf("failed to delete company: %w", err)
		}
	}

	return nil
}

// applyDealChange applies a deal change from vault.
func (s *VaultSyncer) applyDealChange(ctx context.Context, c vault.Change) error {
	var payload DealPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal deal payload: %w", err)
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("failed to parse deal ID: %w", err)
	}

	switch c.Op {
	case vault.OpUpsert:
		company, err := s.resolveCompanyForDeal(ctx, payload)
		if err != nil {
			return err
		}

		deal := &models.Deal{
			ID:        id,
			Title:     payload.Title,
			Amount:    payload.Amount,
			Currency:  payload.Currency,
			Stage:     payload.Stage,
			CompanyID: company.ID,
		}

		if payload.ExpectedCloseDate != nil {
			if t, err := time.Parse(time.RFC3339, *payload.ExpectedCloseDate); err == nil {
				deal.ExpectedCloseDate = &t
			}
		}

		if payload.ContactID != "" {
			if parsed, err := uuid.Parse(payload.ContactID); err == nil {
				if contact, err := s.ensureContact(ctx, parsed, payload.ContactName, &company.ID); err == nil && contact != nil {
					deal.ContactID = &contact.ID
				} else if err != nil {
					return fmt.Errorf("ensure contact: %w", err)
				}
			}
		} else if payload.ContactName != "" {
			contacts, err := db.FindContacts(s.appDB, payload.ContactName, &company.ID, 1)
			if err != nil {
				return fmt.Errorf("failed to find contact: %w", err)
			}
			if len(contacts) > 0 {
				deal.ContactID = &contacts[0].ID
			}
		}

		existing, err := db.GetDeal(s.appDB, id)
		if err != nil {
			return fmt.Errorf("failed to check existing deal: %w", err)
		}

		if existing == nil {
			if err := db.CreateDeal(s.appDB, deal); err != nil {
				return fmt.Errorf("failed to create deal: %w", err)
			}
		} else {
			deal.CreatedAt = existing.CreatedAt
			if err := db.UpdateDeal(s.appDB, deal); err != nil {
				return fmt.Errorf("failed to update deal: %w", err)
			}
		}

	case vault.OpDelete:
		if err := db.DeleteDeal(s.appDB, id); err != nil {
			return fmt.Errorf("failed to delete deal: %w", err)
		}
	}

	return nil
}

// applyDealNoteChange applies a deal note change from vault.
func (s *VaultSyncer) applyDealNoteChange(ctx context.Context, c vault.Change) error {
	var payload DealNotePayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal deal note payload: %w", err)
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("failed to parse deal note ID: %w", err)
	}

	switch c.Op {
	case vault.OpUpsert:
		dealID, err := uuid.Parse(payload.DealID)
		if err != nil {
			return fmt.Errorf("invalid deal ID: %w", err)
		}

		deal, err := db.GetDeal(s.appDB, dealID)
		if err != nil {
			return fmt.Errorf("failed to fetch deal: %w", err)
		}
		if deal == nil {
			company, err := s.resolveNoteCompany(ctx, payload)
			if err != nil {
				return err
			}
			placeholder := &models.Deal{
				ID:        dealID,
				Title:     payload.DealTitle,
				CompanyID: company.ID,
				Currency:  "USD",
				Stage:     "unknown",
			}
			if err := db.CreateDeal(s.appDB, placeholder); err != nil {
				return fmt.Errorf("create placeholder deal: %w", err)
			}
			deal = placeholder
		}

		createdAt, err := time.Parse(time.RFC3339, payload.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse created_at: %w", err)
		}

		note := &models.DealNote{
			ID:        id,
			DealID:    deal.ID,
			Content:   payload.Content,
			CreatedAt: createdAt,
		}

		if err := db.AddDealNote(s.appDB, note); err != nil {
			return fmt.Errorf("failed to add deal note: %w", err)
		}

	case vault.OpDelete:
		return nil
	}

	return nil
}

// applyRelationshipChange applies a relationship change from vault.
func (s *VaultSyncer) applyRelationshipChange(ctx context.Context, c vault.Change) error {
	var payload RelationshipPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal relationship payload: %w", err)
	}

	if c.Op == vault.OpDelete {
		repo := db.NewRelationshipsRepository(s.appDB)
		if err := repo.Delete(ctx, payload.ID); err != nil && !errors.Is(err, db.ErrRelationshipNotFound) {
			return fmt.Errorf("delete relationship: %w", err)
		}
		return nil
	}

	contact1ID, err := s.resolveContactIdentifier(ctx, payload.ContactID1, payload.Contact1Name)
	if err != nil {
		return err
	}
	contact2ID, err := s.resolveContactIdentifier(ctx, payload.ContactID2, payload.Contact2Name)
	if err != nil {
		return err
	}

	if contact1ID == uuid.Nil || contact2ID == uuid.Nil {
		return fmt.Errorf("unable to resolve contacts for relationship %s", payload.ID)
	}

	if contact1ID.String() > contact2ID.String() {
		contact1ID, contact2ID = contact2ID, contact1ID
	}

	repo := db.NewRelationshipsRepository(s.appDB)
	meta := map[string]interface{}{
		"relationship_type": payload.RelationshipType,
		"context":           payload.Context,
	}

	rel := &db.Relationship{
		ID:       payload.ID,
		SourceID: contact1ID.String(),
		TargetID: contact2ID.String(),
		Type:     db.RelTypeKnows,
		Metadata: meta,
	}

	if err := repo.Update(ctx, rel); err != nil {
		if errors.Is(err, db.ErrRelationshipNotFound) {
			if err := repo.Create(ctx, rel); err != nil {
				return fmt.Errorf("create relationship: %w", err)
			}
		} else {
			return fmt.Errorf("update relationship: %w", err)
		}
	}

	return nil
}

// applyInteractionLogChange applies an interaction log change from vault.
func (s *VaultSyncer) applyInteractionLogChange(ctx context.Context, c vault.Change) error {
	var payload InteractionLogPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal interaction log payload: %w", err)
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("failed to parse interaction log ID: %w", err)
	}

	switch c.Op {
	case vault.OpUpsert:
		var contactID uuid.UUID
		if payload.ContactID != "" {
			if parsed, err := uuid.Parse(payload.ContactID); err == nil {
				resolved, err := s.ensureContact(ctx, parsed, payload.ContactName, nil)
				if err != nil {
					return fmt.Errorf("ensure contact: %w", err)
				}
				contactID = resolved.ID
			}
		}
		if contactID == uuid.Nil {
			contact, err := s.findContactByName(payload.ContactName, nil)
			if err != nil {
				return fmt.Errorf("failed to find contact: %w", err)
			}
			if contact == nil {
				return fmt.Errorf("contact not found: %s", payload.ContactName)
			}
			contactID = contact.ID
		}

		timestamp, err := time.Parse(time.RFC3339, payload.InteractedAt)
		if err != nil {
			return fmt.Errorf("failed to parse timestamp: %w", err)
		}

		interaction := &models.InteractionLog{
			ID:              id,
			ContactID:       contactID,
			InteractionType: payload.InteractionType,
			Timestamp:       timestamp,
			Sentiment:       payload.Sentiment,
			Metadata:        payload.Metadata,
		}

		if err := db.LogInteraction(s.appDB, interaction); err != nil {
			return fmt.Errorf("failed to log interaction: %w", err)
		}

	case vault.OpDelete:
		return nil
	}

	return nil
}

// applyContactCadenceChange applies a contact cadence change from vault.
func (s *VaultSyncer) applyContactCadenceChange(ctx context.Context, c vault.Change) error {
	var payload ContactCadencePayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal contact cadence payload: %w", err)
	}

	contactID, err := uuid.Parse(payload.ID)
	if err != nil {
		return fmt.Errorf("failed to parse contact ID: %w", err)
	}

	switch c.Op {
	case vault.OpUpsert:
		if _, err := s.ensureContact(ctx, contactID, payload.ContactName, nil); err != nil {
			return fmt.Errorf("ensure contact for cadence: %w", err)
		}

		cadence := &models.ContactCadence{
			ContactID:            contactID,
			CadenceDays:          payload.CadenceDays,
			RelationshipStrength: payload.RelationshipStrength,
			PriorityScore:        float64(payload.PriorityScore),
		}

		if err := db.CreateContactCadence(s.appDB, cadence); err != nil {
			return fmt.Errorf("failed to create contact cadence: %w", err)
		}

	case vault.OpDelete:
		// Contact cadence deletion can be implemented if needed
		return nil
	}

	return nil
}

// applySuggestionChange applies a suggestion change from vault.
func (s *VaultSyncer) applySuggestionChange(ctx context.Context, c vault.Change) error {
	var payload SuggestionPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal suggestion payload: %w", err)
	}

	// Suggestions are typically not synced back from vault to local,
	// as they're generated locally. Skip for now.
	log.Printf("vault sync: skipping suggestion change %s (suggestions are local-only)", payload.ID)
	return nil
}

func fallbackName(name string, id uuid.UUID) string {
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		return trimmed
	}
	return id.String()
}

func (s *VaultSyncer) ensureCompany(ctx context.Context, id uuid.UUID, name string) (*models.Company, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	company, err := db.GetCompany(s.appDB, id)
	if err != nil {
		return nil, err
	}
	if company != nil {
		return company, nil
	}

	placeholder := &models.Company{
		ID:   id,
		Name: fallbackName(name, id),
	}
	if err := db.CreateCompany(s.appDB, placeholder); err != nil {
		return nil, err
	}
	return placeholder, nil
}

func (s *VaultSyncer) ensureContact(ctx context.Context, id uuid.UUID, name string, companyID *uuid.UUID) (*models.Contact, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	contact, err := db.GetContact(s.appDB, id)
	if err != nil {
		return nil, err
	}
	if contact != nil {
		return contact, nil
	}

	placeholder := &models.Contact{
		ID:   id,
		Name: fallbackName(name, id),
	}
	if companyID != nil {
		placeholder.CompanyID = companyID
	}
	if err := db.CreateContact(s.appDB, placeholder); err != nil {
		return nil, err
	}
	return placeholder, nil
}

func (s *VaultSyncer) findContactByName(name string, companyID *uuid.UUID) (*models.Contact, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	contacts, err := db.FindContacts(s.appDB, name, companyID, 1)
	if err != nil {
		return nil, err
	}
	if len(contacts) == 0 {
		return nil, nil
	}
	return &contacts[0], nil
}

func (s *VaultSyncer) resolveContactIdentifier(ctx context.Context, idStr, name string) (uuid.UUID, error) {
	if idStr != "" {
		if parsed, err := uuid.Parse(idStr); err == nil {
			contact, err := s.ensureContact(ctx, parsed, name, nil)
			if err != nil {
				return uuid.Nil, err
			}
			return contact.ID, nil
		}
	}
	if name != "" {
		contact, err := s.findContactByName(name, nil)
		if err != nil {
			return uuid.Nil, err
		}
		if contact != nil {
			return contact.ID, nil
		}
		placeholder, err := s.ensureContact(ctx, uuid.New(), name, nil)
		if err != nil {
			return uuid.Nil, err
		}
		return placeholder.ID, nil
	}
	return uuid.Nil, fmt.Errorf("missing contact information")
}

func (s *VaultSyncer) resolveCompanyForDeal(ctx context.Context, payload DealPayload) (*models.Company, error) {
	if payload.CompanyID != "" {
		if parsed, err := uuid.Parse(payload.CompanyID); err == nil {
			return s.ensureCompany(ctx, parsed, payload.CompanyName)
		}
	}
	if payload.CompanyName != "" {
		company, err := db.FindCompanyByName(s.appDB, payload.CompanyName)
		if err != nil {
			return nil, fmt.Errorf("failed to find company: %w", err)
		}
		if company != nil {
			return company, nil
		}
	}
	return nil, fmt.Errorf("company not found for deal %s", payload.ID)
}

func (s *VaultSyncer) resolveNoteCompany(ctx context.Context, payload DealNotePayload) (*models.Company, error) {
	if payload.DealCompanyID != "" {
		if parsed, err := uuid.Parse(payload.DealCompanyID); err == nil {
			return s.ensureCompany(ctx, parsed, payload.DealCompanyName)
		}
	}
	if payload.DealCompanyName != "" {
		company, err := db.FindCompanyByName(s.appDB, payload.DealCompanyName)
		if err != nil {
			return nil, fmt.Errorf("failed to find company for deal note: %w", err)
		}
		if company != nil {
			return company, nil
		}
	}
	return nil, fmt.Errorf("company information missing for deal note %s", payload.ID)
}
