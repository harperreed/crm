// ABOUTME: In-memory object store implementation for testing
// ABOUTME: Provides thread-safe storage without database dependencies
package officeos

import (
	"errors"
	"sync"
)

var (
	// ErrNotFound is returned when an object is not found.
	ErrNotFound = errors.New("object not found")
	// ErrAlreadyExists is returned when trying to create an object that already exists.
	ErrAlreadyExists = errors.New("object already exists")
)

// MemoryStore is an in-memory implementation of ObjectStore for testing.
type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]*BaseObject
}

// NewMemoryStore creates a new in-memory object store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		objects: make(map[string]*BaseObject),
	}
}

// Create adds a new object to the store.
func (m *MemoryStore) Create(obj *BaseObject) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[obj.ID]; exists {
		return ErrAlreadyExists
	}

	// Store a copy to avoid external mutations
	copy := *obj
	m.objects[obj.ID] = &copy

	return nil
}

// Get retrieves an object by ID.
func (m *MemoryStore) Get(id string) (*BaseObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, exists := m.objects[id]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to avoid external mutations
	copy := *obj
	return &copy, nil
}

// Update modifies an existing object.
func (m *MemoryStore) Update(obj *BaseObject) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[obj.ID]; !exists {
		return ErrNotFound
	}

	// Store a copy to avoid external mutations
	copy := *obj
	m.objects[obj.ID] = &copy

	return nil
}

// Delete removes an object from the store.
func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[id]; !exists {
		return ErrNotFound
	}

	delete(m.objects, id)
	return nil
}

// Query retrieves objects matching the given filters.
func (m *MemoryStore) Query(kind ObjectKind, filters map[string]interface{}) ([]*BaseObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*BaseObject
	for _, obj := range m.objects {
		if obj.Kind != kind {
			continue
		}

		// TODO: Apply additional filters from the filters map
		// For now, we only filter by kind

		// Return a copy to avoid external mutations
		copy := *obj
		results = append(results, &copy)
	}

	return results, nil
}

// Clear removes all objects from the store (useful for testing).
func (m *MemoryStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects = make(map[string]*BaseObject)
}

// Count returns the number of objects in the store.
func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objects)
}
