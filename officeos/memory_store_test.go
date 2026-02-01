// ABOUTME: Tests for in-memory object store
// ABOUTME: Covers CRUD operations, query, and concurrency
package officeos

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemoryStoreCreate(t *testing.T) {
	store := NewMemoryStore()

	obj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Fields: map[string]interface{}{
			"name": "Test",
		},
	}

	err := store.Create(obj)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected count 1, got %d", store.Count())
	}
}

func TestMemoryStoreCreateDuplicate(t *testing.T) {
	store := NewMemoryStore()

	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
	}

	err := store.Create(obj)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	err = store.Create(obj)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("Expected ErrAlreadyExists, got %v", err)
	}
}

func TestMemoryStoreGet(t *testing.T) {
	store := NewMemoryStore()

	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Fields: map[string]interface{}{
			"name": "Test Object",
		},
	}

	_ = store.Create(obj)

	fetched, err := store.Get("test-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if fetched.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got %s", fetched.ID)
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreUpdate(t *testing.T) {
	store := NewMemoryStore()

	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Fields: map[string]interface{}{
			"name": "Original",
		},
	}

	_ = store.Create(obj)

	obj.Fields = map[string]interface{}{
		"name": "Updated",
	}

	err := store.Update(obj)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	fetched, _ := store.Get("test-1")
	if fetched.Fields.(map[string]interface{})["name"] != "Updated" {
		t.Error("Update did not persist")
	}
}

func TestMemoryStoreUpdateNotFound(t *testing.T) {
	store := NewMemoryStore()

	obj := &BaseObject{
		ID:   "nonexistent",
		Kind: KindRecord,
	}

	err := store.Update(obj)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore()

	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
	}

	_ = store.Create(obj)

	err := store.Delete("test-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Expected count 0, got %d", store.Count())
	}
}

func TestMemoryStoreDeleteNotFound(t *testing.T) {
	store := NewMemoryStore()

	err := store.Delete("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreQuery(t *testing.T) {
	store := NewMemoryStore()

	// Create objects of different kinds
	_ = store.Create(&BaseObject{ID: "record-1", Kind: KindRecord})
	_ = store.Create(&BaseObject{ID: "record-2", Kind: KindRecord})
	_ = store.Create(&BaseObject{ID: "task-1", Kind: KindTask})

	// Query by kind
	records, err := store.Query(KindRecord, nil)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}

	tasks, err := store.Query(KindTask, nil)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
}

func TestMemoryStoreQueryEmpty(t *testing.T) {
	store := NewMemoryStore()

	results, err := store.Query(KindRecord, nil)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestMemoryStoreClear(t *testing.T) {
	store := NewMemoryStore()

	_ = store.Create(&BaseObject{ID: "1", Kind: KindRecord})
	_ = store.Create(&BaseObject{ID: "2", Kind: KindRecord})
	_ = store.Create(&BaseObject{ID: "3", Kind: KindRecord})

	if store.Count() != 3 {
		t.Errorf("Expected count 3, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", store.Count())
	}
}

func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore()
	var wg sync.WaitGroup

	// Create objects concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			obj := &BaseObject{
				ID:   string(rune('a'+id%26)) + string(rune('0'+id)),
				Kind: KindRecord,
			}
			_ = store.Create(obj)
		}(i)
	}

	wg.Wait()

	// Read concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.Query(KindRecord, nil)
		}()
	}

	wg.Wait()

	// Test passed if no race condition detected
}

func TestMemoryStoreIsolation(t *testing.T) {
	store := NewMemoryStore()

	// Create an object
	original := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Fields: map[string]interface{}{
			"name": "Original",
		},
	}
	_ = store.Create(original)

	// Modify the original after creating
	original.Fields = map[string]interface{}{
		"name": "Modified External",
	}

	// Get should return a copy, not affected by external modification
	fetched, _ := store.Get("test-1")

	// The stored version should still have the original value
	if fetched.Fields.(map[string]interface{})["name"] == "Modified External" {
		t.Error("Store should copy objects on create to prevent external mutation")
	}
}

func TestMemoryStoreGetIsolation(t *testing.T) {
	store := NewMemoryStore()

	_ = store.Create(&BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Fields: map[string]interface{}{
			"name": "Original",
		},
	})

	// Get the object
	fetched, _ := store.Get("test-1")

	// Modify the fetched object
	fetched.Fields = map[string]interface{}{
		"name": "Modified After Fetch",
	}

	// Get again should return original value
	fetched2, _ := store.Get("test-1")
	if fetched2.Fields.(map[string]interface{})["name"] == "Modified After Fetch" {
		t.Error("Store should return copies on get to prevent external mutation")
	}
}
