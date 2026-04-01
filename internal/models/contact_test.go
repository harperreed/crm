// ABOUTME: Tests for the Contact model struct and its constructor/methods.
// ABOUTME: Verifies ID generation, field initialization, timestamps, and Touch behavior.
package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewContact(t *testing.T) {
	name := "Alice Smith"
	c := NewContact(name)

	if c == nil {
		t.Fatal("NewContact returned nil")
	}

	if c.ID == uuid.Nil {
		t.Error("expected non-nil UUID, got uuid.Nil")
	}

	if c.Name != name {
		t.Errorf("expected Name %q, got %q", name, c.Name)
	}

	if c.Email != "" {
		t.Errorf("expected empty Email, got %q", c.Email)
	}

	if c.Phone != "" {
		t.Errorf("expected empty Phone, got %q", c.Phone)
	}

	if c.Fields == nil {
		t.Error("expected Fields to be initialized, got nil")
	}

	if len(c.Fields) != 0 {
		t.Errorf("expected empty Fields map, got %d entries", len(c.Fields))
	}

	if c.Tags == nil {
		t.Error("expected Tags to be initialized, got nil")
	}

	if len(c.Tags) != 0 {
		t.Errorf("expected empty Tags slice, got %d entries", len(c.Tags))
	}

	if c.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero time")
	}

	if c.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set, got zero time")
	}

	if c.CreatedAt != c.UpdatedAt {
		t.Error("expected CreatedAt and UpdatedAt to be equal on creation")
	}
}

func TestNewContact_UniqueIDs(t *testing.T) {
	c1 := NewContact("Alice")
	c2 := NewContact("Bob")

	if c1.ID == c2.ID {
		t.Error("expected different UUIDs for different contacts")
	}
}

func TestContact_Touch(t *testing.T) {
	c := NewContact("Alice")
	originalUpdated := c.UpdatedAt
	originalCreated := c.CreatedAt

	// Sleep briefly to ensure time advances
	time.Sleep(2 * time.Millisecond)
	c.Touch()

	if !c.UpdatedAt.After(originalUpdated) {
		t.Error("expected Touch() to advance UpdatedAt")
	}

	if c.CreatedAt != originalCreated {
		t.Error("Touch() should not modify CreatedAt")
	}
}
