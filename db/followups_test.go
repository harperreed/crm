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
