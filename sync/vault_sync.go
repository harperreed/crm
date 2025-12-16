// ABOUTME: Vault sync implementation for bidirectional CRM entity synchronization
// ABOUTME: Queues local changes, applies remote changes, and handles encrypted sync with vault server

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

// VaultSyncer manages bidirectional synchronization with vault server.
type VaultSyncer struct {
	config *VaultConfig
	store  *vault.Store
	keys   vault.Keys
	client *vault.Client
	appDB  *sql.DB
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
		BaseURL:      cfg.Server,
		DeviceID:     cfg.DeviceID,
		AuthToken:    cfg.Token,
		RefreshToken: cfg.RefreshToken,
	})

	return &VaultSyncer{
		config: cfg,
		store:  store,
		keys:   keys,
		client: client,
		appDB:  appDB,
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
	return vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange)
}

// SyncWithEvents performs a full bidirectional sync with event callbacks.
func (s *VaultSyncer) SyncWithEvents(ctx context.Context, events vault.SyncEvents) error {
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

// QueueContactChange queues a contact change for sync.
func (s *VaultSyncer) QueueContactChange(ctx context.Context, contact *models.Contact, companyName string, op vault.Op) error {
	var lastContactedAt *string
	if contact.LastContactedAt != nil {
		ts := contact.LastContactedAt.Format(time.RFC3339)
		lastContactedAt = &ts
	}

	payload := ContactPayload{
		ID:              contact.ID.String(),
		Name:            contact.Name,
		Email:           contact.Email,
		Phone:           contact.Phone,
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

	payload := DealPayload{
		ID:                deal.ID.String(),
		Title:             deal.Title,
		Amount:            deal.Amount,
		Currency:          deal.Currency,
		Stage:             deal.Stage,
		CompanyName:       companyName,
		ContactName:       contactName,
		ExpectedCloseDate: expectedCloseDate,
	}
	return s.queueChange(ctx, EntityDeal, deal.ID.String(), op, payload)
}

// QueueDealNoteChange queues a deal note change for sync.
func (s *VaultSyncer) QueueDealNoteChange(ctx context.Context, note *models.DealNote, dealTitle string, op vault.Op) error {
	payload := DealNotePayload{
		ID:        note.ID.String(),
		DealTitle: dealTitle,
		Content:   note.Content,
		CreatedAt: note.CreatedAt.Format(time.RFC3339),
	}
	return s.queueChange(ctx, EntityDealNote, note.ID.String(), op, payload)
}

// QueueRelationshipChange queues a relationship change for sync.
func (s *VaultSyncer) QueueRelationshipChange(ctx context.Context, rel *models.Relationship, contact1Name, contact2Name string, op vault.Op) error {
	payload := RelationshipPayload{
		ID:               rel.ID.String(),
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
	// Create change
	change, err := vault.NewChange(entity, entityID, op, payload)
	if err != nil {
		return fmt.Errorf("failed to create change: %w", err)
	}

	// Build AAD for authenticated encryption
	aad := change.AAD(s.config.UserID, s.config.DeviceID)

	// Marshal payload to JSON
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Encrypt payload
	env, err := vault.Encrypt(s.keys.EncKey, plaintext, aad)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	// Enqueue encrypted change
	if err := s.store.EnqueueEncryptedChange(ctx, change, s.config.UserID, s.config.DeviceID, env); err != nil {
		return fmt.Errorf("failed to enqueue change: %w", err)
	}

	// Auto-sync if enabled
	if s.config.AutoSync {
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = s.Sync(syncCtx)
		}()
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
		// Build contact model
		contact := &models.Contact{
			ID:    id,
			Name:  payload.Name,
			Email: payload.Email,
			Phone: payload.Phone,
			Notes: payload.Notes,
		}

		// Parse last contacted timestamp
		if payload.LastContactedAt != nil {
			t, err := time.Parse(time.RFC3339, *payload.LastContactedAt)
			if err == nil {
				contact.LastContactedAt = &t
			}
		}

		// Look up company by name if provided
		if payload.CompanyName != "" {
			company, err := db.FindCompanyByName(s.appDB, payload.CompanyName)
			if err != nil {
				return fmt.Errorf("failed to find company: %w", err)
			}
			if company != nil {
				contact.CompanyID = &company.ID
			}
		}

		// Check if contact exists
		existing, err := db.GetContact(s.appDB, id)
		if err != nil {
			return fmt.Errorf("failed to check existing contact: %w", err)
		}

		if existing == nil {
			// Create new contact
			if err := db.CreateContact(s.appDB, contact); err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}
		} else {
			// Update existing contact
			if err := db.UpdateContact(s.appDB, id, contact); err != nil {
				return fmt.Errorf("failed to update contact: %w", err)
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
		// Look up company by name
		company, err := db.FindCompanyByName(s.appDB, payload.CompanyName)
		if err != nil {
			return fmt.Errorf("failed to find company: %w", err)
		}
		if company == nil {
			return fmt.Errorf("company not found: %s", payload.CompanyName)
		}

		deal := &models.Deal{
			ID:        id,
			Title:     payload.Title,
			Amount:    payload.Amount,
			Currency:  payload.Currency,
			Stage:     payload.Stage,
			CompanyID: company.ID,
		}

		// Parse expected close date
		if payload.ExpectedCloseDate != nil {
			t, err := time.Parse(time.RFC3339, *payload.ExpectedCloseDate)
			if err == nil {
				deal.ExpectedCloseDate = &t
			}
		}

		// Look up contact by name if provided
		if payload.ContactName != "" {
			contacts, err := db.FindContacts(s.appDB, payload.ContactName, &company.ID, 1)
			if err != nil {
				return fmt.Errorf("failed to find contact: %w", err)
			}
			if len(contacts) > 0 {
				deal.ContactID = &contacts[0].ID
			}
		}

		// Check if deal exists
		existing, err := db.GetDeal(s.appDB, id)
		if err != nil {
			return fmt.Errorf("failed to check existing deal: %w", err)
		}

		if existing == nil {
			if err := db.CreateDeal(s.appDB, deal); err != nil {
				return fmt.Errorf("failed to create deal: %w", err)
			}
		} else {
			// Update existing deal
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
		// Find deal by title
		deals, err := db.FindDeals(s.appDB, "", nil, 100)
		if err != nil {
			return fmt.Errorf("failed to find deals: %w", err)
		}

		var dealID uuid.UUID
		for _, deal := range deals {
			if deal.Title == payload.DealTitle {
				dealID = deal.ID
				break
			}
		}

		if dealID == uuid.Nil {
			return fmt.Errorf("deal not found: %s", payload.DealTitle)
		}

		createdAt, err := time.Parse(time.RFC3339, payload.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse created_at: %w", err)
		}

		note := &models.DealNote{
			ID:        id,
			DealID:    dealID,
			Content:   payload.Content,
			CreatedAt: createdAt,
		}

		if err := db.AddDealNote(s.appDB, note); err != nil {
			return fmt.Errorf("failed to add deal note: %w", err)
		}

	case vault.OpDelete:
		// Deal notes are deleted with their parent deal in the current implementation
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

	// Note: Relationship sync is complex as it depends on two contacts existing.
	// For now, we'll skip implementing this to avoid errors.
	// This can be implemented when there's a real use case.
	log.Printf("vault sync: skipping relationship change %s (not yet implemented)", payload.ID)
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
		// Find contact by name
		contacts, err := db.FindContacts(s.appDB, payload.ContactName, nil, 1)
		if err != nil {
			return fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return fmt.Errorf("contact not found: %s", payload.ContactName)
		}

		timestamp, err := time.Parse(time.RFC3339, payload.InteractedAt)
		if err != nil {
			return fmt.Errorf("failed to parse timestamp: %w", err)
		}

		interaction := &models.InteractionLog{
			ID:              id,
			ContactID:       contacts[0].ID,
			InteractionType: payload.InteractionType,
			Timestamp:       timestamp,
			Sentiment:       payload.Sentiment,
			Metadata:        payload.Metadata,
		}

		if err := db.LogInteraction(s.appDB, interaction); err != nil {
			return fmt.Errorf("failed to log interaction: %w", err)
		}

	case vault.OpDelete:
		// Interaction logs are typically not deleted, skip for now
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
