// ABOUTME: Tests for SQLite contact CRUD operations including FTS5 search.
// ABOUTME: Covers create, get, get-by-prefix, list with filters, update, and delete.
package storage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestCreateContact(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Alice Smith")
	c.Email = "alice@example.com"
	c.Phone = "+1-555-0100"
	c.Fields = map[string]any{"title": "Engineer", "level": float64(5)}
	c.Tags = []string{"vip", "engineering"}

	if err := store.CreateContact(c); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	got, err := store.GetContact(c.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}

	if got.Name != "Alice Smith" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice Smith")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}
	if got.Phone != "+1-555-0100" {
		t.Errorf("Phone = %q, want %q", got.Phone, "+1-555-0100")
	}

	// Verify fields
	title, ok := got.Fields["title"]
	if !ok || title != "Engineer" {
		t.Errorf("Fields[title] = %v, want %q", title, "Engineer")
	}
	level, ok := got.Fields["level"]
	if !ok || level != float64(5) {
		t.Errorf("Fields[level] = %v, want %v", level, float64(5))
	}

	// Verify tags
	if len(got.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(got.Tags))
	}
	if got.Tags[0] != "vip" || got.Tags[1] != "engineering" {
		t.Errorf("Tags = %v, want [vip engineering]", got.Tags)
	}

	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestGetContactNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetContact(uuid.New())
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestGetContactByPrefix(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Bob Jones")
	if err := store.CreateContact(c); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	prefix := c.ID.String()[:8]
	got, err := store.GetContactByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetContactByPrefix(%q): %v", prefix, err)
	}
	if got.ID != c.ID {
		t.Errorf("ID = %s, want %s", got.ID, c.ID)
	}
}

func TestGetContactByPrefixTooShort(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetContactByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestUpdateContact(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Carol White")
	if err := store.CreateContact(c); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	c.Name = "Carol Black"
	c.Email = "carol@new.com"
	c.Tags = []string{"updated"}
	c.Touch()

	if err := store.UpdateContact(c); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}

	got, err := store.GetContact(c.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if got.Name != "Carol Black" {
		t.Errorf("Name = %q, want %q", got.Name, "Carol Black")
	}
	if got.Email != "carol@new.com" {
		t.Errorf("Email = %q, want %q", got.Email, "carol@new.com")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "updated" {
		t.Errorf("Tags = %v, want [updated]", got.Tags)
	}
}

func TestUpdateContactNotFound(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Ghost")
	if err := store.UpdateContact(c); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestDeleteContact(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Dave Delete")
	if err := store.CreateContact(c); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	if err := store.DeleteContact(c.ID); err != nil {
		t.Fatalf("DeleteContact: %v", err)
	}

	_, err := store.GetContact(c.ID)
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound after delete, got %v", err)
	}
}

func TestDeleteContactNotFound(t *testing.T) {
	store := newTestStore(t)

	if err := store.DeleteContact(uuid.New()); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestListContacts(t *testing.T) {
	store := newTestStore(t)

	c1 := models.NewContact("Eve Alpha")
	c1.Tags = []string{"team-a"}
	c2 := models.NewContact("Frank Beta")
	c2.Tags = []string{"team-b"}
	c3 := models.NewContact("Grace Alpha")
	c3.Tags = []string{"team-a"}

	for _, c := range []*models.Contact{c1, c2, c3} {
		if err := store.CreateContact(c); err != nil {
			t.Fatalf("CreateContact(%s): %v", c.Name, err)
		}
	}

	// List all
	all, err := store.ListContacts(nil)
	if err != nil {
		t.Fatalf("ListContacts(nil): %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListContacts(nil) len = %d, want 3", len(all))
	}

	// Filter by tag
	tag := "team-a"
	filtered, err := store.ListContacts(&ContactFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("ListContacts(tag=team-a): %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListContacts(tag=team-a) len = %d, want 2", len(filtered))
	}

	// Limit
	limited, err := store.ListContacts(&ContactFilter{Limit: 1})
	if err != nil {
		t.Fatalf("ListContacts(limit=1): %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("ListContacts(limit=1) len = %d, want 1", len(limited))
	}
}

func TestListContactsSearch(t *testing.T) {
	store := newTestStore(t)

	c1 := models.NewContact("Heidi Searchable")
	c1.Email = "heidi@searchtest.com"
	c2 := models.NewContact("Ivan Normal")

	for _, c := range []*models.Contact{c1, c2} {
		if err := store.CreateContact(c); err != nil {
			t.Fatalf("CreateContact(%s): %v", c.Name, err)
		}
	}

	results, err := store.ListContacts(&ContactFilter{Search: "Searchable"})
	if err != nil {
		t.Fatalf("ListContacts(search=Searchable): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("search len = %d, want 1", len(results))
	}
	if results[0].Name != "Heidi Searchable" {
		t.Errorf("Name = %q, want %q", results[0].Name, "Heidi Searchable")
	}
}
