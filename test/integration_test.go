// ABOUTME: End-to-end integration tests exercising the full CRM workflow.
// ABOUTME: Runs identical scenarios against both SQLite and Markdown storage backends.
package test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/harperreed/crm/internal/config"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
)

// workflowFixture holds entities created during setup so each sub-step can use them.
type workflowFixture struct {
	store   storage.Storage
	contact *models.Contact
	company *models.Company
	rel     *models.Relationship
}

// setupWorkflow creates a contact, company, and relationship for workflow tests.
func setupWorkflow(t *testing.T, store storage.Storage) *workflowFixture {
	t.Helper()

	contact := models.NewContact("Jane Doe")
	contact.Email = "jane@example.com"
	contact.Tags = []string{"vip", "engineering"}
	if err := store.CreateContact(contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	company := models.NewCompany("Acme Corp")
	company.Domain = "acme.com"
	if err := store.CreateCompany(company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	rel := models.NewRelationship(contact.ID, company.ID, "works_at", "engineer")
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}

	return &workflowFixture{store: store, contact: contact, company: company, rel: rel}
}

func verifyGetContact(t *testing.T, f *workflowFixture) {
	t.Helper()
	got, err := f.store.GetContact(f.contact.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if got.Name != "Jane Doe" {
		t.Errorf("contact name = %q, want %q", got.Name, "Jane Doe")
	}
	if got.Email != "jane@example.com" {
		t.Errorf("contact email = %q, want %q", got.Email, "jane@example.com")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "vip" || got.Tags[1] != "engineering" {
		t.Errorf("contact tags = %v, want [vip engineering]", got.Tags)
	}
}

func verifyRelationships(t *testing.T, f *workflowFixture) {
	t.Helper()
	rels, err := f.store.ListRelationships(f.contact.ID)
	if err != nil {
		t.Fatalf("ListRelationships: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("ListRelationships count = %d, want 1", len(rels))
	}
	if rels[0].Type != "works_at" {
		t.Errorf("relationship type = %q, want %q", rels[0].Type, "works_at")
	}
}

func verifySearch(t *testing.T, f *workflowFixture) {
	t.Helper()
	results, err := f.store.Search("acme")
	if err != nil {
		t.Fatalf("Search(acme): %v", err)
	}
	if len(results.Companies) != 1 {
		t.Errorf("search companies count = %d, want 1", len(results.Companies))
	}

	results, err = f.store.Search("jane")
	if err != nil {
		t.Fatalf("Search(jane): %v", err)
	}
	if len(results.Contacts) != 1 {
		t.Errorf("search contacts count = %d, want 1", len(results.Contacts))
	}
}

func verifyUpdateContact(t *testing.T, f *workflowFixture) {
	t.Helper()
	got, err := f.store.GetContact(f.contact.ID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	got.Name = "Jane Smith"
	got.Touch()
	if err := f.store.UpdateContact(got); err != nil {
		t.Fatalf("UpdateContact: %v", err)
	}

	updated, err := f.store.GetContact(f.contact.ID)
	if err != nil {
		t.Fatalf("GetContact after update: %v", err)
	}
	if updated.Name != "Jane Smith" {
		t.Errorf("updated name = %q, want %q", updated.Name, "Jane Smith")
	}
	if updated.Email != "jane@example.com" {
		t.Errorf("updated email = %q, want %q", updated.Email, "jane@example.com")
	}
}

func verifyDeletion(t *testing.T, f *workflowFixture) {
	t.Helper()
	if err := f.store.DeleteRelationship(f.rel.ID); err != nil {
		t.Fatalf("DeleteRelationship: %v", err)
	}
	remainingRels, err := f.store.ListRelationships(f.contact.ID)
	if err != nil {
		t.Fatalf("ListRelationships after delete: %v", err)
	}
	if len(remainingRels) != 0 {
		t.Errorf("remaining relationships = %d, want 0", len(remainingRels))
	}

	if err := f.store.DeleteContact(f.contact.ID); err != nil {
		t.Fatalf("DeleteContact: %v", err)
	}
	_, err = f.store.GetContact(f.contact.ID)
	if !errors.Is(err, storage.ErrContactNotFound) {
		t.Errorf("GetContact after delete: got %v, want ErrContactNotFound", err)
	}

	if err := f.store.DeleteCompany(f.company.ID); err != nil {
		t.Fatalf("DeleteCompany: %v", err)
	}
	_, err = f.store.GetCompany(f.company.ID)
	if !errors.Is(err, storage.ErrCompanyNotFound) {
		t.Errorf("GetCompany after delete: got %v, want ErrCompanyNotFound", err)
	}
}

func TestFullWorkflow(t *testing.T) {
	backends := []string{"sqlite", "markdown"}
	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			tmpDir := t.TempDir()
			cfg := &config.Config{Backend: backend, DataDir: tmpDir}

			store, err := cfg.OpenStorage()
			if err != nil {
				t.Fatalf("OpenStorage(%s): %v", backend, err)
			}
			defer func() { _ = store.Close() }()

			f := setupWorkflow(t, store)
			verifyGetContact(t, f)
			verifyRelationships(t, f)
			verifySearch(t, f)
			verifyUpdateContact(t, f)
			verifyDeletion(t, f)
		})
	}
}

func TestConfigOpenStorageBothBackends(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &config.Config{Backend: "sqlite", DataDir: dir}

		store, err := cfg.OpenStorage()
		if err != nil {
			t.Fatalf("OpenStorage(sqlite): %v", err)
		}
		defer func() { _ = store.Close() }()

		// Verify crm.db file was created
		dbPath := filepath.Join(dir, "crm.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("expected database file at %q", dbPath)
		}
	})

	t.Run("markdown", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &config.Config{Backend: "markdown", DataDir: dir}

		store, err := cfg.OpenStorage()
		if err != nil {
			t.Fatalf("OpenStorage(markdown): %v", err)
		}
		defer func() { _ = store.Close() }()

		// Verify contacts/ and companies/ directories were created
		contactsDir := filepath.Join(dir, "contacts")
		if _, err := os.Stat(contactsDir); os.IsNotExist(err) {
			t.Errorf("expected contacts directory at %q", contactsDir)
		}

		companiesDir := filepath.Join(dir, "companies")
		if _, err := os.Stat(companiesDir); os.IsNotExist(err) {
			t.Errorf("expected companies directory at %q", companiesDir)
		}
	})
}
