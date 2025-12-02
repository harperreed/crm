// ABOUTME: Legacy API compatibility layer using Office OS foundation
// ABOUTME: Provides old function signatures while using ObjectsRepository underneath

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

// Legacy Company Functions

func CreateCompany(db *sql.DB, company *models.Company) error {
	// Generate ID and timestamps if not set
	if company.ID == uuid.Nil {
		company.ID = uuid.New()
	}
	if company.CreatedAt.IsZero() {
		now := time.Now()
		company.CreatedAt = now
		company.UpdatedAt = now
	}

	repo := NewObjectsRepository(db)
	obj := CompanyToObject(company)
	return repo.Create(context.Background(), obj)
}

func GetCompany(db *sql.DB, id uuid.UUID) (*models.Company, error) {
	repo := NewObjectsRepository(db)
	obj, err := repo.Get(context.Background(), id.String())
	if errors.Is(err, ErrObjectNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ObjectToCompany(obj)
}

func FindCompanies(db *sql.DB, query string, limit int) ([]models.Company, error) {
	if limit <= 0 {
		limit = 10
	}

	repo := NewObjectsRepository(db)
	objects, err := repo.List(context.Background(), ObjectTypeCompany)
	if err != nil {
		return nil, err
	}

	var companies []models.Company
	queryLower := strings.ToLower(query)

	for _, obj := range objects {
		// Apply search filter if query is provided
		if query != "" {
			nameLower := strings.ToLower(getStringFromMetadata(obj.Fields, "name"))
			domainLower := strings.ToLower(getStringFromMetadata(obj.Fields, "domain"))

			if !strings.Contains(nameLower, queryLower) && !strings.Contains(domainLower, queryLower) {
				continue
			}
		}

		company, err := ObjectToCompany(obj)
		if err != nil {
			continue // Skip malformed objects
		}

		companies = append(companies, *company)

		if len(companies) >= limit {
			break
		}
	}

	return companies, nil
}

func FindCompanyByName(db *sql.DB, name string) (*models.Company, error) {
	repo := NewObjectsRepository(db)
	objects, err := repo.List(context.Background(), ObjectTypeCompany)
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, obj := range objects {
		objName := getStringFromMetadata(obj.Fields, "name")
		if strings.ToLower(objName) == nameLower {
			return ObjectToCompany(obj)
		}
	}

	return nil, nil
}

func UpdateCompany(db *sql.DB, id uuid.UUID, updates *models.Company) error {
	repo := NewObjectsRepository(db)

	// Get existing object to preserve ID and timestamps
	existing, err := repo.Get(context.Background(), id.String())
	if err != nil {
		return err
	}

	// Update fields from the updates parameter
	existing.Fields["name"] = updates.Name
	existing.Fields["domain"] = updates.Domain
	existing.Fields["industry"] = updates.Industry
	existing.Fields["notes"] = updates.Notes

	return repo.Update(context.Background(), existing)
}

func DeleteCompany(db *sql.DB, id uuid.UUID) error {
	// Check if company has deals
	repo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)

	// Find all deals related to this company
	deals, err := repo.List(context.Background(), ObjectTypeDeal)
	if err != nil {
		return fmt.Errorf("failed to check deals: %w", err)
	}

	dealCount := 0
	for _, deal := range deals {
		companyIDStr := getStringFromMetadata(deal.Fields, "company_id")
		if companyIDStr == id.String() {
			dealCount++
		}
	}

	if dealCount > 0 {
		return fmt.Errorf("cannot delete company with %d active deals", dealCount)
	}

	// Remove company_id from contacts that reference this company
	contacts, err := repo.List(context.Background(), ObjectTypeContact)
	if err != nil {
		return fmt.Errorf("failed to update contacts: %w", err)
	}

	for _, contact := range contacts {
		companyIDStr := getStringFromMetadata(contact.Fields, "company_id")
		if companyIDStr == id.String() {
			delete(contact.Fields, "company_id")
			if err := repo.Update(context.Background(), contact); err != nil {
				return fmt.Errorf("failed to update contact: %w", err)
			}
		}
	}

	// Delete works_at relationships
	rels, err := relRepo.FindByTarget(context.Background(), id.String(), RelTypeWorksAt)
	if err != nil {
		return fmt.Errorf("failed to query relationships: %w", err)
	}
	for _, rel := range rels {
		if err := relRepo.Delete(context.Background(), rel.ID); err != nil {
			return fmt.Errorf("failed to delete relationship: %w", err)
		}
	}

	// Delete the company
	return repo.Delete(context.Background(), id.String())
}

// Legacy Contact Functions

func CreateContact(db *sql.DB, contact *models.Contact) error {
	// Generate ID and timestamps if not set
	if contact.ID == uuid.Nil {
		contact.ID = uuid.New()
	}
	if contact.CreatedAt.IsZero() {
		now := time.Now()
		contact.CreatedAt = now
		contact.UpdatedAt = now
	}

	repo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)

	obj := ContactToObject(contact)
	if err := repo.Create(context.Background(), obj); err != nil {
		return err
	}

	// Create works_at relationship if company is set
	if contact.CompanyID != nil {
		rel := &Relationship{
			SourceID: contact.ID.String(),
			TargetID: contact.CompanyID.String(),
			Type:     RelTypeWorksAt,
			Metadata: make(map[string]interface{}),
		}
		if err := relRepo.Create(context.Background(), rel); err != nil {
			// Log but don't fail the contact creation
			fmt.Printf("Warning: failed to create works_at relationship: %v\n", err)
		}
	}

	return nil
}

func GetContact(db *sql.DB, id uuid.UUID) (*models.Contact, error) {
	repo := NewObjectsRepository(db)
	obj, err := repo.Get(context.Background(), id.String())
	if errors.Is(err, ErrObjectNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ObjectToContact(obj)
}

func FindContacts(db *sql.DB, query string, companyID *uuid.UUID, limit int) ([]models.Contact, error) {
	if limit <= 0 {
		limit = 10
	}

	repo := NewObjectsRepository(db)
	objects, err := repo.List(context.Background(), ObjectTypeContact)
	if err != nil {
		return nil, err
	}

	var contacts []models.Contact
	queryLower := strings.ToLower(query)

	for _, obj := range objects {
		// Apply company filter if provided
		if companyID != nil {
			objCompanyID := getStringFromMetadata(obj.Fields, "company_id")
			if objCompanyID != companyID.String() {
				continue
			}
		}

		// Apply search filter if query is provided
		if query != "" {
			nameLower := strings.ToLower(getStringFromMetadata(obj.Fields, "name"))
			emailLower := strings.ToLower(getStringFromMetadata(obj.Fields, "email"))

			if !strings.Contains(nameLower, queryLower) && !strings.Contains(emailLower, queryLower) {
				continue
			}
		}

		contact, err := ObjectToContact(obj)
		if err != nil {
			continue // Skip malformed objects
		}

		contacts = append(contacts, *contact)

		if len(contacts) >= limit {
			break
		}
	}

	return contacts, nil
}

func UpdateContact(db *sql.DB, id uuid.UUID, updates *models.Contact) error {
	repo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)

	// Get existing object
	existing, err := repo.Get(context.Background(), id.String())
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return fmt.Errorf("contact not found: %s", id)
		}
		return err
	}

	// Get old company ID before update
	oldCompanyIDStr := getStringFromMetadata(existing.Fields, "company_id")

	// Update fields
	existing.Fields["name"] = updates.Name
	existing.Fields["email"] = updates.Email
	existing.Fields["phone"] = updates.Phone
	existing.Fields["notes"] = updates.Notes

	if updates.CompanyID != nil {
		existing.Fields["company_id"] = updates.CompanyID.String()
	} else {
		delete(existing.Fields, "company_id")
	}

	if err := repo.Update(context.Background(), existing); err != nil {
		return err
	}

	// Update works_at relationship if company changed
	newCompanyIDStr := ""
	if updates.CompanyID != nil {
		newCompanyIDStr = updates.CompanyID.String()
	}

	if oldCompanyIDStr != newCompanyIDStr {
		// Delete old relationship
		if oldCompanyIDStr != "" {
			rels, err := relRepo.FindBySource(context.Background(), id.String(), RelTypeWorksAt)
			if err == nil {
				for _, rel := range rels {
					if rel.TargetID == oldCompanyIDStr {
						_ = relRepo.Delete(context.Background(), rel.ID)
					}
				}
			}
		}

		// Create new relationship
		if newCompanyIDStr != "" {
			rel := &Relationship{
				SourceID: id.String(),
				TargetID: newCompanyIDStr,
				Type:     RelTypeWorksAt,
				Metadata: make(map[string]interface{}),
			}
			_ = relRepo.Create(context.Background(), rel)
		}
	}

	return nil
}

func DeleteContact(db *sql.DB, id uuid.UUID) error {
	repo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)

	// Delete all relationships involving this contact
	// The relationships table has ON DELETE CASCADE, but we'll be explicit
	sourceRels, err := relRepo.FindBySource(context.Background(), id.String(), "")
	if err == nil {
		for _, rel := range sourceRels {
			_ = relRepo.Delete(context.Background(), rel.ID)
		}
	}

	targetRels, err := relRepo.FindByTarget(context.Background(), id.String(), "")
	if err == nil {
		for _, rel := range targetRels {
			_ = relRepo.Delete(context.Background(), rel.ID)
		}
	}

	// Remove contact_id from deals that reference this contact
	deals, err := repo.List(context.Background(), ObjectTypeDeal)
	if err != nil {
		return fmt.Errorf("failed to update deals: %w", err)
	}

	for _, deal := range deals {
		contactIDStr := getStringFromMetadata(deal.Fields, "contact_id")
		if contactIDStr == id.String() {
			delete(deal.Fields, "contact_id")
			if err := repo.Update(context.Background(), deal); err != nil {
				return fmt.Errorf("failed to update deal: %w", err)
			}
		}
	}

	// Delete the contact
	return repo.Delete(context.Background(), id.String())
}

func UpdateContactLastContacted(db *sql.DB, contactID uuid.UUID, timestamp time.Time) error {
	repo := NewObjectsRepository(db)

	obj, err := repo.Get(context.Background(), contactID.String())
	if err != nil {
		return err
	}

	obj.Fields["last_contacted_at"] = timestamp.Format(time.RFC3339Nano)
	return repo.Update(context.Background(), obj)
}

// Legacy Deal Functions

func CreateDeal(db *sql.DB, deal *models.Deal) error {
	// Generate ID and timestamps if not set
	if deal.ID == uuid.Nil {
		deal.ID = uuid.New()
	}
	if deal.CreatedAt.IsZero() {
		now := time.Now()
		deal.CreatedAt = now
		deal.UpdatedAt = now
		deal.LastActivityAt = now
	}
	if deal.Currency == "" {
		deal.Currency = "USD"
	}

	repo := NewObjectsRepository(db)
	obj := DealToObject(deal)
	return repo.Create(context.Background(), obj)
}

func GetDeal(db *sql.DB, id uuid.UUID) (*models.Deal, error) {
	repo := NewObjectsRepository(db)
	obj, err := repo.Get(context.Background(), id.String())
	if errors.Is(err, ErrObjectNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ObjectToDeal(obj)
}

func UpdateDeal(db *sql.DB, deal *models.Deal) error {
	// Update last_activity_at timestamp
	deal.LastActivityAt = time.Now().UTC()

	repo := NewObjectsRepository(db)
	obj := DealToObject(deal)
	return repo.Update(context.Background(), obj)
}

func FindDeals(db *sql.DB, stage string, companyID *uuid.UUID, limit int) ([]models.Deal, error) {
	if limit <= 0 {
		limit = 10
	}

	repo := NewObjectsRepository(db)
	objects, err := repo.List(context.Background(), ObjectTypeDeal)
	if err != nil {
		return nil, err
	}

	var deals []models.Deal

	for _, obj := range objects {
		// Apply company filter if provided
		if companyID != nil {
			objCompanyID := getStringFromMetadata(obj.Fields, "company_id")
			if objCompanyID != companyID.String() {
				continue
			}
		}

		// Apply stage filter if provided
		if stage != "" {
			objStage := getStringFromMetadata(obj.Fields, "stage")
			if objStage != stage {
				continue
			}
		}

		deal, err := ObjectToDeal(obj)
		if err != nil {
			continue // Skip malformed objects
		}

		deals = append(deals, *deal)

		if len(deals) >= limit {
			break
		}
	}

	return deals, nil
}

func DeleteDeal(db *sql.DB, id uuid.UUID) error {
	repo := NewObjectsRepository(db)
	// Note: Deal notes would be stored as separate objects with relationships
	// For now, just delete the deal itself
	return repo.Delete(context.Background(), id.String())
}

// Deal Note functions
// Note: Deal notes are stored as metadata for now since they're tightly coupled to deals

func AddDealNote(db *sql.DB, note *models.DealNote) error {
	note.ID = uuid.New()
	note.CreatedAt = time.Now()

	// Get the deal
	repo := NewObjectsRepository(db)
	deal, err := repo.Get(context.Background(), note.DealID.String())
	if err != nil {
		return err
	}

	// Add note to deal fields
	// Notes are stored as an array in fields
	var notes []map[string]interface{}
	if existingNotes, ok := deal.Fields["notes"].([]interface{}); ok {
		for _, n := range existingNotes {
			if noteMap, ok := n.(map[string]interface{}); ok {
				notes = append(notes, noteMap)
			}
		}
	}

	newNote := map[string]interface{}{
		"id":         note.ID.String(),
		"content":    note.Content,
		"created_at": note.CreatedAt.Format(time.RFC3339Nano),
	}
	notes = append(notes, newNote)
	deal.Fields["notes"] = notes

	// Update last_activity_at
	deal.Fields["last_activity_at"] = note.CreatedAt.Format(time.RFC3339Nano)

	if err := repo.Update(context.Background(), deal); err != nil {
		return err
	}

	// Update contact's last_contacted_at if deal has a contact
	if contactIDStr := getStringFromMetadata(deal.Fields, "contact_id"); contactIDStr != "" {
		contact, err := repo.Get(context.Background(), contactIDStr)
		if err == nil {
			contact.Fields["last_contacted_at"] = note.CreatedAt.Format(time.RFC3339Nano)
			_ = repo.Update(context.Background(), contact)
		}
	}

	return nil
}

func GetDealNotes(db *sql.DB, dealID uuid.UUID) ([]models.DealNote, error) {
	repo := NewObjectsRepository(db)
	deal, err := repo.Get(context.Background(), dealID.String())
	if err != nil {
		return nil, err
	}

	var notes []models.DealNote

	// Extract notes from fields
	if existingNotes, ok := deal.Fields["notes"].([]interface{}); ok {
		for _, n := range existingNotes {
			if noteMap, ok := n.(map[string]interface{}); ok {
				note := models.DealNote{
					DealID: dealID,
				}

				if idStr, ok := noteMap["id"].(string); ok {
					id, err := uuid.Parse(idStr)
					if err == nil {
						note.ID = id
					}
				}

				if content, ok := noteMap["content"].(string); ok {
					note.Content = content
				}

				if createdStr, ok := noteMap["created_at"].(string); ok {
					created, err := time.Parse(time.RFC3339Nano, createdStr)
					if err == nil {
						note.CreatedAt = created
					}
				}

				notes = append(notes, note)
			}
		}
	}

	// Return in descending order (newest first)
	for i, j := 0, len(notes)-1; i < j; i, j = i+1, j-1 {
		notes[i], notes[j] = notes[j], notes[i]
	}

	return notes, nil
}

// Legacy Relationship Functions (for the old contacts-to-contacts relationships)
// Note: these are internal helpers - the public API is in relationships.go

func GetRelationshipsBetween(db *sql.DB, contactID1, contactID2 uuid.UUID) ([]models.Relationship, error) {
	relRepo := NewRelationshipsRepository(db)

	rels, err := relRepo.FindBetween(context.Background(), contactID1.String(), contactID2.String())
	if err != nil {
		return nil, err
	}

	var relationships []models.Relationship
	for _, rel := range rels {
		// Only return "knows" type relationships for legacy compatibility
		if rel.Type != RelTypeKnows {
			continue
		}

		sourceID, _ := uuid.Parse(rel.SourceID)
		targetID, _ := uuid.Parse(rel.TargetID)
		id, _ := uuid.Parse(rel.ID)

		relationship := models.Relationship{
			ID:         id,
			ContactID1: sourceID,
			ContactID2: targetID,
			CreatedAt:  rel.CreatedAt,
			UpdatedAt:  rel.UpdatedAt,
		}

		if relType, ok := rel.Metadata["relationship_type"].(string); ok {
			relationship.RelationshipType = relType
		}
		if context, ok := rel.Metadata["context"].(string); ok {
			relationship.Context = context
		}

		relationships = append(relationships, relationship)
	}

	return relationships, nil
}

func GetContactRelationships(db *sql.DB, contactID uuid.UUID) ([]models.Relationship, error) {
	relRepo := NewRelationshipsRepository(db)

	// Get relationships where contact is source
	sourceRels, err := relRepo.FindBySource(context.Background(), contactID.String(), RelTypeKnows)
	if err != nil {
		return nil, err
	}

	// Get relationships where contact is target
	targetRels, err := relRepo.FindByTarget(context.Background(), contactID.String(), RelTypeKnows)
	if err != nil {
		return nil, err
	}

	var relationships []models.Relationship

	// Convert source relationships
	for _, rel := range sourceRels {
		sourceID, _ := uuid.Parse(rel.SourceID)
		targetID, _ := uuid.Parse(rel.TargetID)
		id, _ := uuid.Parse(rel.ID)

		relationship := models.Relationship{
			ID:         id,
			ContactID1: sourceID,
			ContactID2: targetID,
			CreatedAt:  rel.CreatedAt,
			UpdatedAt:  rel.UpdatedAt,
		}

		if relType, ok := rel.Metadata["relationship_type"].(string); ok {
			relationship.RelationshipType = relType
		}
		if context, ok := rel.Metadata["context"].(string); ok {
			relationship.Context = context
		}

		relationships = append(relationships, relationship)
	}

	// Convert target relationships
	for _, rel := range targetRels {
		sourceID, _ := uuid.Parse(rel.SourceID)
		targetID, _ := uuid.Parse(rel.TargetID)
		id, _ := uuid.Parse(rel.ID)

		relationship := models.Relationship{
			ID:         id,
			ContactID1: sourceID,
			ContactID2: targetID,
			CreatedAt:  rel.CreatedAt,
			UpdatedAt:  rel.UpdatedAt,
		}

		if relType, ok := rel.Metadata["relationship_type"].(string); ok {
			relationship.RelationshipType = relType
		}
		if context, ok := rel.Metadata["context"].(string); ok {
			relationship.Context = context
		}

		relationships = append(relationships, relationship)
	}

	return relationships, nil
}
