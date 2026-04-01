// ABOUTME: Markdown file-based storage for CRM data.
// ABOUTME: Provides constructor and helpers for the markdown backend.
package storage

import "fmt"

// MarkdownStore implements Storage using markdown files on disk.
type MarkdownStore struct {
	DataDir string
}

// NewMarkdownStore creates a new MarkdownStore backed by the given directory.
func NewMarkdownStore(dataDir string) (*MarkdownStore, error) {
	return nil, fmt.Errorf("markdown backend not yet implemented")
}
