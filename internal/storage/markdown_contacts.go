// ABOUTME: Markdown storage CRUD operations for contacts.
// ABOUTME: Stores each contact as a .md file with YAML frontmatter in the contacts/ directory.
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

// contactFrontmatter is the YAML representation of a contact stored in frontmatter.
type contactFrontmatter struct {
	ID        string         `yaml:"id"`
	Name      string         `yaml:"name"`
	Email     string         `yaml:"email,omitempty"`
	Phone     string         `yaml:"phone,omitempty"`
	Fields    map[string]any `yaml:"fields,omitempty"`
	Tags      []string       `yaml:"tags,omitempty"`
	CreatedAt string         `yaml:"created_at"`
	UpdatedAt string         `yaml:"updated_at"`
}

// contactToFrontmatter converts a models.Contact to its YAML frontmatter representation.
func contactToFrontmatter(c *models.Contact) contactFrontmatter {
	return contactFrontmatter{
		ID:        c.ID.String(),
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		Fields:    c.Fields,
		Tags:      c.Tags,
		CreatedAt: mdstore.FormatTime(c.CreatedAt),
		UpdatedAt: mdstore.FormatTime(c.UpdatedAt),
	}
}

// frontmatterToContact converts a contactFrontmatter back to a models.Contact.
func frontmatterToContact(fm contactFrontmatter) (*models.Contact, error) {
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
	return &models.Contact{
		ID:        id,
		Name:      fm.Name,
		Email:     fm.Email,
		Phone:     fm.Phone,
		Fields:    fields,
		Tags:      tags,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// writeContact writes a contact as a .md file with YAML frontmatter.
func (s *MarkdownStore) writeContact(c *models.Contact, filename string) error {
	fm := contactToFrontmatter(c)
	content, err := mdstore.RenderFrontmatter(fm, "")
	if err != nil {
		return err
	}
	return mdstore.AtomicWrite(filepath.Join(s.contactsDir(), filename), []byte(content))
}

// readContactFile reads a single contact .md file and returns the contact and filename.
func readContactFile(path string) (*models.Contact, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) //nolint:gosec // path is constructed from trusted directory listing
	if err != nil {
		return nil, err
	}
	yamlStr, _ := mdstore.ParseFrontmatter(string(data))
	var fm contactFrontmatter
	if err := yamlUnmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, err
	}
	return frontmatterToContact(fm)
}

// findContactFile walks the contacts directory and returns the path of the file
// containing the contact with the given ID, or empty string if not found.
func (s *MarkdownStore) findContactFile(id uuid.UUID) (string, *models.Contact, error) {
	idStr := id.String()
	entries, err := os.ReadDir(s.contactsDir())
	if err != nil {
		return "", nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.contactsDir(), entry.Name())
		c, err := readContactFile(path)
		if err != nil {
			continue
		}
		if c.ID.String() == idStr {
			return path, c, nil
		}
	}
	return "", nil, nil
}

// CreateContact writes a new contact as a markdown file.
func (s *MarkdownStore) CreateContact(contact *models.Contact) error {
	filename := slugForName(contact.Name, contact.ID.String(), s.contactsDir())
	return s.writeContact(contact, filename)
}

// GetContact retrieves a contact by its UUID.
func (s *MarkdownStore) GetContact(id uuid.UUID) (*models.Contact, error) {
	_, c, err := s.findContactFile(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrContactNotFound
	}
	return c, nil
}

// GetContactByPrefix retrieves a contact by a UUID prefix (minimum 6 characters).
func (s *MarkdownStore) GetContactByPrefix(prefix string) (*models.Contact, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}
	entries, err := os.ReadDir(s.contactsDir())
	if err != nil {
		return nil, err
	}
	var matches []*models.Contact
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.contactsDir(), entry.Name())
		c, err := readContactFile(path)
		if err != nil {
			continue
		}
		if strings.HasPrefix(c.ID.String(), prefix) {
			matches = append(matches, c)
		}
	}
	switch len(matches) {
	case 0:
		return nil, ErrContactNotFound
	case 1:
		return matches[0], nil
	default:
		return nil, ErrAmbiguousPrefix
	}
}

// ListContacts returns contacts matching the optional filter.
func (s *MarkdownStore) ListContacts(filter *ContactFilter) ([]*models.Contact, error) {
	entries, err := os.ReadDir(s.contactsDir())
	if err != nil {
		return nil, err
	}
	results := make([]*models.Contact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.contactsDir(), entry.Name())
		c, err := readContactFile(path)
		if err != nil {
			continue
		}
		if filter != nil && !contactMatchesFilter(c, filter) {
			continue
		}
		results = append(results, c)
		if filter != nil && filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// contactMatchesFilter checks if a contact passes the given filter criteria.
func contactMatchesFilter(c *models.Contact, f *ContactFilter) bool {
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
		return contactMatchesSearch(c, f.Search)
	}
	return true
}

// contactMatchesSearch checks if any contact field contains the search string.
func contactMatchesSearch(c *models.Contact, query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(c.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Email), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Phone), q) {
		return true
	}
	for _, v := range c.Fields {
		if strings.Contains(strings.ToLower(anyToString(v)), q) {
			return true
		}
	}
	return false
}

// UpdateContact updates an existing contact. Returns ErrContactNotFound if
// the contact does not exist.
func (s *MarkdownStore) UpdateContact(contact *models.Contact) error {
	path, existing, err := s.findContactFile(contact.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrContactNotFound
	}
	// Preserve original created_at
	if contact.CreatedAt.IsZero() {
		contact.CreatedAt = existing.CreatedAt
	}
	if contact.UpdatedAt.IsZero() || contact.UpdatedAt.Before(existing.UpdatedAt) {
		contact.UpdatedAt = time.Now()
	}
	// If name changed, we might need a new filename
	filename := filepath.Base(path)
	if existing.Name != contact.Name {
		// Remove old file, write with new slug
		if err := os.Remove(path); err != nil {
			return err
		}
		filename = slugForName(contact.Name, contact.ID.String(), s.contactsDir())
	}
	return s.writeContact(contact, filename)
}

// DeleteContact removes the markdown file for the given contact ID.
func (s *MarkdownStore) DeleteContact(id uuid.UUID) error {
	path, c, err := s.findContactFile(id)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrContactNotFound
	}
	return os.Remove(path)
}
