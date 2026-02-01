// ABOUTME: Extended tests for Gmail importer helper functions
// ABOUTME: Covers email parsing, domain handling, and utility functions
package sync

import (
	"testing"

	"github.com/harperreed/pagen/db"
)

func TestIsCommonEmailDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"gmail.com", true},
		{"Gmail.com", true},
		{"GMAIL.COM", true},
		{"yahoo.com", true},
		{"hotmail.com", true},
		{"outlook.com", true},
		{"icloud.com", true},
		{"protonmail.com", true},
		{"company.com", false},
		{"acme.io", false},
		{"startup.tech", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isCommonEmailDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isCommonEmailDomain(%q) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestCapitalizeCompanyName(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"acme.com", "Acme"},
		{"tech-startup.com", "Tech Startup"},
		{"my.company.com", "My Company"},
		{"simple", "Simple"},
		{"UPPERCASE.org", "UPPERCASE"},
		{"mixed.CASE.net", "Mixed CASE"},
		{"example.io", "Example"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := capitalizeCompanyName(tt.domain)
			if result != tt.expected {
				t.Errorf("capitalizeCompanyName(%q) = %q, want %q", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestParseEmailDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
	}{
		{
			name:    "RFC1123Z format",
			dateStr: "Mon, 02 Jan 2006 15:04:05 -0700",
			wantErr: false,
		},
		{
			name:    "single digit day with timezone",
			dateStr: "Mon, 2 Jan 2006 15:04:05 -0700",
			wantErr: false,
		},
		{
			name:    "with timezone name",
			dateStr: "Mon, 02 Jan 2006 15:04:05 -0700 (PST)",
			wantErr: false,
		},
		{
			name:    "RFC3339",
			dateStr: "2006-01-02T15:04:05Z",
			wantErr: false,
		},
		{
			name:    "empty string",
			dateStr: "",
			wantErr: false, // returns time.Now()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEmailDate(tt.dateStr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseEmailDate(%q) expected error, got nil", tt.dateStr)
				}
			} else {
				if err != nil {
					t.Errorf("parseEmailDate(%q) unexpected error: %v", tt.dateStr, err)
				}
				if result.IsZero() {
					t.Errorf("parseEmailDate(%q) returned zero time", tt.dateStr)
				}
			}
		})
	}
}

func TestJsonEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", `"simple"`},
		{"with spaces", `"with spaces"`},
		{`with "quotes"`, `"with \"quotes\""`},
		{"with\nnewline", `"with\nnewline"`},
		{"", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := jsonEscape(tt.input)
			if result != tt.expected {
				t.Errorf("jsonEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindOrCreateCompanyFromDomain(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create new company from domain
	company, err := findOrCreateCompanyFromDomain(database, "testcorp.com")
	if err != nil {
		t.Fatalf("findOrCreateCompanyFromDomain failed: %v", err)
	}

	if company == nil {
		t.Fatal("expected company, got nil")
	}

	if company.Name != "Testcorp" {
		t.Errorf("expected company name 'Testcorp', got %s", company.Name)
	}

	if company.Domain != "testcorp.com" {
		t.Errorf("expected domain 'testcorp.com', got %s", company.Domain)
	}

	// Find existing company by domain (via name lookup)
	company2, err := findOrCreateCompanyFromDomain(database, "testcorp.com")
	if err != nil {
		t.Fatalf("second findOrCreateCompanyFromDomain failed: %v", err)
	}

	if company2.ID != company.ID {
		t.Error("expected to find existing company, not create new")
	}
}

func TestFindOrCreateEmailContact(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create matcher with empty contacts
	matcher := NewContactMatcher(nil)

	// Create new contact
	contactID, isNew, err := findOrCreateEmailContact(database, matcher, "Test User", "test@example.com", "example.com")
	if err != nil {
		t.Fatalf("findOrCreateEmailContact failed: %v", err)
	}

	if !isNew {
		t.Error("expected new contact to be created")
	}

	if contactID.String() == "" {
		t.Error("expected valid contact ID")
	}

	// Find existing contact
	contactID2, isNew2, err := findOrCreateEmailContact(database, matcher, "Test User", "test@example.com", "example.com")
	if err != nil {
		t.Fatalf("second findOrCreateEmailContact failed: %v", err)
	}

	if isNew2 {
		t.Error("expected to find existing contact")
	}

	if contactID2 != contactID {
		t.Error("expected same contact ID")
	}
}

func TestFindOrCreateEmailContactNoName(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	matcher := NewContactMatcher(nil)

	// Create contact without name - should use email username
	contactID, isNew, err := findOrCreateEmailContact(database, matcher, "", "noname@example.com", "example.com")
	if err != nil {
		t.Fatalf("findOrCreateEmailContact failed: %v", err)
	}

	if !isNew {
		t.Error("expected new contact")
	}

	// Verify the contact uses email username as name
	contact, err := db.GetContact(database, contactID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if contact.Name != "noname" {
		t.Errorf("expected name 'noname', got %s", contact.Name)
	}
}

func TestFindOrCreateEmailContactWithCompany(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	matcher := NewContactMatcher(nil)

	// Create contact with company domain
	contactID, isNew, err := findOrCreateEmailContact(database, matcher, "Employee", "emp@acme.io", "acme.io")
	if err != nil {
		t.Fatalf("findOrCreateEmailContact failed: %v", err)
	}

	if !isNew {
		t.Error("expected new contact")
	}

	// Verify company was created and associated
	contact, err := db.GetContact(database, contactID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if contact.CompanyID == nil {
		t.Error("expected company to be associated")
	}
}

func TestFindOrCreateEmailContactCommonDomain(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	matcher := NewContactMatcher(nil)

	// Create contact with gmail domain - should NOT create company
	contactID, _, err := findOrCreateEmailContact(database, matcher, "Personal User", "user@gmail.com", "gmail.com")
	if err != nil {
		t.Fatalf("findOrCreateEmailContact failed: %v", err)
	}

	// Verify no company associated
	contact, err := db.GetContact(database, contactID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if contact.CompanyID != nil {
		t.Error("expected no company for gmail domain")
	}
}

func TestIsCommonEmailDomainAllVariants(t *testing.T) {
	// Test all common domains listed in the function
	commonDomains := []string{
		"gmail.com",
		"googlemail.com",
		"yahoo.com",
		"hotmail.com",
		"outlook.com",
		"live.com",
		"msn.com",
		"icloud.com",
		"me.com",
		"mac.com",
		"aol.com",
		"protonmail.com",
		"pm.me",
	}

	for _, domain := range commonDomains {
		t.Run(domain, func(t *testing.T) {
			if !isCommonEmailDomain(domain) {
				t.Errorf("isCommonEmailDomain(%q) = false, want true", domain)
			}
		})
	}
}

func TestParseEmailDateInvalidFormat(t *testing.T) {
	// Test invalid date format - should return time.Now() and error
	result, err := parseEmailDate("not-a-valid-date-format")
	if err == nil {
		t.Error("expected error for invalid date format")
	}
	// Should still return a time (time.Now())
	if result.IsZero() {
		t.Error("should return current time even for invalid format")
	}
}

func TestJsonEscapeSpecialChars(t *testing.T) {
	// Test various special characters
	tests := []struct {
		input    string
		expected string
	}{
		{"tab\there", `"tab\there"`},
		{"backslash\\here", `"backslash\\here"`},
		{"unicode \u0000", `"unicode \u0000"`},
	}

	for _, tt := range tests {
		result := jsonEscape(tt.input)
		if result != tt.expected {
			t.Errorf("jsonEscape(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFindOrCreateEmailContactEmptyDomain(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	matcher := NewContactMatcher(nil)

	// Create contact with empty domain - should not create company
	contactID, isNew, err := findOrCreateEmailContact(database, matcher, "No Domain", "nodomain@test.com", "")
	if err != nil {
		t.Fatalf("findOrCreateEmailContact failed: %v", err)
	}

	if !isNew {
		t.Error("expected new contact")
	}

	// Verify no company associated
	contact, err := db.GetContact(database, contactID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if contact.CompanyID != nil {
		t.Error("expected no company for empty domain")
	}
}
