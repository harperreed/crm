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

func TestExportToJSON(t *testing.T) {
	db := setupTestDB(t)

	// Create data
	company := &Company{Name: "Acme Corp"}
	err := db.CreateCompany(company)
	require.NoError(t, err)

	jsonData, err := db.ExportToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "Acme Corp")
	assert.Contains(t, string(jsonData), "pagen")
}

func TestConfig(t *testing.T) {
	db := setupTestDB(t)

	config := db.Config()
	require.NotNil(t, config)
	assert.Equal(t, "sqlite://local", config.Host)
}

// ============================================================================
// Additional Contact Tests
// ============================================================================

func TestUpdateNonexistentContact(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{
		ID:   uuid.New(),
		Name: "Nonexistent",
	}

	err := db.UpdateContact(contact)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteNonexistentContact(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteContact(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetNonexistentContact(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetContact(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestContactWithLastContactedAt(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()
	contact := &Contact{
		Name:            "John Doe",
		LastContactedAt: &now,
	}

	err := db.CreateContact(contact)
	require.NoError(t, err)

	fetched, err := db.GetContact(contact.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.LastContactedAt)
}

// ============================================================================
// Additional Company Tests
// ============================================================================

func TestUpdateNonexistentCompany(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{
		ID:   uuid.New(),
		Name: "Nonexistent",
	}

	err := db.UpdateCompany(company)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteNonexistentCompany(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteCompany(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListCompaniesWithQuery(t *testing.T) {
	db := setupTestDB(t)

	_ = db.CreateCompany(&Company{Name: "Acme Corp", Domain: "acme.com"})
	_ = db.CreateCompany(&Company{Name: "Beta Inc", Domain: "beta.io"})

	companies, err := db.ListCompanies(&CompanyFilter{Query: "acme"})
	require.NoError(t, err)
	assert.Len(t, companies, 1)
	assert.Equal(t, "Acme Corp", companies[0].Name)
}

// ============================================================================
// Additional Deal Tests
// ============================================================================

func TestUpdateNonexistentDeal(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{Name: "Test"}
	_ = db.CreateCompany(company)

	deal := &Deal{
		ID:        uuid.New(),
		Title:     "Nonexistent",
		CompanyID: company.ID,
		Stage:     StageProspecting,
	}

	err := db.UpdateDeal(deal)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteNonexistentDeal(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteDeal(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDealWithContact(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	contact := &Contact{Name: "John Doe", CompanyID: &company.ID}
	_ = db.CreateContact(contact)

	closeDate := time.Now().AddDate(0, 1, 0)
	deal := &Deal{
		Title:             "Big Deal",
		CompanyID:         company.ID,
		ContactID:         &contact.ID,
		ContactName:       contact.Name,
		Stage:             StageProspecting,
		ExpectedCloseDate: &closeDate,
	}

	err := db.CreateDeal(deal)
	require.NoError(t, err)

	fetched, err := db.GetDeal(deal.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.ContactID)
	assert.Equal(t, contact.ID, *fetched.ContactID)
	require.NotNil(t, fetched.ExpectedCloseDate)
}

func TestListDealsWithFilters(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{Name: "Test Corp"}
	_ = db.CreateCompany(company)

	contact := &Contact{Name: "John"}
	_ = db.CreateContact(contact)

	_ = db.CreateDeal(&Deal{Title: "Deal A", CompanyID: company.ID, ContactID: &contact.ID, Stage: StageProspecting, Amount: 10000})
	_ = db.CreateDeal(&Deal{Title: "Deal B", CompanyID: company.ID, Stage: StageNegotiation, Amount: 50000})
	_ = db.CreateDeal(&Deal{Title: "Deal C", CompanyID: company.ID, Stage: StageProspecting, Amount: 100000})

	// Filter by contact
	deals, err := db.ListDeals(&DealFilter{ContactID: &contact.ID})
	require.NoError(t, err)
	assert.Len(t, deals, 1)

	// Filter by max amount
	deals, err = db.ListDeals(&DealFilter{MaxAmount: 50000})
	require.NoError(t, err)
	assert.Len(t, deals, 2)

	// Filter by query
	deals, err = db.ListDeals(&DealFilter{Query: "Deal A"})
	require.NoError(t, err)
	assert.Len(t, deals, 1)
}

// ============================================================================
// Additional DealNote Tests
// ============================================================================

func TestDeleteNonexistentDealNote(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteDealNote(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetNonexistentDealNote(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetDealNote(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// Additional Relationship Tests
// ============================================================================

func TestUpdateNonexistentRelationship(t *testing.T) {
	db := setupTestDB(t)

	rel := &Relationship{
		ID:         uuid.New(),
		ContactID1: uuid.New(),
		ContactID2: uuid.New(),
	}

	err := db.UpdateRelationship(rel)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteNonexistentRelationship(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteRelationship(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListRelationshipsWithFilter(t *testing.T) {
	db := setupTestDB(t)

	contact1 := &Contact{Name: "Alice"}
	contact2 := &Contact{Name: "Bob"}
	contact3 := &Contact{Name: "Charlie"}
	_ = db.CreateContact(contact1)
	_ = db.CreateContact(contact2)
	_ = db.CreateContact(contact3)

	_ = db.CreateRelationship(&Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: "colleague",
	})
	_ = db.CreateRelationship(&Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact3.ID,
		RelationshipType: "friend",
	})

	// Filter by contact
	rels, err := db.ListRelationships(&RelationshipFilter{ContactID: &contact1.ID})
	require.NoError(t, err)
	assert.Len(t, rels, 2)

	// Filter by type
	rels, err = db.ListRelationships(&RelationshipFilter{RelationshipType: "colleague"})
	require.NoError(t, err)
	assert.Len(t, rels, 1)

	// Filter with limit
	rels, err = db.ListRelationships(&RelationshipFilter{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, rels, 1)
}

func TestGetRelationshipBetweenNotFound(t *testing.T) {
	db := setupTestDB(t)

	rel, err := db.GetRelationshipBetween(uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, rel)
}

// ============================================================================
// Additional InteractionLog Tests
// ============================================================================

func TestDeleteNonexistentInteractionLog(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteInteractionLog(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetNonexistentInteractionLog(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetInteractionLog(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestInteractionLogWithSentiment(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{Name: "John"}
	_ = db.CreateContact(contact)

	sentiment := "positive"
	log := &InteractionLog{
		ContactID:       contact.ID,
		InteractionType: InteractionMeeting,
		Notes:           "Great meeting",
		Sentiment:       &sentiment,
	}

	err := db.CreateInteractionLog(log)
	require.NoError(t, err)

	fetched, err := db.GetInteractionLog(log.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.Sentiment)
	assert.Equal(t, "positive", *fetched.Sentiment)
}

func TestListInteractionLogsWithFilters(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{Name: "John"}
	_ = db.CreateContact(contact)

	sentiment := SentimentPositive
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	_ = db.CreateInteractionLog(&InteractionLog{
		ContactID:       contact.ID,
		InteractionType: InteractionMeeting,
		Timestamp:       now,
	})
	_ = db.CreateInteractionLog(&InteractionLog{
		ContactID:       contact.ID,
		InteractionType: InteractionEmail,
		Timestamp:       yesterday,
		Sentiment:       &sentiment,
	})

	// Filter by type
	logs, err := db.ListInteractionLogs(&InteractionFilter{InteractionType: InteractionMeeting})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	// Filter by since
	logs, err = db.ListInteractionLogs(&InteractionFilter{Since: &now})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	// Filter by sentiment
	logs, err = db.ListInteractionLogs(&InteractionFilter{Sentiment: SentimentPositive})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

// ============================================================================
// Additional Suggestion Tests
// ============================================================================

func TestDeleteNonexistentSuggestion(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteSuggestion(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateNonexistentSuggestion(t *testing.T) {
	db := setupTestDB(t)

	suggestion := &Suggestion{
		ID:            uuid.New(),
		Type:          SuggestionTypeDeal,
		SourceService: "test",
		Status:        SuggestionStatusPending,
	}

	err := db.UpdateSuggestion(suggestion)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListSuggestionsWithFilters(t *testing.T) {
	db := setupTestDB(t)

	_ = db.CreateSuggestion(&Suggestion{Type: SuggestionTypeDeal, Confidence: 0.9, SourceService: "gmail", Status: SuggestionStatusPending})
	_ = db.CreateSuggestion(&Suggestion{Type: SuggestionTypeCompany, Confidence: 0.5, SourceService: "gmail", Status: SuggestionStatusAccepted})

	// Filter by type
	suggestions, err := db.ListSuggestions(&SuggestionFilter{Type: SuggestionTypeDeal})
	require.NoError(t, err)
	assert.Len(t, suggestions, 1)

	// Filter by min confidence
	suggestions, err = db.ListSuggestions(&SuggestionFilter{MinConfidence: 0.8})
	require.NoError(t, err)
	assert.Len(t, suggestions, 1)

	// Filter with limit
	suggestions, err = db.ListSuggestions(&SuggestionFilter{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, suggestions, 1)
}

// ============================================================================
// Additional ContactCadence Tests
// ============================================================================

func TestUpdateCadenceAfterInteraction(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{Name: "John"}
	_ = db.CreateContact(contact)

	// Create initial cadence
	_ = db.SaveContactCadence(&ContactCadence{
		ContactID:            contact.ID,
		CadenceDays:          14,
		RelationshipStrength: StrengthStrong,
	})

	// Update after interaction
	err := db.UpdateCadenceAfterInteraction(contact.ID, time.Now())
	require.NoError(t, err)

	// Verify
	cadence, err := db.GetContactCadence(contact.ID)
	require.NoError(t, err)
	require.NotNil(t, cadence)
	require.NotNil(t, cadence.LastInteractionDate)
	require.NotNil(t, cadence.NextFollowupDate)
}

func TestUpdateCadenceAfterInteractionCreatesNew(t *testing.T) {
	db := setupTestDB(t)

	contact := &Contact{Name: "John"}
	_ = db.CreateContact(contact)

	// Update without existing cadence should create one
	err := db.UpdateCadenceAfterInteraction(contact.ID, time.Now())
	require.NoError(t, err)

	cadence, err := db.GetContactCadence(contact.ID)
	require.NoError(t, err)
	require.NotNil(t, cadence)
	assert.Equal(t, 30, cadence.CadenceDays)                      // Default
	assert.Equal(t, StrengthMedium, cadence.RelationshipStrength) // Default
}

// ============================================================================
// SyncState Non-existent Tests
// ============================================================================

func TestGetNonexistentSyncState(t *testing.T) {
	db := setupTestDB(t)

	state, err := db.GetSyncState("nonexistent-service")
	require.NoError(t, err)
	assert.Nil(t, state)
}

// ============================================================================
// SyncLog Non-existent Tests
// ============================================================================

func TestFindNonexistentSyncLog(t *testing.T) {
	db := setupTestDB(t)

	log, err := db.FindSyncLogBySource("nonexistent", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, log)
}

// ============================================================================
// GetAllHelpers Tests
// ============================================================================

func TestGetAllContacts(t *testing.T) {
	db := setupTestDB(t)

	_ = db.CreateContact(&Contact{Name: "Alice"})
	_ = db.CreateContact(&Contact{Name: "Bob"})

	contacts, err := db.GetAllContacts()
	require.NoError(t, err)
	assert.Len(t, contacts, 2)
}

func TestGetAllCompanies(t *testing.T) {
	db := setupTestDB(t)

	_ = db.CreateCompany(&Company{Name: "Acme"})
	_ = db.CreateCompany(&Company{Name: "Beta"})

	companies, err := db.GetAllCompanies()
	require.NoError(t, err)
	assert.Len(t, companies, 2)
}

func TestGetAllDeals(t *testing.T) {
	db := setupTestDB(t)

	company := &Company{Name: "Acme"}
	_ = db.CreateCompany(company)

	_ = db.CreateDeal(&Deal{Title: "Deal 1", CompanyID: company.ID, Stage: StageProspecting})
	_ = db.CreateDeal(&Deal{Title: "Deal 2", CompanyID: company.ID, Stage: StageProspecting})

	deals, err := db.GetAllDeals()
	require.NoError(t, err)
	assert.Len(t, deals, 2)
}

func TestGetAllRelationships(t *testing.T) {
	db := setupTestDB(t)

	c1 := &Contact{Name: "Alice"}
	c2 := &Contact{Name: "Bob"}
	_ = db.CreateContact(c1)
	_ = db.CreateContact(c2)

	_ = db.CreateRelationship(&Relationship{ContactID1: c1.ID, ContactID2: c2.ID})

	rels, err := db.GetAllRelationships()
	require.NoError(t, err)
	assert.Len(t, rels, 1)
}
