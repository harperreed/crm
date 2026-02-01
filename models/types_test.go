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

func TestSyncStateDefaults(t *testing.T) {
	state := &SyncState{
		Service: "contacts",
		Status:  SyncStatusIdle,
	}

	if state.Status != SyncStatusIdle {
		t.Errorf("expected idle status, got %s", state.Status)
	}
}

func TestSuggestionCreation(t *testing.T) {
	suggestion := &Suggestion{
		ID:         uuid.New(),
		Type:       SuggestionTypeDeal,
		Confidence: 0.85,
		Status:     SuggestionStatusPending,
	}

	if suggestion.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %.2f", suggestion.Confidence)
	}
	if suggestion.Status != SuggestionStatusPending {
		t.Errorf("expected pending status, got %s", suggestion.Status)
	}
}

func TestComputePriorityScoreNilLastInteractionDate(t *testing.T) {
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthMedium,
		LastInteractionDate:  nil,
	}

	score := cadence.ComputePriorityScore()
	if score != 0.0 {
		t.Errorf("expected priority score 0.0 for nil LastInteractionDate, got %.1f", score)
	}
}

func TestComputePriorityScoreNotOverdue(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -15) // 15 days ago, cadence is 30
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthMedium,
		LastInteractionDate:  &lastContact,
	}

	score := cadence.ComputePriorityScore()
	if score != 0.0 {
		t.Errorf("expected priority score 0.0 when not overdue, got %.1f", score)
	}
}

func TestComputePriorityScoreMediumStrength(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -45) // 45 days ago
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthMedium,
		LastInteractionDate:  &lastContact,
	}

	score := cadence.ComputePriorityScore()
	// 45 - 30 = 15 days overdue
	// 15 * 2 = 30 base score
	// 30 * 1.5 (medium multiplier) = 45
	expected := 45.0
	if score != expected {
		t.Errorf("expected priority score %.1f for medium strength, got %.1f", expected, score)
	}
}

func TestComputePriorityScoreWeakStrength(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -45) // 45 days ago
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthWeak,
		LastInteractionDate:  &lastContact,
	}

	score := cadence.ComputePriorityScore()
	// 45 - 30 = 15 days overdue
	// 15 * 2 = 30 base score
	// 30 * 1.0 (weak multiplier) = 30
	expected := 30.0
	if score != expected {
		t.Errorf("expected priority score %.1f for weak strength, got %.1f", expected, score)
	}
}

func TestComputePriorityScoreUnknownStrength(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -45) // 45 days ago
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: "unknown",
		LastInteractionDate:  &lastContact,
	}

	score := cadence.ComputePriorityScore()
	// 45 - 30 = 15 days overdue
	// 15 * 2 = 30 base score
	// 30 * 1.0 (default multiplier) = 30
	expected := 30.0
	if score != expected {
		t.Errorf("expected priority score %.1f for unknown strength, got %.1f", expected, score)
	}
}

func TestUpdateNextFollowup(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -10)
	cadence := &ContactCadence{
		ContactID:           uuid.New(),
		CadenceDays:         30,
		LastInteractionDate: &lastContact,
	}

	cadence.UpdateNextFollowup()

	if cadence.NextFollowupDate == nil {
		t.Error("expected NextFollowupDate to be set")
	} else {
		expected := lastContact.AddDate(0, 0, 30)
		if !cadence.NextFollowupDate.Equal(expected) {
			t.Errorf("expected NextFollowupDate %v, got %v", expected, *cadence.NextFollowupDate)
		}
	}
}

func TestUpdateNextFollowupNilLastInteraction(t *testing.T) {
	cadence := &ContactCadence{
		ContactID:           uuid.New(),
		CadenceDays:         30,
		LastInteractionDate: nil,
	}

	cadence.UpdateNextFollowup()

	if cadence.NextFollowupDate != nil {
		t.Error("expected NextFollowupDate to remain nil when LastInteractionDate is nil")
	}
}

func TestContactStruct(t *testing.T) {
	companyID := uuid.New()
	now := time.Now()
	contact := &Contact{
		ID:              uuid.New(),
		Name:            "John Doe",
		Email:           "john@example.com",
		Phone:           "555-1234",
		CompanyID:       &companyID,
		Notes:           "Test notes",
		LastContactedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if contact.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %s", contact.Name)
	}
	if contact.Email != "john@example.com" {
		t.Errorf("expected email john@example.com, got %s", contact.Email)
	}
}

func TestCompanyStruct(t *testing.T) {
	now := time.Now()
	company := &Company{
		ID:        uuid.New(),
		Name:      "Acme Corp",
		Domain:    "acme.com",
		Industry:  "Technology",
		Notes:     "Test company",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if company.Name != "Acme Corp" {
		t.Errorf("expected name Acme Corp, got %s", company.Name)
	}
	if company.Domain != "acme.com" {
		t.Errorf("expected domain acme.com, got %s", company.Domain)
	}
}

func TestDealStruct(t *testing.T) {
	companyID := uuid.New()
	contactID := uuid.New()
	now := time.Now()
	closeDate := now.AddDate(0, 1, 0)
	deal := &Deal{
		ID:                uuid.New(),
		Title:             "Big Deal",
		Amount:            100000,
		Currency:          "USD",
		Stage:             StageProspecting,
		CompanyID:         companyID,
		ContactID:         &contactID,
		ExpectedCloseDate: &closeDate,
		CreatedAt:         now,
		UpdatedAt:         now,
		LastActivityAt:    now,
	}

	if deal.Title != "Big Deal" {
		t.Errorf("expected title Big Deal, got %s", deal.Title)
	}
	if deal.Amount != 100000 {
		t.Errorf("expected amount 100000, got %d", deal.Amount)
	}
	if deal.Stage != StageProspecting {
		t.Errorf("expected stage prospecting, got %s", deal.Stage)
	}
}

func TestDealNoteStruct(t *testing.T) {
	now := time.Now()
	note := &DealNote{
		ID:        uuid.New(),
		DealID:    uuid.New(),
		Content:   "Test note content",
		CreatedAt: now,
	}

	if note.Content != "Test note content" {
		t.Errorf("expected content 'Test note content', got %s", note.Content)
	}
}

func TestRelationshipStruct(t *testing.T) {
	now := time.Now()
	rel := &Relationship{
		ID:               uuid.New(),
		ContactID1:       uuid.New(),
		ContactID2:       uuid.New(),
		RelationshipType: "colleague",
		Context:          "Work together",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if rel.RelationshipType != "colleague" {
		t.Errorf("expected relationship type colleague, got %s", rel.RelationshipType)
	}
	if rel.Context != "Work together" {
		t.Errorf("expected context 'Work together', got %s", rel.Context)
	}
}

func TestStageConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{StageProspecting, "prospecting"},
		{StageQualification, "qualification"},
		{StageProposal, "proposal"},
		{StageNegotiation, "negotiation"},
		{StageClosedWon, "closed_won"},
		{StageClosedLost, "closed_lost"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestStrengthConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{StrengthWeak, "weak"},
		{StrengthMedium, "medium"},
		{StrengthStrong, "strong"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestInteractionTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{InteractionMeeting, "meeting"},
		{InteractionCall, "call"},
		{InteractionEmail, "email"},
		{InteractionMessage, "message"},
		{InteractionEvent, "event"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestSentimentConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{SentimentPositive, "positive"},
		{SentimentNeutral, "neutral"},
		{SentimentNegative, "negative"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestSyncStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{SyncStatusIdle, "idle"},
		{SyncStatusSyncing, "syncing"},
		{SyncStatusError, "error"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestSuggestionTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{SuggestionTypeDeal, "deal"},
		{SuggestionTypeRelationship, "relationship"},
		{SuggestionTypeCompany, "company"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestSuggestionStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{SuggestionStatusPending, "pending"},
		{SuggestionStatusAccepted, "accepted"},
		{SuggestionStatusRejected, "rejected"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestSyncLogStruct(t *testing.T) {
	now := time.Now()
	syncLog := &SyncLog{
		ID:            uuid.New(),
		SourceService: "gmail",
		SourceID:      "msg123",
		EntityType:    "contact",
		EntityID:      uuid.New(),
		ImportedAt:    now,
		Metadata:      `{"key": "value"}`,
	}

	if syncLog.SourceService != "gmail" {
		t.Errorf("expected source service gmail, got %s", syncLog.SourceService)
	}
	if syncLog.EntityType != "contact" {
		t.Errorf("expected entity type contact, got %s", syncLog.EntityType)
	}
}

func TestFollowupContactStruct(t *testing.T) {
	now := time.Now()
	nextFollowup := now.AddDate(0, 0, 7)
	fc := &FollowupContact{
		Contact: Contact{
			ID:   uuid.New(),
			Name: "Test Contact",
		},
		CadenceDays:          30,
		RelationshipStrength: StrengthStrong,
		PriorityScore:        75.5,
		DaysSinceContact:     45,
		NextFollowupDate:     &nextFollowup,
	}

	if fc.Name != "Test Contact" {
		t.Errorf("expected name Test Contact, got %s", fc.Name)
	}
	if fc.PriorityScore != 75.5 {
		t.Errorf("expected priority score 75.5, got %.1f", fc.PriorityScore)
	}
	if fc.DaysSinceContact != 45 {
		t.Errorf("expected days since contact 45, got %d", fc.DaysSinceContact)
	}
}
