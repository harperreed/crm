// ABOUTME: Tests for legacy adapter conversions
// ABOUTME: Validates bidirectional conversion between legacy models and Office OS objects
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func TestCompanyToObject(t *testing.T) {
	now := time.Now()
	company := &models.Company{
		ID:        uuid.New(),
		Name:      "Test Corp",
		Domain:    "test.com",
		Industry:  "Technology",
		Notes:     "Test notes",
		CreatedAt: now,
		UpdatedAt: now,
	}

	obj := CompanyToObject(company)

	if obj.ID != company.ID.String() {
		t.Errorf("Expected ID %s, got %s", company.ID.String(), obj.ID)
	}
	if obj.Kind != ObjectTypeCompany {
		t.Errorf("Expected kind %s, got %s", ObjectTypeCompany, obj.Kind)
	}
	if obj.Fields["name"] != company.Name {
		t.Errorf("Expected name %s, got %v", company.Name, obj.Fields["name"])
	}
	if obj.Fields["domain"] != company.Domain {
		t.Errorf("Expected domain %s, got %v", company.Domain, obj.Fields["domain"])
	}
}

func TestObjectToCompany(t *testing.T) {
	now := time.Now()
	obj := &Object{
		ID:        uuid.New().String(),
		Kind:      ObjectTypeCompany,
		CreatedAt: now,
		UpdatedAt: now,
		Fields: map[string]interface{}{
			"name":     "Test Corp",
			"domain":   "test.com",
			"industry": "Tech",
			"notes":    "Notes",
		},
	}

	company, err := ObjectToCompany(obj)
	if err != nil {
		t.Fatalf("ObjectToCompany failed: %v", err)
	}

	if company.Name != "Test Corp" {
		t.Errorf("Expected name 'Test Corp', got %s", company.Name)
	}
	if company.Domain != "test.com" {
		t.Errorf("Expected domain 'test.com', got %s", company.Domain)
	}
}

func TestObjectToCompanyWrongKind(t *testing.T) {
	obj := &Object{
		ID:     uuid.New().String(),
		Kind:   ObjectTypeContact,
		Fields: map[string]interface{}{},
	}

	_, err := ObjectToCompany(obj)
	if err == nil {
		t.Error("Expected error for wrong kind")
	}
}

func TestObjectToCompanyInvalidID(t *testing.T) {
	obj := &Object{
		ID:     "invalid-uuid",
		Kind:   ObjectTypeCompany,
		Fields: map[string]interface{}{},
	}

	_, err := ObjectToCompany(obj)
	if err == nil {
		t.Error("Expected error for invalid ID")
	}
}

func TestContactToObject(t *testing.T) {
	now := time.Now()
	companyID := uuid.New()
	contact := &models.Contact{
		ID:              uuid.New(),
		Name:            "John Doe",
		Email:           "john@example.com",
		Phone:           "555-1234",
		Notes:           "Test contact",
		CompanyID:       &companyID,
		LastContactedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	obj := ContactToObject(contact)

	if obj.ID != contact.ID.String() {
		t.Errorf("Expected ID %s, got %s", contact.ID.String(), obj.ID)
	}
	if obj.Kind != ObjectTypeContact {
		t.Errorf("Expected kind %s, got %s", ObjectTypeContact, obj.Kind)
	}
	if obj.Fields["name"] != contact.Name {
		t.Errorf("Expected name %s, got %v", contact.Name, obj.Fields["name"])
	}
	if obj.Fields["company_id"] != companyID.String() {
		t.Errorf("Expected company_id %s, got %v", companyID.String(), obj.Fields["company_id"])
	}
}

func TestContactToObjectWithoutCompany(t *testing.T) {
	now := time.Now()
	contact := &models.Contact{
		ID:        uuid.New(),
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	obj := ContactToObject(contact)

	if obj.Fields["company_id"] != nil {
		t.Errorf("Expected company_id to be nil, got %v", obj.Fields["company_id"])
	}
}

func TestObjectToContact(t *testing.T) {
	now := time.Now()
	companyID := uuid.New()
	obj := &Object{
		ID:        uuid.New().String(),
		Kind:      ObjectTypeContact,
		CreatedAt: now,
		UpdatedAt: now,
		Fields: map[string]interface{}{
			"name":              "John Doe",
			"email":             "john@example.com",
			"phone":             "555-1234",
			"notes":             "Notes",
			"company_id":        companyID.String(),
			"last_contacted_at": now.Format(time.RFC3339Nano),
		},
	}

	contact, err := ObjectToContact(obj)
	if err != nil {
		t.Fatalf("ObjectToContact failed: %v", err)
	}

	if contact.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %s", contact.Name)
	}
	if contact.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %s", contact.Email)
	}
	if contact.CompanyID == nil || *contact.CompanyID != companyID {
		t.Errorf("Expected company ID %s", companyID.String())
	}
}

func TestObjectToContactWrongKind(t *testing.T) {
	obj := &Object{
		ID:     uuid.New().String(),
		Kind:   ObjectTypeCompany,
		Fields: map[string]interface{}{},
	}

	_, err := ObjectToContact(obj)
	if err == nil {
		t.Error("Expected error for wrong kind")
	}
}

func TestObjectToContactInvalidID(t *testing.T) {
	obj := &Object{
		ID:     "invalid-uuid",
		Kind:   ObjectTypeContact,
		Fields: map[string]interface{}{},
	}

	_, err := ObjectToContact(obj)
	if err == nil {
		t.Error("Expected error for invalid ID")
	}
}

func TestDealToObject(t *testing.T) {
	now := time.Now()
	companyID := uuid.New()
	contactID := uuid.New()
	closeDate := now.AddDate(0, 1, 0)

	deal := &models.Deal{
		ID:                uuid.New(),
		Title:             "Big Deal",
		Amount:            100000,
		Currency:          "USD",
		Stage:             "prospecting",
		CompanyID:         companyID,
		ContactID:         &contactID,
		ExpectedCloseDate: &closeDate,
		CreatedAt:         now,
		UpdatedAt:         now,
		LastActivityAt:    now,
	}

	obj := DealToObject(deal)

	if obj.ID != deal.ID.String() {
		t.Errorf("Expected ID %s, got %s", deal.ID.String(), obj.ID)
	}
	if obj.Kind != ObjectTypeDeal {
		t.Errorf("Expected kind %s, got %s", ObjectTypeDeal, obj.Kind)
	}
	if obj.Fields["title"] != deal.Title {
		t.Errorf("Expected title %s, got %v", deal.Title, obj.Fields["title"])
	}
}

func TestDealToObjectWithoutContact(t *testing.T) {
	now := time.Now()
	companyID := uuid.New()

	deal := &models.Deal{
		ID:        uuid.New(),
		Title:     "Simple Deal",
		Amount:    50000,
		Currency:  "USD",
		Stage:     "prospecting",
		CompanyID: companyID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	obj := DealToObject(deal)

	if obj.Fields["contact_id"] != nil {
		t.Errorf("Expected contact_id to be nil, got %v", obj.Fields["contact_id"])
	}
}

func TestObjectToDeal(t *testing.T) {
	now := time.Now()
	companyID := uuid.New()
	contactID := uuid.New()
	closeDate := now.AddDate(0, 1, 0)

	obj := &Object{
		ID:        uuid.New().String(),
		Kind:      ObjectTypeDeal,
		CreatedAt: now,
		UpdatedAt: now,
		Fields: map[string]interface{}{
			"title":               "Big Deal",
			"amount":              float64(100000),
			"currency":            "USD",
			"stage":               "prospecting",
			"company_id":          companyID.String(),
			"contact_id":          contactID.String(),
			"expected_close_date": closeDate.Format(time.RFC3339Nano),
			"last_activity_at":    now.Format(time.RFC3339Nano),
		},
	}

	deal, err := ObjectToDeal(obj)
	if err != nil {
		t.Fatalf("ObjectToDeal failed: %v", err)
	}

	if deal.Title != "Big Deal" {
		t.Errorf("Expected title 'Big Deal', got %s", deal.Title)
	}
	if deal.Amount != 100000 {
		t.Errorf("Expected amount 100000, got %d", deal.Amount)
	}
	if deal.CompanyID != companyID {
		t.Errorf("Expected company ID %s", companyID.String())
	}
	if deal.ContactID == nil || *deal.ContactID != contactID {
		t.Errorf("Expected contact ID %s", contactID.String())
	}
}

func TestObjectToDealWrongKind(t *testing.T) {
	obj := &Object{
		ID:     uuid.New().String(),
		Kind:   ObjectTypeCompany,
		Fields: map[string]interface{}{},
	}

	_, err := ObjectToDeal(obj)
	if err == nil {
		t.Error("Expected error for wrong kind")
	}
}

func TestObjectToDealInvalidID(t *testing.T) {
	obj := &Object{
		ID:     "invalid-uuid",
		Kind:   ObjectTypeDeal,
		Fields: map[string]interface{}{},
	}

	_, err := ObjectToDeal(obj)
	if err == nil {
		t.Error("Expected error for invalid ID")
	}
}

func TestGetStringFromMetadata(t *testing.T) {
	fields := map[string]interface{}{
		"name":   "Test",
		"number": 123,
		"nil":    nil,
	}

	if getStringFromMetadata(fields, "name") != "Test" {
		t.Error("Expected 'Test' for string field")
	}
	if getStringFromMetadata(fields, "number") != "" {
		t.Error("Expected empty string for non-string field")
	}
	if getStringFromMetadata(fields, "nil") != "" {
		t.Error("Expected empty string for nil field")
	}
	if getStringFromMetadata(fields, "missing") != "" {
		t.Error("Expected empty string for missing field")
	}
}
