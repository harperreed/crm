// ABOUTME: Cascade delete operations for parent-child relationships
// ABOUTME: KV has no foreign keys, so we handle cascades manually

package charm

import (
	"github.com/google/uuid"
)

// DeleteCompanyWithCascade deletes a company and all related entities
// Cascades: contacts, deals, deal notes (via deals).
func (c *Client) DeleteCompanyWithCascade(id uuid.UUID) error {
	// 1. Delete all contacts for this company
	contacts, err := c.ListContacts(&ContactFilter{CompanyID: &id})
	if err != nil {
		return err
	}
	for _, contact := range contacts {
		// This will cascade delete contact's children too
		if err := c.DeleteContactWithCascade(contact.ID); err != nil {
			return err
		}
	}

	// 2. Delete all deals for this company
	deals, err := c.ListDeals(&DealFilter{CompanyID: &id})
	if err != nil {
		return err
	}
	for _, deal := range deals {
		// This will cascade delete deal's notes too
		if err := c.DeleteDealWithCascade(deal.ID); err != nil {
			return err
		}
	}

	// 3. Delete the company itself
	return c.DeleteCompany(id)
}

// DeleteContactWithCascade deletes a contact and all related entities
// Cascades: relationships, interaction logs, cadence settings.
func (c *Client) DeleteContactWithCascade(id uuid.UUID) error {
	// 1. Delete all relationships involving this contact
	rels, err := c.ListRelationshipsForContact(id)
	if err != nil {
		return err
	}
	for _, rel := range rels {
		if err := c.DeleteRelationship(rel.ID); err != nil {
			return err
		}
	}

	// 2. Delete all interaction logs for this contact
	logs, err := c.ListInteractionLogs(&InteractionFilter{ContactID: &id})
	if err != nil {
		return err
	}
	for _, log := range logs {
		if err := c.DeleteInteractionLog(log.ID); err != nil {
			return err
		}
	}

	// 3. Delete cadence settings for this contact (ignore error - cadence may not exist)
	_ = c.DeleteContactCadence(id)

	// 4. Update deals that reference this contact (nullify the contact_id)
	deals, err := c.ListDeals(&DealFilter{ContactID: &id})
	if err != nil {
		return err
	}
	for _, deal := range deals {
		deal.ContactID = nil
		deal.ContactName = ""
		if err := c.UpdateDeal(deal); err != nil {
			return err
		}
	}

	// 5. Delete the contact itself
	return c.DeleteContact(id)
}

// DeleteDealWithCascade deletes a deal and all related entities
// Cascades: deal notes.
func (c *Client) DeleteDealWithCascade(id uuid.UUID) error {
	// 1. Delete all notes for this deal
	notes, err := c.ListDealNotes(id)
	if err != nil {
		return err
	}
	for _, note := range notes {
		if err := c.DeleteDealNote(note.ID); err != nil {
			return err
		}
	}

	// 2. Delete the deal itself
	return c.DeleteDeal(id)
}

// UpdateCompanyDenormalizedNames updates all entities that have denormalized company name
// Call this when a company name changes.
func (c *Client) UpdateCompanyDenormalizedNames(companyID uuid.UUID, newName string) error {
	// Update contacts
	contacts, err := c.ListContacts(&ContactFilter{CompanyID: &companyID})
	if err != nil {
		return err
	}
	for _, contact := range contacts {
		contact.CompanyName = newName
		if err := c.UpdateContact(contact); err != nil {
			return err
		}
	}

	// Update deals
	deals, err := c.ListDeals(&DealFilter{CompanyID: &companyID})
	if err != nil {
		return err
	}
	for _, deal := range deals {
		deal.CompanyName = newName
		if err := c.UpdateDeal(deal); err != nil {
			return err
		}

		// Update deal notes too
		notes, err := c.ListDealNotes(deal.ID)
		if err != nil {
			continue
		}
		for _, note := range notes {
			note.DealCompanyName = newName
			data, _ := c.Get(DealNoteKey(note.ID.String()))
			if data != nil {
				_ = c.Set(DealNoteKey(note.ID.String()), data) // Just to trigger sync
			}
		}
	}

	return nil
}

// UpdateContactDenormalizedNames updates all entities that have denormalized contact name
// Call this when a contact name changes.
func (c *Client) UpdateContactDenormalizedNames(contactID uuid.UUID, newName string) error {
	// Update relationships where this contact is contact1 or contact2
	rels, err := c.ListRelationshipsForContact(contactID)
	if err != nil {
		return err
	}
	for _, rel := range rels {
		if rel.ContactID1 == contactID {
			rel.Contact1Name = newName
		}
		if rel.ContactID2 == contactID {
			rel.Contact2Name = newName
		}
		if err := c.UpdateRelationship(rel); err != nil {
			return err
		}
	}

	// Update interaction logs
	logs, err := c.ListInteractionLogs(&InteractionFilter{ContactID: &contactID})
	if err != nil {
		return err
	}
	for _, log := range logs {
		log.ContactName = newName
		// Re-save with updated name
		data, _ := c.Get(InteractionLogKey(log.ID.String()))
		if data != nil {
			_ = c.Set(InteractionLogKey(log.ID.String()), data)
		}
	}

	// Update deals where this contact is the contact
	deals, err := c.ListDeals(&DealFilter{ContactID: &contactID})
	if err != nil {
		return err
	}
	for _, deal := range deals {
		deal.ContactName = newName
		if err := c.UpdateDeal(deal); err != nil {
			return err
		}
	}

	// Update cadence
	cadence, err := c.GetContactCadence(contactID)
	if err == nil && cadence != nil {
		cadence.ContactName = newName
		if err := c.SaveContactCadence(cadence); err != nil {
			return err
		}
	}

	return nil
}

// UpdateDealDenormalizedNames updates all entities that have denormalized deal info
// Call this when a deal title changes.
func (c *Client) UpdateDealDenormalizedNames(dealID uuid.UUID, newTitle string) error {
	// Update deal notes
	notes, err := c.ListDealNotes(dealID)
	if err != nil {
		return err
	}
	for _, note := range notes {
		note.DealTitle = newTitle
		// Re-save with updated title
		data, _ := c.Get(DealNoteKey(note.ID.String()))
		if data != nil {
			_ = c.Set(DealNoteKey(note.ID.String()), data)
		}
	}

	return nil
}
