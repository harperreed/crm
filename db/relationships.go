// ABOUTME: Relationship database operations
// ABOUTME: Handles CRUD operations and bidirectional relationship queries between contacts
package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

// orderContactIDs ensures contact_id_1 < contact_id_2 for consistent bidirectional storage
func orderContactIDs(id1, id2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if id1.String() < id2.String() {
		return id1, id2
	}
	return id2, id1
}

func CreateRelationship(db *sql.DB, relationship *models.Relationship) error {
	relationship.ID = uuid.New()
	now := time.Now()
	relationship.CreatedAt = now
	relationship.UpdatedAt = now

	// Ensure proper ordering
	relationship.ContactID1, relationship.ContactID2 = orderContactIDs(relationship.ContactID1, relationship.ContactID2)

	_, err := db.Exec(`
		INSERT INTO relationships (id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, relationship.ID.String(), relationship.ContactID1.String(), relationship.ContactID2.String(),
		relationship.RelationshipType, relationship.Context, relationship.CreatedAt, relationship.UpdatedAt)

	return err
}

func GetRelationship(db *sql.DB, id uuid.UUID) (*models.Relationship, error) {
	relationship := &models.Relationship{}

	err := db.QueryRow(`
		SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
		FROM relationships WHERE id = ?
	`, id.String()).Scan(
		&relationship.ID,
		&relationship.ContactID1,
		&relationship.ContactID2,
		&relationship.RelationshipType,
		&relationship.Context,
		&relationship.CreatedAt,
		&relationship.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return relationship, nil
}

func FindRelationshipsBetween(db *sql.DB, contactID1, contactID2 uuid.UUID) ([]models.Relationship, error) {
	// Order the IDs to match storage pattern
	orderedID1, orderedID2 := orderContactIDs(contactID1, contactID2)

	rows, err := db.Query(`
		SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
		FROM relationships
		WHERE contact_id_1 = ? AND contact_id_2 = ?
	`, orderedID1.String(), orderedID2.String())

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []models.Relationship
	for rows.Next() {
		var r models.Relationship
		if err := rows.Scan(&r.ID, &r.ContactID1, &r.ContactID2, &r.RelationshipType, &r.Context, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		relationships = append(relationships, r)
	}

	return relationships, rows.Err()
}

func FindContactRelationships(db *sql.DB, contactID uuid.UUID, relationshipType string) ([]models.Relationship, error) {
	var rows *sql.Rows
	var err error

	// Search both contact_id_1 and contact_id_2 columns since relationships are bidirectional
	if relationshipType != "" {
		rows, err = db.Query(`
			SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
			FROM relationships
			WHERE (contact_id_1 = ? OR contact_id_2 = ?) AND relationship_type = ?
			ORDER BY created_at DESC
		`, contactID.String(), contactID.String(), relationshipType)
	} else {
		rows, err = db.Query(`
			SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
			FROM relationships
			WHERE contact_id_1 = ? OR contact_id_2 = ?
			ORDER BY created_at DESC
		`, contactID.String(), contactID.String())
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []models.Relationship
	for rows.Next() {
		var r models.Relationship
		if err := rows.Scan(&r.ID, &r.ContactID1, &r.ContactID2, &r.RelationshipType, &r.Context, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		relationships = append(relationships, r)
	}

	return relationships, rows.Err()
}

func UpdateRelationship(db *sql.DB, id uuid.UUID, relType, context string) error {
	_, err := db.Exec(`
		UPDATE relationships
		SET relationship_type = ?, context = ?, updated_at = ?
		WHERE id = ?
	`, relType, context, time.Now(), id.String())
	return err
}

func DeleteRelationship(db *sql.DB, id uuid.UUID) error {
	_, err := db.Exec(`DELETE FROM relationships WHERE id = ?`, id.String())
	return err
}
