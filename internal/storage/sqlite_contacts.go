// ABOUTME: SQLite contact CRUD operations with FTS5 search support.
// ABOUTME: Implements create, read, update, delete, list, and prefix-lookup for contacts.
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/models"
)

// CreateContact inserts a new contact, marshaling Fields and Tags to JSON.
func (s *SqliteStore) CreateContact(c *models.Contact) error {
	after, err := sqliteContactSnapshot(c)
	if err != nil {
		return fmt.Errorf("snapshot contact: %w", err)
	}
	event, err := history.NewEvent(
		history.EntityContact,
		c.ID,
		[]uuid.UUID{c.ID},
		history.ActionCreate,
		s.source,
		nil,
		after,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create contact history event: %w", err)
	}
	return s.commitHistoryEvent(event)
}

// GetContact retrieves a contact by UUID, returning ErrContactNotFound on miss.
func (s *SqliteStore) GetContact(id uuid.UUID) (*models.Contact, error) {
	row := s.db.QueryRow(`
		SELECT id, name, email, phone, fields, tags, created_at, updated_at
		FROM contacts WHERE id = ?`, id.String())
	return scanContact(row)
}

// GetContactByPrefix finds a contact whose ID starts with the given prefix.
// Returns ErrPrefixTooShort if prefix is under 6 chars, ErrAmbiguousPrefix
// if multiple contacts match.
func (s *SqliteStore) GetContactByPrefix(prefix string) (*models.Contact, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := s.db.Query(`
		SELECT id, name, email, phone, fields, tags, created_at, updated_at
		FROM contacts WHERE id LIKE ?`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query by prefix: %w", err)
	}

	contacts, err := scanContactRows(rows)
	if err != nil {
		return nil, err
	}

	switch len(contacts) {
	case 0:
		return nil, ErrContactNotFound
	case 1:
		return contacts[0], nil
	default:
		return nil, ErrAmbiguousPrefix
	}
}

// ListContacts returns contacts matching the optional filter criteria.
func (s *SqliteStore) ListContacts(filter *ContactFilter) ([]*models.Contact, error) {
	if filter != nil && filter.Search != "" {
		return s.listContactsFTS(filter)
	}

	query := "SELECT id, name, email, phone, fields, tags, created_at, updated_at FROM contacts"
	var args []any
	var clauses []string

	if filter != nil && filter.Tag != nil {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM json_each(tags) WHERE json_each.value = ?)")
		args = append(args, *filter.Tag)
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ") // #nosec G202 -- clauses are fixed SQL; filter values remain parameterized.
	}

	query += " ORDER BY created_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	return scanContactRows(rows)
}

// listContactsFTS searches contacts using the FTS5 index.
func (s *SqliteStore) listContactsFTS(filter *ContactFilter) ([]*models.Contact, error) {
	escaped := escapeFTS5Query(filter.Search)

	query := `
		SELECT c.id, c.name, c.email, c.phone, c.fields, c.tags, c.created_at, c.updated_at
		FROM contacts c
		JOIN contacts_fts fts ON c.rowid = fts.rowid
		WHERE contacts_fts MATCH ?`
	args := []any{escaped}

	if filter.Tag != nil {
		query += " AND EXISTS (SELECT 1 FROM json_each(c.tags) WHERE json_each.value = ?)"
		args = append(args, *filter.Tag)
	}

	query += " ORDER BY rank"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("fts search contacts: %w", err)
	}
	return scanContactRows(rows)
}

// UpdateContact updates an existing contact, returning ErrContactNotFound
// if no row matches.
func (s *SqliteStore) UpdateContact(c *models.Contact) error {
	current, err := s.GetContact(c.ID)
	if err != nil {
		return err
	}
	before, err := sqliteContactSnapshot(current)
	if err != nil {
		return fmt.Errorf("snapshot current contact: %w", err)
	}
	candidate := *c
	candidate.CreatedAt = current.CreatedAt
	after, err := sqliteContactSnapshot(&candidate)
	if err != nil {
		return fmt.Errorf("snapshot updated contact: %w", err)
	}
	changed, err := history.ChangedFields(history.EntityContact, before, after)
	if err != nil {
		return fmt.Errorf("compare contact snapshots: %w", err)
	}
	if len(changed) == 0 {
		return nil
	}
	event, err := history.NewEvent(
		history.EntityContact,
		c.ID,
		[]uuid.UUID{c.ID},
		history.ActionUpdate,
		s.source,
		before,
		after,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create contact history event: %w", err)
	}
	return s.commitHistoryEvent(event)
}

// DeleteContact removes a contact by UUID, returning ErrContactNotFound
// if no row matches.
func (s *SqliteStore) DeleteContact(id uuid.UUID) error {
	current, err := s.GetContact(id)
	if err != nil {
		return err
	}
	before, err := sqliteContactSnapshot(current)
	if err != nil {
		return fmt.Errorf("snapshot contact: %w", err)
	}
	event, err := history.NewEvent(
		history.EntityContact,
		id,
		[]uuid.UUID{id},
		history.ActionDelete,
		s.source,
		before,
		nil,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create contact history event: %w", err)
	}
	return s.commitHistoryEvent(event)
}

// scanContact scans a single contact row and unmarshals JSON fields.
func scanContact(row rowScanner) (*models.Contact, error) {
	var c models.Contact
	var idStr, fieldsStr, tagsStr string
	var createdAt, updatedAt time.Time

	err := row.Scan(&idStr, &c.Name, &c.Email, &c.Phone, &fieldsStr, &tagsStr, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan contact: %w", err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse contact id: %w", err)
	}
	c.ID = id
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	if err := json.Unmarshal([]byte(fieldsStr), &c.Fields); err != nil {
		return nil, fmt.Errorf("unmarshal fields: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsStr), &c.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	return &c, nil
}

func sqliteContactSnapshot(contact *models.Contact) (json.RawMessage, error) {
	normalized := *contact
	normalized.CreatedAt = contact.CreatedAt.UTC()
	normalized.UpdatedAt = contact.UpdatedAt.UTC()
	return history.SnapshotContact(&normalized)
}

// scanContactRows scans multiple contact rows and closes the result set.
func scanContactRows(rows *sql.Rows) ([]*models.Contact, error) {
	defer func() { _ = rows.Close() }()

	var contacts []*models.Contact
	for rows.Next() {
		var c models.Contact
		var idStr, fieldsStr, tagsStr string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&idStr, &c.Name, &c.Email, &c.Phone, &fieldsStr, &tagsStr, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan contact row: %w", err)
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("parse contact id: %w", err)
		}
		c.ID = id
		c.CreatedAt = createdAt
		c.UpdatedAt = updatedAt

		if err := json.Unmarshal([]byte(fieldsStr), &c.Fields); err != nil {
			return nil, fmt.Errorf("unmarshal fields: %w", err)
		}
		if err := json.Unmarshal([]byte(tagsStr), &c.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}

		contacts = append(contacts, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contact rows: %w", err)
	}

	return contacts, nil
}
