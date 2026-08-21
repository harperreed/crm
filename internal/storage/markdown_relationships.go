// ABOUTME: Markdown storage CRUD operations for relationships.
// ABOUTME: Stores all relationships as a YAML list in _relationships.yaml.
package storage

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/mdstore"
	"gopkg.in/yaml.v3"
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
	data, err := yaml.Marshal(entries)
	if err != nil {
		return fmt.Errorf("marshal relationships: %w", err)
	}
	if err := mdstore.AtomicWrite(s.relationshipsFile(), data); err != nil {
		return fmt.Errorf("write relationships: %w", err)
	}
	return syncMarkdownHistoryDirectory(s.dataDir)
}

// CreateRelationship appends a new relationship to the YAML file.
func (s *MarkdownStore) CreateRelationship(rel *models.Relationship) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.recoverPendingHistoryLocked(); err != nil {
		return err
	}
	if _, err := s.findRelationshipStrict(rel.ID); err == nil {
		return fmt.Errorf("%w: relationship %s already exists", ErrHistoryConflict, rel.ID)
	} else if !errors.Is(err, ErrRelationshipNotFound) {
		return err
	}
	after, err := markdownRelationshipSnapshot(rel)
	if err != nil {
		return fmt.Errorf("snapshot relationship: %w", err)
	}
	event, err := history.NewEvent(
		history.EntityRelationship,
		rel.ID,
		[]uuid.UUID{rel.ID, rel.SourceID, rel.TargetID},
		history.ActionCreate,
		s.source,
		nil,
		after,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create relationship history event: %w", err)
	}
	return s.commitHistoryEvent(event)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.recoverPendingHistoryLocked(); err != nil {
		return err
	}
	relationship, err := s.findRelationshipStrict(id)
	if errors.Is(err, ErrRelationshipNotFound) {
		return ErrRelationshipNotFound
	}
	if err != nil {
		return err
	}
	before, err := markdownRelationshipSnapshot(relationship)
	if err != nil {
		return fmt.Errorf("snapshot relationship: %w", err)
	}
	event, err := history.NewEvent(
		history.EntityRelationship,
		id,
		[]uuid.UUID{id, relationship.SourceID, relationship.TargetID},
		history.ActionDelete,
		s.source,
		before,
		nil,
		s.now(),
	)
	if err != nil {
		return fmt.Errorf("create relationship history event: %w", err)
	}
	return s.commitHistoryEvent(event)
}
