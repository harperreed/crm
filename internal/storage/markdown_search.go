// ABOUTME: Cross-entity search for the markdown storage backend.
// ABOUTME: Provides substring search across all contact and company files.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harperreed/crm/internal/models"
	"gopkg.in/yaml.v3"
)

// Search performs a substring search across all contacts and companies.
func (s *MarkdownStore) Search(query string) (*SearchResults, error) {
	contacts, err := s.searchContacts(query)
	if err != nil {
		return nil, err
	}
	companies, err := s.searchCompanies(query)
	if err != nil {
		return nil, err
	}
	return &SearchResults{
		Contacts:  contacts,
		Companies: companies,
	}, nil
}

// searchContacts finds contacts matching the query via substring search.
func (s *MarkdownStore) searchContacts(query string) ([]*models.Contact, error) {
	entries, err := os.ReadDir(s.contactsDir())
	if err != nil {
		return nil, err
	}
	var results []*models.Contact
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.contactsDir(), entry.Name())
		c, err := readContactFile(path)
		if err != nil {
			continue
		}
		if contactMatchesSearch(c, query) {
			results = append(results, c)
		}
	}
	return results, nil
}

// searchCompanies finds companies matching the query via substring search.
func (s *MarkdownStore) searchCompanies(query string) ([]*models.Company, error) {
	entries, err := os.ReadDir(s.companiesDir())
	if err != nil {
		return nil, err
	}
	var results []*models.Company
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.companiesDir(), entry.Name())
		c, err := readCompanyFile(path)
		if err != nil {
			continue
		}
		if companyMatchesSearch(c, query) {
			results = append(results, c)
		}
	}
	return results, nil
}

// anyToString converts an arbitrary value to a string for search matching.
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// yamlUnmarshal is a thin wrapper around yaml.Unmarshal for consistency.
func yamlUnmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
