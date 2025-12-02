// ABOUTME: This file contains tests for the RelationshipsRepository.
// ABOUTME: It verifies CRUD operations, relationship queries, and foreign key constraints.

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelationshipsRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	// Create test objects
	source := &Object{Type: "Person", Name: "Alice"}
	target := &Object{Type: "Company", Name: "Acme Corp"}
	require.NoError(t, objRepo.Create(ctx, source))
	require.NoError(t, objRepo.Create(ctx, target))

	t.Run("create relationship with all fields", func(t *testing.T) {
		rel := &Relationship{
			SourceID: source.ID,
			TargetID: target.ID,
			Type:     "works_at",
			Metadata: map[string]interface{}{
				"role":       "Engineer",
				"start_date": "2024-01-01",
				"department": "Engineering",
			},
		}

		err := relRepo.Create(ctx, rel)
		require.NoError(t, err)

		assert.NotEmpty(t, rel.ID, "ID should be auto-generated")
		assert.NotZero(t, rel.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, rel.UpdatedAt, "UpdatedAt should be set")

		_, err = uuid.Parse(rel.ID)
		assert.NoError(t, err, "ID should be a valid UUID")
	})

	t.Run("create relationship with predefined ID", func(t *testing.T) {
		customID := "custom-rel-123"
		rel := &Relationship{
			ID:       customID,
			SourceID: source.ID,
			TargetID: target.ID,
			Type:     "manages",
		}

		err := relRepo.Create(ctx, rel)
		require.NoError(t, err)

		assert.Equal(t, customID, rel.ID, "Should preserve custom ID")
	})

	t.Run("create nil relationship returns error", func(t *testing.T) {
		err := relRepo.Create(ctx, nil)
		assert.ErrorIs(t, err, ErrInvalidRelationship)
	})

	t.Run("create relationship without source returns error", func(t *testing.T) {
		rel := &Relationship{
			TargetID: target.ID,
			Type:     "test",
		}
		err := relRepo.Create(ctx, rel)
		assert.ErrorIs(t, err, ErrInvalidRelationship)
	})

	t.Run("create relationship without target returns error", func(t *testing.T) {
		rel := &Relationship{
			SourceID: source.ID,
			Type:     "test",
		}
		err := relRepo.Create(ctx, rel)
		assert.ErrorIs(t, err, ErrInvalidRelationship)
	})

	t.Run("create relationship without type returns error", func(t *testing.T) {
		rel := &Relationship{
			SourceID: source.ID,
			TargetID: target.ID,
		}
		err := relRepo.Create(ctx, rel)
		assert.ErrorIs(t, err, ErrInvalidRelationship)
	})
}

func TestRelationshipsRepository_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	source := &Object{Type: "Person", Name: "Alice"}
	target := &Object{Type: "Project", Name: "Project X"}
	require.NoError(t, objRepo.Create(ctx, source))
	require.NoError(t, objRepo.Create(ctx, target))

	t.Run("get existing relationship", func(t *testing.T) {
		original := &Relationship{
			SourceID: source.ID,
			TargetID: target.ID,
			Type:     "assigned_to",
			Metadata: map[string]interface{}{
				"priority": "high",
				"hours":    40,
			},
		}

		err := relRepo.Create(ctx, original)
		require.NoError(t, err)

		retrieved, err := relRepo.Get(ctx, original.ID)
		require.NoError(t, err)

		assert.Equal(t, original.ID, retrieved.ID)
		assert.Equal(t, original.SourceID, retrieved.SourceID)
		assert.Equal(t, original.TargetID, retrieved.TargetID)
		assert.Equal(t, original.Type, retrieved.Type)
		assert.Equal(t, "high", retrieved.Metadata["priority"])
		assert.Equal(t, float64(40), retrieved.Metadata["hours"])
	})

	t.Run("get non-existent relationship returns error", func(t *testing.T) {
		_, err := relRepo.Get(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrRelationshipNotFound)
	})
}

func TestRelationshipsRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	source := &Object{Type: "Person", Name: "Alice"}
	target1 := &Object{Type: "Project", Name: "Project X"}
	target2 := &Object{Type: "Project", Name: "Project Y"}
	require.NoError(t, objRepo.Create(ctx, source))
	require.NoError(t, objRepo.Create(ctx, target1))
	require.NoError(t, objRepo.Create(ctx, target2))

	t.Run("update existing relationship", func(t *testing.T) {
		rel := &Relationship{
			SourceID: source.ID,
			TargetID: target1.ID,
			Type:     "assigned_to",
			Metadata: map[string]interface{}{
				"status": "active",
			},
		}

		err := relRepo.Create(ctx, rel)
		require.NoError(t, err)

		originalCreatedAt := rel.CreatedAt
		time.Sleep(10 * time.Millisecond)

		rel.TargetID = target2.ID
		rel.Type = "leads"
		rel.Metadata["status"] = "completed"

		err = relRepo.Update(ctx, rel)
		require.NoError(t, err)

		assert.True(t, rel.UpdatedAt.After(originalCreatedAt))

		retrieved, err := relRepo.Get(ctx, rel.ID)
		require.NoError(t, err)

		assert.Equal(t, target2.ID, retrieved.TargetID)
		assert.Equal(t, "leads", retrieved.Type)
		assert.Equal(t, "completed", retrieved.Metadata["status"])
	})

	t.Run("update non-existent relationship returns error", func(t *testing.T) {
		rel := &Relationship{
			ID:       "non-existent-id",
			SourceID: source.ID,
			TargetID: target1.ID,
			Type:     "test",
		}

		err := relRepo.Update(ctx, rel)
		assert.ErrorIs(t, err, ErrRelationshipNotFound)
	})

	t.Run("update nil relationship returns error", func(t *testing.T) {
		err := relRepo.Update(ctx, nil)
		assert.ErrorIs(t, err, ErrInvalidRelationship)
	})
}

func TestRelationshipsRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	source := &Object{Type: "Person", Name: "Alice"}
	target := &Object{Type: "Project", Name: "Project X"}
	require.NoError(t, objRepo.Create(ctx, source))
	require.NoError(t, objRepo.Create(ctx, target))

	t.Run("delete existing relationship", func(t *testing.T) {
		rel := &Relationship{
			SourceID: source.ID,
			TargetID: target.ID,
			Type:     "test",
		}

		err := relRepo.Create(ctx, rel)
		require.NoError(t, err)

		err = relRepo.Delete(ctx, rel.ID)
		require.NoError(t, err)

		_, err = relRepo.Get(ctx, rel.ID)
		assert.ErrorIs(t, err, ErrRelationshipNotFound)
	})

	t.Run("delete non-existent relationship returns error", func(t *testing.T) {
		err := relRepo.Delete(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrRelationshipNotFound)
	})
}

func TestRelationshipsRepository_FindBySource(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	alice := &Object{Type: "Person", Name: "Alice"}
	bob := &Object{Type: "Person", Name: "Bob"}
	acme := &Object{Type: "Company", Name: "Acme"}
	techCorp := &Object{Type: "Company", Name: "TechCorp"}

	require.NoError(t, objRepo.Create(ctx, alice))
	require.NoError(t, objRepo.Create(ctx, bob))
	require.NoError(t, objRepo.Create(ctx, acme))
	require.NoError(t, objRepo.Create(ctx, techCorp))

	// Alice works at Acme
	rel1 := &Relationship{SourceID: alice.ID, TargetID: acme.ID, Type: "works_at"}
	// Alice manages Bob
	rel2 := &Relationship{SourceID: alice.ID, TargetID: bob.ID, Type: "manages"}
	// Alice advises TechCorp
	rel3 := &Relationship{SourceID: alice.ID, TargetID: techCorp.ID, Type: "advises"}

	require.NoError(t, relRepo.Create(ctx, rel1))
	time.Sleep(1 * time.Millisecond)
	require.NoError(t, relRepo.Create(ctx, rel2))
	time.Sleep(1 * time.Millisecond)
	require.NoError(t, relRepo.Create(ctx, rel3))

	t.Run("find all relationships from source", func(t *testing.T) {
		rels, err := relRepo.FindBySource(ctx, alice.ID, "")
		require.NoError(t, err)

		assert.Len(t, rels, 3)
		// Should be ordered by created_at DESC
		assert.Equal(t, "advises", rels[0].Type)
		assert.Equal(t, "manages", rels[1].Type)
		assert.Equal(t, "works_at", rels[2].Type)
	})

	t.Run("find relationships from source filtered by type", func(t *testing.T) {
		rels, err := relRepo.FindBySource(ctx, alice.ID, "works_at")
		require.NoError(t, err)

		assert.Len(t, rels, 1)
		assert.Equal(t, "works_at", rels[0].Type)
		assert.Equal(t, acme.ID, rels[0].TargetID)
	})

	t.Run("find with non-existent source returns empty", func(t *testing.T) {
		rels, err := relRepo.FindBySource(ctx, "non-existent", "")
		require.NoError(t, err)
		assert.Empty(t, rels)
	})
}

func TestRelationshipsRepository_FindByTarget(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	alice := &Object{Type: "Person", Name: "Alice"}
	bob := &Object{Type: "Person", Name: "Bob"}
	carol := &Object{Type: "Person", Name: "Carol"}
	acme := &Object{Type: "Company", Name: "Acme"}

	require.NoError(t, objRepo.Create(ctx, alice))
	require.NoError(t, objRepo.Create(ctx, bob))
	require.NoError(t, objRepo.Create(ctx, carol))
	require.NoError(t, objRepo.Create(ctx, acme))

	// Alice works at Acme
	rel1 := &Relationship{SourceID: alice.ID, TargetID: acme.ID, Type: "works_at"}
	// Bob works at Acme
	rel2 := &Relationship{SourceID: bob.ID, TargetID: acme.ID, Type: "works_at"}
	// Carol advises Acme
	rel3 := &Relationship{SourceID: carol.ID, TargetID: acme.ID, Type: "advises"}

	require.NoError(t, relRepo.Create(ctx, rel1))
	time.Sleep(1 * time.Millisecond)
	require.NoError(t, relRepo.Create(ctx, rel2))
	time.Sleep(1 * time.Millisecond)
	require.NoError(t, relRepo.Create(ctx, rel3))

	t.Run("find all relationships to target", func(t *testing.T) {
		rels, err := relRepo.FindByTarget(ctx, acme.ID, "")
		require.NoError(t, err)

		assert.Len(t, rels, 3)
	})

	t.Run("find relationships to target filtered by type", func(t *testing.T) {
		rels, err := relRepo.FindByTarget(ctx, acme.ID, "works_at")
		require.NoError(t, err)

		assert.Len(t, rels, 2)
		for _, rel := range rels {
			assert.Equal(t, "works_at", rel.Type)
		}
	})

	t.Run("find with non-existent target returns empty", func(t *testing.T) {
		rels, err := relRepo.FindByTarget(ctx, "non-existent", "")
		require.NoError(t, err)
		assert.Empty(t, rels)
	})
}

func TestRelationshipsRepository_FindBetween(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	alice := &Object{Type: "Person", Name: "Alice"}
	bob := &Object{Type: "Person", Name: "Bob"}
	carol := &Object{Type: "Person", Name: "Carol"}

	require.NoError(t, objRepo.Create(ctx, alice))
	require.NoError(t, objRepo.Create(ctx, bob))
	require.NoError(t, objRepo.Create(ctx, carol))

	// Alice manages Bob
	rel1 := &Relationship{SourceID: alice.ID, TargetID: bob.ID, Type: "manages"}
	// Bob reports to Alice
	rel2 := &Relationship{SourceID: bob.ID, TargetID: alice.ID, Type: "reports_to"}

	require.NoError(t, relRepo.Create(ctx, rel1))
	require.NoError(t, relRepo.Create(ctx, rel2))

	t.Run("find relationships between two objects", func(t *testing.T) {
		rels, err := relRepo.FindBetween(ctx, alice.ID, bob.ID)
		require.NoError(t, err)

		assert.Len(t, rels, 2)
	})

	t.Run("find relationships works in both directions", func(t *testing.T) {
		rels1, err := relRepo.FindBetween(ctx, alice.ID, bob.ID)
		require.NoError(t, err)

		rels2, err := relRepo.FindBetween(ctx, bob.ID, alice.ID)
		require.NoError(t, err)

		assert.Equal(t, len(rels1), len(rels2))
	})

	t.Run("find between objects with no relationships", func(t *testing.T) {
		rels, err := relRepo.FindBetween(ctx, alice.ID, carol.ID)
		require.NoError(t, err)
		assert.Empty(t, rels)
	})
}

func TestRelationshipsRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	alice := &Object{Type: "Person", Name: "Alice"}
	bob := &Object{Type: "Person", Name: "Bob"}
	acme := &Object{Type: "Company", Name: "Acme"}

	require.NoError(t, objRepo.Create(ctx, alice))
	require.NoError(t, objRepo.Create(ctx, bob))
	require.NoError(t, objRepo.Create(ctx, acme))

	rel1 := &Relationship{SourceID: alice.ID, TargetID: acme.ID, Type: "works_at"}
	rel2 := &Relationship{SourceID: bob.ID, TargetID: acme.ID, Type: "works_at"}
	rel3 := &Relationship{SourceID: alice.ID, TargetID: bob.ID, Type: "manages"}

	require.NoError(t, relRepo.Create(ctx, rel1))
	require.NoError(t, relRepo.Create(ctx, rel2))
	require.NoError(t, relRepo.Create(ctx, rel3))

	t.Run("list all relationships", func(t *testing.T) {
		rels, err := relRepo.List(ctx, "")
		require.NoError(t, err)
		assert.Len(t, rels, 3)
	})

	t.Run("list filtered by type", func(t *testing.T) {
		rels, err := relRepo.List(ctx, "works_at")
		require.NoError(t, err)

		assert.Len(t, rels, 2)
		for _, rel := range rels {
			assert.Equal(t, "works_at", rel.Type)
		}
	})

	t.Run("list with non-existent type returns empty", func(t *testing.T) {
		rels, err := relRepo.List(ctx, "non_existent_type")
		require.NoError(t, err)
		assert.Empty(t, rels)
	})
}

func TestRelationshipsRepository_CascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	objRepo := NewObjectsRepository(db)
	relRepo := NewRelationshipsRepository(db)
	ctx := context.Background()

	t.Run("deleting object cascades to relationships", func(t *testing.T) {
		alice := &Object{Type: "Person", Name: "Alice"}
		bob := &Object{Type: "Person", Name: "Bob"}

		require.NoError(t, objRepo.Create(ctx, alice))
		require.NoError(t, objRepo.Create(ctx, bob))

		rel := &Relationship{
			SourceID: alice.ID,
			TargetID: bob.ID,
			Type:     "knows",
		}
		require.NoError(t, relRepo.Create(ctx, rel))

		// Delete Alice
		require.NoError(t, objRepo.Delete(ctx, alice.ID))

		// Relationship should be deleted
		_, err := relRepo.Get(ctx, rel.ID)
		assert.ErrorIs(t, err, ErrRelationshipNotFound)
	})
}
