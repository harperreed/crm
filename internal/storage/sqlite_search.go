// ABOUTME: Cross-entity search combining contacts and companies via FTS5.
// ABOUTME: Returns unified SearchResults from both entity types.
package storage

import (
	"fmt"

	"github.com/harperreed/crm/internal/models"
)

// Search queries both contacts and companies using FTS5 and returns combined results.
func (s *SqliteStore) Search(query string) (*SearchResults, error) {
	contacts, err := s.searchContacts(query)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}

	companies, err := s.searchCompanies(query)
	if err != nil {
		return nil, fmt.Errorf("search companies: %w", err)
	}

	return &SearchResults{
		Contacts:  contacts,
		Companies: companies,
	}, nil
}

// searchContacts performs an FTS5 search on the contacts table.
func (s *SqliteStore) searchContacts(query string) ([]*models.Contact, error) {
	escaped := escapeFTS5Query(query)

	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.email, c.phone, c.fields, c.tags, c.created_at, c.updated_at
		FROM contacts c
		JOIN contacts_fts fts ON c.rowid = fts.rowid
		WHERE contacts_fts MATCH ?
		ORDER BY rank`, escaped)
	if err != nil {
		return nil, fmt.Errorf("fts query contacts: %w", err)
	}
	return scanContactRows(rows)
}

// searchCompanies performs an FTS5 search on the companies table.
func (s *SqliteStore) searchCompanies(query string) ([]*models.Company, error) {
	escaped := escapeFTS5Query(query)

	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.domain, c.fields, c.tags, c.created_at, c.updated_at
		FROM companies c
		JOIN companies_fts fts ON c.rowid = fts.rowid
		WHERE companies_fts MATCH ?
		ORDER BY rank`, escaped)
	if err != nil {
		return nil, fmt.Errorf("fts query companies: %w", err)
	}
	return scanCompanyRows(rows)
}
