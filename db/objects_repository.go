// ABOUTME: This file provides the repository interface for Office OS objects.
// ABOUTME: It implements CRUD operations for the objects table with JSON metadata support.

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
	ErrObjectNotFound = errors.New("object not found")
	ErrInvalidObject  = errors.New("invalid object")
)

// ObjectsRepository provides CRUD operations for Office OS objects.
type ObjectsRepository struct {
	db *sql.DB
}

// NewObjectsRepository creates a new objects repository.
func NewObjectsRepository(db *sql.DB) *ObjectsRepository {
	return &ObjectsRepository{db: db}
}

// Create creates a new object in the database.
func (r *ObjectsRepository) Create(ctx context.Context, obj *Object) error {
	if obj == nil {
		return ErrInvalidObject
	}

	if obj.ID == "" {
		obj.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	obj.CreatedAt = now
	obj.UpdatedAt = now

	metadataJSON, err := json.Marshal(obj.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO objects (id, type, name, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		obj.ID,
		obj.Type,
		obj.Name,
		metadataJSON,
		obj.CreatedAt,
		obj.UpdatedAt,
	)

	return err
}

// Get retrieves an object by ID.
func (r *ObjectsRepository) Get(ctx context.Context, id string) (*Object, error) {
	query := `
		SELECT id, type, name, metadata, created_at, updated_at
		FROM objects
		WHERE id = ?
	`

	var obj Object
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&obj.ID,
		&obj.Type,
		&obj.Name,
		&metadataJSON,
		&obj.CreatedAt,
		&obj.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &obj.Metadata); err != nil {
			return nil, err
		}
	}

	return &obj, nil
}

// Update updates an existing object.
func (r *ObjectsRepository) Update(ctx context.Context, obj *Object) error {
	if obj == nil || obj.ID == "" {
		return ErrInvalidObject
	}

	obj.UpdatedAt = time.Now().UTC()

	metadataJSON, err := json.Marshal(obj.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE objects
		SET type = ?, name = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		obj.Type,
		obj.Name,
		metadataJSON,
		obj.UpdatedAt,
		obj.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrObjectNotFound
	}

	return nil
}

// Delete deletes an object by ID.
func (r *ObjectsRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM objects WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrObjectNotFound
	}

	return nil
}

// List retrieves all objects, optionally filtered by type.
func (r *ObjectsRepository) List(ctx context.Context, objectType string) ([]*Object, error) {
	var query string
	var args []interface{}

	if objectType != "" {
		query = `
			SELECT id, type, name, metadata, created_at, updated_at
			FROM objects
			WHERE type = ?
			ORDER BY created_at DESC
		`
		args = append(args, objectType)
	} else {
		query = `
			SELECT id, type, name, metadata, created_at, updated_at
			FROM objects
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := make([]*Object, 0)

	for rows.Next() {
		var obj Object
		var metadataJSON []byte

		err := rows.Scan(
			&obj.ID,
			&obj.Type,
			&obj.Name,
			&metadataJSON,
			&obj.CreatedAt,
			&obj.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &obj.Metadata); err != nil {
				return nil, err
			}
		}

		objects = append(objects, &obj)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return objects, nil
}
