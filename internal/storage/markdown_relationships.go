// ABOUTME: Markdown storage CRUD operations for relationships.
// ABOUTME: Stores all relationships as a YAML list in _relationships.yaml.
package storage

import (
	"os"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/mdstore"
)

// relationshipEntry is the YAML representation of a relationship in the list.
type relationshipEntry struct {
	ID        string `yaml:"id"`
	SourceID  string `yaml:"source_id"`
	TargetID  string `yaml:"target_id"`
	Type      string `yaml:"type"`
	Context   string `yaml:"context,omitempty"`
	CreatedAt string `yaml:"created_at"`
}

// relationshipToEntry converts a models.Relationship to its YAML entry.
func relationshipToEntry(r *models.Relationship) relationshipEntry {
	return relationshipEntry{
		ID:        r.ID.String(),
		SourceID:  r.SourceID.String(),
		TargetID:  r.TargetID.String(),
		Type:      r.Type,
		Context:   r.Context,
		CreatedAt: mdstore.FormatTime(r.CreatedAt),
	}
}

// entryToRelationship converts a YAML entry back to a models.Relationship.
func entryToRelationship(e relationshipEntry) (*models.Relationship, error) {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return nil, err
	}
	sourceID, err := uuid.Parse(e.SourceID)
	if err != nil {
		return nil, err
	}
	targetID, err := uuid.Parse(e.TargetID)
	if err != nil {
		return nil, err
	}
	createdAt, err := mdstore.ParseTime(e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &models.Relationship{
		ID:        id,
		SourceID:  sourceID,
		TargetID:  targetID,
		Type:      e.Type,
		Context:   e.Context,
		CreatedAt: createdAt,
	}, nil
}

// readRelationships reads all relationships from the YAML file.
func (s *MarkdownStore) readRelationships() ([]relationshipEntry, error) {
	var entries []relationshipEntry
	err := mdstore.ReadYAML(s.relationshipsFile(), &entries)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return entries, nil
}

// writeRelationships writes the full relationships list to the YAML file.
func (s *MarkdownStore) writeRelationships(entries []relationshipEntry) error {
	return mdstore.WriteYAML(s.relationshipsFile(), entries)
}

// CreateRelationship appends a new relationship to the YAML file.
func (s *MarkdownStore) CreateRelationship(rel *models.Relationship) error {
	entries, err := s.readRelationships()
	if err != nil {
		return err
	}
	entries = append(entries, relationshipToEntry(rel))
	return s.writeRelationships(entries)
}

// ListRelationships returns all relationships where the given entity ID appears
// as either source_id or target_id.
func (s *MarkdownStore) ListRelationships(entityID uuid.UUID) ([]*models.Relationship, error) {
	entries, err := s.readRelationships()
	if err != nil {
		return nil, err
	}
	idStr := entityID.String()
	var results []*models.Relationship
	for _, e := range entries {
		if e.SourceID == idStr || e.TargetID == idStr {
			r, err := entryToRelationship(e)
			if err != nil {
				continue
			}
			results = append(results, r)
		}
	}
	return results, nil
}

// DeleteRelationship removes a relationship by its ID from the YAML file.
// Returns ErrRelationshipNotFound if the ID does not exist.
func (s *MarkdownStore) DeleteRelationship(id uuid.UUID) error {
	entries, err := s.readRelationships()
	if err != nil {
		return err
	}
	idStr := id.String()
	found := false
	var remaining []relationshipEntry
	for _, e := range entries {
		if e.ID == idStr {
			found = true
			continue
		}
		remaining = append(remaining, e)
	}
	if !found {
		return ErrRelationshipNotFound
	}
	return s.writeRelationships(remaining)
}
