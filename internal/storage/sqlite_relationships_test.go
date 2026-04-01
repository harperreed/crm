// ABOUTME: Tests for SQLite relationship CRUD operations.
// ABOUTME: Covers create, bidirectional list, delete, and not-found scenarios.
package storage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestCreateRelationship(t *testing.T) {
	store := newTestStore(t)

	srcID := uuid.New()
	tgtID := uuid.New()
	rel := models.NewRelationship(srcID, tgtID, "works_at", "engineering team")

	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}

	rels, err := store.ListRelationships(srcID)
	if err != nil {
		t.Fatalf("ListRelationships: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("len = %d, want 1", len(rels))
	}
	if rels[0].Type != "works_at" {
		t.Errorf("Type = %q, want %q", rels[0].Type, "works_at")
	}
	if rels[0].Context != "engineering team" {
		t.Errorf("Context = %q, want %q", rels[0].Context, "engineering team")
	}
}

func TestListRelationshipsBidirectional(t *testing.T) {
	store := newTestStore(t)

	a := uuid.New()
	b := uuid.New()
	c := uuid.New()

	r1 := models.NewRelationship(a, b, "knows", "")
	r2 := models.NewRelationship(c, a, "manages", "")

	for _, r := range []*models.Relationship{r1, r2} {
		if err := store.CreateRelationship(r); err != nil {
			t.Fatalf("CreateRelationship: %v", err)
		}
	}

	// Query by entity A should find both relationships
	rels, err := store.ListRelationships(a)
	if err != nil {
		t.Fatalf("ListRelationships(a): %v", err)
	}
	if len(rels) != 2 {
		t.Errorf("ListRelationships(a) len = %d, want 2", len(rels))
	}

	// Query by entity B should find only r1
	rels, err = store.ListRelationships(b)
	if err != nil {
		t.Fatalf("ListRelationships(b): %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("ListRelationships(b) len = %d, want 1", len(rels))
	}
}

func TestDeleteRelationship(t *testing.T) {
	store := newTestStore(t)

	rel := models.NewRelationship(uuid.New(), uuid.New(), "partner", "")
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}

	if err := store.DeleteRelationship(rel.ID); err != nil {
		t.Fatalf("DeleteRelationship: %v", err)
	}

	rels, err := store.ListRelationships(rel.SourceID)
	if err != nil {
		t.Fatalf("ListRelationships: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(rels))
	}
}

func TestDeleteRelationshipNotFound(t *testing.T) {
	store := newTestStore(t)

	if err := store.DeleteRelationship(uuid.New()); !errors.Is(err, ErrRelationshipNotFound) {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}
}
