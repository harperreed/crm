// ABOUTME: Markdown file-based storage backend for CRM data.
// ABOUTME: Stores contacts and companies as .md files with YAML frontmatter, relationships as YAML.
package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/mdstore"
)

// Compile-time check that MarkdownStore implements Storage.
var _ Storage = (*MarkdownStore)(nil)

// MarkdownStore implements Storage using markdown files on disk.
type MarkdownStore struct {
	dataDir string
	source  history.Source
	now     func() time.Time
	mu      sync.Mutex
}

// NewMarkdownStore creates a new MarkdownStore backed by the given directory.
// It creates the data and committed-history subdirectories if needed.
func NewMarkdownStore(dataDir string, source history.Source) (*MarkdownStore, error) {
	if err := validateHistorySource(source); err != nil {
		return nil, err
	}
	for _, dir := range []string{
		dataDir,
		filepath.Join(dataDir, "contacts"),
		filepath.Join(dataDir, "companies"),
		filepath.Join(dataDir, "_history", "events"),
		filepath.Join(dataDir, "_history", "pending"),
	} {
		if err := ensureDurableDirectory(dir); err != nil {
			return nil, err
		}
	}
	store := &MarkdownStore{dataDir: dataDir, source: source, now: time.Now}
	if err := store.recoverPendingHistory(); err != nil {
		return nil, err
	}
	return store, nil
}

func ensureDurableDirectory(path string) error {
	path = filepath.Clean(path)
	missing := make([]string, 0)
	current := path
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("directory path %s is not a directory", current)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect directory path %s: %w", current, err)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("find existing parent for directory %s", path)
		}
		current = parent
	}
	for index := len(missing) - 1; index >= 0; index-- {
		dir := missing[index]
		if err := os.Mkdir(dir, 0o750); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
		if err := syncMarkdownHistoryDirectory(filepath.Dir(dir)); err != nil {
			return fmt.Errorf("make directory %s durable: %w", dir, err)
		}
	}
	return nil
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

func availableSlugForName(name, id, dir, currentPath string) (string, error) {
	base := mdstore.Slugify(name) + ".md"
	candidates := []string{base, id[:8] + "-" + base, id + "-" + base}
	for suffix := 0; ; suffix++ {
		var candidate string
		if suffix < len(candidates) {
			candidate = candidates[suffix]
		} else {
			candidate = fmt.Sprintf("%s-%s-%d.md", id, strings.TrimSuffix(base, ".md"), suffix-len(candidates)+2)
		}
		path := filepath.Join(dir, candidate)
		if path == currentPath {
			return candidate, nil
		}
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect filename candidate %s: %w", path, err)
		}
	}
}
