// ABOUTME: Google Contacts API importer
// ABOUTME: Fetches and imports contacts from Google People API with deduplication
package sync

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

type ContactsImporter struct {
	db      *sql.DB
	matcher *ContactMatcher
}

type GoogleContact struct {
	ResourceName string
	Name         string
	Email        string
	Phone        string
	Company      string
	JobTitle     string
	Notes        string
}

func NewContactsImporter(database *sql.DB) *ContactsImporter {
	return &ContactsImporter{
		db: database,
	}
}

// ImportContact imports a single contact from Google
func (ci *ContactsImporter) ImportContact(gc *GoogleContact) (bool, error) {
	// Reload matcher with current contacts
	contacts, err := db.FindContacts(ci.db, "", nil, 10000)
	if err != nil {
		return false, fmt.Errorf("failed to load contacts: %w", err)
	}
	ci.matcher = NewContactMatcher(contacts)

	// Check for existing contact
	existing, found := ci.matcher.FindMatch(gc.Email, gc.Name)
	if found {
		// Update existing contact if needed
		if err := ci.updateContact(existing, gc); err != nil {
			return false, err
		}
		return false, nil
	}

	// Create new contact
	contact := &models.Contact{
		Name:  gc.Name,
		Email: gc.Email,
		Phone: gc.Phone,
		Notes: gc.Notes,
	}

	// Handle company
	if gc.Company != "" {
		company, err := ci.findOrCreateCompany(gc.Company)
		if err != nil {
			return false, fmt.Errorf("failed to handle company: %w", err)
		}
		contact.CompanyID = &company.ID
	}

	// Create contact
	if err := db.CreateContact(ci.db, contact); err != nil {
		return false, fmt.Errorf("failed to create contact: %w", err)
	}

	// Log sync
	if err := ci.logSync(gc.ResourceName, contact.ID); err != nil {
		return false, fmt.Errorf("failed to log sync: %w", err)
	}

	return true, nil
}

func (ci *ContactsImporter) updateContact(existing *models.Contact, gc *GoogleContact) error {
	// Only update if Google data is more complete
	updated := false

	if gc.Phone != "" && existing.Phone == "" {
		existing.Phone = gc.Phone
		updated = true
	}

	if gc.Notes != "" && existing.Notes == "" {
		existing.Notes = gc.Notes
		updated = true
	}

	if !updated {
		return nil
	}

	return db.UpdateContact(ci.db, existing.ID, existing)
}

func (ci *ContactsImporter) findOrCreateCompany(name string) (*models.Company, error) {
	// Try to find existing company
	company, err := db.FindCompanyByName(ci.db, name)
	if err != nil {
		return nil, err
	}

	if company != nil {
		return company, nil
	}

	// Create new company
	newCompany := &models.Company{
		Name: name,
	}

	if err := db.CreateCompany(ci.db, newCompany); err != nil {
		return nil, err
	}

	return newCompany, nil
}

func (ci *ContactsImporter) logSync(sourceID string, entityID uuid.UUID) error {
	syncLog := &models.SyncLog{
		ID:            uuid.New(),
		SourceService: "contacts",
		SourceID:      sourceID,
		EntityType:    "contact",
		EntityID:      entityID,
	}

	query := `
		INSERT INTO sync_log (id, source_service, source_id, entity_type, entity_id)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := ci.db.Exec(query,
		syncLog.ID.String(),
		syncLog.SourceService,
		syncLog.SourceID,
		syncLog.EntityType,
		syncLog.EntityID.String(),
	)

	return err
}
