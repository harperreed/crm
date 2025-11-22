// ABOUTME: Tests for deal and deal note database operations
// ABOUTME: Covers CRUD operations, stage updates, and note management
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func TestCreateDeal(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create company
	company := &models.Company{Name: "Deal Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	deal := &models.Deal{
		Title:     "Big Deal",
		Amount:    100000,
		Currency:  "USD",
		Stage:     models.StageProspecting,
		CompanyID: company.ID,
	}

	if err := CreateDeal(db, deal); err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	if deal.ID == uuid.Nil {
		t.Error("Deal ID was not set")
	}

	if deal.LastActivityAt.IsZero() {
		t.Error("LastActivityAt was not set")
	}
}

func TestUpdateDeal(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	company := &models.Company{Name: "Deal Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	deal := &models.Deal{
		Title:     "Test Deal",
		Stage:     models.StageProspecting,
		CompanyID: company.ID,
		Currency:  "USD",
	}

	if err := CreateDeal(db, deal); err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	// Update stage
	deal.Stage = models.StageNegotiation
	deal.Amount = 50000

	if err := UpdateDeal(db, deal); err != nil {
		t.Fatalf("UpdateDeal failed: %v", err)
	}

	// Verify update
	found, err := GetDeal(db, deal.ID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}

	if found.Stage != models.StageNegotiation {
		t.Errorf("Expected stage %s, got %s", models.StageNegotiation, found.Stage)
	}

	if found.Amount != 50000 {
		t.Errorf("Expected amount 50000, got %d", found.Amount)
	}
}

func TestAddDealNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	company := &models.Company{Name: "Note Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	deal := &models.Deal{
		Title:     "Note Deal",
		Stage:     models.StageProspecting,
		CompanyID: company.ID,
		Currency:  "USD",
	}

	if err := CreateDeal(db, deal); err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	originalLastActivity := deal.LastActivityAt

	note := &models.DealNote{
		DealID:  deal.ID,
		Content: "Had a great call today",
	}

	if err := AddDealNote(db, note); err != nil {
		t.Fatalf("AddDealNote failed: %v", err)
	}

	if note.ID == uuid.Nil {
		t.Error("Note ID was not set")
	}

	// Verify note
	notes, err := GetDealNotes(db, deal.ID)
	if err != nil {
		t.Fatalf("GetDealNotes failed: %v", err)
	}

	if len(notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(notes))
	}

	if notes[0].Content != note.Content {
		t.Error("Note content mismatch")
	}

	// Reload deal to check timestamp update
	updatedDeal, err := GetDeal(db, deal.ID)
	if err != nil {
		t.Fatalf("GetDeal after note failed: %v", err)
	}
	if !updatedDeal.LastActivityAt.After(originalLastActivity) {
		t.Error("Deal's LastActivityAt was not updated after adding note")
	}
}

func TestAddDealNoteUpdatesContact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create company
	company := &models.Company{Name: "Contact Update Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contact
	contact := &models.Contact{
		Name:  "Alice Smith",
		Email: "alice@example.com",
	}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Create deal linked to contact
	deal := &models.Deal{
		Title:     "Contact Deal",
		Stage:     models.StageProspecting,
		CompanyID: company.ID,
		Currency:  "USD",
		ContactID: &contact.ID,
	}
	if err := CreateDeal(db, deal); err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	originalContactLastContacted := contact.LastContactedAt

	// Add note to deal
	note := &models.DealNote{
		DealID:  deal.ID,
		Content: "Initial contact discussion",
	}
	if err := AddDealNote(db, note); err != nil {
		t.Fatalf("AddDealNote failed: %v", err)
	}

	// Reload contact to check timestamp update
	updatedContact, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact after note failed: %v", err)
	}

	if updatedContact.LastContactedAt == nil {
		t.Error("Contact's LastContactedAt was not set after adding note")
	} else if originalContactLastContacted != nil && !updatedContact.LastContactedAt.After(*originalContactLastContacted) {
		t.Error("Contact's LastContactedAt was not updated after adding note")
	} else if originalContactLastContacted == nil {
		// First contact interaction should have a timestamp
		if updatedContact.LastContactedAt.Before(time.Now().Add(-1 * time.Second)) {
			t.Error("Contact's LastContactedAt seems too old")
		}
	}
}

func TestFindDeals(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create company
	company := &models.Company{Name: "Search Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create multiple deals with different stages
	deal1 := &models.Deal{
		Title:     "Deal in Prospecting",
		Stage:     models.StageProspecting,
		CompanyID: company.ID,
		Currency:  "USD",
	}
	if err := CreateDeal(db, deal1); err != nil {
		t.Fatalf("CreateDeal 1 failed: %v", err)
	}

	deal2 := &models.Deal{
		Title:     "Deal in Negotiation",
		Stage:     models.StageNegotiation,
		CompanyID: company.ID,
		Currency:  "USD",
	}
	if err := CreateDeal(db, deal2); err != nil {
		t.Fatalf("CreateDeal 2 failed: %v", err)
	}

	deal3 := &models.Deal{
		Title:     "Deal in Closed",
		Stage:     models.StageClosedWon,
		CompanyID: company.ID,
		Currency:  "USD",
	}
	if err := CreateDeal(db, deal3); err != nil {
		t.Fatalf("CreateDeal 3 failed: %v", err)
	}

	// Test 1: Filter by stage
	prospectingDeals, err := FindDeals(db, models.StageProspecting, nil, 10)
	if err != nil {
		t.Fatalf("FindDeals by stage failed: %v", err)
	}
	if len(prospectingDeals) != 1 {
		t.Errorf("Expected 1 prospecting deal, got %d", len(prospectingDeals))
	}
	if prospectingDeals[0].Stage != models.StageProspecting {
		t.Error("Found deal with wrong stage")
	}

	// Test 2: Filter by company
	companyDeals, err := FindDeals(db, "", &company.ID, 10)
	if err != nil {
		t.Fatalf("FindDeals by company failed: %v", err)
	}
	if len(companyDeals) != 3 {
		t.Errorf("Expected 3 deals for company, got %d", len(companyDeals))
	}

	// Test 3: Filter by stage and company
	negotiationDeals, err := FindDeals(db, models.StageNegotiation, &company.ID, 10)
	if err != nil {
		t.Fatalf("FindDeals by stage and company failed: %v", err)
	}
	if len(negotiationDeals) != 1 {
		t.Errorf("Expected 1 negotiation deal for company, got %d", len(negotiationDeals))
	}
	if negotiationDeals[0].Stage != models.StageNegotiation {
		t.Error("Found deal with wrong stage")
	}

	// Test 4: Pagination with limit
	limitedDeals, err := FindDeals(db, "", &company.ID, 2)
	if err != nil {
		t.Fatalf("FindDeals with limit failed: %v", err)
	}
	if len(limitedDeals) != 2 {
		t.Errorf("Expected 2 deals with limit=2, got %d", len(limitedDeals))
	}

	// Test 5: No filters returns all deals
	allDeals, err := FindDeals(db, "", nil, 10)
	if err != nil {
		t.Fatalf("FindDeals without filters failed: %v", err)
	}
	if len(allDeals) < 3 {
		t.Errorf("Expected at least 3 deals total, got %d", len(allDeals))
	}
}
