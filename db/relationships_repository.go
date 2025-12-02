// ABOUTME: This file provides the repository interface for Office OS relationships.
// ABOUTME: It implements CRUD operations for managing relationships between objects.

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRelationshipNotFound = errors.New("relationship not found")
	ErrInvalidRelationship  = errors.New("invalid relationship")
)

// Relationship represents a connection between two objects.
type Relationship struct {
	ID         string                 `json:"id"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	Type       string                 `json:"type"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// RelationshipsRepository provides CRUD operations for relationships.
type RelationshipsRepository struct {
	db *sql.DB
}

// NewRelationshipsRepository creates a new relationships repository.
func NewRelationshipsRepository(db *sql.DB) *RelationshipsRepository {
	return &RelationshipsRepository{db: db}
}

// Create creates a new relationship.
func (r *RelationshipsRepository) Create(ctx context.Context, rel *Relationship) error {
	if rel == nil {
		return ErrInvalidRelationship
	}

	if rel.SourceID == "" || rel.TargetID == "" || rel.Type == "" {
		return ErrInvalidRelationship
	}

	if rel.ID == "" {
		rel.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	rel.CreatedAt = now
	rel.UpdatedAt = now

	metadataJSON, err := json.Marshal(rel.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO relationships (id, source_id, target_id, type, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		rel.ID,
		rel.SourceID,
		rel.TargetID,
		rel.Type,
		metadataJSON,
		rel.CreatedAt,
		rel.UpdatedAt,
	)

	return err
}

// Get retrieves a relationship by ID.
func (r *RelationshipsRepository) Get(ctx context.Context, id string) (*Relationship, error) {
	query := `
		SELECT id, source_id, target_id, type, metadata, created_at, updated_at
		FROM relationships
		WHERE id = ?
	`

	var rel Relationship
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rel.ID,
		&rel.SourceID,
		&rel.TargetID,
		&rel.Type,
		&metadataJSON,
		&rel.CreatedAt,
		&rel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRelationshipNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &rel.Metadata); err != nil {
			return nil, err
		}
	}

	return &rel, nil
}

// Update updates an existing relationship.
func (r *RelationshipsRepository) Update(ctx context.Context, rel *Relationship) error {
	if rel == nil || rel.ID == "" {
		return ErrInvalidRelationship
	}

	rel.UpdatedAt = time.Now().UTC()

	metadataJSON, err := json.Marshal(rel.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE relationships
		SET source_id = ?, target_id = ?, type = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		rel.SourceID,
		rel.TargetID,
		rel.Type,
		metadataJSON,
		rel.UpdatedAt,
		rel.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrRelationshipNotFound
	}

	return nil
}

// Delete deletes a relationship by ID.
func (r *RelationshipsRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM relationships WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrRelationshipNotFound
	}

	return nil
}

// FindBySource retrieves all relationships originating from a source object.
func (r *RelationshipsRepository) FindBySource(ctx context.Context, sourceID string, relType string) ([]*Relationship, error) {
	var query string
	var args []interface{}

	if relType != "" {
		query = `
			SELECT id, source_id, target_id, type, metadata, created_at, updated_at
			FROM relationships
			WHERE source_id = ? AND type = ?
			ORDER BY created_at DESC
		`
		args = append(args, sourceID, relType)
	} else {
		query = `
			SELECT id, source_id, target_id, type, metadata, created_at, updated_at
			FROM relationships
			WHERE source_id = ?
			ORDER BY created_at DESC
		`
		args = append(args, sourceID)
	}

	return r.queryRelationships(ctx, query, args...)
}

// FindByTarget retrieves all relationships pointing to a target object.
func (r *RelationshipsRepository) FindByTarget(ctx context.Context, targetID string, relType string) ([]*Relationship, error) {
	var query string
	var args []interface{}

	if relType != "" {
		query = `
			SELECT id, source_id, target_id, type, metadata, created_at, updated_at
			FROM relationships
			WHERE target_id = ? AND type = ?
			ORDER BY created_at DESC
		`
		args = append(args, targetID, relType)
	} else {
		query = `
			SELECT id, source_id, target_id, type, metadata, created_at, updated_at
			FROM relationships
			WHERE target_id = ?
			ORDER BY created_at DESC
		`
		args = append(args, targetID)
	}

	return r.queryRelationships(ctx, query, args...)
}

// FindBetween retrieves all relationships between two objects (in either direction).
func (r *RelationshipsRepository) FindBetween(ctx context.Context, objectID1, objectID2 string) ([]*Relationship, error) {
	query := `
		SELECT id, source_id, target_id, type, metadata, created_at, updated_at
		FROM relationships
		WHERE (source_id = ? AND target_id = ?) OR (source_id = ? AND target_id = ?)
		ORDER BY created_at DESC
	`

	return r.queryRelationships(ctx, query, objectID1, objectID2, objectID2, objectID1)
}

// List retrieves all relationships, optionally filtered by type.
func (r *RelationshipsRepository) List(ctx context.Context, relType string) ([]*Relationship, error) {
	var query string
	var args []interface{}

	if relType != "" {
		query = `
			SELECT id, source_id, target_id, type, metadata, created_at, updated_at
			FROM relationships
			WHERE type = ?
			ORDER BY created_at DESC
		`
		args = append(args, relType)
	} else {
		query = `
			SELECT id, source_id, target_id, type, metadata, created_at, updated_at
			FROM relationships
			ORDER BY created_at DESC
		`
	}

	return r.queryRelationships(ctx, query, args...)
}

// queryRelationships is a helper that executes a query and scans relationships.
func (r *RelationshipsRepository) queryRelationships(ctx context.Context, query string, args ...interface{}) ([]*Relationship, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	relationships := make([]*Relationship, 0)

	for rows.Next() {
		var rel Relationship
		var metadataJSON []byte

		err := rows.Scan(
			&rel.ID,
			&rel.SourceID,
			&rel.TargetID,
			&rel.Type,
			&metadataJSON,
			&rel.CreatedAt,
			&rel.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &rel.Metadata); err != nil {
				return nil, err
			}
		}

		relationships = append(relationships, &rel)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return relationships, nil
}
