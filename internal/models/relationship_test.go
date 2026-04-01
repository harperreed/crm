// ABOUTME: Tests for the Relationship model struct and its constructor.
// ABOUTME: Verifies ID generation, field assignment, and timestamp initialization.
package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewRelationship(t *testing.T) {
	srcID := uuid.New()
	tgtID := uuid.New()
	relType := "works_at"
	ctx := "current employer"

	r := NewRelationship(srcID, tgtID, relType, ctx)

	if r == nil {
		t.Fatal("NewRelationship returned nil")
	}

	if r.ID == uuid.Nil {
		t.Error("expected non-nil UUID, got uuid.Nil")
	}

	if r.SourceID != srcID {
		t.Errorf("expected SourceID %v, got %v", srcID, r.SourceID)
	}

	if r.TargetID != tgtID {
		t.Errorf("expected TargetID %v, got %v", tgtID, r.TargetID)
	}

	if r.Type != relType {
		t.Errorf("expected Type %q, got %q", relType, r.Type)
	}

	if r.Context != ctx {
		t.Errorf("expected Context %q, got %q", ctx, r.Context)
	}

	if r.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero time")
	}
}

func TestNewRelationship_UniqueIDs(t *testing.T) {
	srcID := uuid.New()
	tgtID := uuid.New()

	r1 := NewRelationship(srcID, tgtID, "works_at", "")
	r2 := NewRelationship(srcID, tgtID, "works_at", "")

	if r1.ID == r2.ID {
		t.Error("expected different UUIDs for different relationships")
	}
}
