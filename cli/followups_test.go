// ABOUTME: Tests for followup CLI commands
// ABOUTME: Validates followup list and interaction logging commands
package cli

import (
	"testing"
	"time"

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

func TestFollowupStatsCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Command should run without error on empty database
	err = FollowupStatsCommand(db, []string{})
	if err != nil {
		t.Errorf("FollowupStatsCommand failed on empty DB: %v", err)
	}
}

func TestFollowupStatsCommandWithData(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts with cadences
	contact1 := &repository.Contact{Name: "Bob", Email: "bob@example.com"}
	contact2 := &repository.Contact{Name: "Carol", Email: "carol@example.com"}
	if err := db.CreateContact(contact1); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}
	if err := db.CreateContact(contact2); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Set up cadences with different strengths
	now := time.Now()
	lastWeek := now.AddDate(0, 0, -7)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact1.ID,
		ContactName:          contact1.Name,
		CadenceDays:          14,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &lastWeek,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact2.ID,
		ContactName:          contact2.Name,
		CadenceDays:          30,
		RelationshipStrength: repository.StrengthMedium,
		LastInteractionDate:  &lastWeek,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}

	err = FollowupStatsCommand(db, []string{})
	if err != nil {
		t.Errorf("FollowupStatsCommand failed: %v", err)
	}
}

func TestSetCadenceCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create a contact
	contact := &repository.Contact{Name: "Dave", Email: "dave@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	args := []string{
		"--contact", contact.ID.String(),
		"--days", "14",
		"--strength", "strong",
	}

	err = SetCadenceCommand(db, args)
	if err != nil {
		t.Errorf("SetCadenceCommand failed: %v", err)
	}

	// Verify cadence was set
	cadence, err := db.GetContactCadence(contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if cadence == nil {
		t.Fatal("cadence should not be nil")
	}
	if cadence.CadenceDays != 14 {
		t.Errorf("expected cadence days 14, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != "strong" {
		t.Errorf("expected strength 'strong', got %s", cadence.RelationshipStrength)
	}
}

func TestSetCadenceCommandByName(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create a contact
	contact := &repository.Contact{Name: "Emily", Email: "emily@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Use name instead of UUID
	args := []string{
		"--contact", "Emily",
		"--days", "7",
		"--strength", "weak",
	}

	err = SetCadenceCommand(db, args)
	if err != nil {
		t.Errorf("SetCadenceCommand by name failed: %v", err)
	}

	// Verify cadence was set
	cadence, err := db.GetContactCadence(contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if cadence.CadenceDays != 7 {
		t.Errorf("expected cadence days 7, got %d", cadence.CadenceDays)
	}
}

func TestSetCadenceCommandNoContact(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Missing --contact flag
	args := []string{
		"--days", "14",
	}

	err = SetCadenceCommand(db, args)
	if err == nil {
		t.Error("Expected error for missing contact")
	}
}

func TestSetCadenceCommandContactNotFound(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	args := []string{
		"--contact", "NonExistentPerson",
		"--days", "14",
	}

	err = SetCadenceCommand(db, args)
	if err == nil {
		t.Error("Expected error for non-existent contact")
	}
}

func TestSetCadenceCommandWithExistingCadence(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contact with existing cadence
	contact := &repository.Contact{Name: "Frank", Email: "frank@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	now := time.Now()
	lastWeek := now.AddDate(0, 0, -7)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          30,
		RelationshipStrength: repository.StrengthWeak,
		LastInteractionDate:  &lastWeek,
	}); err != nil {
		t.Fatalf("failed to set initial cadence: %v", err)
	}

	// Update cadence
	args := []string{
		"--contact", contact.ID.String(),
		"--days", "14",
		"--strength", "strong",
	}

	err = SetCadenceCommand(db, args)
	if err != nil {
		t.Errorf("SetCadenceCommand failed to update: %v", err)
	}

	// Verify cadence was updated
	cadence, err := db.GetContactCadence(contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if cadence.CadenceDays != 14 {
		t.Errorf("expected cadence days 14, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != "strong" {
		t.Errorf("expected strength 'strong', got %s", cadence.RelationshipStrength)
	}
}

func TestDigestCommand(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Command should run without error on empty database
	err = DigestCommand(db, []string{})
	if err != nil {
		t.Errorf("DigestCommand failed on empty DB: %v", err)
	}
}

func TestDigestCommandTextFormat(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	args := []string{"--format", "text"}
	err = DigestCommand(db, args)
	if err != nil {
		t.Errorf("DigestCommand text format failed: %v", err)
	}
}

func TestDigestCommandJSONFormat(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	args := []string{"--format", "json"}
	err = DigestCommand(db, args)
	if err != nil {
		t.Errorf("DigestCommand json format failed: %v", err)
	}
}

func TestDigestCommandHTMLFormat(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	args := []string{"--format", "html"}
	err = DigestCommand(db, args)
	if err != nil {
		t.Errorf("DigestCommand html format failed: %v", err)
	}
}

func TestDigestCommandUnsupportedFormat(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	args := []string{"--format", "xml"}
	err = DigestCommand(db, args)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestDigestCommandWithData(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts with different followup states
	contact1 := &repository.Contact{Name: "Overdue Contact", Email: "overdue@example.com"}
	contact2 := &repository.Contact{Name: "DueSoon Contact", Email: "duesoon@example.com"}
	if err := db.CreateContact(contact1); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}
	if err := db.CreateContact(contact2); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Set up cadences - one overdue, one due soon
	now := time.Now()
	twoWeeksAgo := now.AddDate(0, 0, -14)
	fiveDaysAgo := now.AddDate(0, 0, -5)

	// Contact 1: overdue (14 days since contact, 7 day cadence = 7 days overdue)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact1.ID,
		ContactName:          contact1.Name,
		CadenceDays:          7,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &twoWeeksAgo,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}
	// Contact 2: due soon (5 days since contact, 7 day cadence = within 3 days of due)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact2.ID,
		ContactName:          contact2.Name,
		CadenceDays:          7,
		RelationshipStrength: repository.StrengthMedium,
		LastInteractionDate:  &fiveDaysAgo,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}

	// Test all formats with data
	for _, format := range []string{"text", "json", "html"} {
		args := []string{"--format", format}
		err = DigestCommand(db, args)
		if err != nil {
			t.Errorf("DigestCommand %s format with data failed: %v", format, err)
		}
	}
}

func TestLogInteractionCommandByName(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create a contact
	contact := &repository.Contact{Name: "Grace", Email: "grace@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Use name instead of UUID
	args := []string{
		"--contact", "Grace",
		"--type", "call",
		"--notes", "Phone call",
	}

	err = LogInteractionCommand(db, args)
	if err != nil {
		t.Errorf("LogInteractionCommand by name failed: %v", err)
	}
}

func TestLogInteractionCommandNoContact(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Missing --contact flag
	args := []string{
		"--type", "meeting",
	}

	err = LogInteractionCommand(db, args)
	if err == nil {
		t.Error("Expected error for missing contact")
	}
}

func TestLogInteractionCommandContactNotFound(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	args := []string{
		"--contact", "NonExistentPerson",
		"--type", "meeting",
	}

	err = LogInteractionCommand(db, args)
	if err == nil {
		t.Error("Expected error for non-existent contact")
	}
}

func TestLogInteractionCommandWithSentiment(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	contact := &repository.Contact{Name: "Henry", Email: "henry@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	args := []string{
		"--contact", contact.ID.String(),
		"--type", "email",
		"--notes", "Great discussion",
		"--sentiment", "positive",
	}

	err = LogInteractionCommand(db, args)
	if err != nil {
		t.Errorf("LogInteractionCommand with sentiment failed: %v", err)
	}

	// Verify interaction was logged with sentiment
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
	if logs[0].Sentiment == nil || *logs[0].Sentiment != "positive" {
		t.Error("sentiment was not set correctly")
	}
}

func TestFollowupListCommandWithFilters(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts with cadences
	contact := &repository.Contact{Name: "Isaac", Email: "isaac@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	now := time.Now()
	lastWeek := now.AddDate(0, 0, -7)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          14,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &lastWeek,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}

	// Test with overdue-only filter
	err = FollowupListCommand(db, []string{"--overdue-only"})
	if err != nil {
		t.Errorf("FollowupListCommand with overdue-only failed: %v", err)
	}

	// Test with strength filter
	err = FollowupListCommand(db, []string{"--strength", "strong"})
	if err != nil {
		t.Errorf("FollowupListCommand with strength filter failed: %v", err)
	}

	// Test with limit
	err = FollowupListCommand(db, []string{"--limit", "5"})
	if err != nil {
		t.Errorf("FollowupListCommand with limit failed: %v", err)
	}
}
