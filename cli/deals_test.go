// ABOUTME: Tests for deal CLI commands
// ABOUTME: Validates add, list, and delete deal operations
package cli

import (
	"os"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestAddDealCommand(t *testing.T) {
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
			name:    "valid deal with all fields",
			args:    []string{"--title", "Big Sale", "--company", "Acme Corp", "--amount", "100000", "--currency", "USD", "--stage", "prospecting", "--notes", "Hot lead"},
			wantErr: false,
		},
		{
			name:    "valid deal with minimal fields",
			args:    []string{"--title", "Simple Deal", "--company", "Beta Inc"},
			wantErr: false,
		},
		{
			name:        "missing required title",
			args:        []string{"--company", "Test Corp"},
			wantErr:     true,
			errContains: "--title is required",
		},
		{
			name:        "missing required company",
			args:        []string{"--title", "Test Deal"},
			wantErr:     true,
			errContains: "--company is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := AddDealCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddDealCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("AddDealCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddDealCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAddDealWithExistingCompany(t *testing.T) {
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

	err = AddDealCommand(db, []string{"--title", "Test Deal", "--company", "Existing Corp", "--amount", "50000"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("AddDealCommand() unexpected error: %v", err)
	}

	// Verify deal was created with correct company
	deals, err := db.ListDeals(&repository.DealFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list deals: %v", err)
	}
	if len(deals) != 1 {
		t.Fatalf("Expected 1 deal, got %d", len(deals))
	}
	if deals[0].CompanyID != company.ID {
		t.Errorf("Deal should be linked to existing company")
	}
}

func TestAddDealWithContact(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	// Create contact
	contact := &repository.Contact{Name: "John Doe", CompanyID: &company.ID, CompanyName: company.Name}
	_ = db.CreateContact(contact)

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = AddDealCommand(db, []string{"--title", "Deal With Contact", "--company", "Test Corp", "--contact", "John Doe"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("AddDealCommand() unexpected error: %v", err)
	}

	// Verify deal was created with contact
	deals, err := db.ListDeals(&repository.DealFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list deals: %v", err)
	}
	if len(deals) != 1 {
		t.Fatalf("Expected 1 deal, got %d", len(deals))
	}
	if deals[0].ContactID == nil || *deals[0].ContactID != contact.ID {
		t.Errorf("Deal should be linked to contact")
	}
}

func TestAddDealCreatesNewContact(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = AddDealCommand(db, []string{"--title", "Deal With New Contact", "--company", "Test Corp", "--contact", "New Person"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("AddDealCommand() unexpected error: %v", err)
	}

	// Verify contact was created
	contacts, err := db.ListContacts(&repository.ContactFilter{Query: "New Person", Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact to be created, got %d", len(contacts))
	}
}

func TestListDealsCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test company and deals
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)
	_ = db.CreateDeal(&repository.Deal{Title: "Deal 1", CompanyID: company.ID, CompanyName: company.Name, Amount: 100000, Currency: "USD", Stage: "prospecting"})
	_ = db.CreateDeal(&repository.Deal{Title: "Deal 2", CompanyID: company.ID, CompanyName: company.Name, Amount: 200000, Currency: "USD", Stage: "qualification"})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list all deals",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "list with stage filter",
			args:    []string{"--stage", "prospecting"},
			wantErr: false,
		},
		{
			name:    "list with company filter",
			args:    []string{"--company", "Test Corp"},
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
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := ListDealsCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListDealsCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ListDealsCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListDealsNoResults(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ListDealsCommand(db, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListDealsCommand() unexpected error: %v", err)
	}
}

func TestDeleteDealCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company and deal to delete
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)
	deal := &repository.Deal{Title: "To Delete", CompanyID: company.ID, CompanyName: company.Name, Amount: 50000, Currency: "USD", Stage: "prospecting"}
	_ = db.CreateDeal(deal)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing deal",
			args:    []string{deal.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing deal ID",
			args:        []string{},
			wantErr:     true,
			errContains: "usage",
		},
		{
			name:        "invalid deal ID",
			args:        []string{"invalid-id"},
			wantErr:     true,
			errContains: "invalid deal ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := DeleteDealCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteDealCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteDealCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteDealCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteNonexistentDeal(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = DeleteDealCommand(db, []string{"00000000-0000-0000-0000-000000000000"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Errorf("DeleteDealCommand() expected error for nonexistent deal")
	}
}
