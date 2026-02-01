// ABOUTME: Tests for export CLI commands
// ABOUTME: Validates YAML and Markdown export for contacts, companies, and deals
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestExportContactsCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test contacts
	_ = db.CreateContact(&repository.Contact{Name: "Alice", Email: "alice@example.com", Notes: "Test notes"})
	_ = db.CreateContact(&repository.Contact{Name: "Bob", Email: "bob@example.com", Phone: "555-1234"})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "export as yaml",
			args:    []string{"--format", "yaml"},
			wantErr: false,
		},
		{
			name:    "export as markdown",
			args:    []string{"--format", "markdown"},
			wantErr: false,
		},
		{
			name:    "default format",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "unknown format",
			args:    []string{"--format", "json"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := ExportContactsCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportContactsCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ExportContactsCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExportContactsCommandToFile(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	_ = db.CreateContact(&repository.Contact{Name: "Alice"})

	// Create temp file
	tmpFile, err := os.CreateTemp("", "export-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportContactsCommand(db, []string{"--output", tmpPath})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportContactsCommand() unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestExportCompaniesCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test companies
	_ = db.CreateCompany(&repository.Company{Name: "Acme Corp", Domain: "acme.com", Industry: "Tech", Notes: "Test"})
	_ = db.CreateCompany(&repository.Company{Name: "Beta Inc", Domain: "beta.io"})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "export as yaml",
			args:    []string{"--format", "yaml"},
			wantErr: false,
		},
		{
			name:    "export as markdown",
			args:    []string{"--format", "markdown"},
			wantErr: false,
		},
		{
			name:    "unknown format",
			args:    []string{"--format", "csv"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := ExportCompaniesCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportCompaniesCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ExportCompaniesCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExportDealsCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Acme Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&repository.Deal{
		Title:       "Big Deal",
		CompanyID:   company.ID,
		CompanyName: company.Name,
		Stage:       "prospecting",
		Amount:      100000,
		Currency:    "USD",
	})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "export as yaml",
			args:    []string{"--format", "yaml"},
			wantErr: false,
		},
		{
			name:    "export as markdown",
			args:    []string{"--format", "markdown"},
			wantErr: false,
		},
		{
			name:    "unknown format",
			args:    []string{"--format", "xml"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := ExportDealsCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportDealsCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ExportDealsCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExportAllCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Acme Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateContact(&repository.Contact{Name: "Alice", CompanyID: &company.ID})
	_ = db.CreateDeal(&repository.Deal{Title: "Deal", CompanyID: company.ID, Stage: "prospecting"})

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "export-all-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportAllCommand(db, []string{"--output-dir", tmpDir})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportAllCommand() unexpected error: %v", err)
	}

	// Verify files were created
	for _, file := range []string{"contacts.yaml", "companies.yaml", "deals.yaml"} {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
}

func TestExportAllCommandUnsupportedFormat(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportAllCommand(db, []string{"--format", "json"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Error("ExportAllCommand() expected error for unsupported format")
	}
}
