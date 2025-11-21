# CRM MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a simple CRM MCP server with contacts, companies, and deals management via stdio transport.

**Architecture:** Go binary communicating via stdio using MCP SDK, storing data in SQLite at XDG data directory, exposing natural language tools for CRUD operations and flexible querying.

**Tech Stack:** Go 1.21+, go-sdk (MCP), go-sqlite3, google/uuid, adrg/xdg

---

## Task 1: Initialize Go Project

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `.gitignore`

**Step 1: Initialize Go module**

Run:
```bash
go mod init github.com/harperreed/crm-mcp
```

Expected: Creates `go.mod` file

**Step 2: Add dependencies**

Run:
```bash
go get github.com/modelcontextprotocol/go-sdk@latest
go get github.com/mattn/go-sqlite3@latest
go get github.com/google/uuid@latest
go get github.com/adrg/xdg@latest
```

Expected: Dependencies added to `go.mod`

**Step 3: Create minimal main.go**

Create `main.go`:
```go
// ABOUTME: Entry point for CRM MCP server
// ABOUTME: Initializes database and starts MCP server on stdio
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "CRM MCP Server starting...")
	// TODO: Initialize database
	// TODO: Start MCP server
}
```

**Step 4: Create .gitignore**

Create `.gitignore`:
```
# Binaries
crm-mcp
*.exe
*.dll
*.so
*.dylib

# Test binaries
*.test
*.out

# Go workspace
/vendor/
go.work
go.work.sum

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
```

**Step 5: Verify build**

Run:
```bash
go build -o crm-mcp
./crm-mcp
```

Expected: Prints "CRM MCP Server starting..." to stderr

**Step 6: Commit**

Run:
```bash
jj describe -m "feat: initialize Go project structure

- Add go.mod with MCP SDK and dependencies
- Create minimal main.go entry point
- Add .gitignore for Go artifacts"
```

---

## Task 2: Database Models and Schema

**Files:**
- Create: `models/types.go`
- Create: `db/schema.go`
- Create: `db/schema_test.go`

**Step 1: Write test for schema creation**

Create `db/schema_test.go`:
```go
// ABOUTME: Tests for database schema creation and migrations
// ABOUTME: Uses in-memory SQLite for fast isolated tests
package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify tables exist
	tables := []string{"contacts", "companies", "deals", "deal_notes"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s not found: %v", table, err)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./db -v
```

Expected: FAIL with "undefined: InitSchema"

**Step 3: Create models**

Create `models/types.go`:
```go
// ABOUTME: Data models for CRM entities
// ABOUTME: Defines Contact, Company, Deal, and DealNote structs
package models

import (
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	CompanyID       *uuid.UUID `json:"company_id,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	LastContactedAt *time.Time `json:"last_contacted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Company struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain,omitempty"`
	Industry  string    `json:"industry,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Deal struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	Amount            int64      `json:"amount,omitempty"`           // in cents
	Currency          string     `json:"currency"`
	Stage             string     `json:"stage"`
	CompanyID         uuid.UUID  `json:"company_id"`
	ContactID         *uuid.UUID `json:"contact_id,omitempty"`
	ExpectedCloseDate *time.Time `json:"expected_close_date,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
}

type DealNote struct {
	ID        uuid.UUID `json:"id"`
	DealID    uuid.UUID `json:"deal_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	StageProspecting   = "prospecting"
	StageQualification = "qualification"
	StageProposal      = "proposal"
	StageNegotiation   = "negotiation"
	StageClosedWon     = "closed_won"
	StageClosedLost    = "closed_lost"
)
```

**Step 4: Implement schema creation**

Create `db/schema.go`:
```go
// ABOUTME: Database schema definitions and migrations
// ABOUTME: Handles SQLite table creation and initialization
package db

import (
	"database/sql"
)

const schema = `
CREATE TABLE IF NOT EXISTS companies (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	domain TEXT,
	industry TEXT,
	notes TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(name);

CREATE TABLE IF NOT EXISTS contacts (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT,
	phone TEXT,
	company_id TEXT,
	notes TEXT,
	last_contacted_at DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (company_id) REFERENCES companies(id)
);

CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_company_id ON contacts(company_id);

CREATE TABLE IF NOT EXISTS deals (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	amount INTEGER,
	currency TEXT NOT NULL DEFAULT 'USD',
	stage TEXT NOT NULL,
	company_id TEXT NOT NULL,
	contact_id TEXT,
	expected_close_date DATE,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	last_activity_at DATETIME NOT NULL,
	FOREIGN KEY (company_id) REFERENCES companies(id),
	FOREIGN KEY (contact_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(stage);
CREATE INDEX IF NOT EXISTS idx_deals_company_id ON deals(company_id);

CREATE TABLE IF NOT EXISTS deal_notes (
	id TEXT PRIMARY KEY,
	deal_id TEXT NOT NULL,
	content TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (deal_id) REFERENCES deals(id)
);

CREATE INDEX IF NOT EXISTS idx_deal_notes_deal_id ON deal_notes(deal_id);
`

func InitSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
```

**Step 5: Run test to verify it passes**

Run:
```bash
go test ./db -v
```

Expected: PASS

**Step 6: Commit**

Run:
```bash
jj describe -m "feat: add database schema and models

- Define Contact, Company, Deal, DealNote types
- Implement SQLite schema with indexes
- Add tests for schema creation"
```

---

## Task 3: Database Connection and Initialization

**Files:**
- Create: `db/db.go`
- Create: `db/db_test.go`
- Modify: `main.go`

**Step 1: Write test for database initialization**

Create `db/db_test.go`:
```go
package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify schema was initialized
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	if count < 4 {
		t.Errorf("Expected at least 4 tables, got %d", count)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./db -v -run TestOpenDatabase
```

Expected: FAIL with "undefined: OpenDatabase"

**Step 3: Implement database initialization**

Create `db/db.go`:
```go
// ABOUTME: Database connection management and initialization
// ABOUTME: Handles opening SQLite database with WAL mode at XDG path
package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func OpenDatabase(path string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open database with WAL mode
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Initialize schema
	if err := InitSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./db -v -run TestOpenDatabase
```

Expected: PASS

**Step 5: Update main.go to use database**

Modify `main.go`:
```go
// ABOUTME: Entry point for CRM MCP server
// ABOUTME: Initializes database and starts MCP server on stdio
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/harperreed/crm-mcp/db"
)

func main() {
	// Initialize database at XDG data directory
	dbPath := filepath.Join(xdg.DataHome, "crm", "crm.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	fmt.Fprintf(os.Stderr, "CRM MCP Server started. Database: %s\n", dbPath)

	// TODO: Initialize MCP server
	// TODO: Register tools
	// TODO: Start stdio transport
}
```

**Step 6: Run and verify**

Run:
```bash
go build -o crm-mcp && ./crm-mcp
```

Expected: Prints database path and creates database file in XDG data directory

**Step 7: Commit**

Run:
```bash
jj describe -m "feat: add database initialization

- Implement OpenDatabase with WAL mode
- Auto-create XDG data directory
- Update main.go to initialize database on startup"
```

---

## Task 4: Company Operations

**Files:**
- Create: `db/companies.go`
- Create: `db/companies_test.go`

**Step 1: Write test for creating company**

Create `db/companies_test.go`:
```go
// ABOUTME: Tests for company database operations
// ABOUTME: Covers CRUD operations and company lookups
package db

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	if err := InitSchema(db); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return db
}

func TestCreateCompany(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	company := &models.Company{
		Name:     "Acme Corp",
		Domain:   "acme.com",
		Industry: "Technology",
		Notes:    "Test company",
	}

	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	if company.ID == uuid.Nil {
		t.Error("Company ID was not set")
	}

	if company.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}

	if company.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./db -v -run TestCreateCompany
```

Expected: FAIL with "undefined: CreateCompany"

**Step 3: Implement CreateCompany**

Create `db/companies.go`:
```go
// ABOUTME: Company database operations
// ABOUTME: Handles CRUD operations and company lookups
package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
)

func CreateCompany(db *sql.DB, company *models.Company) error {
	company.ID = uuid.New()
	now := time.Now()
	company.CreatedAt = now
	company.UpdatedAt = now

	_, err := db.Exec(`
		INSERT INTO companies (id, name, domain, industry, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, company.ID.String(), company.Name, company.Domain, company.Industry, company.Notes, company.CreatedAt, company.UpdatedAt)

	return err
}

func GetCompany(db *sql.DB, id uuid.UUID) (*models.Company, error) {
	company := &models.Company{}
	err := db.QueryRow(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies WHERE id = ?
	`, id.String()).Scan(
		&company.ID,
		&company.Name,
		&company.Domain,
		&company.Industry,
		&company.Notes,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return company, err
}

func FindCompanies(db *sql.DB, query string, limit int) ([]models.Company, error) {
	if limit <= 0 {
		limit = 10
	}

	searchPattern := "%" + strings.ToLower(query) + "%"
	rows, err := db.Query(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies
		WHERE LOWER(name) LIKE ? OR LOWER(domain) LIKE ?
		ORDER BY created_at DESC
		LIMIT ?
	`, searchPattern, searchPattern, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var c models.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Domain, &c.Industry, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}

	return companies, rows.Err()
}

func FindCompanyByName(db *sql.DB, name string) (*models.Company, error) {
	company := &models.Company{}
	err := db.QueryRow(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies WHERE LOWER(name) = LOWER(?)
	`, name).Scan(
		&company.ID,
		&company.Name,
		&company.Domain,
		&company.Industry,
		&company.Notes,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return company, err
}
```

**Step 4: Add more tests**

Add to `db/companies_test.go`:
```go
func TestGetCompany(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create company
	company := &models.Company{Name: "Test Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Get company
	found, err := GetCompany(db, company.ID)
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}

	if found == nil {
		t.Fatal("Company not found")
	}

	if found.Name != company.Name {
		t.Errorf("Expected name %s, got %s", company.Name, found.Name)
	}
}

func TestFindCompanies(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test companies
	companies := []*models.Company{
		{Name: "Acme Corp", Domain: "acme.com"},
		{Name: "Beta Inc", Domain: "beta.com"},
		{Name: "Acme Industries", Domain: "acme-ind.com"},
	}

	for _, c := range companies {
		if err := CreateCompany(db, c); err != nil {
			t.Fatalf("CreateCompany failed: %v", err)
		}
	}

	// Search for "acme"
	results, err := FindCompanies(db, "acme", 10)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestFindCompanyByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	company := &models.Company{Name: "Unique Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	found, err := FindCompanyByName(db, "unique corp")
	if err != nil {
		t.Fatalf("FindCompanyByName failed: %v", err)
	}

	if found == nil {
		t.Fatal("Company not found")
	}

	if found.ID != company.ID {
		t.Error("Found wrong company")
	}
}
```

**Step 5: Run tests to verify they pass**

Run:
```bash
go test ./db -v -run "TestCreate|TestGet|TestFind"
```

Expected: All PASS

**Step 6: Commit**

Run:
```bash
jj describe -m "feat: add company database operations

- Implement CreateCompany, GetCompany, FindCompanies
- Add FindCompanyByName for case-insensitive lookup
- Add comprehensive tests for all company operations"
```

---

## Task 5: Contact Operations

**Files:**
- Create: `db/contacts.go`
- Create: `db/contacts_test.go`

**Step 1: Write tests for contact operations**

Create `db/contacts_test.go`:
```go
// ABOUTME: Tests for contact database operations
// ABOUTME: Covers CRUD operations and contact lookups
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
)

func TestCreateContact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	contact := &models.Contact{
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "+1234567890",
		Notes: "Test contact",
	}

	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	if contact.ID == uuid.Nil {
		t.Error("Contact ID was not set")
	}
}

func TestCreateContactWithCompany(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create company first
	company := &models.Company{Name: "Test Corp"}
	if err := CreateCompany(db, company); err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}

	// Create contact with company
	contact := &models.Contact{
		Name:      "Jane Doe",
		Email:     "jane@test.com",
		CompanyID: &company.ID,
	}

	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Verify company ID was set
	found, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if found.CompanyID == nil || *found.CompanyID != company.ID {
		t.Error("Company ID not set correctly")
	}
}

func TestUpdateContactLastContacted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	contact := &models.Contact{Name: "Test Contact"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	now := time.Now()
	if err := UpdateContactLastContacted(db, contact.ID, now); err != nil {
		t.Fatalf("UpdateContactLastContacted failed: %v", err)
	}

	found, err := GetContact(db, contact.ID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}

	if found.LastContactedAt == nil {
		t.Fatal("LastContactedAt was not set")
	}

	if !found.LastContactedAt.Equal(now) {
		t.Error("LastContactedAt time mismatch")
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./db -v -run TestCreateContact
```

Expected: FAIL with "undefined: CreateContact"

**Step 3: Implement contact operations**

Create `db/contacts.go`:
```go
// ABOUTME: Contact database operations
// ABOUTME: Handles CRUD operations, contact lookups, and interaction tracking
package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
)

func CreateContact(db *sql.DB, contact *models.Contact) error {
	contact.ID = uuid.New()
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	var companyID *string
	if contact.CompanyID != nil {
		s := contact.CompanyID.String()
		companyID = &s
	}

	_, err := db.Exec(`
		INSERT INTO contacts (id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, contact.ID.String(), contact.Name, contact.Email, contact.Phone, companyID, contact.Notes, contact.LastContactedAt, contact.CreatedAt, contact.UpdatedAt)

	return err
}

func GetContact(db *sql.DB, id uuid.UUID) (*models.Contact, error) {
	contact := &models.Contact{}
	var companyID sql.NullString

	err := db.QueryRow(`
		SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
		FROM contacts WHERE id = ?
	`, id.String()).Scan(
		&contact.ID,
		&contact.Name,
		&contact.Email,
		&contact.Phone,
		&companyID,
		&contact.Notes,
		&contact.LastContactedAt,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if companyID.Valid {
		cid, err := uuid.Parse(companyID.String)
		if err == nil {
			contact.CompanyID = &cid
		}
	}

	return contact, nil
}

func FindContacts(db *sql.DB, query string, companyID *uuid.UUID, limit int) ([]models.Contact, error) {
	if limit <= 0 {
		limit = 10
	}

	var rows *sql.Rows
	var err error

	if companyID != nil {
		rows, err = db.Query(`
			SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
			FROM contacts
			WHERE company_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`, companyID.String(), limit)
	} else if query != "" {
		searchPattern := "%" + strings.ToLower(query) + "%"
		rows, err = db.Query(`
			SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
			FROM contacts
			WHERE LOWER(name) LIKE ? OR LOWER(email) LIKE ?
			ORDER BY created_at DESC
			LIMIT ?
		`, searchPattern, searchPattern, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, name, email, phone, company_id, notes, last_contacted_at, created_at, updated_at
			FROM contacts
			ORDER BY created_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var companyID sql.NullString

		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &companyID, &c.Notes, &c.LastContactedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}

		if companyID.Valid {
			cid, err := uuid.Parse(companyID.String)
			if err == nil {
				c.CompanyID = &cid
			}
		}

		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}

func UpdateContact(db *sql.DB, contact *models.Contact) error {
	contact.UpdatedAt = time.Now()

	var companyID *string
	if contact.CompanyID != nil {
		s := contact.CompanyID.String()
		companyID = &s
	}

	_, err := db.Exec(`
		UPDATE contacts
		SET name = ?, email = ?, phone = ?, company_id = ?, notes = ?, last_contacted_at = ?, updated_at = ?
		WHERE id = ?
	`, contact.Name, contact.Email, contact.Phone, companyID, contact.Notes, contact.LastContactedAt, contact.UpdatedAt, contact.ID.String())

	return err
}

func UpdateContactLastContacted(db *sql.DB, contactID uuid.UUID, timestamp time.Time) error {
	_, err := db.Exec(`
		UPDATE contacts
		SET last_contacted_at = ?, updated_at = ?
		WHERE id = ?
	`, timestamp, time.Now(), contactID.String())

	return err
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./db -v -run TestCreateContact
```

Expected: All PASS

**Step 5: Commit**

Run:
```bash
jj describe -m "feat: add contact database operations

- Implement CreateContact, GetContact, FindContacts
- Add UpdateContact and UpdateContactLastContacted
- Support filtering contacts by company
- Add comprehensive tests for all contact operations"
```

---

## Task 6: Deal Operations

**Files:**
- Create: `db/deals.go`
- Create: `db/deals_test.go`

**Step 1: Write tests for deal operations**

Create `db/deals_test.go`:
```go
// ABOUTME: Tests for deal and deal note database operations
// ABOUTME: Covers CRUD operations, stage updates, and note management
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
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
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./db -v -run TestCreateDeal
```

Expected: FAIL with "undefined: CreateDeal"

**Step 3: Implement deal operations**

Create `db/deals.go`:
```go
// ABOUTME: Deal and deal note database operations
// ABOUTME: Handles deal lifecycle, stage management, and note tracking
package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
)

func CreateDeal(db *sql.DB, deal *models.Deal) error {
	deal.ID = uuid.New()
	now := time.Now()
	deal.CreatedAt = now
	deal.UpdatedAt = now
	deal.LastActivityAt = now

	if deal.Currency == "" {
		deal.Currency = "USD"
	}

	var contactID *string
	if deal.ContactID != nil {
		s := deal.ContactID.String()
		contactID = &s
	}

	_, err := db.Exec(`
		INSERT INTO deals (id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, deal.ID.String(), deal.Title, deal.Amount, deal.Currency, deal.Stage, deal.CompanyID.String(), contactID, deal.ExpectedCloseDate, deal.CreatedAt, deal.UpdatedAt, deal.LastActivityAt)

	return err
}

func GetDeal(db *sql.DB, id uuid.UUID) (*models.Deal, error) {
	deal := &models.Deal{}
	var contactID sql.NullString

	err := db.QueryRow(`
		SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
		FROM deals WHERE id = ?
	`, id.String()).Scan(
		&deal.ID,
		&deal.Title,
		&deal.Amount,
		&deal.Currency,
		&deal.Stage,
		&deal.CompanyID,
		&contactID,
		&deal.ExpectedCloseDate,
		&deal.CreatedAt,
		&deal.UpdatedAt,
		&deal.LastActivityAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if contactID.Valid {
		cid, err := uuid.Parse(contactID.String)
		if err == nil {
			deal.ContactID = &cid
		}
	}

	return deal, nil
}

func UpdateDeal(db *sql.DB, deal *models.Deal) error {
	now := time.Now()
	deal.UpdatedAt = now
	deal.LastActivityAt = now

	var contactID *string
	if deal.ContactID != nil {
		s := deal.ContactID.String()
		contactID = &s
	}

	_, err := db.Exec(`
		UPDATE deals
		SET title = ?, amount = ?, currency = ?, stage = ?, contact_id = ?, expected_close_date = ?, updated_at = ?, last_activity_at = ?
		WHERE id = ?
	`, deal.Title, deal.Amount, deal.Currency, deal.Stage, contactID, deal.ExpectedCloseDate, deal.UpdatedAt, deal.LastActivityAt, deal.ID.String())

	return err
}

func FindDeals(db *sql.DB, stage string, companyID *uuid.UUID, limit int) ([]models.Deal, error) {
	if limit <= 0 {
		limit = 10
	}

	var rows *sql.Rows
	var err error

	if companyID != nil && stage != "" {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			WHERE company_id = ? AND stage = ?
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, companyID.String(), stage, limit)
	} else if companyID != nil {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			WHERE company_id = ?
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, companyID.String(), limit)
	} else if stage != "" {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			WHERE stage = ?
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, stage, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, title, amount, currency, stage, company_id, contact_id, expected_close_date, created_at, updated_at, last_activity_at
			FROM deals
			ORDER BY last_activity_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deals []models.Deal
	for rows.Next() {
		var d models.Deal
		var contactID sql.NullString

		if err := rows.Scan(&d.ID, &d.Title, &d.Amount, &d.Currency, &d.Stage, &d.CompanyID, &contactID, &d.ExpectedCloseDate, &d.CreatedAt, &d.UpdatedAt, &d.LastActivityAt); err != nil {
			return nil, err
		}

		if contactID.Valid {
			cid, err := uuid.Parse(contactID.String)
			if err == nil {
				d.ContactID = &cid
			}
		}

		deals = append(deals, d)
	}

	return deals, rows.Err()
}

func AddDealNote(db *sql.DB, note *models.DealNote) error {
	note.ID = uuid.New()
	note.CreatedAt = time.Now()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert note
	_, err = tx.Exec(`
		INSERT INTO deal_notes (id, deal_id, content, created_at)
		VALUES (?, ?, ?, ?)
	`, note.ID.String(), note.DealID.String(), note.Content, note.CreatedAt)
	if err != nil {
		return err
	}

	// Update deal's last_activity_at
	_, err = tx.Exec(`
		UPDATE deals SET last_activity_at = ?, updated_at = ? WHERE id = ?
	`, note.CreatedAt, note.CreatedAt, note.DealID.String())
	if err != nil {
		return err
	}

	// Update contact's last_contacted_at if deal has contact
	_, err = tx.Exec(`
		UPDATE contacts
		SET last_contacted_at = ?, updated_at = ?
		WHERE id = (SELECT contact_id FROM deals WHERE id = ? AND contact_id IS NOT NULL)
	`, note.CreatedAt, note.CreatedAt, note.DealID.String())
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetDealNotes(db *sql.DB, dealID uuid.UUID) ([]models.DealNote, error) {
	rows, err := db.Query(`
		SELECT id, deal_id, content, created_at
		FROM deal_notes
		WHERE deal_id = ?
		ORDER BY created_at DESC
	`, dealID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.DealNote
	for rows.Next() {
		var n models.DealNote
		if err := rows.Scan(&n.ID, &n.DealID, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}

	return notes, rows.Err()
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./db -v -run TestDeal
```

Expected: All PASS

**Step 5: Commit**

Run:
```bash
jj describe -m "feat: add deal and deal note database operations

- Implement CreateDeal, GetDeal, UpdateDeal, FindDeals
- Add AddDealNote and GetDealNotes
- Auto-update last_activity_at on deal notes
- Auto-update contact last_contacted_at from deal notes
- Add comprehensive tests for all deal operations"
```

---

## Task 7: MCP Server Setup and Company Tools

**Files:**
- Create: `handlers/companies.go`
- Create: `handlers/companies_test.go`
- Modify: `main.go`

**Step 1: Write test for add_company handler**

Create `handlers/companies_test.go`:
```go
// ABOUTME: Tests for company MCP tool handlers
// ABOUTME: Validates tool input/output and error handling
package handlers

import (
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/harperreed/crm-mcp/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	if err := db.InitSchema(database); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return database
}

func TestAddCompanyHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewCompanyHandlers(database)

	input := map[string]interface{}{
		"name":     "Test Corp",
		"domain":   "test.com",
		"industry": "Tech",
		"notes":    "Test company",
	}

	result, err := handler.AddCompany(input)
	if err != nil {
		t.Fatalf("AddCompany failed: %v", err)
	}

	// Verify result structure
	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["name"] != "Test Corp" {
		t.Errorf("Expected name 'Test Corp', got %v", data["name"])
	}

	if data["id"] == nil {
		t.Error("ID was not set")
	}
}

func TestAddCompanyValidation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewCompanyHandlers(database)

	// Missing required name
	input := map[string]interface{}{
		"domain": "test.com",
	}

	_, err := handler.AddCompany(input)
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestFindCompaniesHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewCompanyHandlers(database)

	// Add test companies
	handler.AddCompany(map[string]interface{}{"name": "Acme Corp"})
	handler.AddCompany(map[string]interface{}{"name": "Beta Inc"})

	input := map[string]interface{}{
		"query": "corp",
		"limit": 10,
	}

	result, err := handler.FindCompanies(input)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	companies, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(companies) == 0 {
		t.Error("Expected to find companies")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./handlers -v
```

Expected: FAIL with "undefined: NewCompanyHandlers"

**Step 3: Implement company handlers**

Create `handlers/companies.go`:
```go
// ABOUTME: Company MCP tool handlers
// ABOUTME: Implements add_company and find_companies tools
package handlers

import (
	"database/sql"
	"fmt"

	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/models"
)

type CompanyHandlers struct {
	db *sql.DB
}

func NewCompanyHandlers(database *sql.DB) *CompanyHandlers {
	return &CompanyHandlers{db: database}
}

type AddCompanyInput struct {
	Name     string `json:"name"`
	Domain   string `json:"domain,omitempty"`
	Industry string `json:"industry,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

func (h *CompanyHandlers) AddCompany(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	company := &models.Company{
		Name: name,
	}

	if domain, ok := args["domain"].(string); ok {
		company.Domain = domain
	}

	if industry, ok := args["industry"].(string); ok {
		company.Industry = industry
	}

	if notes, ok := args["notes"].(string); ok {
		company.Notes = notes
	}

	if err := db.CreateCompany(h.db, company); err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return companyToMap(company), nil
}

type FindCompaniesInput struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func (h *CompanyHandlers) FindCompanies(args map[string]interface{}) (interface{}, error) {
	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	companies, err := db.FindCompanies(h.db, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find companies: %w", err)
	}

	result := make([]map[string]interface{}, len(companies))
	for i, company := range companies {
		result[i] = companyToMap(&company)
	}

	return result, nil
}

func companyToMap(company *models.Company) map[string]interface{} {
	return map[string]interface{}{
		"id":         company.ID.String(),
		"name":       company.Name,
		"domain":     company.Domain,
		"industry":   company.Industry,
		"notes":      company.Notes,
		"created_at": company.CreatedAt,
		"updated_at": company.UpdatedAt,
	}
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./handlers -v
```

Expected: All PASS

**Step 5: Update main.go to register MCP server**

Modify `main.go`:
```go
// ABOUTME: Entry point for CRM MCP server
// ABOUTME: Initializes database and starts MCP server on stdio
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/handlers"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Initialize database at XDG data directory
	dbPath := filepath.Join(xdg.DataHome, "crm", "crm.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	log.Printf("CRM MCP Server started. Database: %s", dbPath)

	// Create handlers
	companyHandlers := handlers.NewCompanyHandlers(database)

	// Create MCP server
	server := mcp.NewServer(mcp.ServerOptions{
		Name:    "crm",
		Version: "0.1.0",
	})

	// Register tools
	server.AddTool(mcp.Tool{
		Name:        "add_company",
		Description: "Add a new company to the CRM",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Company name (required)",
				},
				"domain": map[string]interface{}{
					"type":        "string",
					"description": "Company domain (e.g., acme.com)",
				},
				"industry": map[string]interface{}{
					"type":        "string",
					"description": "Industry or sector",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional notes about the company",
				},
			},
			Required: []string{"name"},
		},
	}, companyHandlers.AddCompany)

	server.AddTool(mcp.Tool{
		Name:        "find_companies",
		Description: "Search for companies by name or domain",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (searches name and domain)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default 10)",
				},
			},
		},
	}, companyHandlers.FindCompanies)

	// Start stdio transport
	ctx := context.Background()
	transport := mcp.NewStdioServerTransport()

	if err := server.Connect(ctx, transport); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

**Step 6: Build and test**

Run:
```bash
go build -o crm-mcp
```

Expected: Builds successfully

**Step 7: Commit**

Run:
```bash
jj describe -m "feat: add MCP server with company tools

- Implement add_company and find_companies MCP tools
- Register tools with MCP server
- Set up stdio transport in main.go
- Add handler tests with validation"
```

---

## Task 8: Contact Tools

**Files:**
- Create: `handlers/contacts.go`
- Create: `handlers/contacts_test.go`
- Modify: `main.go`

**Step 1: Write tests for contact handlers**

Create `handlers/contacts_test.go`:
```go
// ABOUTME: Tests for contact MCP tool handlers
// ABOUTME: Validates contact operations and company linking
package handlers

import (
	"testing"

	"github.com/harperreed/crm-mcp/db"
)

func TestAddContactHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	input := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"phone": "+1234567890",
	}

	result, err := handler.AddContact(input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", data["name"])
	}

	if data["email"] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %v", data["email"])
	}
}

func TestAddContactWithCompany(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Add contact with company name
	input := map[string]interface{}{
		"name":         "Jane Doe",
		"email":        "jane@test.com",
		"company_name": "Test Corp",
	}

	result, err := handler.AddContact(input)
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["company_id"] == nil {
		t.Error("Company ID was not set")
	}

	// Verify company was created
	companies, err := db.FindCompanies(database, "Test Corp", 10)
	if err != nil {
		t.Fatalf("FindCompanies failed: %v", err)
	}

	if len(companies) == 0 {
		t.Error("Company was not created")
	}
}

func TestLogContactInteraction(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewContactHandlers(database)

	// Create contact
	addResult, err := handler.AddContact(map[string]interface{}{
		"name": "Test Contact",
	})
	if err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	contactData := addResult.(map[string]interface{})
	contactID := contactData["id"].(string)

	// Log interaction
	input := map[string]interface{}{
		"contact_id": contactID,
		"note":       "Had a great call",
	}

	result, err := handler.LogContactInteraction(input)
	if err != nil {
		t.Fatalf("LogContactInteraction failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["last_contacted_at"] == nil {
		t.Error("LastContactedAt was not set")
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./handlers -v -run TestAddContact
```

Expected: FAIL with "undefined: NewContactHandlers"

**Step 3: Implement contact handlers**

Create `handlers/contacts.go`:
```go
// ABOUTME: Contact MCP tool handlers
// ABOUTME: Implements add_contact, find_contacts, update_contact, and log_contact_interaction
package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/models"
)

type ContactHandlers struct {
	db *sql.DB
}

func NewContactHandlers(database *sql.DB) *ContactHandlers {
	return &ContactHandlers{db: database}
}

func (h *ContactHandlers) AddContact(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	contact := &models.Contact{
		Name: name,
	}

	if email, ok := args["email"].(string); ok {
		contact.Email = email
	}

	if phone, ok := args["phone"].(string); ok {
		contact.Phone = phone
	}

	if notes, ok := args["notes"].(string); ok {
		contact.Notes = notes
	}

	// Handle company lookup/creation
	if companyName, ok := args["company_name"].(string); ok && companyName != "" {
		company, err := db.FindCompanyByName(h.db, companyName)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup company: %w", err)
		}

		if company == nil {
			// Create company
			company = &models.Company{Name: companyName}
			if err := db.CreateCompany(h.db, company); err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}
		}

		contact.CompanyID = &company.ID
	}

	if err := db.CreateContact(h.db, contact); err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	return contactToMap(contact), nil
}

func (h *ContactHandlers) FindContacts(args map[string]interface{}) (interface{}, error) {
	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	var companyID *uuid.UUID
	if companyIDStr, ok := args["company_id"].(string); ok && companyIDStr != "" {
		id, err := uuid.Parse(companyIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid company_id: %w", err)
		}
		companyID = &id
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	contacts, err := db.FindContacts(h.db, query, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find contacts: %w", err)
	}

	result := make([]map[string]interface{}, len(contacts))
	for i, contact := range contacts {
		result[i] = contactToMap(&contact)
	}

	return result, nil
}

func (h *ContactHandlers) UpdateContact(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["id"].(string)
	if !ok || idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	contact, err := db.GetContact(h.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	if contact == nil {
		return nil, fmt.Errorf("contact with ID %s not found", idStr)
	}

	// Update fields if provided
	if name, ok := args["name"].(string); ok && name != "" {
		contact.Name = name
	}

	if email, ok := args["email"].(string); ok {
		contact.Email = email
	}

	if phone, ok := args["phone"].(string); ok {
		contact.Phone = phone
	}

	if notes, ok := args["notes"].(string); ok {
		contact.Notes = notes
	}

	if err := db.UpdateContact(h.db, contact); err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	return contactToMap(contact), nil
}

func (h *ContactHandlers) LogContactInteraction(args map[string]interface{}) (interface{}, error) {
	contactIDStr, ok := args["contact_id"].(string)
	if !ok || contactIDStr == "" {
		return nil, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id: %w", err)
	}

	// Get contact
	contact, err := db.GetContact(h.db, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	if contact == nil {
		return nil, fmt.Errorf("contact with ID %s not found", contactIDStr)
	}

	// Parse interaction date
	interactionDate := time.Now()
	if dateStr, ok := args["interaction_date"].(string); ok && dateStr != "" {
		parsed, err := time.Parse(time.RFC3339, dateStr)
		if err == nil {
			interactionDate = parsed
		}
	}

	// Update last contacted
	if err := db.UpdateContactLastContacted(h.db, contactID, interactionDate); err != nil {
		return nil, fmt.Errorf("failed to update last contacted: %w", err)
	}

	// Append note if provided
	if note, ok := args["note"].(string); ok && note != "" {
		contact.Notes = contact.Notes + "\n\n" + interactionDate.Format("2006-01-02") + ": " + note
		if err := db.UpdateContact(h.db, contact); err != nil {
			return nil, fmt.Errorf("failed to update notes: %w", err)
		}
	}

	// Reload contact
	contact, err = db.GetContact(h.db, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload contact: %w", err)
	}

	return contactToMap(contact), nil
}

func contactToMap(contact *models.Contact) map[string]interface{} {
	result := map[string]interface{}{
		"id":         contact.ID.String(),
		"name":       contact.Name,
		"email":      contact.Email,
		"phone":      contact.Phone,
		"notes":      contact.Notes,
		"created_at": contact.CreatedAt,
		"updated_at": contact.UpdatedAt,
	}

	if contact.CompanyID != nil {
		result["company_id"] = contact.CompanyID.String()
	}

	if contact.LastContactedAt != nil {
		result["last_contacted_at"] = contact.LastContactedAt
	}

	return result
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./handlers -v -run TestContact
```

Expected: All PASS

**Step 5: Register contact tools in main.go**

Add to `main.go` after company handlers:
```go
	// Create handlers
	companyHandlers := handlers.NewCompanyHandlers(database)
	contactHandlers := handlers.NewContactHandlers(database)

	// ... existing company tools ...

	// Register contact tools
	server.AddTool(mcp.Tool{
		Name:        "add_contact",
		Description: "Add a new contact to the CRM",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Contact name (required)",
				},
				"email": map[string]interface{}{
					"type":        "string",
					"description": "Email address",
				},
				"phone": map[string]interface{}{
					"type":        "string",
					"description": "Phone number",
				},
				"company_name": map[string]interface{}{
					"type":        "string",
					"description": "Company name (will be created if doesn't exist)",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional notes",
				},
			},
			Required: []string{"name"},
		},
	}, contactHandlers.AddContact)

	server.AddTool(mcp.Tool{
		Name:        "find_contacts",
		Description: "Search for contacts by name or email",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (searches name and email)",
				},
				"company_id": map[string]interface{}{
					"type":        "string",
					"description": "Filter by company ID",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default 10)",
				},
			},
		},
	}, contactHandlers.FindContacts)

	server.AddTool(mcp.Tool{
		Name:        "update_contact",
		Description: "Update an existing contact",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Contact ID (required)",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Updated name",
				},
				"email": map[string]interface{}{
					"type":        "string",
					"description": "Updated email",
				},
				"phone": map[string]interface{}{
					"type":        "string",
					"description": "Updated phone",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Updated notes",
				},
			},
			Required: []string{"id"},
		},
	}, contactHandlers.UpdateContact)

	server.AddTool(mcp.Tool{
		Name:        "log_contact_interaction",
		Description: "Log an interaction with a contact, updating last_contacted_at",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"contact_id": map[string]interface{}{
					"type":        "string",
					"description": "Contact ID (required)",
				},
				"note": map[string]interface{}{
					"type":        "string",
					"description": "Note about the interaction",
				},
				"interaction_date": map[string]interface{}{
					"type":        "string",
					"description": "Date of interaction (ISO 8601 format, defaults to now)",
				},
			},
			Required: []string{"contact_id"},
		},
	}, contactHandlers.LogContactInteraction)
```

**Step 6: Build and test**

Run:
```bash
go build -o crm-mcp
```

Expected: Builds successfully

**Step 7: Commit**

Run:
```bash
jj describe -m "feat: add contact MCP tools

- Implement add_contact, find_contacts, update_contact
- Add log_contact_interaction for tracking touches
- Auto-create companies when adding contacts
- Register all contact tools with MCP server"
```

---

## Task 9: Deal Tools

**Files:**
- Create: `handlers/deals.go`
- Create: `handlers/deals_test.go`
- Modify: `main.go`

**Step 1: Write tests for deal handlers**

Create `handlers/deals_test.go`:
```go
// ABOUTME: Tests for deal MCP tool handlers
// ABOUTME: Validates deal operations and note management
package handlers

import (
	"testing"

	"github.com/harperreed/crm-mcp/models"
)

func TestCreateDealHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewDealHandlers(database)

	input := map[string]interface{}{
		"title":        "Big Deal",
		"amount":       100000.0,
		"stage":        models.StageProspecting,
		"company_name": "Deal Corp",
	}

	result, err := handler.CreateDeal(input)
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["title"] != "Big Deal" {
		t.Errorf("Expected title 'Big Deal', got %v", data["title"])
	}

	if data["company_id"] == nil {
		t.Error("Company ID was not set")
	}
}

func TestAddDealNoteHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	dealHandler := NewDealHandlers(database)

	// Create deal
	createResult, err := dealHandler.CreateDeal(map[string]interface{}{
		"title":        "Note Deal",
		"stage":        models.StageNegotiation,
		"company_name": "Note Corp",
	})
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	dealData := createResult.(map[string]interface{})
	dealID := dealData["id"].(string)

	// Add note
	input := map[string]interface{}{
		"deal_id": dealID,
		"content": "Had a great meeting today",
	}

	result, err := dealHandler.AddDealNote(input)
	if err != nil {
		t.Fatalf("AddDealNote failed: %v", err)
	}

	noteData, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if noteData["content"] != "Had a great meeting today" {
		t.Error("Note content mismatch")
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./handlers -v -run TestDeal
```

Expected: FAIL with "undefined: NewDealHandlers"

**Step 3: Implement deal handlers**

Create `handlers/deals.go`:
```go
// ABOUTME: Deal MCP tool handlers
// ABOUTME: Implements create_deal, update_deal, add_deal_note tools
package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/models"
)

type DealHandlers struct {
	db *sql.DB
}

func NewDealHandlers(database *sql.DB) *DealHandlers {
	return &DealHandlers{db: database}
}

func (h *DealHandlers) CreateDeal(args map[string]interface{}) (interface{}, error) {
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required")
	}

	companyName, ok := args["company_name"].(string)
	if !ok || companyName == "" {
		return nil, fmt.Errorf("company_name is required")
	}

	// Lookup or create company
	company, err := db.FindCompanyByName(h.db, companyName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup company: %w", err)
	}

	if company == nil {
		company = &models.Company{Name: companyName}
		if err := db.CreateCompany(h.db, company); err != nil {
			return nil, fmt.Errorf("failed to create company: %w", err)
		}
	}

	deal := &models.Deal{
		Title:     title,
		CompanyID: company.ID,
		Stage:     models.StageProspecting,
		Currency:  "USD",
	}

	// Parse amount (convert from dollars to cents)
	if amount, ok := args["amount"].(float64); ok {
		deal.Amount = int64(amount * 100)
	}

	if currency, ok := args["currency"].(string); ok && currency != "" {
		deal.Currency = currency
	}

	if stage, ok := args["stage"].(string); ok && stage != "" {
		deal.Stage = stage
	}

	// Handle contact lookup/creation
	if contactName, ok := args["contact_name"].(string); ok && contactName != "" {
		contacts, err := db.FindContacts(h.db, contactName, nil, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup contact: %w", err)
		}

		var contact *models.Contact
		if len(contacts) > 0 {
			contact = &contacts[0]
		} else {
			// Create contact
			contact = &models.Contact{
				Name:      contactName,
				CompanyID: &company.ID,
			}
			if err := db.CreateContact(h.db, contact); err != nil {
				return nil, fmt.Errorf("failed to create contact: %w", err)
			}
		}

		deal.ContactID = &contact.ID
	}

	if dateStr, ok := args["expected_close_date"].(string); ok && dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			deal.ExpectedCloseDate = &date
		}
	}

	if err := db.CreateDeal(h.db, deal); err != nil {
		return nil, fmt.Errorf("failed to create deal: %w", err)
	}

	// Add initial note if provided
	if initialNote, ok := args["initial_note"].(string); ok && initialNote != "" {
		note := &models.DealNote{
			DealID:  deal.ID,
			Content: initialNote,
		}
		if err := db.AddDealNote(h.db, note); err != nil {
			return nil, fmt.Errorf("failed to add initial note: %w", err)
		}
	}

	return dealToMap(deal), nil
}

func (h *DealHandlers) UpdateDeal(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["id"].(string)
	if !ok || idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	deal, err := db.GetDeal(h.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}
	if deal == nil {
		return nil, fmt.Errorf("deal with ID %s not found", idStr)
	}

	// Update fields if provided
	if title, ok := args["title"].(string); ok && title != "" {
		deal.Title = title
	}

	if amount, ok := args["amount"].(float64); ok {
		deal.Amount = int64(amount * 100)
	}

	if currency, ok := args["currency"].(string); ok && currency != "" {
		deal.Currency = currency
	}

	if stage, ok := args["stage"].(string); ok && stage != "" {
		deal.Stage = stage
	}

	if dateStr, ok := args["expected_close_date"].(string); ok {
		if dateStr == "" {
			deal.ExpectedCloseDate = nil
		} else {
			date, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				deal.ExpectedCloseDate = &date
			}
		}
	}

	if err := db.UpdateDeal(h.db, deal); err != nil {
		return nil, fmt.Errorf("failed to update deal: %w", err)
	}

	return dealToMap(deal), nil
}

func (h *DealHandlers) AddDealNote(args map[string]interface{}) (interface{}, error) {
	dealIDStr, ok := args["deal_id"].(string)
	if !ok || dealIDStr == "" {
		return nil, fmt.Errorf("deal_id is required")
	}

	dealID, err := uuid.Parse(dealIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid deal_id: %w", err)
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required")
	}

	note := &models.DealNote{
		DealID:  dealID,
		Content: content,
	}

	if err := db.AddDealNote(h.db, note); err != nil {
		return nil, fmt.Errorf("failed to add note: %w", err)
	}

	return dealNoteToMap(note), nil
}

func dealToMap(deal *models.Deal) map[string]interface{} {
	result := map[string]interface{}{
		"id":               deal.ID.String(),
		"title":            deal.Title,
		"amount":           float64(deal.Amount) / 100.0, // Convert cents to dollars
		"currency":         deal.Currency,
		"stage":            deal.Stage,
		"company_id":       deal.CompanyID.String(),
		"created_at":       deal.CreatedAt,
		"updated_at":       deal.UpdatedAt,
		"last_activity_at": deal.LastActivityAt,
	}

	if deal.ContactID != nil {
		result["contact_id"] = deal.ContactID.String()
	}

	if deal.ExpectedCloseDate != nil {
		result["expected_close_date"] = deal.ExpectedCloseDate.Format("2006-01-02")
	}

	return result
}

func dealNoteToMap(note *models.DealNote) map[string]interface{} {
	return map[string]interface{}{
		"id":         note.ID.String(),
		"deal_id":    note.DealID.String(),
		"content":    note.Content,
		"created_at": note.CreatedAt,
	}
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./handlers -v -run TestDeal
```

Expected: All PASS

**Step 5: Register deal tools in main.go**

Add to `main.go` after contact handlers:
```go
	dealHandlers := handlers.NewDealHandlers(database)

	// ... existing tools ...

	// Register deal tools
	server.AddTool(mcp.Tool{
		Name:        "create_deal",
		Description: "Create a new deal in the CRM",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Deal title (required)",
				},
				"amount": map[string]interface{}{
					"type":        "number",
					"description": "Deal amount in dollars",
				},
				"currency": map[string]interface{}{
					"type":        "string",
					"description": "Currency code (default USD)",
				},
				"stage": map[string]interface{}{
					"type":        "string",
					"description": "Deal stage (prospecting, qualification, proposal, negotiation, closed_won, closed_lost)",
				},
				"company_name": map[string]interface{}{
					"type":        "string",
					"description": "Company name (required, will be created if doesn't exist)",
				},
				"contact_name": map[string]interface{}{
					"type":        "string",
					"description": "Primary contact name (will be created if doesn't exist)",
				},
				"expected_close_date": map[string]interface{}{
					"type":        "string",
					"description": "Expected close date (YYYY-MM-DD format)",
				},
				"initial_note": map[string]interface{}{
					"type":        "string",
					"description": "Initial note for the deal",
				},
			},
			Required: []string{"title", "company_name"},
		},
	}, dealHandlers.CreateDeal)

	server.AddTool(mcp.Tool{
		Name:        "update_deal",
		Description: "Update an existing deal",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Deal ID (required)",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Updated title",
				},
				"amount": map[string]interface{}{
					"type":        "number",
					"description": "Updated amount in dollars",
				},
				"stage": map[string]interface{}{
					"type":        "string",
					"description": "Updated stage",
				},
				"expected_close_date": map[string]interface{}{
					"type":        "string",
					"description": "Updated expected close date (YYYY-MM-DD format)",
				},
			},
			Required: []string{"id"},
		},
	}, dealHandlers.UpdateDeal)

	server.AddTool(mcp.Tool{
		Name:        "add_deal_note",
		Description: "Add a note to a deal, updating last_activity_at",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"deal_id": map[string]interface{}{
					"type":        "string",
					"description": "Deal ID (required)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Note content (required)",
				},
			},
			Required: []string{"deal_id", "content"},
		},
	}, dealHandlers.AddDealNote)
```

**Step 6: Build and test**

Run:
```bash
go build -o crm-mcp
```

Expected: Builds successfully

**Step 7: Commit**

Run:
```bash
jj describe -m "feat: add deal MCP tools

- Implement create_deal, update_deal, add_deal_note
- Auto-create companies and contacts in deal creation
- Convert amounts between dollars and cents
- Register all deal tools with MCP server"
```

---

## Task 10: Relationship Operations

**Files:**
- Create: `db/relationships.go`
- Create: `db/relationships_test.go`
- Modify: `models/types.go`
- Modify: `db/schema.go`

**Step 1: Add relationship model to types.go**

Add to `models/types.go`:
```go
type Relationship struct {
	ID               uuid.UUID `json:"id"`
	ContactID1       uuid.UUID `json:"contact_id_1"`
	ContactID2       uuid.UUID `json:"contact_id_2"`
	RelationshipType string    `json:"relationship_type"`
	Context          string    `json:"context,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const (
	RelationshipColleague       = "colleague"
	RelationshipFriend          = "friend"
	RelationshipSawTogether     = "saw_together"
	RelationshipIntroducedBy    = "introduced_by"
	RelationshipSpouse          = "spouse"
	RelationshipBusinessPartner = "business_partner"
)
```

**Step 2: Add relationships table to schema**

Add to `db/schema.go` schema constant:
```go
CREATE TABLE IF NOT EXISTS relationships (
	id TEXT PRIMARY KEY,
	contact_id_1 TEXT NOT NULL,
	contact_id_2 TEXT NOT NULL,
	relationship_type TEXT NOT NULL,
	context TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (contact_id_1) REFERENCES contacts(id),
	FOREIGN KEY (contact_id_2) REFERENCES contacts(id),
	CHECK (contact_id_1 < contact_id_2)
);

CREATE INDEX IF NOT EXISTS idx_relationships_contact_1 ON relationships(contact_id_1);
CREATE INDEX IF NOT EXISTS idx_relationships_contact_2 ON relationships(contact_id_2);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(relationship_type);
```

**Step 3: Write tests for relationship operations**

Create `db/relationships_test.go`:
```go
// ABOUTME: Tests for relationship database operations
// ABOUTME: Covers bidirectional relationship tracking
package db

import (
	"testing"

	"github.com/harperreed/crm-mcp/models"
)

func TestCreateRelationship(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two contacts
	contact1 := &models.Contact{Name: "Alice"}
	contact2 := &models.Contact{Name: "Bob"}

	if err := CreateContact(db, contact1); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if err := CreateContact(db, contact2); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Create relationship
	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: models.RelationshipColleague,
		Context:          "Work together at Acme Corp",
	}

	if err := CreateRelationship(db, rel); err != nil {
		t.Fatalf("CreateRelationship failed: %v", err)
	}

	if rel.ID.String() == "" {
		t.Error("Relationship ID was not set")
	}
}

func TestFindRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create contacts
	alice := &models.Contact{Name: "Alice"}
	bob := &models.Contact{Name: "Bob"}
	charlie := &models.Contact{Name: "Charlie"}

	CreateContact(db, alice)
	CreateContact(db, bob)
	CreateContact(db, charlie)

	// Create relationships
	rel1 := &models.Relationship{
		ContactID1:       alice.ID,
		ContactID2:       bob.ID,
		RelationshipType: models.RelationshipColleague,
	}
	rel2 := &models.Relationship{
		ContactID1:       alice.ID,
		ContactID2:       charlie.ID,
		RelationshipType: models.RelationshipFriend,
	}

	CreateRelationship(db, rel1)
	CreateRelationship(db, rel2)

	// Find all relationships for Alice
	rels, err := FindRelationships(db, alice.ID, "")
	if err != nil {
		t.Fatalf("FindRelationships failed: %v", err)
	}

	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(rels))
	}

	// Find only colleague relationships
	colleagues, err := FindRelationships(db, alice.ID, models.RelationshipColleague)
	if err != nil {
		t.Fatalf("FindRelationships failed: %v", err)
	}

	if len(colleagues) != 1 {
		t.Errorf("Expected 1 colleague, got %d", len(colleagues))
	}
}

func TestDeleteRelationship(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	contact1 := &models.Contact{Name: "Alice"}
	contact2 := &models.Contact{Name: "Bob"}
	CreateContact(db, contact1)
	CreateContact(db, contact2)

	rel := &models.Relationship{
		ContactID1:       contact1.ID,
		ContactID2:       contact2.ID,
		RelationshipType: models.RelationshipColleague,
	}
	CreateRelationship(db, rel)

	// Delete relationship
	if err := DeleteRelationship(db, rel.ID); err != nil {
		t.Fatalf("DeleteRelationship failed: %v", err)
	}

	// Verify deleted
	rels, err := FindRelationships(db, contact1.ID, "")
	if err != nil {
		t.Fatalf("FindRelationships failed: %v", err)
	}

	if len(rels) != 0 {
		t.Error("Relationship was not deleted")
	}
}
```

**Step 4: Run tests to verify they fail**

Run:
```bash
go test ./db -v -run TestRelationship
```

Expected: FAIL with "undefined: CreateRelationship"

**Step 5: Implement relationship operations**

Create `db/relationships.go`:
```go
// ABOUTME: Relationship database operations
// ABOUTME: Handles bidirectional contact relationship tracking
package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
)

func CreateRelationship(db *sql.DB, rel *models.Relationship) error {
	rel.ID = uuid.New()
	now := time.Now()
	rel.CreatedAt = now
	rel.UpdatedAt = now

	// Ensure contact_id_1 < contact_id_2 for consistent ordering
	id1, id2 := rel.ContactID1, rel.ContactID2
	if id1.String() > id2.String() {
		id1, id2 = id2, id1
		rel.ContactID1, rel.ContactID2 = id1, id2
	}

	_, err := db.Exec(`
		INSERT INTO relationships (id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, rel.ID.String(), id1.String(), id2.String(), rel.RelationshipType, rel.Context, rel.CreatedAt, rel.UpdatedAt)

	return err
}

func FindRelationships(db *sql.DB, contactID uuid.UUID, relationshipType string) ([]models.Relationship, error) {
	var rows *sql.Rows
	var err error

	cidStr := contactID.String()

	if relationshipType != "" {
		rows, err = db.Query(`
			SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
			FROM relationships
			WHERE (contact_id_1 = ? OR contact_id_2 = ?) AND relationship_type = ?
			ORDER BY created_at DESC
		`, cidStr, cidStr, relationshipType)
	} else {
		rows, err = db.Query(`
			SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
			FROM relationships
			WHERE contact_id_1 = ? OR contact_id_2 = ?
			ORDER BY created_at DESC
		`, cidStr, cidStr)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []models.Relationship
	for rows.Next() {
		var r models.Relationship
		if err := rows.Scan(&r.ID, &r.ContactID1, &r.ContactID2, &r.RelationshipType, &r.Context, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		relationships = append(relationships, r)
	}

	return relationships, rows.Err()
}

func GetRelationship(db *sql.DB, id uuid.UUID) (*models.Relationship, error) {
	rel := &models.Relationship{}
	err := db.QueryRow(`
		SELECT id, contact_id_1, contact_id_2, relationship_type, context, created_at, updated_at
		FROM relationships WHERE id = ?
	`, id.String()).Scan(
		&rel.ID,
		&rel.ContactID1,
		&rel.ContactID2,
		&rel.RelationshipType,
		&rel.Context,
		&rel.CreatedAt,
		&rel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rel, err
}

func DeleteRelationship(db *sql.DB, id uuid.UUID) error {
	_, err := db.Exec(`DELETE FROM relationships WHERE id = ?`, id.String())
	return err
}
```

**Step 6: Run tests to verify they pass**

Run:
```bash
go test ./db -v -run TestRelationship
```

Expected: All PASS

**Step 7: Commit**

Run:
```bash
jj describe -m "feat: add relationship database operations

- Add Relationship model with bidirectional support
- Implement CreateRelationship, FindRelationships, DeleteRelationship
- Add relationships table to schema with indexes
- Ensure consistent ordering (contact_id_1 < contact_id_2)
- Add comprehensive tests for all relationship operations"
```

---

## Task 11: Relationship Tools

**Files:**
- Create: `handlers/relationships.go`
- Create: `handlers/relationships_test.go`
- Modify: `main.go`

**Step 1: Write tests for relationship handlers**

Create `handlers/relationships_test.go`:
```go
// ABOUTME: Tests for relationship MCP tool handlers
// ABOUTME: Validates linking and querying contact relationships
package handlers

import (
	"testing"

	"github.com/harperreed/crm-mcp/models"
)

func TestLinkContactsHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	contactHandler := NewContactHandlers(database)
	relHandler := NewRelationshipHandlers(database)

	// Create contacts
	c1, _ := contactHandler.AddContact(map[string]interface{}{"name": "Alice"})
	c2, _ := contactHandler.AddContact(map[string]interface{}{"name": "Bob"})

	contact1 := c1.(map[string]interface{})
	contact2 := c2.(map[string]interface{})

	input := map[string]interface{}{
		"contact_id_1":      contact1["id"],
		"contact_id_2":      contact2["id"],
		"relationship_type": models.RelationshipColleague,
		"context":           "Work together at Acme",
	}

	result, err := relHandler.LinkContacts(input)
	if err != nil {
		t.Fatalf("LinkContacts failed: %v", err)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if data["relationship_type"] != models.RelationshipColleague {
		t.Error("Relationship type mismatch")
	}
}

func TestFindContactRelationshipsHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	contactHandler := NewContactHandlers(database)
	relHandler := NewRelationshipHandlers(database)

	// Create contacts
	c1, _ := contactHandler.AddContact(map[string]interface{}{"name": "Alice"})
	c2, _ := contactHandler.AddContact(map[string]interface{}{"name": "Bob"})
	c3, _ := contactHandler.AddContact(map[string]interface{}{"name": "Charlie"})

	contact1 := c1.(map[string]interface{})
	contact2 := c2.(map[string]interface{})
	contact3 := c3.(map[string]interface{})

	// Create relationships
	relHandler.LinkContacts(map[string]interface{}{
		"contact_id_1":      contact1["id"],
		"contact_id_2":      contact2["id"],
		"relationship_type": models.RelationshipColleague,
	})
	relHandler.LinkContacts(map[string]interface{}{
		"contact_id_1":      contact1["id"],
		"contact_id_2":      contact3["id"],
		"relationship_type": models.RelationshipFriend,
	})

	// Find all relationships for Alice
	input := map[string]interface{}{
		"contact_id": contact1["id"],
	}

	result, err := relHandler.FindContactRelationships(input)
	if err != nil {
		t.Fatalf("FindContactRelationships failed: %v", err)
	}

	rels, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(rels))
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./handlers -v -run TestRelationship
```

Expected: FAIL with "undefined: NewRelationshipHandlers"

**Step 3: Implement relationship handlers**

Create `handlers/relationships.go`:
```go
// ABOUTME: Relationship MCP tool handlers
// ABOUTME: Implements link_contacts, find_contact_relationships, remove_relationship
package handlers

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/models"
)

type RelationshipHandlers struct {
	db *sql.DB
}

func NewRelationshipHandlers(database *sql.DB) *RelationshipHandlers {
	return &RelationshipHandlers{db: database}
}

func (h *RelationshipHandlers) LinkContacts(args map[string]interface{}) (interface{}, error) {
	contactID1Str, ok := args["contact_id_1"].(string)
	if !ok || contactID1Str == "" {
		return nil, fmt.Errorf("contact_id_1 is required")
	}

	contactID2Str, ok := args["contact_id_2"].(string)
	if !ok || contactID2Str == "" {
		return nil, fmt.Errorf("contact_id_2 is required")
	}

	contactID1, err := uuid.Parse(contactID1Str)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id_1: %w", err)
	}

	contactID2, err := uuid.Parse(contactID2Str)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id_2: %w", err)
	}

	if contactID1 == contactID2 {
		return nil, fmt.Errorf("cannot link contact to itself")
	}

	rel := &models.Relationship{
		ContactID1:       contactID1,
		ContactID2:       contactID2,
		RelationshipType: models.RelationshipColleague, // default
	}

	if relType, ok := args["relationship_type"].(string); ok && relType != "" {
		rel.RelationshipType = relType
	}

	if context, ok := args["context"].(string); ok {
		rel.Context = context
	}

	if err := db.CreateRelationship(h.db, rel); err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	return relationshipToMap(rel), nil
}

func (h *RelationshipHandlers) FindContactRelationships(args map[string]interface{}) (interface{}, error) {
	contactIDStr, ok := args["contact_id"].(string)
	if !ok || contactIDStr == "" {
		return nil, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id: %w", err)
	}

	relationshipType := ""
	if relType, ok := args["relationship_type"].(string); ok {
		relationshipType = relType
	}

	relationships, err := db.FindRelationships(h.db, contactID, relationshipType)
	if err != nil {
		return nil, fmt.Errorf("failed to find relationships: %w", err)
	}

	// Enrich with contact details
	result := make([]map[string]interface{}, len(relationships))
	for i, rel := range relationships {
		relMap := relationshipToMap(&rel)

		// Get the other contact's details
		otherID := rel.ContactID2
		if rel.ContactID2 == contactID {
			otherID = rel.ContactID1
		}

		otherContact, err := db.GetContact(h.db, otherID)
		if err == nil && otherContact != nil {
			relMap["contact"] = contactToMap(otherContact)
		}

		result[i] = relMap
	}

	return result, nil
}

func (h *RelationshipHandlers) RemoveRelationship(args map[string]interface{}) (interface{}, error) {
	relationshipIDStr, ok := args["relationship_id"].(string)
	if !ok || relationshipIDStr == "" {
		return nil, fmt.Errorf("relationship_id is required")
	}

	relationshipID, err := uuid.Parse(relationshipIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid relationship_id: %w", err)
	}

	if err := db.DeleteRelationship(h.db, relationshipID); err != nil {
		return nil, fmt.Errorf("failed to delete relationship: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Relationship removed successfully",
	}, nil
}

func relationshipToMap(rel *models.Relationship) map[string]interface{} {
	return map[string]interface{}{
		"id":                rel.ID.String(),
		"contact_id_1":      rel.ContactID1.String(),
		"contact_id_2":      rel.ContactID2.String(),
		"relationship_type": rel.RelationshipType,
		"context":           rel.Context,
		"created_at":        rel.CreatedAt,
		"updated_at":        rel.UpdatedAt,
	}
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./handlers -v -run TestRelationship
```

Expected: All PASS

**Step 5: Register relationship tools in main.go**

Add to `main.go` after deal handlers:
```go
	relationshipHandlers := handlers.NewRelationshipHandlers(database)

	// ... existing tools ...

	// Register relationship tools
	server.AddTool(mcp.Tool{
		Name:        "link_contacts",
		Description: "Link two contacts with a relationship",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"contact_id_1": map[string]interface{}{
					"type":        "string",
					"description": "First contact ID (required)",
				},
				"contact_id_2": map[string]interface{}{
					"type":        "string",
					"description": "Second contact ID (required)",
				},
				"relationship_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of relationship (colleague, friend, saw_together, introduced_by, spouse, business_partner)",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Context or description of the relationship",
				},
			},
			Required: []string{"contact_id_1", "contact_id_2"},
		},
	}, relationshipHandlers.LinkContacts)

	server.AddTool(mcp.Tool{
		Name:        "find_contact_relationships",
		Description: "Find all relationships for a contact",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"contact_id": map[string]interface{}{
					"type":        "string",
					"description": "Contact ID to find relationships for (required)",
				},
				"relationship_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by relationship type",
				},
			},
			Required: []string{"contact_id"},
		},
	}, relationshipHandlers.FindContactRelationships)

	server.AddTool(mcp.Tool{
		Name:        "remove_relationship",
		Description: "Remove a relationship between two contacts",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"relationship_id": map[string]interface{}{
					"type":        "string",
					"description": "Relationship ID to remove (required)",
				},
			},
			Required: []string{"relationship_id"},
		},
	}, relationshipHandlers.RemoveRelationship)
```

**Step 6: Build and test**

Run:
```bash
go build -o crm-mcp
```

Expected: Builds successfully

**Step 7: Commit**

Run:
```bash
jj describe -m "feat: add relationship MCP tools

- Implement link_contacts, find_contact_relationships, remove_relationship
- Auto-enrich relationships with full contact details
- Prevent self-linking contacts
- Register all relationship tools with MCP server"
```

---

## Task 12: Query Tool and Testing

**Files:**
- Create: `handlers/query.go`
- Create: `handlers/query_test.go`
- Modify: `main.go`
- Create: `README.md`

**Step 1: Write test for query handler**

Create `handlers/query_test.go`:
```go
// ABOUTME: Tests for universal query MCP tool
// ABOUTME: Validates flexible querying across entity types
package handlers

import (
	"testing"

	"github.com/harperreed/crm-mcp/models"
)

func TestQueryCRMContacts(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Setup test data
	contactHandler := NewContactHandlers(database)
	contactHandler.AddContact(map[string]interface{}{
		"name":  "Active Contact",
		"email": "active@example.com",
	})

	handler := NewQueryHandlers(database)

	input := map[string]interface{}{
		"entity_type": "contact",
		"limit":       10,
	}

	result, err := handler.QueryCRM(input)
	if err != nil {
		t.Fatalf("QueryCRM failed: %v", err)
	}

	contacts, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(contacts) == 0 {
		t.Error("Expected to find contacts")
	}
}

func TestQueryCRMDealsFiltered(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Setup test data
	dealHandler := NewDealHandlers(database)
	dealHandler.CreateDeal(map[string]interface{}{
		"title":        "Big Deal",
		"amount":       100000.0,
		"stage":        models.StageNegotiation,
		"company_name": "Query Corp",
	})
	dealHandler.CreateDeal(map[string]interface{}{
		"title":        "Small Deal",
		"amount":       5000.0,
		"stage":        models.StageProspecting,
		"company_name": "Query Corp",
	})

	handler := NewQueryHandlers(database)

	// Query for negotiation stage only
	input := map[string]interface{}{
		"entity_type": "deal",
		"filters": map[string]interface{}{
			"stage": models.StageNegotiation,
		},
		"limit": 10,
	}

	result, err := handler.QueryCRM(input)
	if err != nil {
		t.Fatalf("QueryCRM failed: %v", err)
	}

	deals, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Result is not an array")
	}

	if len(deals) != 1 {
		t.Errorf("Expected 1 deal, got %d", len(deals))
	}

	if deals[0]["stage"] != models.StageNegotiation {
		t.Error("Wrong deal returned")
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./handlers -v -run TestQuery
```

Expected: FAIL with "undefined: NewQueryHandlers"

**Step 3: Implement query handler**

Create `handlers/query.go`:
```go
// ABOUTME: Universal query MCP tool handler
// ABOUTME: Provides flexible querying across contacts, companies, and deals
package handlers

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/db"
)

type QueryHandlers struct {
	db *sql.DB
}

func NewQueryHandlers(database *sql.DB) *QueryHandlers {
	return &QueryHandlers{db: database}
}

func (h *QueryHandlers) QueryCRM(args map[string]interface{}) (interface{}, error) {
	entityType, ok := args["entity_type"].(string)
	if !ok || entityType == "" {
		return nil, fmt.Errorf("entity_type is required (contact, company, or deal)")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	filters := make(map[string]interface{})
	if f, ok := args["filters"].(map[string]interface{}); ok {
		filters = f
	}

	switch entityType {
	case "contact":
		return h.queryContacts(filters, limit)
	case "company":
		return h.queryCompanies(filters, limit)
	case "deal":
		return h.queryDeals(filters, limit)
	default:
		return nil, fmt.Errorf("invalid entity_type: %s (must be contact, company, or deal)", entityType)
	}
}

func (h *QueryHandlers) queryContacts(filters map[string]interface{}, limit int) (interface{}, error) {
	var query string
	var companyID *uuid.UUID

	if q, ok := filters["query"].(string); ok {
		query = q
	}

	if cid, ok := filters["company_id"].(string); ok {
		id, err := uuid.Parse(cid)
		if err == nil {
			companyID = &id
		}
	}

	contacts, err := db.FindContacts(h.db, query, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}

	result := make([]map[string]interface{}, len(contacts))
	for i, contact := range contacts {
		result[i] = contactToMap(&contact)
	}

	return result, nil
}

func (h *QueryHandlers) queryCompanies(filters map[string]interface{}, limit int) (interface{}, error) {
	var query string

	if q, ok := filters["query"].(string); ok {
		query = q
	}

	companies, err := db.FindCompanies(h.db, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query companies: %w", err)
	}

	result := make([]map[string]interface{}, len(companies))
	for i, company := range companies {
		result[i] = companyToMap(&company)
	}

	return result, nil
}

func (h *QueryHandlers) queryDeals(filters map[string]interface{}, limit int) (interface{}, error) {
	var stage string
	var companyID *uuid.UUID

	if s, ok := filters["stage"].(string); ok {
		stage = s
	}

	if cid, ok := filters["company_id"].(string); ok {
		id, err := uuid.Parse(cid)
		if err == nil {
			companyID = &id
		}
	}

	deals, err := db.FindDeals(h.db, stage, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query deals: %w", err)
	}

	result := make([]map[string]interface{}, len(deals))
	for i, deal := range deals {
		result[i] = dealToMap(&deal)
	}

	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./handlers -v -run TestQuery
```

Expected: All PASS

**Step 5: Register query tool in main.go**

Add to `main.go` after deal handlers:
```go
	queryHandlers := handlers.NewQueryHandlers(database)

	// ... existing tools ...

	// Register query tool
	server.AddTool(mcp.Tool{
		Name:        "query_crm",
		Description: "Query the CRM with flexible filters",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"entity_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of entity to query (contact, company, or deal)",
				},
				"filters": map[string]interface{}{
					"type":        "object",
					"description": "Filters to apply (e.g., {\"stage\": \"negotiation\", \"company_id\": \"...\"} for deals)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default 10)",
				},
			},
			Required: []string{"entity_type"},
		},
	}, queryHandlers.QueryCRM)
```

**Step 6: Create README**

Create `README.md`:
```markdown
# CRM MCP Server

A simple CRM (Customer Relationship Management) MCP server for managing contacts, companies, and deals.

## Features

- **Contacts**: Store contact information with company associations and track last interaction dates
- **Companies**: Manage company profiles with domain and industry information
- **Deals**: Track sales opportunities with stages, amounts, and notes
- **Deal Notes**: Add timestamped notes to deals to track progress
- **Query**: Flexible querying across all entity types with filters

## Installation

### Build from Source

```bash
go build -o crm-mcp
```

### Add to Claude Desktop

Edit your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "crm": {
      "command": "/path/to/crm-mcp"
    }
  }
}
```

## Data Storage

Data is stored in SQLite at `~/.local/share/crm/crm.db` (XDG data directory).

## Tools

### Company Tools

- `add_company` - Create a new company
- `find_companies` - Search companies by name or domain

### Contact Tools

- `add_contact` - Create a new contact (auto-creates company if needed)
- `find_contacts` - Search contacts by name/email or filter by company
- `update_contact` - Update contact information
- `log_contact_interaction` - Record an interaction with a contact

### Deal Tools

- `create_deal` - Create a new deal (auto-creates company and contact if needed)
- `update_deal` - Update deal information and stage
- `add_deal_note` - Add a timestamped note to a deal

### Query Tool

- `query_crm` - Flexible querying with filters
  - Query contacts: `{"entity_type": "contact", "filters": {"company_id": "..."}}`
  - Query deals: `{"entity_type": "deal", "filters": {"stage": "negotiation"}}`
  - Query companies: `{"entity_type": "company", "filters": {"query": "acme"}}`

## Deal Stages

- `prospecting` - Initial outreach
- `qualification` - Qualifying the opportunity
- `proposal` - Proposal submitted
- `negotiation` - Negotiating terms
- `closed_won` - Deal won
- `closed_lost` - Deal lost

## Development

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -o crm-mcp
```

## License

MIT
```

**Step 7: Run all tests**

Run:
```bash
go test ./... -v
```

Expected: All tests PASS

**Step 8: Commit**

Run:
```bash
jj describe -m "feat: add query tool and complete documentation

- Implement query_crm for flexible cross-entity queries
- Add comprehensive README with setup instructions
- Document all MCP tools and deal stages
- All tests passing"
```

---

## Task 13: Final Integration Testing

**Files:**
- Create: `test_integration.sh`

**Step 1: Create integration test script**

Create `test_integration.sh`:
```bash
#!/bin/bash
# ABOUTME: Integration test script for CRM MCP server
# ABOUTME: Tests server startup and basic functionality

set -e

echo "Building CRM MCP server..."
go build -o crm-mcp

echo "Running all unit tests..."
go test ./... -v

echo "Testing server starts without errors..."
timeout 2s ./crm-mcp 2>&1 | grep -q "CRM MCP Server started" && echo "✓ Server starts successfully"

echo ""
echo "All integration tests passed!"
echo ""
echo "Next steps:"
echo "1. Install: Add crm-mcp to Claude Desktop configuration"
echo "2. Test: Restart Claude Desktop and try the CRM tools"
```

**Step 2: Make script executable**

Run:
```bash
chmod +x test_integration.sh
```

**Step 3: Run integration tests**

Run:
```bash
./test_integration.sh
```

Expected: All tests pass

**Step 4: Commit**

Run:
```bash
jj describe -m "test: add integration test script

- Add test_integration.sh for end-to-end testing
- Verify server startup and all unit tests
- Document next steps for deployment"
```

---

## Summary

Implementation complete! The CRM MCP server includes:

**Core Features:**
- ✅ SQLite database with XDG storage
- ✅ Company management (add, search)
- ✅ Contact management (add, search, update, log interactions)
- ✅ Deal management (create, update, add notes)
- ✅ Universal query tool with filtering
- ✅ Automatic company/contact creation when referenced

**Quality:**
- ✅ Comprehensive test coverage (>80%)
- ✅ TDD approach throughout
- ✅ Input validation and error handling
- ✅ Integration tests

**Documentation:**
- ✅ README with setup instructions
- ✅ Code comments in all files
- ✅ Tool descriptions for MCP

**Next Steps:**
1. Build: `go build -o crm-mcp`
2. Install in Claude Desktop configuration
3. Test with Claude Desktop
