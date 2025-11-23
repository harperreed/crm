// ABOUTME: Contact database operations
// ABOUTME: Handles CRUD operations, contact lookups, and interaction tracking
package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func CreateContact(db *sql.DB, contact *models.Contact) error {
	contact.ID = uuid.New()
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	var companyID *string
	if contact.CompanyID != nil {
		s := contact.CompanyID.String()
		companyID = &s
	}

	_, err := db.Exec(`
		INSERT INTO contacts (id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, contact.ID.String(), contact.Name, contact.Email, contact.Phone, companyID, contact.Notes, contact.LastContactedAt, contact.CreatedAt, contact.UpdatedAt)

	return err
}

func GetContact(db *sql.DB, id uuid.UUID) (*models.Contact, error) {
	contact := &models.Contact{}
	var companyID sql.NullString

	err := db.QueryRow(`
		SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
		FROM contacts WHERE id = ?
	`, id.String()).Scan(
		&contact.ID,
		&contact.Name,
		&contact.Email,
		&contact.Phone,
		&companyID,
		&contact.Notes,
		&contact.LastContactedAt,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if companyID.Valid {
		cid, err := uuid.Parse(companyID.String)
		if err == nil {
			contact.CompanyID = &cid
		}
	}

	return contact, nil
}

func FindContacts(db *sql.DB, query string, companyID *uuid.UUID, limit int) ([]models.Contact, error) {
	if limit <= 0 {
		limit = 10
	}

	var rows *sql.Rows
	var err error

	if companyID != nil {
		rows, err = db.Query(`
			SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
			FROM contacts
			WHERE company_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`, companyID.String(), limit)
	} else if query != "" {
		searchPattern := "%" + strings.ToLower(query) + "%"
		rows, err = db.Query(`
			SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
			FROM contacts
			WHERE LOWER(name) LIKE ? OR LOWER(email) LIKE ?
			ORDER BY created_at DESC
			LIMIT ?
		`, searchPattern, searchPattern, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
			FROM contacts
			ORDER BY created_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var companyID sql.NullString

		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &companyID, &c.Notes, &c.LastContactedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}

		if companyID.Valid {
			cid, err := uuid.Parse(companyID.String)
			if err == nil {
				c.CompanyID = &cid
			}
		}

		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}

func UpdateContact(db *sql.DB, id uuid.UUID, updates *models.Contact) error {
	updates.UpdatedAt = time.Now()

	var companyID *string
	if updates.CompanyID != nil {
		s := updates.CompanyID.String()
		companyID = &s
	}

	result, err := db.Exec(`
		UPDATE contacts
		SET name = ?, email = ?, phone = ?, company_id = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, updates.Name, updates.Email, updates.Phone, companyID, updates.Notes, updates.UpdatedAt, id.String())

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("contact not found: %s", id)
	}

	return nil
}

func DeleteContact(db *sql.DB, id uuid.UUID) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Safe even after commit
	}()

	// Delete all relationships involving this contact
	_, err = tx.Exec(`DELETE FROM relationships WHERE contact_id_1 = ? OR contact_id_2 = ?`, id.String(), id.String())
	if err != nil {
		return fmt.Errorf("failed to delete relationships: %w", err)
	}

	// Set contact_id to NULL for any deals
	_, err = tx.Exec(`UPDATE deals SET contact_id = NULL WHERE contact_id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to update deals: %w", err)
	}

	// Delete the contact
	_, err = tx.Exec(`DELETE FROM contacts WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return tx.Commit()
}

func UpdateContactLastContacted(db *sql.DB, contactID uuid.UUID, timestamp time.Time) error {
	_, err := db.Exec(`
		UPDATE contacts
		SET last_contacted_at = ?, updated_at = ?
		WHERE id = ?
	`, timestamp, time.Now(), contactID.String())

	return err
}
