// ABOUTME: SQLite company CRUD operations with FTS5 search support.
// ABOUTME: Implements create, read, update, delete, list, and prefix-lookup for companies.
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

// CreateCompany inserts a new company, marshaling Fields and Tags to JSON.
func (s *SqliteStore) CreateCompany(c *models.Company) error {
	fieldsJSON, err := json.Marshal(c.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	tagsJSON, err := json.Marshal(c.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO companies (id, name, domain, fields, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID.String(), c.Name, c.Domain,
		string(fieldsJSON), string(tagsJSON),
		c.CreatedAt.UTC(), c.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert company: %w", err)
	}
	return nil
}

// GetCompany retrieves a company by UUID, returning ErrCompanyNotFound on miss.
func (s *SqliteStore) GetCompany(id uuid.UUID) (*models.Company, error) {
	row := s.db.QueryRow(`
		SELECT id, name, domain, fields, tags, created_at, updated_at
		FROM companies WHERE id = ?`, id.String())
	return scanCompany(row)
}

// GetCompanyByPrefix finds a company whose ID starts with the given prefix.
// Returns ErrPrefixTooShort if prefix is under 6 chars, ErrAmbiguousPrefix
// if multiple companies match.
func (s *SqliteStore) GetCompanyByPrefix(prefix string) (*models.Company, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := s.db.Query(`
		SELECT id, name, domain, fields, tags, created_at, updated_at
		FROM companies WHERE id LIKE ?`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query by prefix: %w", err)
	}

	companies, err := scanCompanyRows(rows)
	if err != nil {
		return nil, err
	}

	switch len(companies) {
	case 0:
		return nil, ErrCompanyNotFound
	case 1:
		return companies[0], nil
	default:
		return nil, ErrAmbiguousPrefix
	}
}

// ListCompanies returns companies matching the optional filter criteria.
func (s *SqliteStore) ListCompanies(filter *CompanyFilter) ([]*models.Company, error) {
	if filter != nil && filter.Search != "" {
		return s.listCompaniesFTS(filter)
	}

	query := "SELECT id, name, domain, fields, tags, created_at, updated_at FROM companies"
	var args []any
	var clauses []string

	if filter != nil && filter.Tag != nil {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM json_each(tags) WHERE json_each.value = ?)")
		args = append(args, *filter.Tag)
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	return scanCompanyRows(rows)
}

// listCompaniesFTS searches companies using the FTS5 index.
func (s *SqliteStore) listCompaniesFTS(filter *CompanyFilter) ([]*models.Company, error) {
	escaped := escapeFTS5Query(filter.Search)

	query := `
		SELECT c.id, c.name, c.domain, c.fields, c.tags, c.created_at, c.updated_at
		FROM companies c
		JOIN companies_fts fts ON c.rowid = fts.rowid
		WHERE companies_fts MATCH ?`
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
		return nil, fmt.Errorf("fts search companies: %w", err)
	}
	return scanCompanyRows(rows)
}

// UpdateCompany updates an existing company, returning ErrCompanyNotFound
// if no row matches.
func (s *SqliteStore) UpdateCompany(c *models.Company) error {
	fieldsJSON, err := json.Marshal(c.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	tagsJSON, err := json.Marshal(c.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	res, err := s.db.Exec(`
		UPDATE companies SET name=?, domain=?, fields=?, tags=?, updated_at=?
		WHERE id=?`,
		c.Name, c.Domain,
		string(fieldsJSON), string(tagsJSON),
		c.UpdatedAt.UTC(), c.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrCompanyNotFound
	}
	return nil
}

// DeleteCompany removes a company by UUID, returning ErrCompanyNotFound
// if no row matches.
func (s *SqliteStore) DeleteCompany(id uuid.UUID) error {
	res, err := s.db.Exec("DELETE FROM companies WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrCompanyNotFound
	}
	return nil
}

// scanCompany scans a single company row and unmarshals JSON fields.
func scanCompany(row *sql.Row) (*models.Company, error) {
	var c models.Company
	var idStr, fieldsStr, tagsStr string
	var createdAt, updatedAt time.Time

	err := row.Scan(&idStr, &c.Name, &c.Domain, &fieldsStr, &tagsStr, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCompanyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan company: %w", err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse company id: %w", err)
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

// scanCompanyRows scans multiple company rows and closes the result set.
func scanCompanyRows(rows *sql.Rows) ([]*models.Company, error) {
	defer func() { _ = rows.Close() }()

	var companies []*models.Company
	for rows.Next() {
		var c models.Company
		var idStr, fieldsStr, tagsStr string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&idStr, &c.Name, &c.Domain, &fieldsStr, &tagsStr, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan company row: %w", err)
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("parse company id: %w", err)
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

		companies = append(companies, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate company rows: %w", err)
	}

	return companies, nil
}
