// ABOUTME: SQLite relationship CRUD operations for entity connections.
// ABOUTME: Supports bidirectional listing and create/delete operations.
package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

// CreateRelationship inserts a new relationship.
func (s *SqliteStore) CreateRelationship(rel *models.Relationship) error {
	_, err := s.db.Exec(`
		INSERT INTO relationships (id, source_id, target_id, type, context, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		rel.ID.String(), rel.SourceID.String(), rel.TargetID.String(),
		rel.Type, rel.Context, rel.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert relationship: %w", err)
	}
	return nil
}

// ListRelationships returns all relationships where the given entityID appears
// as either source or target (bidirectional lookup).
func (s *SqliteStore) ListRelationships(entityID uuid.UUID) ([]*models.Relationship, error) {
	rows, err := s.db.Query(`
		SELECT id, source_id, target_id, type, context, created_at
		FROM relationships
		WHERE source_id = ? OR target_id = ?`,
		entityID.String(), entityID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rels []*models.Relationship
	for rows.Next() {
		var r models.Relationship
		var idStr, srcStr, tgtStr string
		var createdAt time.Time

		if err := rows.Scan(&idStr, &srcStr, &tgtStr, &r.Type, &r.Context, &createdAt); err != nil {
			return nil, fmt.Errorf("scan relationship: %w", err)
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("parse relationship id: %w", err)
		}
		srcID, err := uuid.Parse(srcStr)
		if err != nil {
			return nil, fmt.Errorf("parse source_id: %w", err)
		}
		tgtID, err := uuid.Parse(tgtStr)
		if err != nil {
			return nil, fmt.Errorf("parse target_id: %w", err)
		}

		r.ID = id
		r.SourceID = srcID
		r.TargetID = tgtID
		r.CreatedAt = createdAt

		rels = append(rels, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relationships: %w", err)
	}

	return rels, nil
}

// DeleteRelationship removes a relationship by UUID, returning
// ErrRelationshipNotFound if no row matches.
func (s *SqliteStore) DeleteRelationship(id uuid.UUID) error {
	res, err := s.db.Exec("DELETE FROM relationships WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("delete relationship: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrRelationshipNotFound
	}
	return nil
}
