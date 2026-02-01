// ABOUTME: Tests for relationship CLI commands
// ABOUTME: Validates update and delete relationship operations
package cli

import (
	"os"
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestUpdateRelationshipCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts and relationship
	contact1 := &repository.Contact{Name: "Alice"}
	contact2 := &repository.Contact{Name: "Bob"}
	_ = db.CreateContact(contact1)
	_ = db.CreateContact(contact2)

	rel := &repository.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		Contact1Name:     contact1.Name,
		Contact2Name:     contact2.Name,
		RelationshipType: "colleague",
		Context:          "Work",
	}
	_ = db.CreateRelationship(rel)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "update type",
			args:    []string{"--type", "friend", rel.ID.String()},
			wantErr: false,
		},
		{
			name:    "update context",
			args:    []string{"--context", "Met at conference", rel.ID.String()},
			wantErr: false,
		},
		{
			name:    "update both",
			args:    []string{"--type", "mentor", "--context", "Career guidance", rel.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing ID",
			args:        []string{"--type", "friend"},
			wantErr:     true,
			errContains: "usage:",
		},
		{
			name:        "invalid ID",
			args:        []string{"--type", "friend", "invalid-uuid"},
			wantErr:     true,
			errContains: "invalid relationship ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := UpdateRelationshipCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateRelationshipCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateRelationshipCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateRelationshipCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteRelationshipCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts and relationship
	contact1 := &repository.Contact{Name: "Alice"}
	contact2 := &repository.Contact{Name: "Bob"}
	_ = db.CreateContact(contact1)
	_ = db.CreateContact(contact2)

	rel := &repository.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
	}
	_ = db.CreateRelationship(rel)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing relationship",
			args:    []string{rel.ID.String()},
			wantErr: false,
		},
		{
			name:        "missing ID",
			args:        []string{},
			wantErr:     true,
			errContains: "usage:",
		},
		{
			name:        "invalid ID",
			args:        []string{"invalid-uuid"},
			wantErr:     true,
			errContains: "invalid relationship ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := DeleteRelationshipCommand(db, tt.args)

			_ = w.Close()
			os.Stdout = oldStdout

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteRelationshipCommand() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteRelationshipCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteRelationshipCommand() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteNonexistentRelationship(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = DeleteRelationshipCommand(db, []string{"00000000-0000-0000-0000-000000000000"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Error("DeleteRelationshipCommand() expected error for nonexistent relationship")
	}
}
