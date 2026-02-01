// ABOUTME: Tests for visualization CLI commands
// ABOUTME: Validates graph generation and dashboard commands
package cli

import (
	"os"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestVizGraphContactsCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	contact1 := &repository.Contact{Name: "Alice", Email: "alice@example.com"}
	contact2 := &repository.Contact{Name: "Bob", Email: "bob@example.com"}
	_ = db.CreateContact(contact1)
	_ = db.CreateContact(contact2)

	_ = db.CreateRelationship(&repository.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
	})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "generate all contacts graph",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "generate single contact graph",
			args:    []string{contact1.ID.String()},
			wantErr: false,
		},
		{
			name:    "invalid contact ID",
			args:    []string{"not-a-uuid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := VizGraphContactsCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("VizGraphContactsCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("VizGraphContactsCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestVizGraphContactsCommandWithOutput(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test contact
	_ = db.CreateContact(&repository.Contact{Name: "Alice"})

	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "viz-output-*.dot")
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

	err = VizGraphContactsCommand(db, []string{"--output", tmpPath})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("VizGraphContactsCommand() unexpected error: %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestVizGraphCompanyCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Acme Corp", Domain: "acme.com"}
	_ = db.CreateCompany(company)

	_ = db.CreateContact(&repository.Contact{Name: "Alice", CompanyID: &company.ID, CompanyName: company.Name})
	_ = db.CreateContact(&repository.Contact{Name: "Bob", CompanyID: &company.ID, CompanyName: company.Name})

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "generate company graph",
			args:    []string{company.ID.String()},
			wantErr: false,
		},
		{
			name:    "missing company ID",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "invalid company ID",
			args:    []string{"not-a-uuid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := VizGraphCompanyCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("VizGraphCompanyCommand() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("VizGraphCompanyCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestVizGraphPipelineCommand(t *testing.T) {
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

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = VizGraphPipelineCommand(db, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("VizGraphPipelineCommand() unexpected error: %v", err)
	}
}

func TestVizGraphAllCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Acme Corp"}
	_ = db.CreateCompany(company)

	contact1 := &repository.Contact{Name: "Alice", CompanyID: &company.ID}
	contact2 := &repository.Contact{Name: "Bob", CompanyID: &company.ID}
	_ = db.CreateContact(contact1)
	_ = db.CreateContact(contact2)

	_ = db.CreateRelationship(&repository.Relationship{
		ContactID1: contact1.ID,
		ContactID2: contact2.ID,
	})

	_ = db.CreateDeal(&repository.Deal{
		Title:     "Deal",
		CompanyID: company.ID,
		Stage:     "prospecting",
	})

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = VizGraphAllCommand(db, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("VizGraphAllCommand() unexpected error: %v", err)
	}
}

func TestVizDashboardCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create test data
	company := &repository.Company{Name: "Acme Corp"}
	_ = db.CreateCompany(company)

	_ = db.CreateContact(&repository.Contact{Name: "Alice"})
	_ = db.CreateContact(&repository.Contact{Name: "Bob"})

	_ = db.CreateDeal(&repository.Deal{
		Title:     "Deal 1",
		CompanyID: company.ID,
		Stage:     "prospecting",
		Amount:    50000,
		Currency:  "USD",
	})

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = VizDashboardCommand(db, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("VizDashboardCommand() unexpected error: %v", err)
	}
}
