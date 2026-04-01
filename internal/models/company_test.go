// ABOUTME: Tests for the Company model struct and its constructor/methods.
// ABOUTME: Verifies ID generation, field initialization, timestamps, and Touch behavior.
package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewCompany(t *testing.T) {
	name := "Acme Corp"
	c := NewCompany(name)

	if c == nil {
		t.Fatal("NewCompany returned nil")
	}

	if c.ID == uuid.Nil {
		t.Error("expected non-nil UUID, got uuid.Nil")
	}

	if c.Name != name {
		t.Errorf("expected Name %q, got %q", name, c.Name)
	}

	if c.Domain != "" {
		t.Errorf("expected empty Domain, got %q", c.Domain)
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

func TestNewCompany_UniqueIDs(t *testing.T) {
	c1 := NewCompany("Acme")
	c2 := NewCompany("Globex")

	if c1.ID == c2.ID {
		t.Error("expected different UUIDs for different companies")
	}
}

func TestCompany_Touch(t *testing.T) {
	c := NewCompany("Acme")
	original := c.UpdatedAt

	time.Sleep(2 * time.Millisecond)
	c.Touch()

	if !c.UpdatedAt.After(original) {
		t.Error("expected Touch() to advance UpdatedAt")
	}
}
