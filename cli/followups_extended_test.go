// ABOUTME: Extended tests for followup CLI commands
// ABOUTME: Covers digest printing, filtering, and edge cases
package cli

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/harperreed/pagen/repository"
)

func TestPrintTextDigestEmpty(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printTextDigest(nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printTextDigest(nil) unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected some output even for empty list")
	}
}

func TestPrintTextDigestWithOverdue(t *testing.T) {
	followups := []*repository.FollowupContact{
		{
			Name:             "Overdue Contact",
			Email:            "overdue@example.com",
			DaysSinceContact: 21,
			CadenceDays:      7,
			PriorityScore:    28.0,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printTextDigest(followups)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printTextDigest() unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "OVERDUE") {
		t.Error("expected OVERDUE section in output")
	}
	if !contains(output, "Overdue Contact") {
		t.Error("expected contact name in output")
	}
}

func TestPrintTextDigestWithDueSoon(t *testing.T) {
	followups := []*repository.FollowupContact{
		{
			Name:             "Due Soon Contact",
			Email:            "duesoon@example.com",
			DaysSinceContact: 5,
			CadenceDays:      7,
			PriorityScore:    5.0,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printTextDigest(followups)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printTextDigest() unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "DUE SOON") {
		t.Error("expected DUE SOON section in output")
	}
}

func TestPrintJSONDigestEmpty(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSONDigest(nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printJSONDigest(nil) unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, `"followups":[]`) {
		t.Errorf("expected empty followups array in JSON, got: %s", output)
	}
}

func TestPrintJSONDigestWithData(t *testing.T) {
	followups := []*repository.FollowupContact{
		{
			Name:             "Test Contact",
			Email:            "test@example.com",
			DaysSinceContact: 10,
			PriorityScore:    15.5,
		},
		{
			Name:             "Another Contact",
			Email:            "another@example.com",
			DaysSinceContact: 5,
			PriorityScore:    8.0,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSONDigest(followups)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printJSONDigest() unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "Test Contact") {
		t.Error("expected contact name in JSON output")
	}
	if !contains(output, "Another Contact") {
		t.Error("expected second contact name in JSON output")
	}
	if !contains(output, `"days":10`) {
		t.Error("expected days field in JSON output")
	}
}

func TestPrintHTMLDigestEmpty(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printHTMLDigest(nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printHTMLDigest(nil) unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "<html>") {
		t.Error("expected HTML tags in output")
	}
	if !contains(output, "</html>") {
		t.Error("expected closing HTML tag")
	}
	if !contains(output, "<table") {
		t.Error("expected table element")
	}
}

func TestPrintHTMLDigestWithData(t *testing.T) {
	followups := []*repository.FollowupContact{
		{
			Name:             "HTML Test",
			Email:            "html@example.com",
			DaysSinceContact: 12,
			PriorityScore:    20.0,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printHTMLDigest(followups)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("printHTMLDigest() unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "HTML Test") {
		t.Error("expected contact name in HTML output")
	}
	if !contains(output, "<tr>") {
		t.Error("expected table row element")
	}
	if !contains(output, "<td>") {
		t.Error("expected table cell element")
	}
}

func TestFollowupListCommandOverdueFilter(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contact with cadence
	contact := &repository.Contact{Name: "Test", Email: "test@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Set cadence with overdue state
	now := time.Now()
	twoWeeksAgo := now.AddDate(0, 0, -14)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          7,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &twoWeeksAgo,
		PriorityScore:        14.0, // Positive priority means overdue
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = FollowupListCommand(db, []string{"--overdue-only"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("FollowupListCommand() unexpected error: %v", err)
	}
}

func TestFollowupListCommandStrengthFilter(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts with different strengths
	strongContact := &repository.Contact{Name: "Strong", Email: "strong@example.com"}
	weakContact := &repository.Contact{Name: "Weak", Email: "weak@example.com"}
	if err := db.CreateContact(strongContact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}
	if err := db.CreateContact(weakContact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	now := time.Now()
	lastWeek := now.AddDate(0, 0, -7)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            strongContact.ID,
		ContactName:          strongContact.Name,
		CadenceDays:          14,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &lastWeek,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            weakContact.ID,
		ContactName:          weakContact.Name,
		CadenceDays:          30,
		RelationshipStrength: repository.StrengthWeak,
		LastInteractionDate:  &lastWeek,
	}); err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}

	// Test strong filter
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = FollowupListCommand(db, []string{"--strength", "strong"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("FollowupListCommand() unexpected error: %v", err)
	}
}

func TestFollowupListCommandCombinedFilters(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Test with combined filters
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = FollowupListCommand(db, []string{"--overdue-only", "--strength", "medium", "--limit", "5"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("FollowupListCommand() unexpected error: %v", err)
	}
}

func TestLogInteractionCommandMultipleMatches(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	// Create contacts with similar names
	contact1 := &repository.Contact{Name: "John Smith", Email: "john1@example.com"}
	contact2 := &repository.Contact{Name: "John Doe", Email: "john2@example.com"}
	if err := db.CreateContact(contact1); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}
	if err := db.CreateContact(contact2); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Search by partial name should fail with multiple matches
	args := []string{
		"--contact", "John",
		"--type", "meeting",
	}

	err = LogInteractionCommand(db, args)
	if err == nil {
		t.Error("Expected error for multiple matching contacts")
	}
}

func TestSetCadenceCommandInvalidStrength(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	contact := &repository.Contact{Name: "Test", Email: "test@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Setting an invalid strength should still work (no validation at CLI level)
	args := []string{
		"--contact", contact.ID.String(),
		"--days", "14",
		"--strength", "invalid",
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SetCadenceCommand(db, args)

	_ = w.Close()
	os.Stdout = oldStdout

	// CLI doesn't validate strength values, so this should succeed
	if err != nil {
		t.Errorf("SetCadenceCommand() unexpected error: %v", err)
	}
}

func TestSetCadenceCommandWithPriorityCalculation(t *testing.T) {
	db, cleanup, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer cleanup()

	contact := &repository.Contact{Name: "Priority Test", Email: "priority@example.com"}
	if err := db.CreateContact(contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Set initial cadence with last interaction
	now := time.Now()
	twoWeeksAgo := now.AddDate(0, 0, -14)
	if err := db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          30,
		RelationshipStrength: repository.StrengthWeak,
		LastInteractionDate:  &twoWeeksAgo,
	}); err != nil {
		t.Fatalf("failed to set initial cadence: %v", err)
	}

	// Update to shorter cadence (should calculate priority)
	args := []string{
		"--contact", contact.ID.String(),
		"--days", "7",
		"--strength", "strong",
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err = SetCadenceCommand(db, args)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SetCadenceCommand() unexpected error: %v", err)
	}

	// Verify priority was calculated (14 days since contact, 7 day cadence = 7 days overdue)
	cadence, err := db.GetContactCadence(contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}

	if cadence.PriorityScore <= 0 {
		t.Error("expected positive priority score for overdue contact")
	}
}
