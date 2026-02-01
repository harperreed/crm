// ABOUTME: SQLite-based repository implementing the same interface as charm.Client
// ABOUTME: Provides CRUD operations for all CRM entities with pure Go SQLite (modernc.org/sqlite)

package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
)

// DB wraps a SQLite database connection for CRM operations.
type DB struct {
	db *sql.DB
}

// Open opens the SQLite database at the given path.
// If path is empty, uses the default XDG data path.
func Open(path string) (*DB, error) {
	if path == "" {
		var err error
		path, err = xdg.DataFile("pagen/pagen.db")
		if err != nil {
			return nil, fmt.Errorf("failed to get data path: %w", err)
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with WAL mode
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for SQLite
	db.SetMaxOpenConns(1)

	repo := &DB{db: db}
	if err := repo.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// NewTestDB creates a test database in a temporary file.
// The caller is responsible for calling cleanup when done.
func NewTestDB() (*DB, func(), error) {
	tmpFile, err := os.CreateTemp("", "pagen-test-*.db")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	db, err := Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.Remove(tmpPath)
		_ = os.Remove(tmpPath + "-wal")
		_ = os.Remove(tmpPath + "-shm")
	}

	return db, cleanup, nil
}

// initSchema creates all required tables.
func (d *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contacts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT,
		phone TEXT,
		company_id TEXT,
		company_name TEXT,
		notes TEXT,
		last_contacted_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_contacts_company ON contacts(company_id);
	CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name);

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

	CREATE TABLE IF NOT EXISTS deals (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		amount INTEGER DEFAULT 0,
		currency TEXT DEFAULT 'USD',
		stage TEXT NOT NULL,
		company_id TEXT NOT NULL,
		company_name TEXT,
		contact_id TEXT,
		contact_name TEXT,
		expected_close_date DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_activity_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_deals_company ON deals(company_id);
	CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(stage);

	CREATE TABLE IF NOT EXISTS deal_notes (
		id TEXT PRIMARY KEY,
		deal_id TEXT NOT NULL,
		deal_title TEXT,
		deal_company_name TEXT,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (deal_id) REFERENCES deals(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_deal_notes_deal ON deal_notes(deal_id);

	CREATE TABLE IF NOT EXISTS relationships (
		id TEXT PRIMARY KEY,
		contact_id_1 TEXT NOT NULL,
		contact_id_2 TEXT NOT NULL,
		contact1_name TEXT,
		contact2_name TEXT,
		relationship_type TEXT,
		context TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (contact_id_1) REFERENCES contacts(id) ON DELETE CASCADE,
		FOREIGN KEY (contact_id_2) REFERENCES contacts(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_relationships_contact1 ON relationships(contact_id_1);
	CREATE INDEX IF NOT EXISTS idx_relationships_contact2 ON relationships(contact_id_2);

	CREATE TABLE IF NOT EXISTS interaction_log (
		id TEXT PRIMARY KEY,
		contact_id TEXT NOT NULL,
		contact_name TEXT,
		interaction_type TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		notes TEXT,
		sentiment TEXT,
		metadata TEXT,
		FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_interaction_contact ON interaction_log(contact_id);
	CREATE INDEX IF NOT EXISTS idx_interaction_timestamp ON interaction_log(timestamp);

	CREATE TABLE IF NOT EXISTS contact_cadence (
		contact_id TEXT PRIMARY KEY,
		contact_name TEXT,
		cadence_days INTEGER NOT NULL,
		relationship_strength TEXT NOT NULL,
		priority_score REAL DEFAULT 0,
		last_interaction_date DATETIME,
		next_followup_date DATETIME,
		FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS suggestions (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		confidence REAL NOT NULL,
		source_service TEXT NOT NULL,
		source_id TEXT,
		source_data TEXT,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		reviewed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_suggestions_status ON suggestions(status);

	CREATE TABLE IF NOT EXISTS sync_state (
		service TEXT PRIMARY KEY,
		last_sync_time DATETIME,
		last_sync_token TEXT,
		status TEXT NOT NULL,
		error_message TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sync_log (
		id TEXT PRIMARY KEY,
		source_service TEXT NOT NULL,
		source_id TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		imported_at DATETIME NOT NULL,
		metadata TEXT,
		UNIQUE(source_service, source_id)
	);
	CREATE INDEX IF NOT EXISTS idx_sync_log_source ON sync_log(source_service, source_id);
	`

	_, err := d.db.Exec(schema)
	return err
}

// ============================================================================
// Contact Operations
// ============================================================================

// Contact represents a contact in the CRM.
type Contact struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	CompanyID       *uuid.UUID `json:"company_id,omitempty"`
	CompanyName     string     `json:"company_name,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	LastContactedAt *time.Time `json:"last_contacted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ContactFilter defines criteria for filtering contacts.
type ContactFilter struct {
	Query     string
	CompanyID *uuid.UUID
	Limit     int
}

// CreateContact creates a new contact.
func (d *DB) CreateContact(contact *Contact) error {
	if contact.ID == uuid.Nil {
		contact.ID = uuid.New()
	}
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	var companyID *string
	if contact.CompanyID != nil {
		s := contact.CompanyID.String()
		companyID = &s
	}

	_, err := d.db.Exec(`
		INSERT INTO contacts (id, name, email, phone, company_id, company_name, notes, last_contacted_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		contact.ID.String(), contact.Name, contact.Email, contact.Phone,
		companyID, contact.CompanyName, contact.Notes, contact.LastContactedAt,
		contact.CreatedAt, contact.UpdatedAt)
	return err
}

// GetContact retrieves a contact by ID.
func (d *DB) GetContact(id uuid.UUID) (*Contact, error) {
	row := d.db.QueryRow(`
		SELECT id, name, email, phone, company_id, company_name, notes, last_contacted_at, created_at, updated_at
		FROM contacts WHERE id = ?`, id.String())

	return scanContact(row)
}

// UpdateContact updates an existing contact.
func (d *DB) UpdateContact(contact *Contact) error {
	contact.UpdatedAt = time.Now()

	var companyID *string
	if contact.CompanyID != nil {
		s := contact.CompanyID.String()
		companyID = &s
	}

	result, err := d.db.Exec(`
		UPDATE contacts SET name = ?, email = ?, phone = ?, company_id = ?, company_name = ?,
		notes = ?, last_contacted_at = ?, updated_at = ?
		WHERE id = ?`,
		contact.Name, contact.Email, contact.Phone, companyID, contact.CompanyName,
		contact.Notes, contact.LastContactedAt, contact.UpdatedAt, contact.ID.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteContact removes a contact by ID.
func (d *DB) DeleteContact(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM contacts WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListContacts returns all contacts matching the filter.
func (d *DB) ListContacts(filter *ContactFilter) ([]*Contact, error) {
	query := `SELECT id, name, email, phone, company_id, company_name, notes, last_contacted_at, created_at, updated_at FROM contacts WHERE 1=1`
	var args []interface{}

	if filter != nil {
		if filter.CompanyID != nil {
			query += ` AND company_id = ?`
			args = append(args, filter.CompanyID.String())
		}
		if filter.Query != "" {
			query += ` AND (LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(notes) LIKE ? OR LOWER(company_name) LIKE ?)`
			q := "%" + strings.ToLower(filter.Query) + "%"
			args = append(args, q, q, q, q)
		}
	}

	query += ` ORDER BY name`

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, filter.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var contacts []*Contact
	for rows.Next() {
		contact, err := scanContactRows(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	return contacts, rows.Err()
}

// FindContactByName finds a contact by exact name match.
func (d *DB) FindContactByName(name string) (*Contact, error) {
	row := d.db.QueryRow(`
		SELECT id, name, email, phone, company_id, company_name, notes, last_contacted_at, created_at, updated_at
		FROM contacts WHERE name = ?`, name)

	contact, err := scanContact(row)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return contact, err
}

func scanContact(row *sql.Row) (*Contact, error) {
	var contact Contact
	var id, companyID sql.NullString
	var lastContactedAt sql.NullTime

	err := row.Scan(&id, &contact.Name, &contact.Email, &contact.Phone,
		&companyID, &contact.CompanyName, &contact.Notes, &lastContactedAt,
		&contact.CreatedAt, &contact.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	contact.ID, _ = uuid.Parse(id.String)
	if companyID.Valid {
		cid, _ := uuid.Parse(companyID.String)
		contact.CompanyID = &cid
	}
	if lastContactedAt.Valid {
		contact.LastContactedAt = &lastContactedAt.Time
	}
	return &contact, nil
}

func scanContactRows(rows *sql.Rows) (*Contact, error) {
	var contact Contact
	var id, companyID sql.NullString
	var lastContactedAt sql.NullTime

	err := rows.Scan(&id, &contact.Name, &contact.Email, &contact.Phone,
		&companyID, &contact.CompanyName, &contact.Notes, &lastContactedAt,
		&contact.CreatedAt, &contact.UpdatedAt)
	if err != nil {
		return nil, err
	}

	contact.ID, _ = uuid.Parse(id.String)
	if companyID.Valid {
		cid, _ := uuid.Parse(companyID.String)
		contact.CompanyID = &cid
	}
	if lastContactedAt.Valid {
		contact.LastContactedAt = &lastContactedAt.Time
	}
	return &contact, nil
}

// ============================================================================
// Company Operations
// ============================================================================

// Company represents a company in the CRM.
type Company struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain,omitempty"`
	Industry  string    `json:"industry,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CompanyFilter defines criteria for filtering companies.
type CompanyFilter struct {
	Query    string
	Industry string
	Limit    int
}

// CreateCompany creates a new company.
func (d *DB) CreateCompany(company *Company) error {
	if company.ID == uuid.Nil {
		company.ID = uuid.New()
	}
	now := time.Now()
	company.CreatedAt = now
	company.UpdatedAt = now

	_, err := d.db.Exec(`
		INSERT INTO companies (id, name, domain, industry, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		company.ID.String(), company.Name, company.Domain, company.Industry,
		company.Notes, company.CreatedAt, company.UpdatedAt)
	return err
}

// GetCompany retrieves a company by ID.
func (d *DB) GetCompany(id uuid.UUID) (*Company, error) {
	row := d.db.QueryRow(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies WHERE id = ?`, id.String())

	var company Company
	var idStr string
	err := row.Scan(&idStr, &company.Name, &company.Domain, &company.Industry,
		&company.Notes, &company.CreatedAt, &company.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	company.ID, _ = uuid.Parse(idStr)
	return &company, nil
}

// UpdateCompany updates an existing company.
func (d *DB) UpdateCompany(company *Company) error {
	company.UpdatedAt = time.Now()

	result, err := d.db.Exec(`
		UPDATE companies SET name = ?, domain = ?, industry = ?, notes = ?, updated_at = ?
		WHERE id = ?`,
		company.Name, company.Domain, company.Industry, company.Notes,
		company.UpdatedAt, company.ID.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCompany removes a company by ID.
func (d *DB) DeleteCompany(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM companies WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListCompanies returns all companies matching the filter.
func (d *DB) ListCompanies(filter *CompanyFilter) ([]*Company, error) {
	query := `SELECT id, name, domain, industry, notes, created_at, updated_at FROM companies WHERE 1=1`
	var args []interface{}

	if filter != nil {
		if filter.Industry != "" {
			query += ` AND LOWER(industry) = LOWER(?)`
			args = append(args, filter.Industry)
		}
		if filter.Query != "" {
			query += ` AND (LOWER(name) LIKE ? OR LOWER(domain) LIKE ? OR LOWER(industry) LIKE ? OR LOWER(notes) LIKE ?)`
			q := "%" + strings.ToLower(filter.Query) + "%"
			args = append(args, q, q, q, q)
		}
	}

	query += ` ORDER BY name`

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, filter.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var companies []*Company
	for rows.Next() {
		var company Company
		var idStr string
		err := rows.Scan(&idStr, &company.Name, &company.Domain, &company.Industry,
			&company.Notes, &company.CreatedAt, &company.UpdatedAt)
		if err != nil {
			return nil, err
		}
		company.ID, _ = uuid.Parse(idStr)
		companies = append(companies, &company)
	}
	return companies, rows.Err()
}

// FindCompanyByName finds a company by exact name match.
func (d *DB) FindCompanyByName(name string) (*Company, error) {
	row := d.db.QueryRow(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies WHERE name = ?`, name)

	var company Company
	var idStr string
	err := row.Scan(&idStr, &company.Name, &company.Domain, &company.Industry,
		&company.Notes, &company.CreatedAt, &company.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	company.ID, _ = uuid.Parse(idStr)
	return &company, nil
}

// ============================================================================
// Deal Operations
// ============================================================================

// Deal represents a deal in the CRM.
type Deal struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	Amount            int64      `json:"amount,omitempty"`
	Currency          string     `json:"currency"`
	Stage             string     `json:"stage"`
	CompanyID         uuid.UUID  `json:"company_id"`
	CompanyName       string     `json:"company_name,omitempty"`
	ContactID         *uuid.UUID `json:"contact_id,omitempty"`
	ContactName       string     `json:"contact_name,omitempty"`
	ExpectedCloseDate *time.Time `json:"expected_close_date,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
}

// DealFilter defines criteria for filtering deals.
type DealFilter struct {
	Query     string
	Stage     string
	CompanyID *uuid.UUID
	ContactID *uuid.UUID
	MinAmount int64
	MaxAmount int64
	Limit     int
}

// Stage constants for deals.
const (
	StageProspecting   = "prospecting"
	StageQualification = "qualification"
	StageProposal      = "proposal"
	StageNegotiation   = "negotiation"
	StageClosedWon     = "closed_won"
	StageClosedLost    = "closed_lost"
)

// CreateDeal creates a new deal.
func (d *DB) CreateDeal(deal *Deal) error {
	if deal.ID == uuid.Nil {
		deal.ID = uuid.New()
	}
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

	_, err := d.db.Exec(`
		INSERT INTO deals (id, title, amount, currency, stage, company_id, company_name,
		contact_id, contact_name, expected_close_date, created_at, updated_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deal.ID.String(), deal.Title, deal.Amount, deal.Currency, deal.Stage,
		deal.CompanyID.String(), deal.CompanyName, contactID, deal.ContactName,
		deal.ExpectedCloseDate, deal.CreatedAt, deal.UpdatedAt, deal.LastActivityAt)
	return err
}

// GetDeal retrieves a deal by ID.
func (d *DB) GetDeal(id uuid.UUID) (*Deal, error) {
	row := d.db.QueryRow(`
		SELECT id, title, amount, currency, stage, company_id, company_name,
		contact_id, contact_name, expected_close_date, created_at, updated_at, last_activity_at
		FROM deals WHERE id = ?`, id.String())

	return scanDeal(row)
}

// UpdateDeal updates an existing deal.
func (d *DB) UpdateDeal(deal *Deal) error {
	deal.UpdatedAt = time.Now()
	deal.LastActivityAt = time.Now()

	var contactID *string
	if deal.ContactID != nil {
		s := deal.ContactID.String()
		contactID = &s
	}

	result, err := d.db.Exec(`
		UPDATE deals SET title = ?, amount = ?, currency = ?, stage = ?,
		company_id = ?, company_name = ?, contact_id = ?, contact_name = ?,
		expected_close_date = ?, updated_at = ?, last_activity_at = ?
		WHERE id = ?`,
		deal.Title, deal.Amount, deal.Currency, deal.Stage,
		deal.CompanyID.String(), deal.CompanyName, contactID, deal.ContactName,
		deal.ExpectedCloseDate, deal.UpdatedAt, deal.LastActivityAt, deal.ID.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteDeal removes a deal by ID.
func (d *DB) DeleteDeal(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM deals WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListDeals returns all deals matching the filter.
func (d *DB) ListDeals(filter *DealFilter) ([]*Deal, error) {
	query := `SELECT id, title, amount, currency, stage, company_id, company_name,
		contact_id, contact_name, expected_close_date, created_at, updated_at, last_activity_at
		FROM deals WHERE 1=1`
	var args []interface{}

	if filter != nil {
		if filter.Stage != "" {
			query += ` AND stage = ?`
			args = append(args, filter.Stage)
		}
		if filter.CompanyID != nil {
			query += ` AND company_id = ?`
			args = append(args, filter.CompanyID.String())
		}
		if filter.ContactID != nil {
			query += ` AND contact_id = ?`
			args = append(args, filter.ContactID.String())
		}
		if filter.MinAmount > 0 {
			query += ` AND amount >= ?`
			args = append(args, filter.MinAmount)
		}
		if filter.MaxAmount > 0 {
			query += ` AND amount <= ?`
			args = append(args, filter.MaxAmount)
		}
		if filter.Query != "" {
			query += ` AND (LOWER(title) LIKE ? OR LOWER(company_name) LIKE ?)`
			q := "%" + strings.ToLower(filter.Query) + "%"
			args = append(args, q, q)
		}
	}

	query += ` ORDER BY last_activity_at DESC`

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, filter.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var deals []*Deal
	for rows.Next() {
		deal, err := scanDealRows(rows)
		if err != nil {
			return nil, err
		}
		deals = append(deals, deal)
	}
	return deals, rows.Err()
}

func scanDeal(row *sql.Row) (*Deal, error) {
	var deal Deal
	var id, companyID, contactID sql.NullString
	var expectedCloseDate sql.NullTime

	err := row.Scan(&id, &deal.Title, &deal.Amount, &deal.Currency, &deal.Stage,
		&companyID, &deal.CompanyName, &contactID, &deal.ContactName,
		&expectedCloseDate, &deal.CreatedAt, &deal.UpdatedAt, &deal.LastActivityAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	deal.ID, _ = uuid.Parse(id.String)
	deal.CompanyID, _ = uuid.Parse(companyID.String)
	if contactID.Valid {
		cid, _ := uuid.Parse(contactID.String)
		deal.ContactID = &cid
	}
	if expectedCloseDate.Valid {
		deal.ExpectedCloseDate = &expectedCloseDate.Time
	}
	return &deal, nil
}

func scanDealRows(rows *sql.Rows) (*Deal, error) {
	var deal Deal
	var id, companyID, contactID sql.NullString
	var expectedCloseDate sql.NullTime

	err := rows.Scan(&id, &deal.Title, &deal.Amount, &deal.Currency, &deal.Stage,
		&companyID, &deal.CompanyName, &contactID, &deal.ContactName,
		&expectedCloseDate, &deal.CreatedAt, &deal.UpdatedAt, &deal.LastActivityAt)
	if err != nil {
		return nil, err
	}

	deal.ID, _ = uuid.Parse(id.String)
	deal.CompanyID, _ = uuid.Parse(companyID.String)
	if contactID.Valid {
		cid, _ := uuid.Parse(contactID.String)
		deal.ContactID = &cid
	}
	if expectedCloseDate.Valid {
		deal.ExpectedCloseDate = &expectedCloseDate.Time
	}
	return &deal, nil
}

// ============================================================================
// DealNote Operations
// ============================================================================

// DealNote represents a note attached to a deal.
type DealNote struct {
	ID              uuid.UUID `json:"id"`
	DealID          uuid.UUID `json:"deal_id"`
	DealTitle       string    `json:"deal_title,omitempty"`
	DealCompanyName string    `json:"deal_company_name,omitempty"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateDealNote creates a new deal note.
func (d *DB) CreateDealNote(note *DealNote) error {
	if note.ID == uuid.Nil {
		note.ID = uuid.New()
	}
	note.CreatedAt = time.Now()

	_, err := d.db.Exec(`
		INSERT INTO deal_notes (id, deal_id, deal_title, deal_company_name, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		note.ID.String(), note.DealID.String(), note.DealTitle, note.DealCompanyName,
		note.Content, note.CreatedAt)
	return err
}

// GetDealNote retrieves a deal note by ID.
func (d *DB) GetDealNote(id uuid.UUID) (*DealNote, error) {
	row := d.db.QueryRow(`
		SELECT id, deal_id, deal_title, deal_company_name, content, created_at
		FROM deal_notes WHERE id = ?`, id.String())

	var note DealNote
	var idStr, dealID string
	err := row.Scan(&idStr, &dealID, &note.DealTitle, &note.DealCompanyName,
		&note.Content, &note.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	note.ID, _ = uuid.Parse(idStr)
	note.DealID, _ = uuid.Parse(dealID)
	return &note, nil
}

// DeleteDealNote removes a deal note by ID.
func (d *DB) DeleteDealNote(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM deal_notes WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListDealNotes returns all notes for a deal.
func (d *DB) ListDealNotes(dealID uuid.UUID) ([]*DealNote, error) {
	rows, err := d.db.Query(`
		SELECT id, deal_id, deal_title, deal_company_name, content, created_at
		FROM deal_notes WHERE deal_id = ? ORDER BY created_at`, dealID.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var notes []*DealNote
	for rows.Next() {
		var note DealNote
		var idStr, dealIDStr string
		err := rows.Scan(&idStr, &dealIDStr, &note.DealTitle, &note.DealCompanyName,
			&note.Content, &note.CreatedAt)
		if err != nil {
			return nil, err
		}
		note.ID, _ = uuid.Parse(idStr)
		note.DealID, _ = uuid.Parse(dealIDStr)
		notes = append(notes, &note)
	}
	return notes, rows.Err()
}

// ============================================================================
// Relationship Operations
// ============================================================================

// Relationship represents a bidirectional relationship between contacts.
type Relationship struct {
	ID               uuid.UUID `json:"id"`
	ContactID1       uuid.UUID `json:"contact_id_1"`
	ContactID2       uuid.UUID `json:"contact_id_2"`
	Contact1Name     string    `json:"contact1_name,omitempty"`
	Contact2Name     string    `json:"contact2_name,omitempty"`
	RelationshipType string    `json:"relationship_type,omitempty"`
	Context          string    `json:"context,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// RelationshipFilter defines criteria for filtering relationships.
type RelationshipFilter struct {
	ContactID        *uuid.UUID
	RelationshipType string
	Limit            int
}

// CreateRelationship creates a new relationship between contacts.
func (d *DB) CreateRelationship(rel *Relationship) error {
	if rel.ID == uuid.Nil {
		rel.ID = uuid.New()
	}
	now := time.Now()
	rel.CreatedAt = now
	rel.UpdatedAt = now

	_, err := d.db.Exec(`
		INSERT INTO relationships (id, contact_id_1, contact_id_2, contact1_name, contact2_name,
		relationship_type, context, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rel.ID.String(), rel.ContactID1.String(), rel.ContactID2.String(),
		rel.Contact1Name, rel.Contact2Name, rel.RelationshipType, rel.Context,
		rel.CreatedAt, rel.UpdatedAt)
	return err
}

// GetRelationship retrieves a relationship by ID.
func (d *DB) GetRelationship(id uuid.UUID) (*Relationship, error) {
	row := d.db.QueryRow(`
		SELECT id, contact_id_1, contact_id_2, contact1_name, contact2_name,
		relationship_type, context, created_at, updated_at
		FROM relationships WHERE id = ?`, id.String())

	return scanRelationship(row)
}

// UpdateRelationship updates an existing relationship.
func (d *DB) UpdateRelationship(rel *Relationship) error {
	rel.UpdatedAt = time.Now()

	result, err := d.db.Exec(`
		UPDATE relationships SET contact_id_1 = ?, contact_id_2 = ?, contact1_name = ?,
		contact2_name = ?, relationship_type = ?, context = ?, updated_at = ?
		WHERE id = ?`,
		rel.ContactID1.String(), rel.ContactID2.String(), rel.Contact1Name, rel.Contact2Name,
		rel.RelationshipType, rel.Context, rel.UpdatedAt, rel.ID.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRelationship removes a relationship by ID.
func (d *DB) DeleteRelationship(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM relationships WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListRelationshipsForContact returns all relationships involving a contact.
func (d *DB) ListRelationshipsForContact(contactID uuid.UUID) ([]*Relationship, error) {
	rows, err := d.db.Query(`
		SELECT id, contact_id_1, contact_id_2, contact1_name, contact2_name,
		relationship_type, context, created_at, updated_at
		FROM relationships WHERE contact_id_1 = ? OR contact_id_2 = ?`,
		contactID.String(), contactID.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanRelationshipRows(rows)
}

// GetRelationshipBetween finds a relationship between two specific contacts.
func (d *DB) GetRelationshipBetween(contactID1, contactID2 uuid.UUID) (*Relationship, error) {
	row := d.db.QueryRow(`
		SELECT id, contact_id_1, contact_id_2, contact1_name, contact2_name,
		relationship_type, context, created_at, updated_at
		FROM relationships
		WHERE (contact_id_1 = ? AND contact_id_2 = ?) OR (contact_id_1 = ? AND contact_id_2 = ?)`,
		contactID1.String(), contactID2.String(), contactID2.String(), contactID1.String())

	rel, err := scanRelationship(row)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return rel, err
}

// ListRelationships returns relationships matching the filter.
func (d *DB) ListRelationships(filter *RelationshipFilter) ([]*Relationship, error) {
	query := `SELECT id, contact_id_1, contact_id_2, contact1_name, contact2_name,
		relationship_type, context, created_at, updated_at FROM relationships WHERE 1=1`
	var args []interface{}

	if filter != nil {
		if filter.ContactID != nil {
			query += ` AND (contact_id_1 = ? OR contact_id_2 = ?)`
			args = append(args, filter.ContactID.String(), filter.ContactID.String())
		}
		if filter.RelationshipType != "" {
			query += ` AND relationship_type = ?`
			args = append(args, filter.RelationshipType)
		}
	}

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, filter.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanRelationshipRows(rows)
}

func scanRelationship(row *sql.Row) (*Relationship, error) {
	var rel Relationship
	var id, contactID1, contactID2 string

	err := row.Scan(&id, &contactID1, &contactID2, &rel.Contact1Name, &rel.Contact2Name,
		&rel.RelationshipType, &rel.Context, &rel.CreatedAt, &rel.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	rel.ID, _ = uuid.Parse(id)
	rel.ContactID1, _ = uuid.Parse(contactID1)
	rel.ContactID2, _ = uuid.Parse(contactID2)
	return &rel, nil
}

func scanRelationshipRows(rows *sql.Rows) ([]*Relationship, error) {
	var rels []*Relationship
	for rows.Next() {
		var rel Relationship
		var id, contactID1, contactID2 string

		err := rows.Scan(&id, &contactID1, &contactID2, &rel.Contact1Name, &rel.Contact2Name,
			&rel.RelationshipType, &rel.Context, &rel.CreatedAt, &rel.UpdatedAt)
		if err != nil {
			return nil, err
		}

		rel.ID, _ = uuid.Parse(id)
		rel.ContactID1, _ = uuid.Parse(contactID1)
		rel.ContactID2, _ = uuid.Parse(contactID2)
		rels = append(rels, &rel)
	}
	return rels, rows.Err()
}

// ============================================================================
// InteractionLog Operations
// ============================================================================

// InteractionLog records an interaction with a contact.
type InteractionLog struct {
	ID              uuid.UUID `json:"id"`
	ContactID       uuid.UUID `json:"contact_id"`
	ContactName     string    `json:"contact_name,omitempty"`
	InteractionType string    `json:"interaction_type"`
	Timestamp       time.Time `json:"timestamp"`
	Notes           string    `json:"notes,omitempty"`
	Sentiment       *string   `json:"sentiment,omitempty"`
	Metadata        string    `json:"metadata,omitempty"`
}

// InteractionFilter defines criteria for filtering interaction logs.
type InteractionFilter struct {
	ContactID       *uuid.UUID
	InteractionType string
	Since           *time.Time
	Before          *time.Time
	Sentiment       string
	Limit           int
}

// InteractionType constants.
const (
	InteractionMeeting = "meeting"
	InteractionCall    = "call"
	InteractionEmail   = "email"
	InteractionMessage = "message"
	InteractionEvent   = "event"
)

// Sentiment constants.
const (
	SentimentPositive = "positive"
	SentimentNeutral  = "neutral"
	SentimentNegative = "negative"
)

// CreateInteractionLog creates a new interaction log entry.
func (d *DB) CreateInteractionLog(log *InteractionLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	_, err := d.db.Exec(`
		INSERT INTO interaction_log (id, contact_id, contact_name, interaction_type, timestamp, notes, sentiment, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID.String(), log.ContactID.String(), log.ContactName, log.InteractionType,
		log.Timestamp, log.Notes, log.Sentiment, log.Metadata)
	return err
}

// GetInteractionLog retrieves an interaction log entry by ID.
func (d *DB) GetInteractionLog(id uuid.UUID) (*InteractionLog, error) {
	row := d.db.QueryRow(`
		SELECT id, contact_id, contact_name, interaction_type, timestamp, notes, sentiment, metadata
		FROM interaction_log WHERE id = ?`, id.String())

	var log InteractionLog
	var idStr, contactID string
	var sentiment sql.NullString

	err := row.Scan(&idStr, &contactID, &log.ContactName, &log.InteractionType,
		&log.Timestamp, &log.Notes, &sentiment, &log.Metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	log.ID, _ = uuid.Parse(idStr)
	log.ContactID, _ = uuid.Parse(contactID)
	if sentiment.Valid {
		log.Sentiment = &sentiment.String
	}
	return &log, nil
}

// DeleteInteractionLog removes an interaction log entry by ID.
func (d *DB) DeleteInteractionLog(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM interaction_log WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListInteractionLogs returns interactions matching the filter.
func (d *DB) ListInteractionLogs(filter *InteractionFilter) ([]*InteractionLog, error) {
	query := `SELECT id, contact_id, contact_name, interaction_type, timestamp, notes, sentiment, metadata FROM interaction_log WHERE 1=1`
	var args []interface{}

	if filter != nil {
		if filter.ContactID != nil {
			query += ` AND contact_id = ?`
			args = append(args, filter.ContactID.String())
		}
		if filter.InteractionType != "" {
			query += ` AND interaction_type = ?`
			args = append(args, filter.InteractionType)
		}
		if filter.Since != nil {
			query += ` AND timestamp >= ?`
			args = append(args, *filter.Since)
		}
		if filter.Before != nil {
			query += ` AND timestamp <= ?`
			args = append(args, *filter.Before)
		}
		if filter.Sentiment != "" {
			query += ` AND sentiment = ?`
			args = append(args, filter.Sentiment)
		}
	}

	query += ` ORDER BY timestamp DESC`

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, filter.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var logs []*InteractionLog
	for rows.Next() {
		var log InteractionLog
		var idStr, contactID string
		var sentiment sql.NullString

		err := rows.Scan(&idStr, &contactID, &log.ContactName, &log.InteractionType,
			&log.Timestamp, &log.Notes, &sentiment, &log.Metadata)
		if err != nil {
			return nil, err
		}

		log.ID, _ = uuid.Parse(idStr)
		log.ContactID, _ = uuid.Parse(contactID)
		if sentiment.Valid {
			log.Sentiment = &sentiment.String
		}
		logs = append(logs, &log)
	}
	return logs, rows.Err()
}

// ============================================================================
// ContactCadence Operations
// ============================================================================

// ContactCadence tracks follow-up settings for a contact.
type ContactCadence struct {
	ContactID            uuid.UUID  `json:"contact_id"`
	ContactName          string     `json:"contact_name,omitempty"`
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	LastInteractionDate  *time.Time `json:"last_interaction_date,omitempty"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}

// RelationshipStrength constants.
const (
	StrengthWeak   = "weak"
	StrengthMedium = "medium"
	StrengthStrong = "strong"
)

// SaveContactCadence saves or updates a contact cadence.
func (d *DB) SaveContactCadence(cadence *ContactCadence) error {
	_, err := d.db.Exec(`
		INSERT INTO contact_cadence (contact_id, contact_name, cadence_days, relationship_strength,
		priority_score, last_interaction_date, next_followup_date)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(contact_id) DO UPDATE SET
		contact_name = excluded.contact_name,
		cadence_days = excluded.cadence_days,
		relationship_strength = excluded.relationship_strength,
		priority_score = excluded.priority_score,
		last_interaction_date = excluded.last_interaction_date,
		next_followup_date = excluded.next_followup_date`,
		cadence.ContactID.String(), cadence.ContactName, cadence.CadenceDays,
		cadence.RelationshipStrength, cadence.PriorityScore,
		cadence.LastInteractionDate, cadence.NextFollowupDate)
	return err
}

// GetContactCadence retrieves a contact cadence by contact ID.
func (d *DB) GetContactCadence(contactID uuid.UUID) (*ContactCadence, error) {
	row := d.db.QueryRow(`
		SELECT contact_id, contact_name, cadence_days, relationship_strength,
		priority_score, last_interaction_date, next_followup_date
		FROM contact_cadence WHERE contact_id = ?`, contactID.String())

	var cadence ContactCadence
	var id string
	var lastInteraction, nextFollowup sql.NullTime

	err := row.Scan(&id, &cadence.ContactName, &cadence.CadenceDays, &cadence.RelationshipStrength,
		&cadence.PriorityScore, &lastInteraction, &nextFollowup)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	cadence.ContactID, _ = uuid.Parse(id)
	if lastInteraction.Valid {
		cadence.LastInteractionDate = &lastInteraction.Time
	}
	if nextFollowup.Valid {
		cadence.NextFollowupDate = &nextFollowup.Time
	}
	return &cadence, nil
}

// DeleteContactCadence removes a contact cadence.
func (d *DB) DeleteContactCadence(contactID uuid.UUID) error {
	_, err := d.db.Exec(`DELETE FROM contact_cadence WHERE contact_id = ?`, contactID.String())
	return err
}

// ListContactCadences returns all contact cadences.
func (d *DB) ListContactCadences() ([]*ContactCadence, error) {
	rows, err := d.db.Query(`
		SELECT contact_id, contact_name, cadence_days, relationship_strength,
		priority_score, last_interaction_date, next_followup_date
		FROM contact_cadence ORDER BY priority_score DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var cadences []*ContactCadence
	for rows.Next() {
		var cadence ContactCadence
		var id string
		var lastInteraction, nextFollowup sql.NullTime

		err := rows.Scan(&id, &cadence.ContactName, &cadence.CadenceDays, &cadence.RelationshipStrength,
			&cadence.PriorityScore, &lastInteraction, &nextFollowup)
		if err != nil {
			return nil, err
		}

		cadence.ContactID, _ = uuid.Parse(id)
		if lastInteraction.Valid {
			cadence.LastInteractionDate = &lastInteraction.Time
		}
		if nextFollowup.Valid {
			cadence.NextFollowupDate = &nextFollowup.Time
		}
		cadences = append(cadences, &cadence)
	}
	return cadences, rows.Err()
}

// FollowupContact combines Contact with cadence info for follow-up views.
type FollowupContact struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Email                string     `json:"email,omitempty"`
	Phone                string     `json:"phone,omitempty"`
	CompanyID            *uuid.UUID `json:"company_id,omitempty"`
	CompanyName          string     `json:"company_name,omitempty"`
	Notes                string     `json:"notes,omitempty"`
	LastContactedAt      *time.Time `json:"last_contacted_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	DaysSinceContact     int        `json:"days_since_contact"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}

// GetFollowupList returns contacts needing follow-up, sorted by priority.
func (d *DB) GetFollowupList(limit int) ([]*FollowupContact, error) {
	cadences, err := d.ListContactCadences()
	if err != nil {
		return nil, err
	}

	var followups []*FollowupContact
	for _, cadence := range cadences {
		if cadence.PriorityScore <= 0 {
			continue
		}

		contact, err := d.GetContact(cadence.ContactID)
		if err != nil {
			continue
		}

		daysSince := 0
		if cadence.LastInteractionDate != nil {
			daysSince = int(time.Since(*cadence.LastInteractionDate).Hours() / 24)
		}

		followup := &FollowupContact{
			ID:                   contact.ID,
			Name:                 contact.Name,
			Email:                contact.Email,
			Phone:                contact.Phone,
			CompanyID:            contact.CompanyID,
			CompanyName:          contact.CompanyName,
			Notes:                contact.Notes,
			LastContactedAt:      contact.LastContactedAt,
			CreatedAt:            contact.CreatedAt,
			UpdatedAt:            contact.UpdatedAt,
			CadenceDays:          cadence.CadenceDays,
			RelationshipStrength: cadence.RelationshipStrength,
			PriorityScore:        cadence.PriorityScore,
			DaysSinceContact:     daysSince,
			NextFollowupDate:     cadence.NextFollowupDate,
		}

		followups = append(followups, followup)

		if limit > 0 && len(followups) >= limit {
			break
		}
	}

	return followups, nil
}

// UpdateCadenceAfterInteraction updates cadence when interaction is logged.
func (d *DB) UpdateCadenceAfterInteraction(contactID uuid.UUID, timestamp time.Time) error {
	cadence, err := d.GetContactCadence(contactID)
	if err != nil {
		return err
	}

	if cadence == nil {
		cadence = &ContactCadence{
			ContactID:            contactID,
			CadenceDays:          30,
			RelationshipStrength: StrengthMedium,
		}
	}

	cadence.LastInteractionDate = &timestamp
	next := timestamp.AddDate(0, 0, cadence.CadenceDays)
	cadence.NextFollowupDate = &next

	daysSinceContact := int(time.Since(*cadence.LastInteractionDate).Hours() / 24)
	daysOverdue := daysSinceContact - cadence.CadenceDays

	if daysOverdue <= 0 {
		cadence.PriorityScore = 0.0
	} else {
		baseScore := float64(daysOverdue * 2)
		multiplier := 1.0
		switch cadence.RelationshipStrength {
		case StrengthStrong:
			multiplier = 2.0
		case StrengthMedium:
			multiplier = 1.5
		case StrengthWeak:
			multiplier = 1.0
		}
		cadence.PriorityScore = baseScore * multiplier
	}

	return d.SaveContactCadence(cadence)
}

// ============================================================================
// Suggestion Operations
// ============================================================================

// Suggestion represents an AI-generated suggestion.
type Suggestion struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Confidence    float64    `json:"confidence"`
	SourceService string     `json:"source_service"`
	SourceID      string     `json:"source_id,omitempty"`
	SourceData    string     `json:"source_data,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
}

// SuggestionFilter defines criteria for filtering suggestions.
type SuggestionFilter struct {
	Type          string
	Status        string
	MinConfidence float64
	Limit         int
}

// SuggestionType constants.
const (
	SuggestionTypeDeal         = "deal"
	SuggestionTypeRelationship = "relationship"
	SuggestionTypeCompany      = "company"
)

// SuggestionStatus constants.
const (
	SuggestionStatusPending  = "pending"
	SuggestionStatusAccepted = "accepted"
	SuggestionStatusRejected = "rejected"
)

// CreateSuggestion creates a new suggestion.
func (d *DB) CreateSuggestion(suggestion *Suggestion) error {
	if suggestion.ID == uuid.Nil {
		suggestion.ID = uuid.New()
	}
	suggestion.CreatedAt = time.Now()

	_, err := d.db.Exec(`
		INSERT INTO suggestions (id, type, confidence, source_service, source_id, source_data, status, created_at, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		suggestion.ID.String(), suggestion.Type, suggestion.Confidence, suggestion.SourceService,
		suggestion.SourceID, suggestion.SourceData, suggestion.Status, suggestion.CreatedAt, suggestion.ReviewedAt)
	return err
}

// GetSuggestion retrieves a suggestion by ID.
func (d *DB) GetSuggestion(id uuid.UUID) (*Suggestion, error) {
	row := d.db.QueryRow(`
		SELECT id, type, confidence, source_service, source_id, source_data, status, created_at, reviewed_at
		FROM suggestions WHERE id = ?`, id.String())

	var suggestion Suggestion
	var idStr string
	var reviewedAt sql.NullTime

	err := row.Scan(&idStr, &suggestion.Type, &suggestion.Confidence, &suggestion.SourceService,
		&suggestion.SourceID, &suggestion.SourceData, &suggestion.Status, &suggestion.CreatedAt, &reviewedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	suggestion.ID, _ = uuid.Parse(idStr)
	if reviewedAt.Valid {
		suggestion.ReviewedAt = &reviewedAt.Time
	}
	return &suggestion, nil
}

// UpdateSuggestion updates an existing suggestion.
func (d *DB) UpdateSuggestion(suggestion *Suggestion) error {
	result, err := d.db.Exec(`
		UPDATE suggestions SET type = ?, confidence = ?, source_service = ?, source_id = ?,
		source_data = ?, status = ?, reviewed_at = ?
		WHERE id = ?`,
		suggestion.Type, suggestion.Confidence, suggestion.SourceService, suggestion.SourceID,
		suggestion.SourceData, suggestion.Status, suggestion.ReviewedAt, suggestion.ID.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSuggestion removes a suggestion by ID.
func (d *DB) DeleteSuggestion(id uuid.UUID) error {
	result, err := d.db.Exec(`DELETE FROM suggestions WHERE id = ?`, id.String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListSuggestions returns suggestions matching the filter.
func (d *DB) ListSuggestions(filter *SuggestionFilter) ([]*Suggestion, error) {
	query := `SELECT id, type, confidence, source_service, source_id, source_data, status, created_at, reviewed_at FROM suggestions WHERE 1=1`
	var args []interface{}

	if filter != nil {
		if filter.Type != "" {
			query += ` AND type = ?`
			args = append(args, filter.Type)
		}
		if filter.Status != "" {
			query += ` AND status = ?`
			args = append(args, filter.Status)
		}
		if filter.MinConfidence > 0 {
			query += ` AND confidence >= ?`
			args = append(args, filter.MinConfidence)
		}
	}

	query += ` ORDER BY confidence DESC`

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, filter.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var suggestions []*Suggestion
	for rows.Next() {
		var suggestion Suggestion
		var idStr string
		var reviewedAt sql.NullTime

		err := rows.Scan(&idStr, &suggestion.Type, &suggestion.Confidence, &suggestion.SourceService,
			&suggestion.SourceID, &suggestion.SourceData, &suggestion.Status, &suggestion.CreatedAt, &reviewedAt)
		if err != nil {
			return nil, err
		}

		suggestion.ID, _ = uuid.Parse(idStr)
		if reviewedAt.Valid {
			suggestion.ReviewedAt = &reviewedAt.Time
		}
		suggestions = append(suggestions, &suggestion)
	}
	return suggestions, rows.Err()
}

// ============================================================================
// SyncState Operations
// ============================================================================

// SyncState tracks sync status for external services.
type SyncState struct {
	Service       string     `json:"service"`
	LastSyncTime  *time.Time `json:"last_sync_time,omitempty"`
	LastSyncToken string     `json:"last_sync_token,omitempty"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SyncStatus constants.
const (
	SyncStatusIdle    = "idle"
	SyncStatusSyncing = "syncing"
	SyncStatusError   = "error"
)

// SaveSyncState saves sync state for a service.
func (d *DB) SaveSyncState(state *SyncState) error {
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	_, err := d.db.Exec(`
		INSERT INTO sync_state (service, last_sync_time, last_sync_token, status, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(service) DO UPDATE SET
		last_sync_time = excluded.last_sync_time,
		last_sync_token = excluded.last_sync_token,
		status = excluded.status,
		error_message = excluded.error_message,
		updated_at = excluded.updated_at`,
		state.Service, state.LastSyncTime, state.LastSyncToken, state.Status,
		state.ErrorMessage, state.CreatedAt, state.UpdatedAt)
	return err
}

// GetSyncState retrieves sync state for a service.
func (d *DB) GetSyncState(service string) (*SyncState, error) {
	row := d.db.QueryRow(`
		SELECT service, last_sync_time, last_sync_token, status, error_message, created_at, updated_at
		FROM sync_state WHERE service = ?`, service)

	var state SyncState
	var lastSyncTime sql.NullTime

	err := row.Scan(&state.Service, &lastSyncTime, &state.LastSyncToken, &state.Status,
		&state.ErrorMessage, &state.CreatedAt, &state.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if lastSyncTime.Valid {
		state.LastSyncTime = &lastSyncTime.Time
	}
	return &state, nil
}

// ============================================================================
// SyncLog Operations
// ============================================================================

// SyncLog records imported entities from external services.
type SyncLog struct {
	ID            uuid.UUID `json:"id"`
	SourceService string    `json:"source_service"`
	SourceID      string    `json:"source_id"`
	EntityType    string    `json:"entity_type"`
	EntityID      uuid.UUID `json:"entity_id"`
	ImportedAt    time.Time `json:"imported_at"`
	Metadata      string    `json:"metadata,omitempty"`
}

// CreateSyncLog creates a sync log entry.
func (d *DB) CreateSyncLog(log *SyncLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.ImportedAt = time.Now()

	_, err := d.db.Exec(`
		INSERT INTO sync_log (id, source_service, source_id, entity_type, entity_id, imported_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log.ID.String(), log.SourceService, log.SourceID, log.EntityType,
		log.EntityID.String(), log.ImportedAt, log.Metadata)
	return err
}

// FindSyncLogBySource finds a sync log by source service and ID.
func (d *DB) FindSyncLogBySource(service, sourceID string) (*SyncLog, error) {
	row := d.db.QueryRow(`
		SELECT id, source_service, source_id, entity_type, entity_id, imported_at, metadata
		FROM sync_log WHERE source_service = ? AND source_id = ?`, service, sourceID)

	var log SyncLog
	var id, entityID string

	err := row.Scan(&id, &log.SourceService, &log.SourceID, &log.EntityType,
		&entityID, &log.ImportedAt, &log.Metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	log.ID, _ = uuid.Parse(id)
	log.EntityID, _ = uuid.Parse(entityID)
	return &log, nil
}

// ============================================================================
// Config (placeholder for charm.Config compatibility)
// ============================================================================

// Config holds database configuration.
type Config struct {
	Host string
}

// Config returns the current configuration.
func (d *DB) Config() *Config {
	return &Config{Host: "sqlite://local"}
}

// ============================================================================
// Dashboard Stats
// ============================================================================

// DashboardStats holds statistics for the dashboard.
type DashboardStats struct {
	TotalContacts     int
	TotalCompanies    int
	TotalDeals        int
	TotalPipeline     int64
	DealsByStage      map[string]int
	RecentContacts    []*Contact
	RecentDeals       []*Deal
	UpcomingFollowups []*FollowupContact
}

// GetDashboardStats returns statistics for the dashboard.
func (d *DB) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{
		DealsByStage: make(map[string]int),
	}

	// Count contacts
	row := d.db.QueryRow(`SELECT COUNT(*) FROM contacts`)
	_ = row.Scan(&stats.TotalContacts)

	// Count companies
	row = d.db.QueryRow(`SELECT COUNT(*) FROM companies`)
	_ = row.Scan(&stats.TotalCompanies)

	// Count deals and pipeline value
	row = d.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM deals WHERE stage NOT IN ('closed_won', 'closed_lost')`)
	_ = row.Scan(&stats.TotalDeals, &stats.TotalPipeline)

	// Deals by stage
	rows, err := d.db.Query(`SELECT stage, COUNT(*) FROM deals GROUP BY stage`)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var stage string
			var count int
			if err := rows.Scan(&stage, &count); err == nil {
				stats.DealsByStage[stage] = count
			}
		}
	}

	// Recent contacts
	stats.RecentContacts, _ = d.ListContacts(&ContactFilter{Limit: 5})

	// Recent deals
	stats.RecentDeals, _ = d.ListDeals(&DealFilter{Limit: 5})

	// Upcoming followups
	stats.UpcomingFollowups, _ = d.GetFollowupList(5)

	return stats, nil
}

// ============================================================================
// Export Helpers
// ============================================================================

// ExportData contains all data for export.
type ExportData struct {
	Version       string          `yaml:"version" json:"version"`
	ExportedAt    time.Time       `yaml:"exported_at" json:"exported_at"`
	Tool          string          `yaml:"tool" json:"tool"`
	Contacts      []*Contact      `yaml:"contacts,omitempty" json:"contacts,omitempty"`
	Companies     []*Company      `yaml:"companies,omitempty" json:"companies,omitempty"`
	Deals         []*Deal         `yaml:"deals,omitempty" json:"deals,omitempty"`
	DealNotes     []*DealNote     `yaml:"deal_notes,omitempty" json:"deal_notes,omitempty"`
	Relationships []*Relationship `yaml:"relationships,omitempty" json:"relationships,omitempty"`
}

// ExportAll exports all data.
func (d *DB) ExportAll() (*ExportData, error) {
	data := &ExportData{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Tool:       "pagen",
	}

	var err error
	data.Contacts, err = d.ListContacts(nil)
	if err != nil {
		return nil, err
	}

	data.Companies, err = d.ListCompanies(nil)
	if err != nil {
		return nil, err
	}

	data.Deals, err = d.ListDeals(nil)
	if err != nil {
		return nil, err
	}

	// Get all deal notes
	for _, deal := range data.Deals {
		notes, err := d.ListDealNotes(deal.ID)
		if err != nil {
			continue
		}
		data.DealNotes = append(data.DealNotes, notes...)
	}

	data.Relationships, err = d.ListRelationships(nil)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ExportToJSON exports all data to JSON.
func (d *DB) ExportToJSON() ([]byte, error) {
	data, err := d.ExportAll()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

// GetAllContacts returns all contacts (for export purposes).
func (d *DB) GetAllContacts() ([]*Contact, error) {
	return d.ListContacts(nil)
}

// GetAllCompanies returns all companies (for export purposes).
func (d *DB) GetAllCompanies() ([]*Company, error) {
	return d.ListCompanies(nil)
}

// GetAllDeals returns all deals (for export purposes).
func (d *DB) GetAllDeals() ([]*Deal, error) {
	return d.ListDeals(nil)
}

// GetAllRelationships returns all relationships (for export purposes).
func (d *DB) GetAllRelationships() ([]*Relationship, error) {
	return d.ListRelationships(nil)
}

// Sort helper for followups by priority score.
type byPriorityScore []*FollowupContact

func (a byPriorityScore) Len() int           { return len(a) }
func (a byPriorityScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriorityScore) Less(i, j int) bool { return a[i].PriorityScore > a[j].PriorityScore }

func init() {
	// Register sort interface
	_ = sort.Interface(byPriorityScore{})
}
