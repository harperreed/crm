// ABOUTME: Tests for company CLI commands
// ABOUTME: Validates add, list, update, and delete company operations
package cli

import (
	"os"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestAddCompanyCommand(t *testing.T) {
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
			name:    "valid company with all fields",
			args:    []string{"--name", "Acme Corp", "--domain", "acme.com", "--industry", "Technology", "--notes", "Test company"},
			wantErr: false,
		},
		{
			name:    "valid company with minimal fields",
			args:    []string{"--name", "Simple Inc"},
			wantErr: false,
		},
		{
			name:        "missing required name",
			args:        []string{"--domain", "test.com"},
			wantErr:     true,
			errContains: "--name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := AddCompanyCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddCompanyCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("AddCompanyCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddCompanyCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListCompaniesCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test companies
	_ = db.CreateCompany(&repository.Company{Name: "Acme Corp", Domain: "acme.com", Industry: "Tech"})
	_ = db.CreateCompany(&repository.Company{Name: "Beta Inc", Domain: "beta.io", Industry: "Finance"})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list all companies",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "list with query",
			args:    []string{"--query", "acme"},
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

			err := ListCompaniesCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListCompaniesCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ListCompaniesCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListCompaniesNoResults(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ListCompaniesCommand(db, []string{"--query", "nonexistent"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListCompaniesCommand() unexpected error: %v", err)
	}
}

func TestUpdateCompanyCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company to update
	company := &repository.Company{Name: "Original Corp", Domain: "original.com"}
	_ = db.CreateCompany(company)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "update name",
			args:    []string{"--name", "Updated Corp", company.ID.String()},
			wantErr: false,
		},
		{
			name:    "update domain and industry",
			args:    []string{"--domain", "new.com", "--industry", "Healthcare", company.ID.String()},
			wantErr: false,
		},
		{
			name:    "update notes",
			args:    []string{"--notes", "Updated notes", company.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing company ID",
			args:        []string{"--name", "Test"},
			wantErr:     true,
			errContains: "company ID is required",
		},
		{
			name:        "invalid company ID",
			args:        []string{"--name", "Test", "invalid-id"},
			wantErr:     true,
			errContains: "invalid company ID",
		},
		{
			name:        "nonexistent company",
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

			err := UpdateCompanyCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateCompanyCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateCompanyCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateCompanyCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteCompanyCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company to delete
	company := &repository.Company{Name: "To Delete"}
	_ = db.CreateCompany(company)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing company",
			args:    []string{company.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing company ID",
			args:        []string{},
			wantErr:     true,
			errContains: "company ID is required",
		},
		{
			name:        "invalid company ID",
			args:        []string{"invalid-id"},
			wantErr:     true,
			errContains: "invalid company ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := DeleteCompanyCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteCompanyCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteCompanyCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteCompanyCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteNonexistentCompany(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = DeleteCompanyCommand(db, []string{"00000000-0000-0000-0000-000000000000"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Errorf("DeleteCompanyCommand() expected error for nonexistent company")
	}
}
