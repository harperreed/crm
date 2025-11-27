// ABOUTME: Database operations for follow-up tracking
// ABOUTME: Handles contact cadence, interaction logging, and follow-up queries
package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

// CreateContactCadence creates or updates a contact's follow-up cadence
func CreateContactCadence(db *sql.DB, cadence *models.ContactCadence) error {
	query := `
		INSERT INTO contact_cadence (
			contact_id, cadence_days, relationship_strength,
			priority_score, last_interaction_date, next_followup_date
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(contact_id) DO UPDATE SET
			cadence_days = excluded.cadence_days,
			relationship_strength = excluded.relationship_strength,
			priority_score = excluded.priority_score,
			last_interaction_date = excluded.last_interaction_date,
			next_followup_date = excluded.next_followup_date
	`

	_, err := db.Exec(query,
		cadence.ContactID.String(),
		cadence.CadenceDays,
		cadence.RelationshipStrength,
		cadence.PriorityScore,
		cadence.LastInteractionDate,
		cadence.NextFollowupDate,
	)
	return err
}

// GetContactCadence retrieves cadence info for a contact
func GetContactCadence(db *sql.DB, contactID uuid.UUID) (*models.ContactCadence, error) {
	query := `
		SELECT contact_id, cadence_days, relationship_strength,
		       priority_score, last_interaction_date, next_followup_date
		FROM contact_cadence
		WHERE contact_id = ?
	`

	cadence := &models.ContactCadence{}
	var contactIDStr string
	err := db.QueryRow(query, contactID.String()).Scan(
		&contactIDStr,
		&cadence.CadenceDays,
		&cadence.RelationshipStrength,
		&cadence.PriorityScore,
		&cadence.LastInteractionDate,
		&cadence.NextFollowupDate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cadence.ContactID, err = uuid.Parse(contactIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse contact ID: %w", err)
	}

	return cadence, nil
}

// GetFollowupList returns contacts needing follow-up, sorted by priority
func GetFollowupList(db *sql.DB, limit int) ([]models.FollowupContact, error) {
	query := `
		SELECT
			c.id, c.name, c.email, c.phone, c.company_id, c.notes,
			c.last_contacted_at, c.created_at, c.updated_at,
			cc.cadence_days, cc.relationship_strength, cc.priority_score,
			cc.next_followup_date,
			CAST((julianday('now') - julianday(cc.last_interaction_date)) AS INTEGER) as days_since
		FROM contacts c
		INNER JOIN contact_cadence cc ON c.id = cc.contact_id
		WHERE cc.priority_score > 0
		ORDER BY cc.priority_score DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var followups []models.FollowupContact
	for rows.Next() {
		var f models.FollowupContact
		var idStr, companyIDStr string
		var companyID *string
		err := rows.Scan(
			&idStr, &f.Name, &f.Email, &f.Phone, &companyID, &f.Notes,
			&f.LastContactedAt, &f.CreatedAt, &f.UpdatedAt,
			&f.CadenceDays, &f.RelationshipStrength, &f.PriorityScore,
			&f.NextFollowupDate, &f.DaysSinceContact,
		)
		if err != nil {
			return nil, err
		}

		f.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse contact ID: %w", err)
		}

		if companyID != nil {
			companyIDStr = *companyID
			parsed, err := uuid.Parse(companyIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse company ID: %w", err)
			}
			f.CompanyID = &parsed
		}

		followups = append(followups, f)
	}

	return followups, rows.Err()
}

// UpdateCadenceAfterInteraction updates cadence when interaction is logged
func UpdateCadenceAfterInteraction(db *sql.DB, contactID uuid.UUID, timestamp time.Time) error {
	// Get or create cadence
	cadence, err := GetContactCadence(db, contactID)
	if err != nil {
		return err
	}

	if cadence == nil {
		// Create default cadence
		cadence = &models.ContactCadence{
			ContactID:            contactID,
			CadenceDays:          30,
			RelationshipStrength: models.StrengthMedium,
		}
	}

	// Update timestamps
	cadence.LastInteractionDate = &timestamp
	cadence.UpdateNextFollowup()
	cadence.PriorityScore = cadence.ComputePriorityScore()

	return CreateContactCadence(db, cadence)
}

// SetContactCadence sets or updates a contact's cadence settings
func SetContactCadence(db *sql.DB, contactID uuid.UUID, days int, strength string) error {
	cadence, err := GetContactCadence(db, contactID)
	if err != nil {
		return err
	}

	if cadence == nil {
		cadence = &models.ContactCadence{
			ContactID: contactID,
		}
	}

	cadence.CadenceDays = days
	cadence.RelationshipStrength = strength
	cadence.PriorityScore = cadence.ComputePriorityScore()
	cadence.UpdateNextFollowup()

	return CreateContactCadence(db, cadence)
}
