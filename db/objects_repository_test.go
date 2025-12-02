// ABOUTME: This file contains tests for the ObjectsRepository.
// ABOUTME: It verifies CRUD operations, error handling, and JSON metadata marshaling.

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectsRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewObjectsRepository(db)
	ctx := context.Background()

	t.Run("create object with all fields", func(t *testing.T) {
		obj := &Object{
			Type: "Activity",
			Name: "Morning Standup",
			Metadata: map[string]interface{}{
				"duration":   30,
				"attendees":  []string{"alice", "bob"},
				"recurring":  true,
				"project_id": "proj-123",
			},
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)

		assert.NotEmpty(t, obj.ID, "ID should be auto-generated")
		assert.NotZero(t, obj.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, obj.UpdatedAt, "UpdatedAt should be set")
		assert.Equal(t, obj.CreatedAt, obj.UpdatedAt, "CreatedAt and UpdatedAt should match on create")

		_, err = uuid.Parse(obj.ID)
		assert.NoError(t, err, "ID should be a valid UUID")
	})

	t.Run("create object with predefined ID", func(t *testing.T) {
		customID := "custom-id-123"
		obj := &Object{
			ID:   customID,
			Type: "Task",
			Name: "Custom Task",
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)

		assert.Equal(t, customID, obj.ID, "Should preserve custom ID")
	})

	t.Run("create object with nil metadata", func(t *testing.T) {
		obj := &Object{
			Type: "SimpleObject",
			Name: "No Metadata",
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)
		assert.NotEmpty(t, obj.ID)
	})

	t.Run("create nil object returns error", func(t *testing.T) {
		err := repo.Create(ctx, nil)
		assert.ErrorIs(t, err, ErrInvalidObject)
	})
}

func TestObjectsRepository_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewObjectsRepository(db)
	ctx := context.Background()

	t.Run("get existing object", func(t *testing.T) {
		original := &Object{
			Type: "Activity",
			Name: "Team Meeting",
			Metadata: map[string]interface{}{
				"location": "Room 404",
				"urgent":   true,
				"priority": 5,
			},
		}

		err := repo.Create(ctx, original)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, original.ID)
		require.NoError(t, err)

		assert.Equal(t, original.ID, retrieved.ID)
		assert.Equal(t, original.Type, retrieved.Type)
		assert.Equal(t, original.Name, retrieved.Name)
		assert.Equal(t, "Room 404", retrieved.Metadata["location"])
		assert.Equal(t, true, retrieved.Metadata["urgent"])
		assert.Equal(t, float64(5), retrieved.Metadata["priority"]) // JSON unmarshals numbers as float64
		assert.Equal(t, original.CreatedAt.Unix(), retrieved.CreatedAt.Unix())
	})

	t.Run("get non-existent object returns error", func(t *testing.T) {
		_, err := repo.Get(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestObjectsRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewObjectsRepository(db)
	ctx := context.Background()

	t.Run("update existing object", func(t *testing.T) {
		obj := &Object{
			Type: "Task",
			Name: "Original Name",
			Metadata: map[string]interface{}{
				"status": "pending",
			},
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)

		originalCreatedAt := obj.CreatedAt
		time.Sleep(10 * time.Millisecond) // Ensure UpdatedAt changes

		obj.Name = "Updated Name"
		obj.Type = "UpdatedTask"
		obj.Metadata["status"] = "completed"
		obj.Metadata["new_field"] = "new_value"

		err = repo.Update(ctx, obj)
		require.NoError(t, err)

		assert.True(t, obj.UpdatedAt.After(originalCreatedAt), "UpdatedAt should be newer")

		retrieved, err := repo.Get(ctx, obj.ID)
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "UpdatedTask", retrieved.Type)
		assert.Equal(t, "completed", retrieved.Metadata["status"])
		assert.Equal(t, "new_value", retrieved.Metadata["new_field"])
		assert.Equal(t, originalCreatedAt.Unix(), retrieved.CreatedAt.Unix(), "CreatedAt should not change")
	})

	t.Run("update non-existent object returns error", func(t *testing.T) {
		obj := &Object{
			ID:   "non-existent-id",
			Type: "Task",
			Name: "Ghost",
		}

		err := repo.Update(ctx, obj)
		assert.ErrorIs(t, err, ErrObjectNotFound)
	})

	t.Run("update nil object returns error", func(t *testing.T) {
		err := repo.Update(ctx, nil)
		assert.ErrorIs(t, err, ErrInvalidObject)
	})

	t.Run("update object with empty ID returns error", func(t *testing.T) {
		obj := &Object{
			Type: "Task",
			Name: "No ID",
		}

		err := repo.Update(ctx, obj)
		assert.ErrorIs(t, err, ErrInvalidObject)
	})
}

func TestObjectsRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewObjectsRepository(db)
	ctx := context.Background()

	t.Run("delete existing object", func(t *testing.T) {
		obj := &Object{
			Type: "Task",
			Name: "To Be Deleted",
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)

		err = repo.Delete(ctx, obj.ID)
		require.NoError(t, err)

		_, err = repo.Get(ctx, obj.ID)
		assert.ErrorIs(t, err, ErrObjectNotFound, "Object should not exist after deletion")
	})

	t.Run("delete non-existent object returns error", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestObjectsRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewObjectsRepository(db)
	ctx := context.Background()

	t.Run("list all objects", func(t *testing.T) {
		objects := []*Object{
			{Type: "Activity", Name: "Activity 1"},
			{Type: "Task", Name: "Task 1"},
			{Type: "Activity", Name: "Activity 2"},
			{Type: "Task", Name: "Task 2"},
		}

		for _, obj := range objects {
			err := repo.Create(ctx, obj)
			require.NoError(t, err)
			time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		}

		allObjects, err := repo.List(ctx, "")
		require.NoError(t, err)

		assert.Len(t, allObjects, 4)
		// Should be ordered by created_at DESC (most recent first)
		assert.Equal(t, "Task 2", allObjects[0].Name)
		assert.Equal(t, "Activity 2", allObjects[1].Name)
		assert.Equal(t, "Task 1", allObjects[2].Name)
		assert.Equal(t, "Activity 1", allObjects[3].Name)
	})

	t.Run("list objects filtered by type", func(t *testing.T) {
		activities, err := repo.List(ctx, "Activity")
		require.NoError(t, err)

		assert.Len(t, activities, 2)
		for _, obj := range activities {
			assert.Equal(t, "Activity", obj.Type)
		}
	})

	t.Run("list with non-existent type returns empty slice", func(t *testing.T) {
		objects, err := repo.List(ctx, "NonExistentType")
		require.NoError(t, err)
		assert.Empty(t, objects)
	})

	t.Run("list empty database returns empty slice", func(t *testing.T) {
		// Create a fresh database
		freshDB := setupTestDB(t)
		defer freshDB.Close()
		freshRepo := NewObjectsRepository(freshDB)

		objects, err := freshRepo.List(ctx, "")
		require.NoError(t, err)
		assert.NotNil(t, objects)
		assert.Empty(t, objects)
	})
}

func TestObjectsRepository_MetadataHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewObjectsRepository(db)
	ctx := context.Background()

	t.Run("handle complex nested metadata", func(t *testing.T) {
		obj := &Object{
			Type: "ComplexObject",
			Name: "Nested Data",
			Metadata: map[string]interface{}{
				"string": "value",
				"number": 42,
				"bool":   true,
				"null":   nil,
				"array":  []interface{}{"a", "b", "c"},
				"nested": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep",
					},
				},
			},
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, obj.ID)
		require.NoError(t, err)

		assert.Equal(t, "value", retrieved.Metadata["string"])
		assert.Equal(t, float64(42), retrieved.Metadata["number"])
		assert.Equal(t, true, retrieved.Metadata["bool"])
		assert.Nil(t, retrieved.Metadata["null"])

		array := retrieved.Metadata["array"].([]interface{})
		assert.Len(t, array, 3)
		assert.Equal(t, "a", array[0])

		nested := retrieved.Metadata["nested"].(map[string]interface{})
		level2 := nested["level2"].(map[string]interface{})
		assert.Equal(t, "deep", level2["level3"])
	})

	t.Run("handle empty metadata", func(t *testing.T) {
		obj := &Object{
			Type:     "EmptyMetadata",
			Name:     "Empty",
			Metadata: map[string]interface{}{},
		}

		err := repo.Create(ctx, obj)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, obj.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)
		assert.Empty(t, retrieved.Metadata)
	})
}
