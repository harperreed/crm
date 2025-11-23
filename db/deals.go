// ABOUTME: Deal and deal note database operations
// ABOUTME: Handles deal lifecycle, stage management, and note tracking
package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func CreateDeal(db *sql.DB, deal *models.Deal) error {
	deal.ID = uuid.New()
	now := time.Now()
	deal.CreatedAt = now
	deal.UpdatedAt = now
	deal.LastActivityAt = now

	if deal.Currency == "" {
		deal.Currency = "USD"
	}

	var contactID *string
	if deal.ContactID != nil {
		s := deal.ContactID.String()
		contactID = &s
	}

	_, err := db.Exec(`
		INSERT INTO deals (id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, deal.ID.String(), deal.Title, deal.Amount, deal.Currency, deal.Stage, deal.CompanyID.String(), contactID, deal.ExpectedCloseDate, deal.CreatedAt, deal.UpdatedAt, deal.LastActivityAt)

	return err
}

func GetDeal(db *sql.DB, id uuid.UUID) (*models.Deal, error) {
	deal := &models.Deal{}
	var contactID sql.NullString

	err := db.QueryRow(`
		SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
		FROM deals WHERE id = ?
	`, id.String()).Scan(
		&deal.ID,
		&deal.Title,
		&deal.Amount,
		&deal.Currency,
		&deal.Stage,
		&deal.CompanyID,
		&contactID,
		&deal.ExpectedCloseDate,
		&deal.CreatedAt,
		&deal.UpdatedAt,
		&deal.LastActivityAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if contactID.Valid {
		cid, err := uuid.Parse(contactID.String)
		if err == nil {
			deal.ContactID = &cid
		}
	}

	return deal, nil
}

func UpdateDeal(db *sql.DB, deal *models.Deal) error {
	now := time.Now()
	deal.UpdatedAt = now
	deal.LastActivityAt = now

	var contactID *string
	if deal.ContactID != nil {
		s := deal.ContactID.String()
		contactID = &s
	}

	_, err := db.Exec(`
		UPDATE deals
		SET title = ?, amount = ?, currency = ?, stage = ?, contact_id = ?, expected_close_date = ?, updated_at = ?, last_activity_at = ?
		WHERE id = ?
	`, deal.Title, deal.Amount, deal.Currency, deal.Stage, contactID, deal.ExpectedCloseDate, deal.UpdatedAt, deal.LastActivityAt, deal.ID.String())

	return err
}

func FindDeals(db *sql.DB, stage string, companyID *uuid.UUID, limit int) ([]models.Deal, error) {
	if limit <= 0 {
		limit = 10
	}

	var rows *sql.Rows
	var err error

	if companyID != nil && stage != "" {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			WHERE company_id = ? AND stage = ?
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, companyID.String(), stage, limit)
	} else if companyID != nil {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			WHERE company_id = ?
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, companyID.String(), limit)
	} else if stage != "" {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			WHERE stage = ?
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, stage, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deals []models.Deal
	for rows.Next() {
		var d models.Deal
		var contactID sql.NullString

		if err := rows.Scan(&d.ID, &d.Title, &d.Amount, &d.Currency, &d.Stage, &d.CompanyID, &contactID, &d.ExpectedCloseDate, &d.CreatedAt, &d.UpdatedAt, &d.LastActivityAt); err != nil {
			return nil, err
		}

		if contactID.Valid {
			cid, err := uuid.Parse(contactID.String)
			if err == nil {
				d.ContactID = &cid
			}
		}

		deals = append(deals, d)
	}

	return deals, rows.Err()
}

func AddDealNote(db *sql.DB, note *models.DealNote) error {
	note.ID = uuid.New()
	note.CreatedAt = time.Now()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Insert note
	_, err = tx.Exec(`
		INSERT INTO deal_notes (id, deal_id, content, created_at)
		VALUES (?, ?, ?, ?)
	`, note.ID.String(), note.DealID.String(), note.Content, note.CreatedAt)
	if err != nil {
		return err
	}

	// Update deal's last_activity_at
	_, err = tx.Exec(`
		UPDATE deals SET last_activity_at = ?, updated_at = ? WHERE id = ?
	`, note.CreatedAt, note.CreatedAt, note.DealID.String())
	if err != nil {
		return err
	}

	// Update contact's last_contacted_at if deal has contact
	_, err = tx.Exec(`
		UPDATE contacts
		SET last_contacted_at = ?, updated_at = ?
		WHERE id = (SELECT contact_id FROM deals WHERE id = ? AND contact_id IS NOT NULL)
	`, note.CreatedAt, note.CreatedAt, note.DealID.String())
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetDealNotes(db *sql.DB, dealID uuid.UUID) ([]models.DealNote, error) {
	rows, err := db.Query(`
		SELECT id, deal_id, content, created_at
		FROM deal_notes
		WHERE deal_id = ?
		ORDER BY created_at DESC
	`, dealID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.DealNote
	for rows.Next() {
		var n models.DealNote
		if err := rows.Scan(&n.ID, &n.DealID, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}

	return notes, rows.Err()
}

func DeleteDeal(db *sql.DB, id uuid.UUID) error {
	// Delete all associated notes
	_, err := db.Exec(`DELETE FROM deal_notes WHERE deal_id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete deal notes: %w", err)
	}

	// Delete the deal
	_, err = db.Exec(`DELETE FROM deals WHERE id = ?`, id.String())
	return err
}
