// ABOUTME: Tests for SQLite company CRUD operations including FTS5 search.
// ABOUTME: Covers create, get, get-by-prefix, list with filters, update, and delete.
package storage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestCreateCompany(t *testing.T) {
	store := newTestStore(t)

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

func TestGetCompanyNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetCompany(uuid.New())
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestGetCompanyByPrefix(t *testing.T) {
	store := newTestStore(t)

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

func TestGetCompanyByPrefixTooShort(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetCompanyByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestUpdateCompany(t *testing.T) {
	store := newTestStore(t)

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

func TestUpdateCompanyNotFound(t *testing.T) {
	store := newTestStore(t)

	c := models.NewCompany("Ghost Corp")
	if err := store.UpdateCompany(c); !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestDeleteCompany(t *testing.T) {
	store := newTestStore(t)

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

func TestDeleteCompanyNotFound(t *testing.T) {
	store := newTestStore(t)

	if err := store.DeleteCompany(uuid.New()); !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestListCompanies(t *testing.T) {
	store := newTestStore(t)

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

func TestListCompaniesSearch(t *testing.T) {
	store := newTestStore(t)

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
