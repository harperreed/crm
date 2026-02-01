// ABOUTME: Tests for MCP follow-up handlers
// ABOUTME: Validates follow-up list, interaction logging, and cadence management
package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
)

func TestNewFollowupHandlers(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewFollowupHandlers(db)
	if handlers == nil {
		t.Error("Expected non-nil FollowupHandlers")
	}
}

func TestGetFollowupList(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contact with cadence
	contact := &repository.Contact{Name: "Test Contact"}
	_ = db.CreateContact(contact)

	lastInteraction := time.Now().AddDate(0, 0, -45) // 45 days ago
	_ = db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          30,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &lastInteraction,
	})

	handlers := NewFollowupHandlers(db)

	_, output, err := handlers.GetFollowupList(context.Background(), nil, GetFollowupListInput{})
	if err != nil {
		t.Fatalf("GetFollowupList failed: %v", err)
	}

	if output.Count < 0 {
		t.Error("Expected non-negative count")
	}
}

func TestGetFollowupListWithLimit(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create multiple contacts
	for i := 0; i < 5; i++ {
		contact := &repository.Contact{Name: "Contact " + string(rune('A'+i))}
		_ = db.CreateContact(contact)
	}

	handlers := NewFollowupHandlers(db)

	limit := 2
	_, output, err := handlers.GetFollowupList(context.Background(), nil, GetFollowupListInput{Limit: &limit})
	if err != nil {
		t.Fatalf("GetFollowupList failed: %v", err)
	}

	if output.Count > limit {
		t.Errorf("Expected count <= %d, got %d", limit, output.Count)
	}
}

func TestGetFollowupListOverdueOnly(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contact with cadence that is overdue
	contact := &repository.Contact{Name: "Overdue Contact"}
	_ = db.CreateContact(contact)

	lastInteraction := time.Now().AddDate(0, 0, -45) // 45 days ago, cadence is 30
	_ = db.SaveContactCadence(&repository.ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          30,
		RelationshipStrength: repository.StrengthStrong,
		LastInteractionDate:  &lastInteraction,
		PriorityScore:        60.0,
	})

	handlers := NewFollowupHandlers(db)

	overdueOnly := true
	_, output, err := handlers.GetFollowupList(context.Background(), nil, GetFollowupListInput{OverdueOnly: &overdueOnly})
	if err != nil {
		t.Fatalf("GetFollowupList failed: %v", err)
	}

	// All returned should have priority > 0
	for _, f := range output.Followups {
		if f.PriorityScore <= 0 {
			t.Errorf("Expected only overdue contacts with priority > 0, got %.1f", f.PriorityScore)
		}
	}
}

func TestGetFollowupListWithMinPriority(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewFollowupHandlers(db)

	minPriority := 50.0
	_, output, err := handlers.GetFollowupList(context.Background(), nil, GetFollowupListInput{MinPriority: &minPriority})
	if err != nil {
		t.Fatalf("GetFollowupList failed: %v", err)
	}

	for _, f := range output.Followups {
		if f.PriorityScore < minPriority {
			t.Errorf("Expected priority >= %.1f, got %.1f", minPriority, f.PriorityScore)
		}
	}
}

func TestLogInteraction(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contact
	contact := &repository.Contact{Name: "Test Contact"}
	_ = db.CreateContact(contact)

	handlers := NewFollowupHandlers(db)

	notes := "Had a great call"
	sentiment := "positive"
	_, output, err := handlers.LogInteraction(context.Background(), nil, LogInteractionInput{
		ContactID:       contact.ID.String(),
		InteractionType: "call",
		Notes:           &notes,
		Sentiment:       &sentiment,
	})
	if err != nil {
		t.Fatalf("LogInteraction failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success")
	}
	if output.InteractionID == "" {
		t.Error("Expected interaction ID")
	}
}

func TestLogInteractionByName(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contact
	contact := &repository.Contact{Name: "Unique Name"}
	_ = db.CreateContact(contact)

	handlers := NewFollowupHandlers(db)

	_, output, err := handlers.LogInteraction(context.Background(), nil, LogInteractionInput{
		ContactID:       "Unique Name",
		InteractionType: "meeting",
	})
	if err != nil {
		t.Fatalf("LogInteraction failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success")
	}
}

func TestLogInteractionNotFound(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewFollowupHandlers(db)

	_, _, err := handlers.LogInteraction(context.Background(), nil, LogInteractionInput{
		ContactID:       "NonexistentContact",
		InteractionType: "call",
	})
	if err == nil {
		t.Error("Expected error for nonexistent contact")
	}
}

func TestLogInteractionInvalidUUID(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewFollowupHandlers(db)

	_, _, err := handlers.LogInteraction(context.Background(), nil, LogInteractionInput{
		ContactID:       uuid.New().String(), // Valid UUID but doesn't exist
		InteractionType: "call",
	})
	if err == nil {
		t.Error("Expected error for nonexistent UUID")
	}
}

func TestSetCadence(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contact
	contact := &repository.Contact{Name: "Test Contact"}
	_ = db.CreateContact(contact)

	handlers := NewFollowupHandlers(db)

	_, output, err := handlers.SetCadence(context.Background(), nil, SetCadenceInput{
		ContactID: contact.ID.String(),
		Days:      14,
		Strength:  "strong",
	})
	if err != nil {
		t.Fatalf("SetCadence failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success")
	}

	// Verify cadence was set
	cadence, err := db.GetContactCadence(contact.ID)
	if err != nil {
		t.Fatalf("Failed to get cadence: %v", err)
	}
	if cadence.CadenceDays != 14 {
		t.Errorf("Expected cadence days 14, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != "strong" {
		t.Errorf("Expected strength 'strong', got %s", cadence.RelationshipStrength)
	}
}

func TestSetCadenceByName(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	// Create contact
	contact := &repository.Contact{Name: "Named Contact"}
	_ = db.CreateContact(contact)

	handlers := NewFollowupHandlers(db)

	_, output, err := handlers.SetCadence(context.Background(), nil, SetCadenceInput{
		ContactID: "Named Contact",
		Days:      7,
		Strength:  "medium",
	})
	if err != nil {
		t.Fatalf("SetCadence failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success")
	}
}

func TestSetCadenceNotFound(t *testing.T) {
	db, cleanup, _ := repository.NewTestDB()
	defer cleanup()

	handlers := NewFollowupHandlers(db)

	_, _, err := handlers.SetCadence(context.Background(), nil, SetCadenceInput{
		ContactID: "NonexistentContact",
		Days:      30,
		Strength:  "weak",
	})
	if err == nil {
		t.Error("Expected error for nonexistent contact")
	}
}
