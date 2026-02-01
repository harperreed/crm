// ABOUTME: Tests for followup CLI commands
// ABOUTME: Validates followup list and interaction logging commands
package cli

import (
	"testing"

	"github.com/harperreed/pagen/repository"
)

func TestFollowupListCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Will test that command runs without error
	// Detailed output testing will be manual
	err = FollowupListCommand(db, []string{})
	if err != nil {
		t.Errorf("FollowupListCommand failed: %v", err)
	}
}

func TestLogInteractionCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create a contact first
	contact := &repository.Contact{Name: "Alice", Email: "alice@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	args := []string{
		"--contact", contact.ID.String(),
		"--type", "meeting",
		"--notes", "Coffee chat",
	}

	err = LogInteractionCommand(db, args)
	if err != nil {
		t.Errorf("LogInteractionCommand failed: %v", err)
	}

	// Verify interaction was logged
	logs, err := db.ListInteractionLogs(&repository.InteractionFilter{
		ContactID: &contact.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 interaction, got %d", len(logs))
	}
}
