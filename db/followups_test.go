package db

import (
	"testing"
	"time"

	"github.com/harperreed/pagen/models"
)

func TestCreateContactCadence(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Create contact first
	contact := &models.Contact{Name: "Alice"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	cadence := &models.ContactCadence{
		ContactID:            contact.ID,
		CadenceDays:          30,
		RelationshipStrength: models.StrengthMedium,
	}

	err := CreateContactCadence(db, cadence)
	if err != nil {
		t.Fatalf("failed to create cadence: %v", err)
	}

	// Verify it was created
	retrieved, err := GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if retrieved.CadenceDays != 30 {
		t.Errorf("expected 30 days, got %d", retrieved.CadenceDays)
	}
}

func TestGetFollowupList(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Create contacts with different cadences
	contact1 := &models.Contact{Name: "Alice"}
	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("failed to create contact1: %v", err)
	}
	lastContact := time.Now().AddDate(0, 0, -45)
	cadence1 := &models.ContactCadence{
		ContactID:            contact1.ID,
		CadenceDays:          30,
		RelationshipStrength: models.StrengthStrong,
		LastInteractionDate:  &lastContact,
		PriorityScore:        60.0,
	}
	if err := CreateContactCadence(db, cadence1); err != nil {
		t.Fatalf("failed to create cadence1: %v", err)
	}

	contact2 := &models.Contact{Name: "Bob"}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("failed to create contact2: %v", err)
	}

	// Get followup list
	followups, err := GetFollowupList(db, 10)
	if err != nil {
		t.Fatalf("failed to get followup list: %v", err)
	}

	if len(followups) == 0 {
		t.Error("expected at least one followup")
	}

	// Should be sorted by priority score descending
	if len(followups) > 1 && followups[0].PriorityScore < followups[1].PriorityScore {
		t.Error("followups not sorted by priority")
	}
}

func TestLogInteraction(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Create contact
	contact := &models.Contact{Name: "Alice"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Create initial cadence
	lastContact := time.Now().AddDate(0, 0, -10)
	cadence := &models.ContactCadence{
		ContactID:            contact.ID,
		CadenceDays:          30,
		RelationshipStrength: models.StrengthMedium,
		LastInteractionDate:  &lastContact,
	}
	cadence.PriorityScore = cadence.ComputePriorityScore()
	if err := CreateContactCadence(db, cadence); err != nil {
		t.Fatalf("failed to create cadence: %v", err)
	}

	// Log interaction
	interaction := &models.InteractionLog{
		ContactID:       contact.ID,
		InteractionType: models.InteractionMeeting,
		Timestamp:       time.Now(),
		Notes:           "Coffee chat",
	}

	err := LogInteraction(db, interaction)
	if err != nil {
		t.Fatalf("failed to log interaction: %v", err)
	}

	// Verify cadence was updated
	updated, err := GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get updated cadence: %v", err)
	}

	if updated.LastInteractionDate.Before(lastContact) {
		t.Error("last interaction date was not updated")
	}

	// Priority score should be 0 now (contact is up-to-date)
	if updated.PriorityScore != 0 {
		t.Errorf("expected priority score 0 after recent interaction, got %.1f", updated.PriorityScore)
	}
}

func TestGetInteractionHistory(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	contact := &models.Contact{Name: "Alice"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Log multiple interactions
	for i := 0; i < 3; i++ {
		interaction := &models.InteractionLog{
			ContactID:       contact.ID,
			InteractionType: models.InteractionEmail,
			Timestamp:       time.Now().AddDate(0, 0, -i),
		}
		if err := LogInteraction(db, interaction); err != nil {
			t.Fatalf("failed to log interaction %d: %v", i, err)
		}
	}

	history, err := GetInteractionHistory(db, contact.ID, 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 interactions, got %d", len(history))
	}

	// Should be sorted newest first
	if len(history) > 1 && history[0].Timestamp.Before(history[1].Timestamp) {
		t.Error("interactions not sorted by timestamp descending")
	}
}
