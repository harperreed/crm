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

	// Default values for new fields
	if obj.CreatedBy == "" {
		obj.CreatedBy = "system"
	}
	if obj.ACL == "" {
		obj.ACL = "[]"
	}
	if obj.Tags == "" {
		obj.Tags = "[]"
	}
	if obj.Fields == nil {
		obj.Fields = make(map[string]interface{})
	}

	fieldsJSON, err := json.Marshal(obj.Fields)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO objects (id, kind, created_at, updated_at, created_by, acl, tags, fields)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		obj.ID,
		obj.Kind,
		obj.CreatedAt,
		obj.UpdatedAt,
		obj.CreatedBy,
		obj.ACL,
		obj.Tags,
		fieldsJSON,
	)

	return err
}

// Get retrieves an object by ID.
func (r *ObjectsRepository) Get(ctx context.Context, id string) (*Object, error) {
	query := `
		SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
		FROM objects
		WHERE id = ?
	`

	var obj Object
	var fieldsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&obj.ID,
		&obj.Kind,
		&obj.CreatedAt,
		&obj.UpdatedAt,
		&obj.CreatedBy,
		&obj.ACL,
		&obj.Tags,
		&fieldsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(fieldsJSON) > 0 && string(fieldsJSON) != "null" {
		if err := json.Unmarshal(fieldsJSON, &obj.Fields); err != nil {
			return nil, err
		}
	} else {
		obj.Fields = make(map[string]interface{})
	}

	return &obj, nil
}

// Update updates an existing object.
func (r *ObjectsRepository) Update(ctx context.Context, obj *Object) error {
	if obj == nil || obj.ID == "" {
		return ErrInvalidObject
	}

	obj.UpdatedAt = time.Now().UTC()

	if obj.Fields == nil {
		obj.Fields = make(map[string]interface{})
	}

	fieldsJSON, err := json.Marshal(obj.Fields)
	if err != nil {
		return err
	}

	query := `
		UPDATE objects
		SET kind = ?, created_by = ?, acl = ?, tags = ?, fields = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		obj.Kind,
		obj.CreatedBy,
		obj.ACL,
		obj.Tags,
		fieldsJSON,
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

// List retrieves all objects, optionally filtered by kind.
func (r *ObjectsRepository) List(ctx context.Context, objectKind string) ([]*Object, error) {
	var query string
	var args []interface{}

	if objectKind != "" {
		query = `
			SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
			FROM objects
			WHERE kind = ?
			ORDER BY created_at DESC
		`
		args = append(args, objectKind)
	} else {
		query = `
			SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
			FROM objects
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	objects := make([]*Object, 0)

	for rows.Next() {
		var obj Object
		var fieldsJSON []byte

		err := rows.Scan(
			&obj.ID,
			&obj.Kind,
			&obj.CreatedAt,
			&obj.UpdatedAt,
			&obj.CreatedBy,
			&obj.ACL,
			&obj.Tags,
			&fieldsJSON,
		)
		if err != nil {
			return nil, err
		}

		if len(fieldsJSON) > 0 && string(fieldsJSON) != "null" {
			if err := json.Unmarshal(fieldsJSON, &obj.Fields); err != nil {
				return nil, err
			}
		} else {
			obj.Fields = make(map[string]interface{})
		}

		objects = append(objects, &obj)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return objects, nil
}
