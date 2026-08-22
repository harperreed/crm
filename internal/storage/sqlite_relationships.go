// ABOUTME: SQLite relationship CRUD operations for entity connections.
// ABOUTME: Supports bidirectional listing and create/delete operations.
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/v2/internal/history"
	"github.com/harperreed/crm/v2/internal/models"
)

// CreateRelationship inserts a new relationship.
func (s *SqliteStore) CreateRelationship(rel *models.Relationship) error {
	after, err := sqliteRelationshipSnapshot(rel)
	if err != nil {
		return fmt.Errorf("snapshot relationship: %w", err)
	}
	event, err := history.NewEvent(
		history.EntityRelationship,
		rel.ID,
		[]uuid.UUID{rel.ID, rel.SourceID, rel.TargetID},
		history.ActionCreate,
		s.source,
		nil,
		after,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create relationship history event: %w", err)
	}
	return s.commitHistoryEvent(event)
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
	relationship, err := s.getRelationship(id)
	if err != nil {
		return err
	}
	before, err := sqliteRelationshipSnapshot(relationship)
	if err != nil {
		return fmt.Errorf("snapshot relationship: %w", err)
	}
	event, err := history.NewEvent(
		history.EntityRelationship,
		id,
		[]uuid.UUID{id, relationship.SourceID, relationship.TargetID},
		history.ActionDelete,
		s.source,
		before,
		nil,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create relationship history event: %w", err)
	}
	return s.commitHistoryEvent(event)
}

func (s *SqliteStore) getRelationship(id uuid.UUID) (*models.Relationship, error) {
	return scanRelationship(s.db.QueryRow(`
		SELECT id, source_id, target_id, type, context, created_at
		FROM relationships WHERE id = ?`, id.String()))
}

func scanRelationship(row rowScanner) (*models.Relationship, error) {
	var relationship models.Relationship
	var id, sourceID, targetID string
	var createdAt time.Time
	if err := row.Scan(
		&id,
		&sourceID,
		&targetID,
		&relationship.Type,
		&relationship.Context,
		&createdAt,
	); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRelationshipNotFound
	} else if err != nil {
		return nil, fmt.Errorf("scan relationship: %w", err)
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse relationship id: %w", err)
	}
	parsedSourceID, err := uuid.Parse(sourceID)
	if err != nil {
		return nil, fmt.Errorf("parse source_id: %w", err)
	}
	parsedTargetID, err := uuid.Parse(targetID)
	if err != nil {
		return nil, fmt.Errorf("parse target_id: %w", err)
	}
	relationship.ID = parsedID
	relationship.SourceID = parsedSourceID
	relationship.TargetID = parsedTargetID
	relationship.CreatedAt = createdAt
	return &relationship, nil
}

func sqliteRelationshipSnapshot(relationship *models.Relationship) (json.RawMessage, error) {
	normalized := *relationship
	normalized.CreatedAt = relationship.CreatedAt.UTC()
	return history.SnapshotRelationship(&normalized)
}
