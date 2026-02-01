// ABOUTME: Tests for SQLite repository implementation
// ABOUTME: Covers CRUD operations for all entity types

package repository

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "pagen-test-*.db")
	require.NoError(t, err)
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	t.Cleanup(func() {
		_ = os.Remove(tmpPath)
		_ = os.Remove(tmpPath + "-wal")
		_ = os.Remove(tmpPath + "-shm")
	})

	db, err := Open(tmpPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestOpenDatabase(t *testing.T) {
	db := setupTestDB(t)
	assert.NotNil(t, db)
}

// ============================================================================
// Contact Tests
// ============================================================================

func TestContactCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create
	contact := &Contact{
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "555-1234",
		Notes: "Test contact",
	}
	err := db.CreateContact(contact)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, contact.ID)
	assert.False(t, contact.CreatedAt.IsZero())

	// Read
	fetched, err := db.GetContact(contact.ID)
	require.NoError(t, err)
	assert.Equal(t, contact.Name, fetched.Name)
	assert.Equal(t, contact.Email, fetched.Email)
	assert.Equal(t, contact.Phone, fetched.Phone)

	// Update
	fetched.Name = "Jane Doe"
	err = db.UpdateContact(fetched)
	require.NoError(t, err)

	updated, err := db.GetContact(contact.ID)
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", updated.Name)

	// Delete
	err = db.DeleteContact(contact.ID)
	require.NoError(t, err)

	_, err = db.GetContact(contact.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListContacts(t *testing.T) {
	db := setupTestDB(t)

	// Create contacts
	for i := 0; i < 5; i++ {
		err := db.CreateContact(&Contact{
			Name:  "Contact " + string(rune('A'+i)),
			Email: "contact" + string(rune('a'+i)) + "@example.com",
		})
		require.NoError(t, err)
	}

	// List all
	contacts, err := db.ListContacts(nil)
	require.NoError(t, err)
	assert.Len(t, contacts, 5)

	// Filter by query
	contacts, err = db.ListContacts(&ContactFilter{Query: "Contact A"})
	require.NoError(t, err)
	assert.Len(t, contacts, 1)
	assert.Equal(t, "Contact A", contacts[0].Name)

	// With limit
	contacts, err = db.ListContacts(&ContactFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, contacts, 2)
}

func TestContactWithCompany(t *testing.T) {
	db := setupTestDB(t)

	// Create company
	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	// Create contact with company
	contact := &Contact{
		Name:        "John Doe",
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	}
	err = db.CreateContact(contact)
	require.NoError(t, err)

	// Fetch and verify
	fetched, err := db.GetContact(contact.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.CompanyID)
	assert.Equal(t, company.ID, *fetched.CompanyID)
	assert.Equal(t, "Acme Corp", fetched.CompanyName)

	// Filter by company
	contacts, err := db.ListContacts(&ContactFilter{CompanyID: &company.ID})
	require.NoError(t, err)
	assert.Len(t, contacts, 1)
}

func TestFindContactByName(t *testing.T) {
	db := setupTestDB(t)

	err := db.CreateContact(&Contact{Name: "John Doe"})
	require.NoError(t, err)

	// Find existing
	contact, err := db.FindContactByName("John Doe")
	require.NoError(t, err)
	require.NotNil(t, contact)
	assert.Equal(t, "John Doe", contact.Name)

	// Not found
	contact, err = db.FindContactByName("Jane Doe")
	require.NoError(t, err)
	assert.Nil(t, contact)
}

// ============================================================================
// Company Tests
// ============================================================================

func TestCompanyCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create
	company := &Company{
		Name:     "Acme Corp",
		Domain:   "acme.com",
		Industry: "Technology",
		Notes:    "Test company",
	}
	err := db.CreateCompany(company)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, company.ID)

	// Read
	fetched, err := db.GetCompany(company.ID)
	require.NoError(t, err)
	assert.Equal(t, company.Name, fetched.Name)
	assert.Equal(t, company.Domain, fetched.Domain)

	// Update
	fetched.Industry = "Software"
	err = db.UpdateCompany(fetched)
	require.NoError(t, err)

	updated, err := db.GetCompany(company.ID)
	require.NoError(t, err)
	assert.Equal(t, "Software", updated.Industry)

	// Delete
	err = db.DeleteCompany(company.ID)
	require.NoError(t, err)

	_, err = db.GetCompany(company.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListCompanies(t *testing.T) {
	db := setupTestDB(t)

	// Create companies
	for i := 0; i < 3; i++ {
		err := db.CreateCompany(&Company{
			Name:     "Company " + string(rune('A'+i)),
			Industry: "Tech",
		})
		require.NoError(t, err)
	}

	// List all
	companies, err := db.ListCompanies(nil)
	require.NoError(t, err)
	assert.Len(t, companies, 3)

	// Filter by industry
	companies, err = db.ListCompanies(&CompanyFilter{Industry: "Tech"})
	require.NoError(t, err)
	assert.Len(t, companies, 3)
}

func TestFindCompanyByName(t *testing.T) {
	db := setupTestDB(t)

	err := db.CreateCompany(&Company{Name: "Acme Corp"})
	require.NoError(t, err)

	company, err := db.FindCompanyByName("Acme Corp")
	require.NoError(t, err)
	require.NotNil(t, company)
	assert.Equal(t, "Acme Corp", company.Name)

	company, err = db.FindCompanyByName("Unknown")
	require.NoError(t, err)
	assert.Nil(t, company)
}

// ============================================================================
// Deal Tests
// ============================================================================

func TestDealCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create company first
	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	// Create deal
	deal := &Deal{
		Title:       "Big Deal",
		Amount:      100000,
		Currency:    "USD",
		Stage:       StageProspecting,
		CompanyID:   company.ID,
		CompanyName: company.Name,
	}
	err = db.CreateDeal(deal)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, deal.ID)

	// Read
	fetched, err := db.GetDeal(deal.ID)
	require.NoError(t, err)
	assert.Equal(t, deal.Title, fetched.Title)
	assert.Equal(t, deal.Amount, fetched.Amount)

	// Update
	fetched.Stage = StageNegotiation
	err = db.UpdateDeal(fetched)
	require.NoError(t, err)

	updated, err := db.GetDeal(deal.ID)
	require.NoError(t, err)
	assert.Equal(t, StageNegotiation, updated.Stage)

	// Delete
	err = db.DeleteDeal(deal.ID)
	require.NoError(t, err)

	_, err = db.GetDeal(deal.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListDeals(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	// Create deals
	for i := 0; i < 3; i++ {
		err := db.CreateDeal(&Deal{
			Title:       "Deal " + string(rune('A'+i)),
			CompanyID:   company.ID,
			CompanyName: company.Name,
			Stage:       StageProspecting,
			Amount:      int64((i + 1) * 10000),
		})
		require.NoError(t, err)
	}

	// List all
	deals, err := db.ListDeals(nil)
	require.NoError(t, err)
	assert.Len(t, deals, 3)

	// Filter by stage
	deals, err = db.ListDeals(&DealFilter{Stage: StageProspecting})
	require.NoError(t, err)
	assert.Len(t, deals, 3)

	// Filter by amount
	deals, err = db.ListDeals(&DealFilter{MinAmount: 20000})
	require.NoError(t, err)
	assert.Len(t, deals, 2)
}

// ============================================================================
// DealNote Tests
// ============================================================================

func TestDealNoteCRUD(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	deal := &Deal{
		Title:     "Big Deal",
		CompanyID: company.ID,
		Stage:     StageProspecting,
	}
	err = db.CreateDeal(deal)
	require.NoError(t, err)

	// Create note
	note := &DealNote{
		DealID:  deal.ID,
		Content: "Initial meeting went well",
	}
	err = db.CreateDealNote(note)
	require.NoError(t, err)

	// Read
	fetched, err := db.GetDealNote(note.ID)
	require.NoError(t, err)
	assert.Equal(t, note.Content, fetched.Content)

	// List
	notes, err := db.ListDealNotes(deal.ID)
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Delete
	err = db.DeleteDealNote(note.ID)
	require.NoError(t, err)

	_, err = db.GetDealNote(note.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// Relationship Tests
// ============================================================================

func TestRelationshipCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create contacts
	contact1 := &Contact{Name: "Alice"}
	contact2 := &Contact{Name: "Bob"}
	err := db.CreateContact(contact1)
	require.NoError(t, err)
	err = db.CreateContact(contact2)
	require.NoError(t, err)

	// Create relationship
	rel := &Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		Contact1Name:     contact1.Name,
		Contact2Name:     contact2.Name,
		RelationshipType: "colleague",
		Context:          "Work together",
	}
	err = db.CreateRelationship(rel)
	require.NoError(t, err)

	// Read
	fetched, err := db.GetRelationship(rel.ID)
	require.NoError(t, err)
	assert.Equal(t, rel.RelationshipType, fetched.RelationshipType)

	// List for contact
	rels, err := db.ListRelationshipsForContact(contact1.ID)
	require.NoError(t, err)
	assert.Len(t, rels, 1)

	// Get between contacts
	found, err := db.GetRelationshipBetween(contact1.ID, contact2.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, rel.ID, found.ID)

	// Delete
	err = db.DeleteRelationship(rel.ID)
	require.NoError(t, err)

	_, err = db.GetRelationship(rel.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// InteractionLog Tests
// ============================================================================

func TestInteractionLogCRUD(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{Name: "John Doe"}
	err := db.CreateContact(contact)
	require.NoError(t, err)

	// Create
	log := &InteractionLog{
		ContactID:       contact.ID,
		ContactName:     contact.Name,
		InteractionType: InteractionMeeting,
		Notes:           "Discussed project",
	}
	err = db.CreateInteractionLog(log)
	require.NoError(t, err)

	// Read
	fetched, err := db.GetInteractionLog(log.ID)
	require.NoError(t, err)
	assert.Equal(t, log.Notes, fetched.Notes)

	// List
	logs, err := db.ListInteractionLogs(&InteractionFilter{
		ContactID: &contact.ID,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	// Delete
	err = db.DeleteInteractionLog(log.ID)
	require.NoError(t, err)

	_, err = db.GetInteractionLog(log.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// ContactCadence Tests
// ============================================================================

func TestContactCadenceCRUD(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{Name: "John Doe"}
	err := db.CreateContact(contact)
	require.NoError(t, err)

	// Save
	now := time.Now()
	next := now.AddDate(0, 0, 14)
	cadence := &ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          14,
		RelationshipStrength: StrengthStrong,
		PriorityScore:        10.5,
		LastInteractionDate:  &now,
		NextFollowupDate:     &next,
	}
	err = db.SaveContactCadence(cadence)
	require.NoError(t, err)

	// Read
	fetched, err := db.GetContactCadence(contact.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, cadence.CadenceDays, fetched.CadenceDays)
	assert.Equal(t, cadence.RelationshipStrength, fetched.RelationshipStrength)

	// List
	cadences, err := db.ListContactCadences()
	require.NoError(t, err)
	assert.Len(t, cadences, 1)

	// Delete
	err = db.DeleteContactCadence(contact.ID)
	require.NoError(t, err)

	fetched, err = db.GetContactCadence(contact.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestGetFollowupList(t *testing.T) {
	db := setupTestDB(t)

	// Create contact with cadence
	contact := &Contact{Name: "John Doe"}
	err := db.CreateContact(contact)
	require.NoError(t, err)

	// Past interaction date to create priority
	pastDate := time.Now().AddDate(0, 0, -45)
	cadence := &ContactCadence{
		ContactID:            contact.ID,
		ContactName:          contact.Name,
		CadenceDays:          30,
		RelationshipStrength: StrengthMedium,
		PriorityScore:        15.0,
		LastInteractionDate:  &pastDate,
	}
	err = db.SaveContactCadence(cadence)
	require.NoError(t, err)

	// Get followup list
	followups, err := db.GetFollowupList(10)
	require.NoError(t, err)
	assert.Len(t, followups, 1)
	assert.Equal(t, contact.ID, followups[0].ID)
	assert.Equal(t, 15.0, followups[0].PriorityScore)
}

// ============================================================================
// Suggestion Tests
// ============================================================================

func TestSuggestionCRUD(t *testing.T) {
	db := setupTestDB(t)

	suggestion := &Suggestion{
		Type:          SuggestionTypeDeal,
		Confidence:    0.85,
		SourceService: "gmail",
		Status:        SuggestionStatusPending,
	}
	err := db.CreateSuggestion(suggestion)
	require.NoError(t, err)

	// Read
	fetched, err := db.GetSuggestion(suggestion.ID)
	require.NoError(t, err)
	assert.Equal(t, suggestion.Confidence, fetched.Confidence)

	// Update
	fetched.Status = SuggestionStatusAccepted
	now := time.Now()
	fetched.ReviewedAt = &now
	err = db.UpdateSuggestion(fetched)
	require.NoError(t, err)

	// List
	suggestions, err := db.ListSuggestions(&SuggestionFilter{
		Status: SuggestionStatusAccepted,
	})
	require.NoError(t, err)
	assert.Len(t, suggestions, 1)

	// Delete
	err = db.DeleteSuggestion(suggestion.ID)
	require.NoError(t, err)

	_, err = db.GetSuggestion(suggestion.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// SyncState Tests
// ============================================================================

func TestSyncStateCRUD(t *testing.T) {
	db := setupTestDB(t)

	state := &SyncState{
		Service: "google-contacts",
		Status:  SyncStatusIdle,
	}
	err := db.SaveSyncState(state)
	require.NoError(t, err)

	// Read
	fetched, err := db.GetSyncState("google-contacts")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, SyncStatusIdle, fetched.Status)

	// Update
	now := time.Now()
	state.LastSyncTime = &now
	state.Status = SyncStatusSyncing
	err = db.SaveSyncState(state)
	require.NoError(t, err)

	updated, err := db.GetSyncState("google-contacts")
	require.NoError(t, err)
	assert.Equal(t, SyncStatusSyncing, updated.Status)
	require.NotNil(t, updated.LastSyncTime)
}

// ============================================================================
// SyncLog Tests
// ============================================================================

func TestSyncLogCRUD(t *testing.T) {
	db := setupTestDB(t)

	entityID := uuid.New()
	log := &SyncLog{
		SourceService: "google-contacts",
		SourceID:      "people/123",
		EntityType:    "contact",
		EntityID:      entityID,
	}
	err := db.CreateSyncLog(log)
	require.NoError(t, err)

	// Find by source
	found, err := db.FindSyncLogBySource("google-contacts", "people/123")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entityID, found.EntityID)

	// Not found
	notFound, err := db.FindSyncLogBySource("google-contacts", "people/999")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

// ============================================================================
// Dashboard Stats Tests
// ============================================================================

func TestGetDashboardStats(t *testing.T) {
	db := setupTestDB(t)

	// Create some data
	for i := 0; i < 3; i++ {
		err := db.CreateContact(&Contact{Name: "Contact " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	err = db.CreateDeal(&Deal{
		Title:     "Deal A",
		CompanyID: company.ID,
		Stage:     StageProspecting,
		Amount:    10000,
	})
	require.NoError(t, err)

	stats, err := db.GetDashboardStats()
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalContacts)
	assert.Equal(t, 1, stats.TotalCompanies)
	assert.Equal(t, 1, stats.TotalDeals)
	assert.Equal(t, int64(10000), stats.TotalPipeline)
}

// ============================================================================
// Export Tests
// ============================================================================

func TestExportAll(t *testing.T) {
	db := setupTestDB(t)

	// Create some data
	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	contact := &Contact{Name: "John Doe", CompanyID: &company.ID}
	err = db.CreateContact(contact)
	require.NoError(t, err)

	deal := &Deal{Title: "Big Deal", CompanyID: company.ID, Stage: StageProspecting}
	err = db.CreateDeal(deal)
	require.NoError(t, err)

	// Export
	data, err := db.ExportAll()
	require.NoError(t, err)
	assert.Equal(t, "1.0", data.Version)
	assert.Equal(t, "pagen", data.Tool)
	assert.Len(t, data.Contacts, 1)
	assert.Len(t, data.Companies, 1)
	assert.Len(t, data.Deals, 1)
}
