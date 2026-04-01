// ABOUTME: Markdown file-based storage backend for CRM data.
// ABOUTME: Stores contacts and companies as .md files with YAML frontmatter, relationships as YAML.
package storage

import (
	"os"
	"path/filepath"

	"github.com/harperreed/mdstore"
)

// Compile-time check that MarkdownStore implements Storage.
var _ Storage = (*MarkdownStore)(nil)

// MarkdownStore implements Storage using markdown files on disk.
type MarkdownStore struct {
	dataDir string
}

// NewMarkdownStore creates a new MarkdownStore backed by the given directory.
// It creates the dataDir, contacts/, and companies/ subdirectories if needed.
func NewMarkdownStore(dataDir string) (*MarkdownStore, error) {
	for _, dir := range []string{
		dataDir,
		filepath.Join(dataDir, "contacts"),
		filepath.Join(dataDir, "companies"),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, err
		}
	}
	return &MarkdownStore{dataDir: dataDir}, nil
}

// Close is a no-op for the file-based backend.
func (s *MarkdownStore) Close() error { return nil }

// contactsDir returns the path to the contacts directory.
func (s *MarkdownStore) contactsDir() string {
	return filepath.Join(s.dataDir, "contacts")
}

// companiesDir returns the path to the companies directory.
func (s *MarkdownStore) companiesDir() string {
	return filepath.Join(s.dataDir, "companies")
}

// relationshipsFile returns the path to the relationships YAML file.
func (s *MarkdownStore) relationshipsFile() string {
	return filepath.Join(s.dataDir, "_relationships.yaml")
}

// slugForName generates a filename-safe slug, appending a UUID prefix on collision.
func slugForName(name, id, dir string) string {
	base := mdstore.Slugify(name)
	candidate := base + ".md"
	path := filepath.Join(dir, candidate)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return candidate
	}
	// Collision: prepend first 8 chars of UUID
	return id[:8] + "-" + base + ".md"
}
