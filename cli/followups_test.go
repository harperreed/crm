// ABOUTME: Tests for followup CLI commands
// ABOUTME: Validates followup list and interaction logging commands
package cli

import (
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
)

func TestFollowupListCommand(t *testing.T) {
	client := charm.NewTestClient(t)

	// Will test that command runs without error
	// Detailed output testing will be manual
	err := FollowupListCommand(client, []string{})
	if err != nil {
		t.Errorf("FollowupListCommand failed: %v", err)
	}
}

func TestLogInteractionCommand(t *testing.T) {
	client := charm.NewTestClient(t)

	// Create a contact first
	contact := &charm.Contact{ID: uuid.New(), Name: "Alice", Email: "alice@example.com"}
	if err := client.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	args := []string{
		"--contact", contact.ID.String(),
		"--type", "meeting",
		"--notes", "Coffee chat",
	}

	err := LogInteractionCommand(client, args)
	if err != nil {
		t.Errorf("LogInteractionCommand failed: %v", err)
	}

	// Verify interaction was logged
	logs, err := client.ListInteractionLogs(&charm.InteractionFilter{
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
