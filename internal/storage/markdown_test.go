// ABOUTME: Tests for the markdown file-based storage backend.
// ABOUTME: Covers contact and company CRUD, relationships, search, prefix lookup, and filters.
package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

// newTestMarkdownStore creates a MarkdownStore in a temp directory.
func newTestMarkdownStore(t *testing.T) *MarkdownStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := NewMarkdownStore(tmpDir)
	if err != nil {
		t.Fatalf("NewMarkdownStore(%q): %v", tmpDir, err)
	}
	return store
}

func TestMarkdownNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "sub", "data")
	store, err := NewMarkdownStore(dataDir)
	if err != nil {
		t.Fatalf("NewMarkdownStore: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	// Verify subdirectories were created
	for _, sub := range []string{"contacts", "companies"} {
		path := filepath.Join(dataDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected %s dir to exist: %v", sub, err)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}
}

func TestMarkdownClose(t *testing.T) {
	store := newTestMarkdownStore(t)
	if err := store.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestMarkdownCreateAndGetContact(t *testing.T) {
	store := newTestMarkdownStore(t)

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

	title, ok := got.Fields["title"]
	if !ok || title != "Engineer" {
		t.Errorf("Fields[title] = %v, want %q", title, "Engineer")
	}

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

func TestMarkdownGetContactNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	_, err := store.GetContact(uuid.New())
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestMarkdownDeleteContact(t *testing.T) {
	store := newTestMarkdownStore(t)

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

func TestMarkdownDeleteContactNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	if err := store.DeleteContact(uuid.New()); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestMarkdownCreateAndGetCompany(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewCompany("Acme Corp")
	c.Domain = "acme.com"
	c.Fields = map[string]any{"industry": "tech", "size": float64(500)}
	c.Tags = []string{"enterprise", "partner"}

	if err := store.CreateCompany(c); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	got, err := store.GetCompany(c.ID)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}

	if got.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", got.Name, "Acme Corp")
	}
	if got.Domain != "acme.com" {
		t.Errorf("Domain = %q, want %q", got.Domain, "acme.com")
	}

	industry, ok := got.Fields["industry"]
	if !ok || industry != "tech" {
		t.Errorf("Fields[industry] = %v, want %q", industry, "tech")
	}

	if len(got.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(got.Tags))
	}
	if got.Tags[0] != "enterprise" || got.Tags[1] != "partner" {
		t.Errorf("Tags = %v, want [enterprise partner]", got.Tags)
	}
}

func TestMarkdownGetCompanyNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	_, err := store.GetCompany(uuid.New())
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestMarkdownDeleteCompany(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewCompany("DeleteMe Corp")
	if err := store.CreateCompany(c); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	if err := store.DeleteCompany(c.ID); err != nil {
		t.Fatalf("DeleteCompany: %v", err)
	}

	_, err := store.GetCompany(c.ID)
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound after delete, got %v", err)
	}
}

func TestMarkdownDeleteCompanyNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	if err := store.DeleteCompany(uuid.New()); !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestMarkdownRelationships(t *testing.T) {
	store := newTestMarkdownStore(t)

	srcID := uuid.New()
	tgtID := uuid.New()
	rel := models.NewRelationship(srcID, tgtID, "works_at", "engineering team")

	// Create
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}

	// List by source
	rels, err := store.ListRelationships(srcID)
	if err != nil {
		t.Fatalf("ListRelationships(src): %v", err)
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

	// List by target
	rels, err = store.ListRelationships(tgtID)
	if err != nil {
		t.Fatalf("ListRelationships(tgt): %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("len = %d, want 1", len(rels))
	}

	// Delete
	if err := store.DeleteRelationship(rel.ID); err != nil {
		t.Fatalf("DeleteRelationship: %v", err)
	}

	rels, err = store.ListRelationships(srcID)
	if err != nil {
		t.Fatalf("ListRelationships after delete: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(rels))
	}
}

func TestMarkdownDeleteRelationshipNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	if err := store.DeleteRelationship(uuid.New()); !errors.Is(err, ErrRelationshipNotFound) {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}
}

func TestMarkdownRelationshipsBidirectional(t *testing.T) {
	store := newTestMarkdownStore(t)

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

func TestMarkdownSearch(t *testing.T) {
	store := newTestMarkdownStore(t)

	contact := models.NewContact("Quantum Findable")
	contact.Email = "quantum@findable.com"
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	company := models.NewCompany("Quantum Enterprises")
	company.Domain = "quantum.com"
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	other := models.NewContact("Invisible Person")
	if err := store.CreateContact(other); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	results, err := store.Search("Quantum")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Contacts) != 1 {
		t.Errorf("Contacts len = %d, want 1", len(results.Contacts))
	}
	if len(results.Companies) != 1 {
		t.Errorf("Companies len = %d, want 1", len(results.Companies))
	}

	if len(results.Contacts) == 1 && results.Contacts[0].Name != "Quantum Findable" {
		t.Errorf("Contact name = %q, want %q", results.Contacts[0].Name, "Quantum Findable")
	}
	if len(results.Companies) == 1 && results.Companies[0].Name != "Quantum Enterprises" {
		t.Errorf("Company name = %q, want %q", results.Companies[0].Name, "Quantum Enterprises")
	}
}

func TestMarkdownSearchNoResults(t *testing.T) {
	store := newTestMarkdownStore(t)

	results, err := store.Search("nonexistent")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Contacts) != 0 {
		t.Errorf("Contacts len = %d, want 0", len(results.Contacts))
	}
	if len(results.Companies) != 0 {
		t.Errorf("Companies len = %d, want 0", len(results.Companies))
	}
}

func TestMarkdownGetContactByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)

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

func TestMarkdownGetContactByPrefixTooShort(t *testing.T) {
	store := newTestMarkdownStore(t)

	_, err := store.GetContactByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestMarkdownGetContactByPrefixNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	_, err := store.GetContactByPrefix("000000")
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestMarkdownGetCompanyByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewCompany("Globex Corp")
	if err := store.CreateCompany(c); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	prefix := c.ID.String()[:8]
	got, err := store.GetCompanyByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetCompanyByPrefix(%q): %v", prefix, err)
	}
	if got.ID != c.ID {
		t.Errorf("ID = %s, want %s", got.ID, c.ID)
	}
}

func TestMarkdownGetCompanyByPrefixTooShort(t *testing.T) {
	store := newTestMarkdownStore(t)

	_, err := store.GetCompanyByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestMarkdownGetCompanyByPrefixNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	_, err := store.GetCompanyByPrefix("000000")
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestMarkdownUpdateContact(t *testing.T) {
	store := newTestMarkdownStore(t)

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

func TestMarkdownUpdateContactNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewContact("Ghost")
	if err := store.UpdateContact(c); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestMarkdownUpdateCompany(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewCompany("OldName Inc")
	if err := store.CreateCompany(c); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	c.Name = "NewName Inc"
	c.Domain = "newname.com"
	c.Tags = []string{"renamed"}
	c.Touch()

	if err := store.UpdateCompany(c); err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}

	got, err := store.GetCompany(c.ID)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if got.Name != "NewName Inc" {
		t.Errorf("Name = %q, want %q", got.Name, "NewName Inc")
	}
	if got.Domain != "newname.com" {
		t.Errorf("Domain = %q, want %q", got.Domain, "newname.com")
	}
}

func TestMarkdownUpdateCompanyNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewCompany("Ghost Corp")
	if err := store.UpdateCompany(c); !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestMarkdownListContacts(t *testing.T) {
	store := newTestMarkdownStore(t)

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

func TestMarkdownListContactsSearch(t *testing.T) {
	store := newTestMarkdownStore(t)

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

func TestMarkdownListCompanies(t *testing.T) {
	store := newTestMarkdownStore(t)

	c1 := models.NewCompany("Alpha Corp")
	c1.Tags = []string{"saas"}
	c2 := models.NewCompany("Beta Corp")
	c2.Tags = []string{"hardware"}
	c3 := models.NewCompany("Gamma Corp")
	c3.Tags = []string{"saas"}

	for _, c := range []*models.Company{c1, c2, c3} {
		if err := store.CreateCompany(c); err != nil {
			t.Fatalf("CreateCompany(%s): %v", c.Name, err)
		}
	}

	// List all
	all, err := store.ListCompanies(nil)
	if err != nil {
		t.Fatalf("ListCompanies(nil): %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListCompanies(nil) len = %d, want 3", len(all))
	}

	// Filter by tag
	tag := "saas"
	filtered, err := store.ListCompanies(&CompanyFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("ListCompanies(tag=saas): %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListCompanies(tag=saas) len = %d, want 2", len(filtered))
	}

	// Limit
	limited, err := store.ListCompanies(&CompanyFilter{Limit: 1})
	if err != nil {
		t.Fatalf("ListCompanies(limit=1): %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("ListCompanies(limit=1) len = %d, want 1", len(limited))
	}
}

func TestMarkdownListCompaniesSearch(t *testing.T) {
	store := newTestMarkdownStore(t)

	c1 := models.NewCompany("Searchable Industries")
	c1.Domain = "searchable.io"
	c2 := models.NewCompany("Hidden Corp")

	for _, c := range []*models.Company{c1, c2} {
		if err := store.CreateCompany(c); err != nil {
			t.Fatalf("CreateCompany(%s): %v", c.Name, err)
		}
	}

	results, err := store.ListCompanies(&CompanyFilter{Search: "Searchable"})
	if err != nil {
		t.Fatalf("ListCompanies(search=Searchable): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("search len = %d, want 1", len(results))
	}
	if results[0].Name != "Searchable Industries" {
		t.Errorf("Name = %q, want %q", results[0].Name, "Searchable Industries")
	}
}

func TestMarkdownContactFieldsRoundTrip(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewContact("Fields Test")
	c.Fields = map[string]any{
		"string_val": "hello",
		"num_val":    float64(42),
		"bool_val":   true,
	}

	if err := store.CreateContact(c); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	got, err := store.GetContact(c.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}

	if got.Fields["string_val"] != "hello" {
		t.Errorf("Fields[string_val] = %v, want %q", got.Fields["string_val"], "hello")
	}
	if got.Fields["bool_val"] != true {
		t.Errorf("Fields[bool_val] = %v, want true", got.Fields["bool_val"])
	}
}

func TestMarkdownCompanyFieldsRoundTrip(t *testing.T) {
	store := newTestMarkdownStore(t)

	c := models.NewCompany("Fields Corp")
	c.Fields = map[string]any{
		"industry": "tech",
		"size":     float64(100),
	}

	if err := store.CreateCompany(c); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	got, err := store.GetCompany(c.ID)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}

	if got.Fields["industry"] != "tech" {
		t.Errorf("Fields[industry] = %v, want %q", got.Fields["industry"], "tech")
	}
}

func TestMarkdownInterfaceCompliance(t *testing.T) {
	// Verify MarkdownStore implements Storage at runtime
	store := newTestMarkdownStore(t)
	var _ Storage = store
}
