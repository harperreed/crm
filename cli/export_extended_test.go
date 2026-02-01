// ABOUTME: Extended tests for export CLI commands
// ABOUTME: Covers edge cases for YAML and Markdown export functions
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harperreed/pagen/repository"
)

func TestExportContactsMarkdownWithAllFields(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company for contact
	company := &repository.Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	// Create contact with all fields
	now := time.Now()
	_ = db.CreateContact(&repository.Contact{
		Name:            "Full Contact",
		Email:           "full@example.com",
		Phone:           "555-1234",
		CompanyID:       &company.ID,
		CompanyName:     company.Name,
		LastContactedAt: &now,
		Notes:           "These are detailed notes about the contact.",
	})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportContactsCommand(db, []string{"--format", "markdown"})

	_ = w.Close()
	var buf strings.Builder
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportContactsCommand() unexpected error: %v", err)
	}

	output := buf.String()
	// Verify all fields are present in markdown
	if !strings.Contains(output, "Full Contact") {
		t.Error("Markdown should contain contact name")
	}
	if !strings.Contains(output, "full@example.com") {
		t.Error("Markdown should contain email")
	}
	if !strings.Contains(output, "555-1234") {
		t.Error("Markdown should contain phone")
	}
	if !strings.Contains(output, "Test Corp") {
		t.Error("Markdown should contain company")
	}
	if !strings.Contains(output, "Notes") {
		t.Error("Markdown should contain notes section")
	}
}

func TestExportCompaniesMarkdownWithAllFields(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company with all fields
	_ = db.CreateCompany(&repository.Company{
		Name:     "Full Company",
		Domain:   "fullcompany.com",
		Industry: "Technology",
		Notes:    "Company notes here",
	})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportCompaniesCommand(db, []string{"--format", "markdown"})

	_ = w.Close()
	var buf strings.Builder
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportCompaniesCommand() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Full Company") {
		t.Error("Markdown should contain company name")
	}
	if !strings.Contains(output, "fullcompany.com") {
		t.Error("Markdown should contain domain")
	}
	if !strings.Contains(output, "Technology") {
		t.Error("Markdown should contain industry")
	}
}

func TestExportDealsMarkdownWithAllFields(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create company and contact for deal
	company := &repository.Company{Name: "Deal Corp"}
	_ = db.CreateCompany(company)

	contact := &repository.Contact{Name: "Deal Contact"}
	_ = db.CreateContact(contact)

	// Create deal with all fields
	closeDate := time.Now().Add(30 * 24 * time.Hour)
	_ = db.CreateDeal(&repository.Deal{
		Title:             "Full Deal",
		CompanyID:         company.ID,
		CompanyName:       company.Name,
		ContactID:         &contact.ID,
		ContactName:       contact.Name,
		Stage:             "negotiation",
		Amount:            500000,
		Currency:          "USD",
		ExpectedCloseDate: &closeDate,
	})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportDealsCommand(db, []string{"--format", "markdown"})

	_ = w.Close()
	var buf strings.Builder
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportDealsCommand() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Full Deal") {
		t.Error("Markdown should contain deal title")
	}
	if !strings.Contains(output, "negotiation") {
		t.Error("Markdown should contain stage")
	}
	if !strings.Contains(output, "Deal Corp") {
		t.Error("Markdown should contain company name")
	}
	if !strings.Contains(output, "Deal Contact") {
		t.Error("Markdown should contain contact name")
	}
	if !strings.Contains(output, "USD") {
		t.Error("Markdown should contain currency")
	}
}

func TestExportCompaniesCommandToFile(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	_ = db.CreateCompany(&repository.Company{Name: "File Export Co"})

	// Create temp file
	tmpFile, err := os.CreateTemp("", "export-companies-*.yaml")
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

	err = ExportCompaniesCommand(db, []string{"--output", tmpPath})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportCompaniesCommand() unexpected error: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "File Export Co") {
		t.Error("Output file should contain company name")
	}
}

func TestExportDealsCommandToFile(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	company := &repository.Company{Name: "Deal Export Co"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&repository.Deal{
		Title:       "File Export Deal",
		CompanyID:   company.ID,
		CompanyName: company.Name,
		Stage:       "prospecting",
	})

	// Create temp file
	tmpFile, err := os.CreateTemp("", "export-deals-*.yaml")
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

	err = ExportDealsCommand(db, []string{"--output", tmpPath})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportDealsCommand() unexpected error: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "File Export Deal") {
		t.Error("Output file should contain deal title")
	}
}

func TestExportAllCommandCreatesDirectory(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create some data
	company := &repository.Company{Name: "Dir Test Corp"}
	_ = db.CreateCompany(company)
	_ = db.CreateContact(&repository.Contact{Name: "Dir Test Contact"})
	_ = db.CreateDeal(&repository.Deal{Title: "Dir Test Deal", CompanyID: company.ID, Stage: "prospecting"})

	// Use a nested directory path that doesn't exist
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "export", "dir")

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = ExportAllCommand(db, []string{"--output-dir", nestedDir})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportAllCommand() unexpected error: %v", err)
	}

	// Verify directory was created and files exist
	for _, file := range []string{"contacts.yaml", "companies.yaml", "deals.yaml"} {
		path := filepath.Join(nestedDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
}

func TestExportEmptyDatabase(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Export contacts from empty database
	err = ExportContactsCommand(db, []string{"--format", "yaml"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ExportContactsCommand() unexpected error with empty DB: %v", err)
	}
}

func TestWriteOutputToStdout(t *testing.T) {
	// Test writeOutput with empty path (should print to stdout)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := writeOutput("test content", "")

	_ = w.Close()
	var buf strings.Builder
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("writeOutput() unexpected error: %v", err)
	}

	if buf.String() != "test content" {
		t.Errorf("writeOutput() output = %q, want %q", buf.String(), "test content")
	}
}
