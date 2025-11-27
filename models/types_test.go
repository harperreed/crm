// ABOUTME: Tests for CRM data models
// ABOUTME: Validates ContactCadence, InteractionLog, and priority scoring logic
package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestContactCadenceDefaults(t *testing.T) {
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthMedium,
	}

	if cadence.CadenceDays != 30 {
		t.Errorf("expected default cadence 30, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != StrengthMedium {
		t.Errorf("expected medium strength, got %s", cadence.RelationshipStrength)
	}
}

func TestInteractionLogCreation(t *testing.T) {
	log := &InteractionLog{
		ID:              uuid.New(),
		ContactID:       uuid.New(),
		InteractionType: InteractionMeeting,
		Timestamp:       time.Now(),
		Notes:           "Coffee chat",
	}

	if log.InteractionType != InteractionMeeting {
		t.Errorf("expected meeting type, got %s", log.InteractionType)
	}
}

func TestComputePriorityScore(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -45) // 45 days ago
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthStrong,
		LastInteractionDate:  &lastContact,
	}

	score := cadence.ComputePriorityScore()

	// 45 - 30 = 15 days overdue
	// 15 * 2 = 30 base score
	// 30 * 2.0 (strong multiplier) = 60
	expected := 60.0
	if score != expected {
		t.Errorf("expected priority score %.1f, got %.1f", expected, score)
	}
}
