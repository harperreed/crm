package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestSetContactCadence(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Create contact first
	contact := &models.Contact{Name: "Bob"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Test setting cadence on contact with no existing cadence
	err := SetContactCadence(db, contact.ID, 14, models.StrengthStrong)
	if err != nil {
		t.Fatalf("failed to set cadence: %v", err)
	}

	// Verify cadence was created
	cadence, err := GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if cadence == nil {
		t.Fatal("expected cadence to be created")
	}
	if cadence.CadenceDays != 14 {
		t.Errorf("expected 14 days, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != models.StrengthStrong {
		t.Errorf("expected strength 'strong', got %s", cadence.RelationshipStrength)
	}

	// Test updating existing cadence
	err = SetContactCadence(db, contact.ID, 7, models.StrengthWeak)
	if err != nil {
		t.Fatalf("failed to update cadence: %v", err)
	}

	// Verify cadence was updated
	cadence, err = GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get updated cadence: %v", err)
	}
	if cadence.CadenceDays != 7 {
		t.Errorf("expected 7 days after update, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != models.StrengthWeak {
		t.Errorf("expected strength 'weak' after update, got %s", cadence.RelationshipStrength)
	}
}

func TestGetRecentInteractions(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Create contact
	contact := &models.Contact{Name: "Charlie"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Log interactions at different times
	// Recent interaction (today)
	interaction1 := &models.InteractionLog{
		ContactID:       contact.ID,
		InteractionType: models.InteractionMeeting,
		Timestamp:       time.Now(),
		Notes:           "Today's meeting",
	}
	if err := LogInteraction(db, interaction1); err != nil {
		t.Fatalf("failed to log interaction1: %v", err)
	}

	// Old interaction (2 days ago)
	interaction2 := &models.InteractionLog{
		ContactID:       contact.ID,
		InteractionType: models.InteractionEmail,
		Timestamp:       time.Now().AddDate(0, 0, -2),
		Notes:           "Email from 2 days ago",
	}
	if err := LogInteraction(db, interaction2); err != nil {
		t.Fatalf("failed to log interaction2: %v", err)
	}

	// Get recent interactions within 7 days
	interactions, err := GetRecentInteractions(db, 7, 10)
	if err != nil {
		t.Fatalf("failed to get recent interactions: %v", err)
	}

	if len(interactions) != 2 {
		t.Errorf("expected 2 recent interactions, got %d", len(interactions))
	}

	// Test with narrow window (1 day)
	interactions, err = GetRecentInteractions(db, 1, 10)
	if err != nil {
		t.Fatalf("failed to get 1-day interactions: %v", err)
	}

	if len(interactions) != 1 {
		t.Errorf("expected 1 interaction within 1 day, got %d", len(interactions))
	}

	// Test with limit
	interactions, err = GetRecentInteractions(db, 7, 1)
	if err != nil {
		t.Fatalf("failed to get limited interactions: %v", err)
	}

	if len(interactions) != 1 {
		t.Errorf("expected 1 interaction with limit, got %d", len(interactions))
	}
}

func TestGetContactCadenceNonexistent(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Try to get cadence for non-existent contact
	nonexistentID := uuid.New()
	cadence, err := GetContactCadence(db, nonexistentID)
	if err != nil {
		t.Fatalf("expected no error for non-existent contact, got: %v", err)
	}
	if cadence != nil {
		t.Error("expected nil cadence for non-existent contact")
	}
}

func TestUpdateCadenceAfterInteractionNewCadence(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	// Create contact without cadence
	contact := &models.Contact{Name: "Diana"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	// Update cadence after interaction (should create default cadence)
	timestamp := time.Now()
	err := UpdateCadenceAfterInteraction(db, contact.ID, timestamp)
	if err != nil {
		t.Fatalf("failed to update cadence after interaction: %v", err)
	}

	// Verify default cadence was created
	cadence, err := GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if cadence == nil {
		t.Fatal("expected cadence to be created")
	}
	if cadence.CadenceDays != 30 {
		t.Errorf("expected default 30 days, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != models.StrengthMedium {
		t.Errorf("expected default 'medium' strength, got %s", cadence.RelationshipStrength)
	}
	if cadence.LastInteractionDate == nil {
		t.Error("expected last interaction date to be set")
	}
}
