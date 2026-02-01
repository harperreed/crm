// ABOUTME: Tests for contact CLI commands
// ABOUTME: Validates add, list, update, and delete contact operations
package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestAddContactCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid contact with all fields",
			args:    []string{"--name", "John Doe", "--email", "john@example.com", "--phone", "555-1234", "--notes", "Test contact"},
			wantErr: false,
		},
		{
			name:    "valid contact with minimal fields",
			args:    []string{"--name", "Jane Smith"},
			wantErr: false,
		},
		{
			name:        "missing required name",
			args:        []string{"--email", "test@example.com"},
			wantErr:     true,
			errContains: "--name is required",
		},
		{
			name:    "contact with company creates new company",
			args:    []string{"--name", "Bob Jones", "--company", "New Corp"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := AddContactCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddContactCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("AddContactCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddContactCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAddContactWithExistingCompany(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create existing company
	company := &repository.Company{Name: "Existing Corp"}
	if err := db.CreateCompany(company); err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = AddContactCommand(db, []string{"--name", "Test Contact", "--company", "Existing Corp"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("AddContactCommand() unexpected error: %v", err)
	}

	// Verify contact was created with correct company
	contacts, err := db.ListContacts(&repository.ContactFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].CompanyID == nil || *contacts[0].CompanyID != company.ID {
		t.Errorf("Contact should be linked to existing company")
	}
}

func TestListContactsCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test contacts
	_ = db.CreateContact(&repository.Contact{Name: "Alice Smith", Email: "alice@example.com"})
	_ = db.CreateContact(&repository.Contact{Name: "Bob Jones", Email: "bob@test.com"})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list all contacts",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "list with query",
			args:    []string{"--query", "alice"},
			wantErr: false,
		},
		{
			name:    "list with limit",
			args:    []string{"--limit", "1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := ListContactsCommand(db, tt.args)

			_ = w.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListContactsCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ListContactsCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListContactsWithCompanyFilter(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company and contact
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)
	_ = db.CreateContact(&repository.Contact{Name: "Company Contact", CompanyID: &company.ID, CompanyName: company.Name})
	_ = db.CreateContact(&repository.Contact{Name: "Other Contact"})

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ListContactsCommand(db, []string{"--company", "Test Corp"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListContactsCommand() unexpected error: %v", err)
	}
}

func TestListContactsNoResults(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ListContactsCommand(db, []string{"--query", "nonexistent"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListContactsCommand() unexpected error: %v", err)
	}
}

func TestUpdateContactCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contact to update
	contact := &repository.Contact{Name: "Original Name", Email: "original@example.com"}
	_ = db.CreateContact(contact)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "update name",
			args:    []string{"--name", "Updated Name", contact.ID.String()},
			wantErr: false,
		},
		{
			name:    "update email and phone",
			args:    []string{"--email", "new@example.com", "--phone", "555-9999", contact.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing contact ID",
			args:        []string{"--name", "Test"},
			wantErr:     true,
			errContains: "contact ID is required",
		},
		{
			name:        "invalid contact ID",
			args:        []string{"--name", "Test", "invalid-id"},
			wantErr:     true,
			errContains: "invalid contact ID",
		},
		{
			name:        "nonexistent contact",
			args:        []string{"--name", "Test", "00000000-0000-0000-0000-000000000000"},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := UpdateContactCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateContactCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateContactCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateContactCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUpdateContactWithCompany(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company
	company := &repository.Company{Name: "New Company"}
	_ = db.CreateCompany(company)

	// Create contact
	contact := &repository.Contact{Name: "Test Contact"}
	_ = db.CreateContact(contact)

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Note: flags must come BEFORE the positional argument (contact ID)
	err = UpdateContactCommand(db, []string{"--company", "New Company", contact.ID.String()})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("UpdateContactCommand() unexpected error: %v", err)
	}

	// Verify contact was updated
	updated, _ := db.GetContact(contact.ID)
	if updated.CompanyID == nil || *updated.CompanyID != company.ID {
		t.Errorf("Contact should be linked to company")
	}
}

func TestUpdateContactWithNonexistentCompany(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contact
	contact := &repository.Contact{Name: "Test Contact"}
	_ = db.CreateContact(contact)

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Note: flags must come BEFORE the positional argument (contact ID)
	err = UpdateContactCommand(db, []string{"--company", "Nonexistent Corp", contact.ID.String()})

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Errorf("UpdateContactCommand() expected error for nonexistent company")
	}
}

func TestDeleteContactCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contact to delete
	contact := &repository.Contact{Name: "To Delete"}
	_ = db.CreateContact(contact)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing contact",
			args:    []string{contact.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing contact ID",
			args:        []string{},
			wantErr:     true,
			errContains: "contact ID is required",
		},
		{
			name:        "invalid contact ID",
			args:        []string{"invalid-id"},
			wantErr:     true,
			errContains: "invalid contact ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := DeleteContactCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteContactCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteContactCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteContactCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteNonexistentContact(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = DeleteContactCommand(db, []string{"00000000-0000-0000-0000-000000000000"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Errorf("DeleteContactCommand() expected error for nonexistent contact")
	}
}

// Note: contains helper function is defined in sync_test.go
