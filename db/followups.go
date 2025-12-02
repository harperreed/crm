// ABOUTME: Database operations for follow-up tracking
// ABOUTME: Handles contact cadence, interaction logging, and follow-up queries
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

// CreateContactCadence creates or updates a contact's follow-up cadence.
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

// GetContactCadence retrieves cadence info for a contact.
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

	if errors.Is(err, sql.ErrNoRows) {
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

// GetFollowupList returns contacts needing follow-up, sorted by priority.
func GetFollowupList(db *sql.DB, limit int) ([]models.FollowupContact, error) {
	query := `
		SELECT
			c.id,
			json_extract(c.fields, '$.name') as name,
			json_extract(c.fields, '$.email') as email,
			json_extract(c.fields, '$.phone') as phone,
			json_extract(c.fields, '$.company_id') as company_id,
			json_extract(c.fields, '$.notes') as notes,
			json_extract(c.fields, '$.last_contacted_at') as last_contacted_at,
			c.created_at, c.updated_at,
			cc.cadence_days, cc.relationship_strength, cc.priority_score,
			cc.next_followup_date,
			CAST((julianday('now') - julianday(cc.last_interaction_date)) AS INTEGER) as days_since
		FROM objects c
		INNER JOIN contact_cadence cc ON c.id = cc.contact_id
		WHERE c.kind = 'Contact' AND cc.priority_score > 0
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
		var email, phone, notes, lastContactedAtStr, companyID *string
		err := rows.Scan(
			&idStr, &f.Name, &email, &phone, &companyID, &notes,
			&lastContactedAtStr, &f.CreatedAt, &f.UpdatedAt,
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

		if email != nil {
			f.Email = *email
		}
		if phone != nil {
			f.Phone = *phone
		}
		if notes != nil {
			f.Notes = *notes
		}
		if lastContactedAtStr != nil {
			parsed, err := time.Parse(time.RFC3339Nano, *lastContactedAtStr)
			if err == nil {
				f.LastContactedAt = &parsed
			}
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

// UpdateCadenceAfterInteraction updates cadence when interaction is logged.
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

// SetContactCadence sets or updates a contact's cadence settings.
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

// LogInteraction records a new interaction and updates contact cadence.
func LogInteraction(db *sql.DB, interaction *models.InteractionLog) error {
	// Generate ID if not set
	if interaction.ID == uuid.Nil {
		interaction.ID = uuid.New()
	}

	// Insert interaction
	query := `
		INSERT INTO interaction_log (
			id, contact_id, interaction_type, timestamp, notes, sentiment, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		interaction.ID.String(),
		interaction.ContactID.String(),
		interaction.InteractionType,
		interaction.Timestamp,
		interaction.Notes,
		interaction.Sentiment,
		interaction.Metadata,
	)
	if err != nil {
		return err
	}

	// Update contact's last_contacted_at in fields
	updateContact := `
		UPDATE objects
		SET fields = json_set(fields, '$.last_contacted_at', ?)
		WHERE id = ? AND kind = 'Contact'
	`
	_, err = db.Exec(updateContact, interaction.Timestamp.Format(time.RFC3339Nano), interaction.ContactID.String())
	if err != nil {
		return err
	}

	// Update cadence
	return UpdateCadenceAfterInteraction(db, interaction.ContactID, interaction.Timestamp)
}

// GetInteractionHistory retrieves interaction history for a contact.
func GetInteractionHistory(db *sql.DB, contactID uuid.UUID, limit int) ([]models.InteractionLog, error) {
	query := `
		SELECT id, contact_id, interaction_type, timestamp, notes, sentiment, metadata
		FROM interaction_log
		WHERE contact_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.Query(query, contactID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var interactions []models.InteractionLog
	for rows.Next() {
		var i models.InteractionLog
		var id, contactID string
		err := rows.Scan(&id, &contactID, &i.InteractionType, &i.Timestamp, &i.Notes, &i.Sentiment, &i.Metadata)
		if err != nil {
			return nil, err
		}
		i.ID, _ = uuid.Parse(id)
		i.ContactID, _ = uuid.Parse(contactID)
		interactions = append(interactions, i)
	}

	return interactions, rows.Err()
}

// GetRecentInteractions gets all recent interactions across all contacts.
func GetRecentInteractions(db *sql.DB, days int, limit int) ([]models.InteractionLog, error) {
	query := `
		SELECT id, contact_id, interaction_type, timestamp, notes, sentiment, metadata
		FROM interaction_log
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.Query(query, days, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var interactions []models.InteractionLog
	for rows.Next() {
		var i models.InteractionLog
		var id, contactID string
		err := rows.Scan(&id, &contactID, &i.InteractionType, &i.Timestamp, &i.Notes, &i.Sentiment, &i.Metadata)
		if err != nil {
			return nil, err
		}
		i.ID, _ = uuid.Parse(id)
		i.ContactID, _ = uuid.Parse(contactID)
		interactions = append(interactions, i)
	}

	return interactions, rows.Err()
}
