// ABOUTME: Markdown storage CRUD operations for companies.
// ABOUTME: Stores each company as a .md file with YAML frontmatter in the companies/ directory.
package storage

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/mdstore"
)

// companyFrontmatter is the YAML representation of a company stored in frontmatter.
type companyFrontmatter struct {
	ID        string         `yaml:"id"`
	Name      string         `yaml:"name"`
	Domain    string         `yaml:"domain,omitempty"`
	Fields    map[string]any `yaml:"fields,omitempty"`
	Tags      []string       `yaml:"tags,omitempty"`
	CreatedAt string         `yaml:"created_at"`
	UpdatedAt string         `yaml:"updated_at"`
}

// companyToFrontmatter converts a models.Company to its YAML frontmatter representation.
func companyToFrontmatter(c *models.Company) companyFrontmatter {
	return companyFrontmatter{
		ID:        c.ID.String(),
		Name:      c.Name,
		Domain:    c.Domain,
		Fields:    c.Fields,
		Tags:      c.Tags,
		CreatedAt: mdstore.FormatTime(c.CreatedAt),
		UpdatedAt: mdstore.FormatTime(c.UpdatedAt),
	}
}

// frontmatterToCompany converts a companyFrontmatter back to a models.Company.
func frontmatterToCompany(fm companyFrontmatter) (*models.Company, error) {
	id, err := uuid.Parse(fm.ID)
	if err != nil {
		return nil, err
	}
	createdAt, err := mdstore.ParseTime(fm.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := mdstore.ParseTime(fm.UpdatedAt)
	if err != nil {
		return nil, err
	}
	fields := fm.Fields
	if fields == nil {
		fields = make(map[string]any)
	}
	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}
	return &models.Company{
		ID:        id,
		Name:      fm.Name,
		Domain:    fm.Domain,
		Fields:    fields,
		Tags:      tags,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// writeCompany writes a company as a .md file with YAML frontmatter.
func (s *MarkdownStore) writeCompany(c *models.Company, filename string) error {
	fm := companyToFrontmatter(c)
	content, err := mdstore.RenderFrontmatter(fm, "")
	if err != nil {
		return err
	}
	return mdstore.AtomicWrite(filepath.Join(s.companiesDir(), filename), []byte(content))
}

// readCompanyFile reads a single company .md file and returns the company.
func readCompanyFile(path string) (*models.Company, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) //nolint:gosec // path is constructed from trusted directory listing
	if err != nil {
		return nil, err
	}
	yamlStr, _ := mdstore.ParseFrontmatter(string(data))
	var fm companyFrontmatter
	if err := yamlUnmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, err
	}
	return frontmatterToCompany(fm)
}

// findCompanyFile walks the companies directory and returns the path of the file
// containing the company with the given ID, or empty string if not found.
func (s *MarkdownStore) findCompanyFile(id uuid.UUID) (string, *models.Company, error) {
	idStr := id.String()
	entries, err := os.ReadDir(s.companiesDir())
	if err != nil {
		return "", nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.companiesDir(), entry.Name())
		c, err := readCompanyFile(path)
		if err != nil {
			continue
		}
		if c.ID.String() == idStr {
			return path, c, nil
		}
	}
	return "", nil, nil
}

// CreateCompany writes a new company as a markdown file.
func (s *MarkdownStore) CreateCompany(company *models.Company) error {
	filename := slugForName(company.Name, company.ID.String(), s.companiesDir())
	return s.writeCompany(company, filename)
}

// GetCompany retrieves a company by its UUID.
func (s *MarkdownStore) GetCompany(id uuid.UUID) (*models.Company, error) {
	_, c, err := s.findCompanyFile(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCompanyNotFound
	}
	return c, nil
}

// GetCompanyByPrefix retrieves a company by a UUID prefix (minimum 6 characters).
func (s *MarkdownStore) GetCompanyByPrefix(prefix string) (*models.Company, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}
	entries, err := os.ReadDir(s.companiesDir())
	if err != nil {
		return nil, err
	}
	var matches []*models.Company
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.companiesDir(), entry.Name())
		c, err := readCompanyFile(path)
		if err != nil {
			continue
		}
		if strings.HasPrefix(c.ID.String(), prefix) {
			matches = append(matches, c)
		}
	}
	switch len(matches) {
	case 0:
		return nil, ErrCompanyNotFound
	case 1:
		return matches[0], nil
	default:
		return nil, ErrAmbiguousPrefix
	}
}

// ListCompanies returns companies matching the optional filter.
func (s *MarkdownStore) ListCompanies(filter *CompanyFilter) ([]*models.Company, error) {
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
		if filter != nil && !companyMatchesFilter(c, filter) {
			continue
		}
		results = append(results, c)
		if filter != nil && filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// companyMatchesFilter checks if a company passes the given filter criteria.
func companyMatchesFilter(c *models.Company, f *CompanyFilter) bool {
	if f.Tag != nil {
		found := false
		for _, tag := range c.Tags {
			if tag == *f.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if f.Search != "" {
		return companyMatchesSearch(c, f.Search)
	}
	return true
}

// companyMatchesSearch checks if any company field contains the search string.
func companyMatchesSearch(c *models.Company, query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(c.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Domain), q) {
		return true
	}
	for _, v := range c.Fields {
		if strings.Contains(strings.ToLower(anyToString(v)), q) {
			return true
		}
	}
	return false
}

// UpdateCompany updates an existing company. Returns ErrCompanyNotFound if
// the company does not exist.
func (s *MarkdownStore) UpdateCompany(company *models.Company) error {
	path, existing, err := s.findCompanyFile(company.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCompanyNotFound
	}
	// Preserve original created_at
	if company.CreatedAt.IsZero() {
		company.CreatedAt = existing.CreatedAt
	}
	if company.UpdatedAt.IsZero() || company.UpdatedAt.Before(existing.UpdatedAt) {
		company.UpdatedAt = time.Now()
	}
	// If name changed, we might need a new filename
	filename := filepath.Base(path)
	if existing.Name != company.Name {
		if err := os.Remove(path); err != nil {
			return err
		}
		filename = slugForName(company.Name, company.ID.String(), s.companiesDir())
	}
	return s.writeCompany(company, filename)
}

// DeleteCompany removes the markdown file for the given company ID.
func (s *MarkdownStore) DeleteCompany(id uuid.UUID) error {
	path, c, err := s.findCompanyFile(id)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrCompanyNotFound
	}
	return os.Remove(path)
}
