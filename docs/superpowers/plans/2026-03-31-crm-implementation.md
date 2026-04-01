# CRM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a simple CRM with contacts, companies, and relationships — accessible via MCP (agents) and CLI (humans), with dual storage backends (SQLite + Markdown).

**Architecture:** Memo-clone pattern. Storage interface in `internal/storage/`, models in `internal/models/`, MCP server in `internal/mcp/`, CLI in `cmd/crm/`, config in `internal/config/`. All entity CRUD flows through the Storage interface.

**Tech Stack:** Go 1.24, modernc.org/sqlite, harperreed/mdstore, modelcontextprotocol/go-sdk v1.1.0, spf13/cobra, google/uuid, fatih/color, gopkg.in/yaml.v3, stretchr/testify

---

## File Structure

| File | Responsibility |
|------|---------------|
| `cmd/crm/main.go` | Entry point, version vars |
| `cmd/crm/root.go` | Cobra root, PersistentPreRunE storage init |
| `cmd/crm/contacts.go` | Contact CLI subcommands (add/list/show/edit/rm) |
| `cmd/crm/companies.go` | Company CLI subcommands (add/list/show/edit/rm) |
| `cmd/crm/relationships.go` | Relationship CLI subcommands (link/unlink) |
| `cmd/crm/mcp.go` | MCP server command |
| `cmd/crm/skill.go` | Claude Code skill installation |
| `internal/models/contact.go` | Contact struct + constructor |
| `internal/models/company.go` | Company struct + constructor |
| `internal/models/relationship.go` | Relationship struct + constructor |
| `internal/storage/interface.go` | Storage interface + filter types + errors |
| `internal/storage/sqlite.go` | SqliteStore init, schema, Close, DataDir |
| `internal/storage/sqlite_contacts.go` | Contact CRUD for SQLite |
| `internal/storage/sqlite_companies.go` | Company CRUD for SQLite |
| `internal/storage/sqlite_relationships.go` | Relationship CRUD for SQLite |
| `internal/storage/sqlite_search.go` | Cross-entity FTS5 search |
| `internal/storage/markdown.go` | MarkdownStore init, helpers |
| `internal/storage/markdown_contacts.go` | Contact CRUD for Markdown |
| `internal/storage/markdown_companies.go` | Company CRUD for Markdown |
| `internal/storage/markdown_relationships.go` | Relationship CRUD for Markdown |
| `internal/storage/markdown_search.go` | Cross-entity text search |
| `internal/mcp/server.go` | MCP server setup |
| `internal/mcp/tools.go` | 12 tool handlers |
| `internal/mcp/resources.go` | 4 resource handlers |
| `internal/mcp/prompts.go` | 3 prompt handlers |
| `internal/config/config.go` | XDG config, backend factory |
| `Makefile` | Build/test/lint targets |
| `.pre-commit-config.yaml` | Pre-commit hooks |
| `.golangci.yml` | Linter config |

Test files follow `<source>_test.go` convention in the same package.

---

### Task 1: Project Bootstrap

**Files:**
- Create: `go.mod`, `cmd/crm/main.go`, `Makefile`, `.pre-commit-config.yaml`, `.golangci.yml`, `CLAUDE.md`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd /Users/harper/Public/src/personal/suite/crm && uv --version 2>/dev/null; go version
```

```bash
cd /Users/harper/Public/src/personal/suite/crm && go mod init github.com/harperreed/crm
```

- [ ] **Step 2: Create main.go**

Create `cmd/crm/main.go`:
```go
// ABOUTME: Entry point for crm CLI application.
// ABOUTME: Initializes and executes the root command.

package main

import (
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create Makefile**

Create `Makefile`:
```makefile
# ABOUTME: Build and test targets for crm CLI
# ABOUTME: Provides standard targets for development, CI, and release

.PHONY: build test test-race test-coverage install clean lint fmt check

build:
	go build -o crm ./cmd/crm

build-dev:
	go build -o bin/crm ./cmd/crm

test:
	go test -v ./...

test-race:
	go test -race -v ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

install:
	go install ./cmd/crm

lint:
	golangci-lint run --timeout=10m

fmt:
	go fmt ./...
	goimports -w .

clean:
	rm -f crm
	rm -rf bin/
	rm -f coverage.out

check: fmt lint test-race
```

- [ ] **Step 4: Create .pre-commit-config.yaml**

Create `.pre-commit-config.yaml`:
```yaml
# ABOUTME: Pre-commit hooks configuration for crm project
# ABOUTME: Runs formatting, linting, and tests before each commit
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
        exclude: ^go\.(mod|sum)$
      - id: end-of-file-fixer
        exclude: ^go\.(mod|sum)$
      - id: check-yaml
      - id: check-added-large-files
        args: ['--maxkb=1000']
        exclude: ^crm$
      - id: check-merge-conflict
      - id: mixed-line-ending
        args: ['--fix=lf']
      - id: check-case-conflict

  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-imports
      - id: go-mod-tidy
      - id: golangci-lint
        args: ['--timeout=10m', '--fix']
      - id: go-unit-tests
        args: ['-race', '-count=1', './...']

  - repo: local
    hooks:
      - id: go-mod-verify
        name: go mod verify
        entry: go mod verify
        language: system
        files: go\.(mod|sum)$
        pass_filenames: false

      - id: go-vet
        name: go vet
        entry: go vet
        language: system
        args: ['./...']
        files: \.go$
        pass_filenames: false
```

- [ ] **Step 5: Create .golangci.yml**

Create `.golangci.yml`:
```yaml
# ABOUTME: Linter configuration for crm project
# ABOUTME: Configures golangci-lint with sensible defaults
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports

linters-settings:
  goimports:
    local-prefixes: github.com/harperreed/crm

issues:
  exclude-dirs:
    - vendor
```

- [ ] **Step 6: Create CLAUDE.md**

Create `CLAUDE.md`:
```markdown
# CRM

Simple CRM with contacts, companies, and relationships.

## Build & Test

- `make build` — build binary
- `make test` — run tests
- `make test-race` — run tests with race detector
- `make lint` — run linter
- `make check` — fmt + lint + test-race

## Architecture

- `cmd/crm/` — CLI commands (Cobra)
- `internal/models/` — data structs (Contact, Company, Relationship)
- `internal/storage/` — Storage interface + SQLite/Markdown backends
- `internal/mcp/` — MCP server (tools, resources, prompts)
- `internal/config/` — XDG config + backend factory

## Patterns

- Storage interface with dual backends (SQLite + Markdown)
- Config-driven initialization via PersistentPreRunE
- Filter objects for list queries
- MCP handlers delegate directly to storage layer
- All files start with ABOUTME comments
```

- [ ] **Step 7: Verify build scaffolding compiles**

We need root.go to compile. Create a minimal stub `cmd/crm/root.go`:
```go
// ABOUTME: Root command for crm CLI with config-driven storage initialization.
// ABOUTME: Handles global flags and persistent pre/post run hooks.

package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crm",
	Short: "A simple CRM for contacts, companies, and relationships",
}

func Execute() error {
	return rootCmd.Execute()
}
```

```bash
cd /Users/harper/Public/src/personal/suite/crm && go get github.com/spf13/cobra@v1.10.2 && go build ./cmd/crm
```

Expected: compiles with exit 0.

- [ ] **Step 8: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add cmd/crm/main.go cmd/crm/root.go go.mod go.sum Makefile .pre-commit-config.yaml .golangci.yml CLAUDE.md
git commit -m "feat: bootstrap crm project with CLI scaffold"
```

---

### Task 2: Models

**Files:**
- Create: `internal/models/contact.go`, `internal/models/contact_test.go`, `internal/models/company.go`, `internal/models/company_test.go`, `internal/models/relationship.go`, `internal/models/relationship_test.go`

- [ ] **Step 1: Write failing test for Contact model**

Create `internal/models/contact_test.go`:
```go
// ABOUTME: Tests for Contact model struct and constructor.
// ABOUTME: Validates field initialization and Touch() behavior.

package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewContact(t *testing.T) {
	c := NewContact("Jane Doe")

	if c.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if c.Name != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got %q", c.Name)
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if c.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
	if c.Fields == nil {
		t.Error("expected non-nil Fields map")
	}
	if c.Tags == nil {
		t.Error("expected non-nil Tags slice")
	}
}

func TestContactTouch(t *testing.T) {
	c := NewContact("Test")
	original := c.UpdatedAt
	time.Sleep(time.Millisecond)
	c.Touch()

	if !c.UpdatedAt.After(original) {
		t.Error("expected UpdatedAt to advance after Touch()")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go get github.com/google/uuid@v1.6.0 && go test ./internal/models/...
```

Expected: FAIL — `NewContact` not defined.

- [ ] **Step 3: Write Contact model**

Create `internal/models/contact.go`:
```go
// ABOUTME: Contact model representing a person in the CRM.
// ABOUTME: Provides constructor and Touch method for lifecycle management.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Phone     string
	Fields    map[string]any
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewContact(name string) *Contact {
	now := time.Now()
	return &Contact{
		ID:        uuid.New(),
		Name:      name,
		Fields:    make(map[string]any),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (c *Contact) Touch() {
	c.UpdatedAt = time.Now()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test ./internal/models/...
```

Expected: PASS

- [ ] **Step 5: Write failing test for Company model**

Create `internal/models/company_test.go`:
```go
// ABOUTME: Tests for Company model struct and constructor.
// ABOUTME: Validates field initialization and Touch() behavior.

package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewCompany(t *testing.T) {
	c := NewCompany("Acme Corp")

	if c.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if c.Name != "Acme Corp" {
		t.Errorf("expected name 'Acme Corp', got %q", c.Name)
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if c.Fields == nil {
		t.Error("expected non-nil Fields map")
	}
	if c.Tags == nil {
		t.Error("expected non-nil Tags slice")
	}
}

func TestCompanyTouch(t *testing.T) {
	c := NewCompany("Test")
	original := c.UpdatedAt
	time.Sleep(time.Millisecond)
	c.Touch()

	if !c.UpdatedAt.After(original) {
		t.Error("expected UpdatedAt to advance after Touch()")
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test ./internal/models/...
```

Expected: FAIL — `NewCompany` not defined.

- [ ] **Step 7: Write Company model**

Create `internal/models/company.go`:
```go
// ABOUTME: Company model representing an organization in the CRM.
// ABOUTME: Provides constructor and Touch method for lifecycle management.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Company struct {
	ID        uuid.UUID
	Name      string
	Domain    string
	Fields    map[string]any
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCompany(name string) *Company {
	now := time.Now()
	return &Company{
		ID:        uuid.New(),
		Name:      name,
		Fields:    make(map[string]any),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (c *Company) Touch() {
	c.UpdatedAt = time.Now()
}
```

- [ ] **Step 8: Write failing test for Relationship model**

Create `internal/models/relationship_test.go`:
```go
// ABOUTME: Tests for Relationship model struct and constructor.
// ABOUTME: Validates field initialization for entity links.

package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewRelationship(t *testing.T) {
	src := uuid.New()
	tgt := uuid.New()
	r := NewRelationship(src, tgt, "works_at", "VP of Engineering")

	if r.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if r.SourceID != src {
		t.Errorf("expected SourceID %s, got %s", src, r.SourceID)
	}
	if r.TargetID != tgt {
		t.Errorf("expected TargetID %s, got %s", tgt, r.TargetID)
	}
	if r.Type != "works_at" {
		t.Errorf("expected Type 'works_at', got %q", r.Type)
	}
	if r.Context != "VP of Engineering" {
		t.Errorf("expected Context 'VP of Engineering', got %q", r.Context)
	}
	if r.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}
```

- [ ] **Step 9: Run test to verify it fails**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test ./internal/models/...
```

Expected: FAIL — `NewRelationship` not defined.

- [ ] **Step 10: Write Relationship model**

Create `internal/models/relationship.go`:
```go
// ABOUTME: Relationship model representing a typed link between CRM entities.
// ABOUTME: Connects contacts and companies with type and context metadata.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Relationship struct {
	ID        uuid.UUID
	SourceID  uuid.UUID
	TargetID  uuid.UUID
	Type      string
	Context   string
	CreatedAt time.Time
}

func NewRelationship(sourceID, targetID uuid.UUID, relType, context string) *Relationship {
	return &Relationship{
		ID:        uuid.New(),
		SourceID:  sourceID,
		TargetID:  targetID,
		Type:      relType,
		Context:   context,
		CreatedAt: time.Now(),
	}
}
```

- [ ] **Step 11: Run all model tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/models/...
```

Expected: all PASS

- [ ] **Step 12: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/models/
git commit -m "feat: add Contact, Company, Relationship models"
```

---

### Task 3: Storage Interface

**Files:**
- Create: `internal/storage/interface.go`

- [ ] **Step 1: Write storage interface**

Create `internal/storage/interface.go`:
```go
// ABOUTME: Storage interface for CRM data backends.
// ABOUTME: Defines the contract that SQLite and Markdown implementations must satisfy.

package storage

import (
	"errors"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

var (
	ErrContactNotFound      = errors.New("contact not found")
	ErrCompanyNotFound      = errors.New("company not found")
	ErrRelationshipNotFound = errors.New("relationship not found")
	ErrPrefixTooShort       = errors.New("prefix must be at least 6 characters")
	ErrAmbiguousPrefix      = errors.New("prefix matches multiple records")
)

// Storage defines the interface for CRM data persistence.
type Storage interface {
	// Contacts
	CreateContact(contact *models.Contact) error
	GetContact(id uuid.UUID) (*models.Contact, error)
	GetContactByPrefix(prefix string) (*models.Contact, error)
	ListContacts(filter *ContactFilter) ([]*models.Contact, error)
	UpdateContact(contact *models.Contact) error
	DeleteContact(id uuid.UUID) error

	// Companies
	CreateCompany(company *models.Company) error
	GetCompany(id uuid.UUID) (*models.Company, error)
	GetCompanyByPrefix(prefix string) (*models.Company, error)
	ListCompanies(filter *CompanyFilter) ([]*models.Company, error)
	UpdateCompany(company *models.Company) error
	DeleteCompany(id uuid.UUID) error

	// Relationships
	CreateRelationship(rel *models.Relationship) error
	ListRelationships(entityID uuid.UUID) ([]*models.Relationship, error)
	DeleteRelationship(id uuid.UUID) error

	// Search
	Search(query string) (*SearchResults, error)

	// Maintenance
	Close() error
}

// ContactFilter defines criteria for filtering contacts.
type ContactFilter struct {
	Tag    *string
	Search string
	Limit  int
}

// CompanyFilter defines criteria for filtering companies.
type CompanyFilter struct {
	Tag    *string
	Search string
	Limit  int
}

// SearchResults holds cross-entity search results.
type SearchResults struct {
	Contacts  []*models.Contact
	Companies []*models.Company
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go build ./internal/storage/
```

Expected: exit 0

- [ ] **Step 3: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/storage/interface.go
git commit -m "feat: add Storage interface with filter and error types"
```

---

### Task 4: SQLite Backend — Schema & Init

**Files:**
- Create: `internal/storage/sqlite.go`, `internal/storage/sqlite_test.go`

- [ ] **Step 1: Write failing test for SQLite store creation**

Create `internal/storage/sqlite_test.go`:
```go
// ABOUTME: Tests for SQLite store initialization and schema.
// ABOUTME: Validates database creation, table existence, and FTS triggers.

package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *SqliteStore {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "crm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	return store
}

func TestNewSqliteStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "crm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestStoreTables(t *testing.T) {
	store := newTestStore(t)

	tables := []string{"contacts", "companies", "relationships", "contacts_fts", "companies_fts"}
	for _, table := range tables {
		var count int
		err := store.db.QueryRow(
			`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestStoreFTSTriggers(t *testing.T) {
	store := newTestStore(t)

	triggers := []string{
		"contacts_ai", "contacts_ad", "contacts_au",
		"companies_ai", "companies_ad", "companies_au",
	}
	for _, trigger := range triggers {
		var count int
		err := store.db.QueryRow(
			`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name=?`, trigger,
		).Scan(&count)
		if err != nil {
			t.Fatalf("failed to check trigger %s: %v", trigger, err)
		}
		if count != 1 {
			t.Errorf("expected trigger %s to exist", trigger)
		}
	}
}

func TestDataDir(t *testing.T) {
	path := DataDir()
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}

func TestStoreCloseNilDB(t *testing.T) {
	store := &SqliteStore{db: nil}
	err := store.Close()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test ./internal/storage/...
```

Expected: FAIL — `SqliteStore` not defined.

- [ ] **Step 3: Write SQLite store with schema**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go get modernc.org/sqlite
```

Create `internal/storage/sqlite.go`:
```go
// ABOUTME: SQLite database initialization and connection management.
// ABOUTME: Uses modernc.org/sqlite (pure Go) with FTS5 for full-text search.

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// SqliteStore provides SQLite-based storage for CRM data.
type SqliteStore struct {
	db     *sql.DB
	dbPath string
}

// Compile-time check that SqliteStore implements Storage.
var _ Storage = (*SqliteStore)(nil)

// NewSqliteStore creates a new SqliteStore with the given database path.
func NewSqliteStore(dbPath string) (*SqliteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &SqliteStore{db: db, dbPath: dbPath}

	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *SqliteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// initSchema creates all required tables, indexes, and triggers.
func (s *SqliteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contacts (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
		id TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		email TEXT DEFAULT '',
		phone TEXT DEFAULT '',
		fields TEXT DEFAULT '{}',
		tags TEXT DEFAULT '[]',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS companies (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
		id TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		domain TEXT DEFAULT '',
		fields TEXT DEFAULT '{}',
		tags TEXT DEFAULT '[]',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS relationships (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		type TEXT NOT NULL,
		context TEXT DEFAULT '',
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_contacts_id ON contacts(id);
	CREATE INDEX IF NOT EXISTS idx_companies_id ON companies(id);
	CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_id);
	CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_id);

	CREATE VIRTUAL TABLE IF NOT EXISTS contacts_fts USING fts5(
		name, email, fields, content=contacts, content_rowid=rowid
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS companies_fts USING fts5(
		name, domain, fields, content=companies, content_rowid=rowid
	);
	`

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	triggers := []struct {
		name string
		sql  string
	}{
		{
			name: "contacts_ai",
			sql: `CREATE TRIGGER IF NOT EXISTS contacts_ai AFTER INSERT ON contacts BEGIN
				INSERT INTO contacts_fts(rowid, name, email, fields) VALUES (new.rowid, new.name, new.email, new.fields);
			END`,
		},
		{
			name: "contacts_ad",
			sql: `CREATE TRIGGER IF NOT EXISTS contacts_ad AFTER DELETE ON contacts BEGIN
				INSERT INTO contacts_fts(contacts_fts, rowid, name, email, fields) VALUES('delete', old.rowid, old.name, old.email, old.fields);
			END`,
		},
		{
			name: "contacts_au",
			sql: `CREATE TRIGGER IF NOT EXISTS contacts_au AFTER UPDATE ON contacts BEGIN
				INSERT INTO contacts_fts(contacts_fts, rowid, name, email, fields) VALUES('delete', old.rowid, old.name, old.email, old.fields);
				INSERT INTO contacts_fts(rowid, name, email, fields) VALUES (new.rowid, new.name, new.email, new.fields);
			END`,
		},
		{
			name: "companies_ai",
			sql: `CREATE TRIGGER IF NOT EXISTS companies_ai AFTER INSERT ON companies BEGIN
				INSERT INTO companies_fts(rowid, name, domain, fields) VALUES (new.rowid, new.name, new.domain, new.fields);
			END`,
		},
		{
			name: "companies_ad",
			sql: `CREATE TRIGGER IF NOT EXISTS companies_ad AFTER DELETE ON companies BEGIN
				INSERT INTO companies_fts(companies_fts, rowid, name, domain, fields) VALUES('delete', old.rowid, old.name, old.domain, old.fields);
			END`,
		},
		{
			name: "companies_au",
			sql: `CREATE TRIGGER IF NOT EXISTS companies_au AFTER UPDATE ON companies BEGIN
				INSERT INTO companies_fts(companies_fts, rowid, name, domain, fields) VALUES('delete', old.rowid, old.name, old.domain, old.fields);
				INSERT INTO companies_fts(rowid, name, domain, fields) VALUES (new.rowid, new.name, new.domain, new.fields);
			END`,
		},
	}

	for _, trigger := range triggers {
		if _, err := tx.Exec(trigger.sql); err != nil {
			return fmt.Errorf("create trigger %s: %w", trigger.name, err)
		}
	}

	return tx.Commit()
}

// DataDir returns the default data directory path.
func DataDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "crm")
}

// escapeFTS5Query escapes special characters in an FTS5 query string.
func escapeFTS5Query(query string) string {
	escaped := strings.ReplaceAll(query, `"`, `""`)
	return `"` + escaped + `"`
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/storage/...
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/storage/sqlite.go internal/storage/sqlite_test.go
git commit -m "feat: add SQLite store with schema, FTS5, and triggers"
```

---

### Task 5: SQLite Contact CRUD

**Files:**
- Create: `internal/storage/sqlite_contacts.go`, `internal/storage/sqlite_contacts_test.go`

- [ ] **Step 1: Write failing tests for contact CRUD**

Create `internal/storage/sqlite_contacts_test.go`:
```go
// ABOUTME: Tests for contact CRUD operations in SQLite storage.
// ABOUTME: Validates creation, retrieval, update, deletion, and filtering.

package storage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestCreateContact(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Jane Doe")
	c.Email = "jane@example.com"
	c.Phone = "+1-555-0123"
	c.Fields["title"] = "VP Engineering"
	c.Tags = []string{"engineering", "vip"}

	if err := store.CreateContact(c); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	got, err := store.GetContact(c.ID)
	if err != nil {
		t.Fatalf("failed to get contact: %v", err)
	}
	if got.Name != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got %q", got.Name)
	}
	if got.Email != "jane@example.com" {
		t.Errorf("expected email 'jane@example.com', got %q", got.Email)
	}
	if got.Fields["title"] != "VP Engineering" {
		t.Errorf("expected field title 'VP Engineering', got %v", got.Fields["title"])
	}
	if len(got.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(got.Tags))
	}
}

func TestGetContactNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetContact(uuid.New())
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestGetContactByPrefix(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Prefix Test")
	_ = store.CreateContact(c)

	got, err := store.GetContactByPrefix(c.ID.String()[:8])
	if err != nil {
		t.Fatalf("failed to get by prefix: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("expected ID %s, got %s", c.ID, got.ID)
	}

	_, err = store.GetContactByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestUpdateContact(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Original")
	_ = store.CreateContact(c)

	c.Name = "Updated"
	c.Email = "updated@example.com"
	c.Tags = []string{"updated"}
	c.Touch()

	if err := store.UpdateContact(c); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	got, err := store.GetContact(c.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("expected 'Updated', got %q", got.Name)
	}
	if got.Email != "updated@example.com" {
		t.Errorf("expected 'updated@example.com', got %q", got.Email)
	}
}

func TestDeleteContact(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Delete Me")
	_ = store.CreateContact(c)

	if err := store.DeleteContact(c.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := store.GetContact(c.ID)
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound after delete, got %v", err)
	}
}

func TestListContacts(t *testing.T) {
	store := newTestStore(t)

	c1 := models.NewContact("Alice")
	c1.Tags = []string{"team-a"}
	_ = store.CreateContact(c1)

	c2 := models.NewContact("Bob")
	c2.Tags = []string{"team-b"}
	_ = store.CreateContact(c2)

	// List all
	all, err := store.ListContacts(&ContactFilter{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(all))
	}

	// Filter by tag
	tag := "team-a"
	filtered, err := store.ListContacts(&ContactFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 contact with tag 'team-a', got %d", len(filtered))
	}

	// Limit
	limited, err := store.ListContacts(&ContactFilter{Limit: 1})
	if err != nil {
		t.Fatalf("failed to limit: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 contact with limit, got %d", len(limited))
	}
}

func TestListContactsSearch(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Searchable Person")
	c.Email = "search@example.com"
	_ = store.CreateContact(c)

	results, err := store.ListContacts(&ContactFilter{Search: "Searchable"})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 search result, got %d", len(results))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test ./internal/storage/...
```

Expected: FAIL — methods not defined on SqliteStore.

- [ ] **Step 3: Implement SQLite contact CRUD**

Create `internal/storage/sqlite_contacts.go`:
```go
// ABOUTME: Contact CRUD operations for SQLite storage.
// ABOUTME: Handles creation, retrieval, update, deletion with FTS5 search and tag filtering.

package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

// CreateContact inserts a new contact into the database.
func (s *SqliteStore) CreateContact(contact *models.Contact) error {
	fieldsJSON, err := json.Marshal(contact.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	tagsJSON, err := json.Marshal(contact.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO contacts (id, name, email, phone, fields, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, contact.ID.String(), contact.Name, contact.Email, contact.Phone,
		string(fieldsJSON), string(tagsJSON), contact.CreatedAt, contact.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert contact: %w", err)
	}

	return nil
}

// GetContact retrieves a contact by UUID.
func (s *SqliteStore) GetContact(id uuid.UUID) (*models.Contact, error) {
	row := s.db.QueryRow(`
		SELECT id, name, email, phone, fields, tags, created_at, updated_at
		FROM contacts WHERE id = ?
	`, id.String())

	c, err := scanContact(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}
	return c, nil
}

// GetContactByPrefix finds a contact by ID prefix (minimum 6 chars).
func (s *SqliteStore) GetContactByPrefix(prefix string) (*models.Contact, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := s.db.Query(`
		SELECT id, name, email, phone, fields, tags, created_at, updated_at
		FROM contacts WHERE id LIKE ?
	`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query contacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var contacts []*models.Contact
	for rows.Next() {
		c, err := scanContactRows(rows)
		if err != nil {
			continue
		}
		contacts = append(contacts, c)
	}

	if len(contacts) == 0 {
		return nil, ErrContactNotFound
	}
	if len(contacts) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(contacts))
	}
	return contacts[0], nil
}

// UpdateContact updates an existing contact.
func (s *SqliteStore) UpdateContact(contact *models.Contact) error {
	fieldsJSON, err := json.Marshal(contact.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	tagsJSON, err := json.Marshal(contact.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	result, err := s.db.Exec(`
		UPDATE contacts SET name=?, email=?, phone=?, fields=?, tags=?, updated_at=?
		WHERE id=?
	`, contact.Name, contact.Email, contact.Phone,
		string(fieldsJSON), string(tagsJSON), contact.UpdatedAt, contact.ID.String())
	if err != nil {
		return fmt.Errorf("update contact: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrContactNotFound
	}
	return nil
}

// DeleteContact removes a contact by ID.
func (s *SqliteStore) DeleteContact(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM contacts WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrContactNotFound
	}
	return nil
}

// ListContacts returns contacts matching the filter.
func (s *SqliteStore) ListContacts(filter *ContactFilter) ([]*models.Contact, error) {
	if filter == nil {
		filter = &ContactFilter{}
	}

	if filter.Search != "" {
		return s.searchContacts(filter)
	}

	query := `SELECT id, name, email, phone, fields, tags, created_at, updated_at FROM contacts`
	var args []interface{}
	var conditions []string

	if filter.Tag != nil {
		conditions = append(conditions, `json_each.value = ?`)
		query = `SELECT c.id, c.name, c.email, c.phone, c.fields, c.tags, c.created_at, c.updated_at
			FROM contacts c, json_each(c.tags)`
		args = append(args, strings.ToLower(*filter.Tag))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY updated_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query contacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var contacts []*models.Contact
	for rows.Next() {
		c, err := scanContactRows(rows)
		if err != nil {
			continue
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (s *SqliteStore) searchContacts(filter *ContactFilter) ([]*models.Contact, error) {
	query := `
		SELECT c.id, c.name, c.email, c.phone, c.fields, c.tags, c.created_at, c.updated_at
		FROM contacts c
		JOIN contacts_fts fts ON c.rowid = fts.rowid
		WHERE contacts_fts MATCH ?
		ORDER BY c.updated_at DESC
	`
	args := []interface{}{escapeFTS5Query(filter.Search)}

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var contacts []*models.Contact
	for rows.Next() {
		c, err := scanContactRows(rows)
		if err != nil {
			continue
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func scanContact(row *sql.Row) (*models.Contact, error) {
	var idStr, name, email, phone, fieldsStr, tagsStr string
	var createdAt, updatedAt time.Time

	if err := row.Scan(&idStr, &name, &email, &phone, &fieldsStr, &tagsStr, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse contact ID: %w", err)
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
		fields = make(map[string]any)
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
		tags = []string{}
	}

	return &models.Contact{
		ID: id, Name: name, Email: email, Phone: phone,
		Fields: fields, Tags: tags, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func scanContactRows(rows *sql.Rows) (*models.Contact, error) {
	var idStr, name, email, phone, fieldsStr, tagsStr string
	var createdAt, updatedAt time.Time

	if err := rows.Scan(&idStr, &name, &email, &phone, &fieldsStr, &tagsStr, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse contact ID: %w", err)
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
		fields = make(map[string]any)
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
		tags = []string{}
	}

	return &models.Contact{
		ID: id, Name: name, Email: email, Phone: phone,
		Fields: fields, Tags: tags, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/storage/...
```

Expected: all contact tests PASS (some may fail because company/relationship methods aren't implemented yet — that's OK, the compile-time interface check will be deferred until all methods exist).

- [ ] **Step 5: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/storage/sqlite_contacts.go internal/storage/sqlite_contacts_test.go
git commit -m "feat: add SQLite contact CRUD with FTS5 search"
```

---

### Task 6: SQLite Company & Relationship CRUD

**Files:**
- Create: `internal/storage/sqlite_companies.go`, `internal/storage/sqlite_companies_test.go`, `internal/storage/sqlite_relationships.go`, `internal/storage/sqlite_relationships_test.go`, `internal/storage/sqlite_search.go`, `internal/storage/sqlite_search_test.go`

- [ ] **Step 1: Write failing tests for company CRUD**

Create `internal/storage/sqlite_companies_test.go`:
```go
// ABOUTME: Tests for company CRUD operations in SQLite storage.
// ABOUTME: Validates creation, retrieval, update, deletion, and filtering.

package storage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestCreateCompany(t *testing.T) {
	store := newTestStore(t)

	c := models.NewCompany("Acme Corp")
	c.Domain = "acme.com"
	c.Fields["industry"] = "Tech"
	c.Tags = []string{"partner"}

	if err := store.CreateCompany(c); err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	got, err := store.GetCompany(c.ID)
	if err != nil {
		t.Fatalf("failed to get company: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", got.Name)
	}
	if got.Domain != "acme.com" {
		t.Errorf("expected 'acme.com', got %q", got.Domain)
	}
	if got.Fields["industry"] != "Tech" {
		t.Errorf("expected field industry 'Tech', got %v", got.Fields["industry"])
	}
}

func TestGetCompanyNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetCompany(uuid.New())
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestGetCompanyByPrefix(t *testing.T) {
	store := newTestStore(t)
	c := models.NewCompany("Prefix Corp")
	_ = store.CreateCompany(c)

	got, err := store.GetCompanyByPrefix(c.ID.String()[:8])
	if err != nil {
		t.Fatalf("failed to get by prefix: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("expected ID %s, got %s", c.ID, got.ID)
	}
}

func TestUpdateCompany(t *testing.T) {
	store := newTestStore(t)
	c := models.NewCompany("Original Corp")
	_ = store.CreateCompany(c)

	c.Name = "Updated Corp"
	c.Domain = "updated.com"
	c.Touch()

	if err := store.UpdateCompany(c); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	got, err := store.GetCompany(c.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if got.Name != "Updated Corp" {
		t.Errorf("expected 'Updated Corp', got %q", got.Name)
	}
}

func TestDeleteCompany(t *testing.T) {
	store := newTestStore(t)
	c := models.NewCompany("Delete Corp")
	_ = store.CreateCompany(c)

	if err := store.DeleteCompany(c.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := store.GetCompany(c.ID)
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestListCompanies(t *testing.T) {
	store := newTestStore(t)
	c1 := models.NewCompany("Alpha")
	c1.Tags = []string{"partner"}
	_ = store.CreateCompany(c1)

	c2 := models.NewCompany("Beta")
	c2.Tags = []string{"customer"}
	_ = store.CreateCompany(c2)

	all, err := store.ListCompanies(&CompanyFilter{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 companies, got %d", len(all))
	}

	tag := "partner"
	filtered, err := store.ListCompanies(&CompanyFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 company, got %d", len(filtered))
	}
}
```

- [ ] **Step 2: Implement SQLite company CRUD**

Create `internal/storage/sqlite_companies.go` — same pattern as `sqlite_contacts.go` but for companies. Uses `scanCompany`/`scanCompanyRows` helpers, `domain` instead of `email`/`phone`:

```go
// ABOUTME: Company CRUD operations for SQLite storage.
// ABOUTME: Handles creation, retrieval, update, deletion with FTS5 search and tag filtering.

package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func (s *SqliteStore) CreateCompany(company *models.Company) error {
	fieldsJSON, err := json.Marshal(company.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	tagsJSON, err := json.Marshal(company.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO companies (id, name, domain, fields, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, company.ID.String(), company.Name, company.Domain,
		string(fieldsJSON), string(tagsJSON), company.CreatedAt, company.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert company: %w", err)
	}
	return nil
}

func (s *SqliteStore) GetCompany(id uuid.UUID) (*models.Company, error) {
	row := s.db.QueryRow(`
		SELECT id, name, domain, fields, tags, created_at, updated_at
		FROM companies WHERE id = ?
	`, id.String())

	c, err := scanCompany(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCompanyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}
	return c, nil
}

func (s *SqliteStore) GetCompanyByPrefix(prefix string) (*models.Company, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := s.db.Query(`
		SELECT id, name, domain, fields, tags, created_at, updated_at
		FROM companies WHERE id LIKE ?
	`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var companies []*models.Company
	for rows.Next() {
		c, err := scanCompanyRows(rows)
		if err != nil {
			continue
		}
		companies = append(companies, c)
	}

	if len(companies) == 0 {
		return nil, ErrCompanyNotFound
	}
	if len(companies) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(companies))
	}
	return companies[0], nil
}

func (s *SqliteStore) UpdateCompany(company *models.Company) error {
	fieldsJSON, err := json.Marshal(company.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	tagsJSON, err := json.Marshal(company.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	result, err := s.db.Exec(`
		UPDATE companies SET name=?, domain=?, fields=?, tags=?, updated_at=?
		WHERE id=?
	`, company.Name, company.Domain, string(fieldsJSON), string(tagsJSON),
		company.UpdatedAt, company.ID.String())
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrCompanyNotFound
	}
	return nil
}

func (s *SqliteStore) DeleteCompany(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM companies WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrCompanyNotFound
	}
	return nil
}

func (s *SqliteStore) ListCompanies(filter *CompanyFilter) ([]*models.Company, error) {
	if filter == nil {
		filter = &CompanyFilter{}
	}

	if filter.Search != "" {
		return s.searchCompanies(filter)
	}

	query := `SELECT id, name, domain, fields, tags, created_at, updated_at FROM companies`
	var args []interface{}
	var conditions []string

	if filter.Tag != nil {
		query = `SELECT c.id, c.name, c.domain, c.fields, c.tags, c.created_at, c.updated_at
			FROM companies c, json_each(c.tags)`
		conditions = append(conditions, `json_each.value = ?`)
		args = append(args, strings.ToLower(*filter.Tag))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY updated_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var companies []*models.Company
	for rows.Next() {
		c, err := scanCompanyRows(rows)
		if err != nil {
			continue
		}
		companies = append(companies, c)
	}
	return companies, nil
}

func (s *SqliteStore) searchCompanies(filter *CompanyFilter) ([]*models.Company, error) {
	query := `
		SELECT c.id, c.name, c.domain, c.fields, c.tags, c.created_at, c.updated_at
		FROM companies c
		JOIN companies_fts fts ON c.rowid = fts.rowid
		WHERE companies_fts MATCH ?
		ORDER BY c.updated_at DESC
	`
	args := []interface{}{escapeFTS5Query(filter.Search)}
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search companies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var companies []*models.Company
	for rows.Next() {
		c, err := scanCompanyRows(rows)
		if err != nil {
			continue
		}
		companies = append(companies, c)
	}
	return companies, nil
}

func scanCompany(row *sql.Row) (*models.Company, error) {
	var idStr, name, domain, fieldsStr, tagsStr string
	var createdAt, updatedAt time.Time
	if err := row.Scan(&idStr, &name, &domain, &fieldsStr, &tagsStr, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse company ID: %w", err)
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
		fields = make(map[string]any)
	}
	var tags []string
	if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
		tags = []string{}
	}
	return &models.Company{
		ID: id, Name: name, Domain: domain,
		Fields: fields, Tags: tags, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func scanCompanyRows(rows *sql.Rows) (*models.Company, error) {
	var idStr, name, domain, fieldsStr, tagsStr string
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&idStr, &name, &domain, &fieldsStr, &tagsStr, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse company ID: %w", err)
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
		fields = make(map[string]any)
	}
	var tags []string
	if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
		tags = []string{}
	}
	return &models.Company{
		ID: id, Name: name, Domain: domain,
		Fields: fields, Tags: tags, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}
```

- [ ] **Step 3: Write failing relationship tests**

Create `internal/storage/sqlite_relationships_test.go`:
```go
// ABOUTME: Tests for relationship CRUD operations in SQLite storage.
// ABOUTME: Validates creation, listing by entity, and deletion.

package storage

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func TestCreateRelationship(t *testing.T) {
	store := newTestStore(t)

	contact := models.NewContact("Jane")
	_ = store.CreateContact(contact)
	company := models.NewCompany("Acme")
	_ = store.CreateCompany(company)

	rel := models.NewRelationship(contact.ID, company.ID, "works_at", "VP Engineering")
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("failed to create relationship: %v", err)
	}

	rels, err := store.ListRelationships(contact.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
	if rels[0].Type != "works_at" {
		t.Errorf("expected type 'works_at', got %q", rels[0].Type)
	}
	if rels[0].Context != "VP Engineering" {
		t.Errorf("expected context 'VP Engineering', got %q", rels[0].Context)
	}
}

func TestListRelationshipsBidirectional(t *testing.T) {
	store := newTestStore(t)

	c1 := models.NewContact("Alice")
	_ = store.CreateContact(c1)
	c2 := models.NewContact("Bob")
	_ = store.CreateContact(c2)

	rel := models.NewRelationship(c1.ID, c2.ID, "knows", "college friends")
	_ = store.CreateRelationship(rel)

	// Should appear when querying from either side
	fromSource, _ := store.ListRelationships(c1.ID)
	if len(fromSource) != 1 {
		t.Errorf("expected 1 relationship from source, got %d", len(fromSource))
	}

	fromTarget, _ := store.ListRelationships(c2.ID)
	if len(fromTarget) != 1 {
		t.Errorf("expected 1 relationship from target, got %d", len(fromTarget))
	}
}

func TestDeleteRelationship(t *testing.T) {
	store := newTestStore(t)

	c1 := models.NewContact("Alice")
	_ = store.CreateContact(c1)
	c2 := models.NewContact("Bob")
	_ = store.CreateContact(c2)

	rel := models.NewRelationship(c1.ID, c2.ID, "knows", "")
	_ = store.CreateRelationship(rel)

	if err := store.DeleteRelationship(rel.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	rels, _ := store.ListRelationships(c1.ID)
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships after delete, got %d", len(rels))
	}
}

func TestDeleteRelationshipNotFound(t *testing.T) {
	store := newTestStore(t)
	err := store.DeleteRelationship(uuid.New())
	if !errors.Is(err, ErrRelationshipNotFound) {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}
}
```

- [ ] **Step 4: Implement SQLite relationship CRUD**

Create `internal/storage/sqlite_relationships.go`:
```go
// ABOUTME: Relationship CRUD operations for SQLite storage.
// ABOUTME: Handles creation, bidirectional listing, and deletion of entity links.

package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func (s *SqliteStore) CreateRelationship(rel *models.Relationship) error {
	_, err := s.db.Exec(`
		INSERT INTO relationships (id, source_id, target_id, type, context, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, rel.ID.String(), rel.SourceID.String(), rel.TargetID.String(),
		rel.Type, rel.Context, rel.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert relationship: %w", err)
	}
	return nil
}

func (s *SqliteStore) ListRelationships(entityID uuid.UUID) ([]*models.Relationship, error) {
	rows, err := s.db.Query(`
		SELECT id, source_id, target_id, type, context, created_at
		FROM relationships
		WHERE source_id = ? OR target_id = ?
		ORDER BY created_at DESC
	`, entityID.String(), entityID.String())
	if err != nil {
		return nil, fmt.Errorf("query relationships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rels []*models.Relationship
	for rows.Next() {
		var idStr, srcStr, tgtStr, relType, context string
		var createdAt time.Time
		if err := rows.Scan(&idStr, &srcStr, &tgtStr, &relType, &context, &createdAt); err != nil {
			continue
		}
		id, _ := uuid.Parse(idStr)
		srcID, _ := uuid.Parse(srcStr)
		tgtID, _ := uuid.Parse(tgtStr)
		rels = append(rels, &models.Relationship{
			ID: id, SourceID: srcID, TargetID: tgtID,
			Type: relType, Context: context, CreatedAt: createdAt,
		})
	}
	return rels, nil
}

func (s *SqliteStore) DeleteRelationship(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM relationships WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete relationship: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrRelationshipNotFound
	}
	return nil
}
```

- [ ] **Step 5: Write cross-entity search test and implementation**

Create `internal/storage/sqlite_search_test.go`:
```go
// ABOUTME: Tests for cross-entity search in SQLite storage.
// ABOUTME: Validates FTS5 search across contacts and companies.

package storage

import (
	"testing"

	"github.com/harperreed/crm/internal/models"
)

func TestSearch(t *testing.T) {
	store := newTestStore(t)

	c := models.NewContact("Findable Person")
	c.Email = "findable@example.com"
	_ = store.CreateContact(c)

	co := models.NewCompany("Findable Corp")
	co.Domain = "findable.com"
	_ = store.CreateCompany(co)

	_ = models.NewContact("Invisible Person")

	results, err := store.Search("Findable")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results.Contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(results.Contacts))
	}
	if len(results.Companies) != 1 {
		t.Errorf("expected 1 company, got %d", len(results.Companies))
	}
}
```

Create `internal/storage/sqlite_search.go`:
```go
// ABOUTME: Cross-entity search for SQLite storage.
// ABOUTME: Searches contacts and companies via FTS5 and merges results.

package storage

func (s *SqliteStore) Search(query string) (*SearchResults, error) {
	contacts, err := s.searchContacts(&ContactFilter{Search: query})
	if err != nil {
		return nil, err
	}
	companies, err := s.searchCompanies(&CompanyFilter{Search: query})
	if err != nil {
		return nil, err
	}
	return &SearchResults{Contacts: contacts, Companies: companies}, nil
}
```

- [ ] **Step 6: Run all storage tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/storage/...
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/storage/sqlite_companies.go internal/storage/sqlite_companies_test.go \
        internal/storage/sqlite_relationships.go internal/storage/sqlite_relationships_test.go \
        internal/storage/sqlite_search.go internal/storage/sqlite_search_test.go
git commit -m "feat: add SQLite company, relationship, and search CRUD"
```

---

### Task 7: Config

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`

- [ ] **Step 1: Write failing config test**

Create `internal/config/config_test.go`:
```go
// ABOUTME: Tests for CRM configuration loading and backend factory.
// ABOUTME: Validates XDG paths, backend selection, and storage creation.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := &Config{}
	if cfg.GetBackend() != "sqlite" {
		t.Errorf("expected default backend 'sqlite', got %q", cfg.GetBackend())
	}
}

func TestGetDataDir(t *testing.T) {
	cfg := &Config{}
	dir := cfg.GetDataDir()
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
}

func TestGetDataDirCustom(t *testing.T) {
	cfg := &Config{DataDir: "~/custom"}
	dir := cfg.GetDataDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "custom")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"~", home},
		{"~/foo", filepath.Join(home, "foo")},
		{"/abs/path", "/abs/path"},
	}
	for _, tc := range tests {
		got := ExpandPath(tc.input)
		if got != tc.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestOpenStorageSqlite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "crm-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{Backend: "sqlite", DataDir: tmpDir}
	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()
}

func TestOpenStorageUnknown(t *testing.T) {
	cfg := &Config{Backend: "unknown"}
	_, err := cfg.OpenStorage()
	if err == nil {
		t.Error("expected error for unknown backend")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test ./internal/config/...
```

Expected: FAIL

- [ ] **Step 3: Implement config**

Create `internal/config/config.go`:
```go
// ABOUTME: CRM configuration management with backend selection.
// ABOUTME: Handles settings, XDG paths, and storage backend factory.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harperreed/crm/internal/storage"
)

// Config stores CRM configuration.
type Config struct {
	Backend string `json:"backend,omitempty"`
	DataDir string `json:"data_dir,omitempty"`
}

// GetBackend returns the configured backend, defaulting to "sqlite".
func (c *Config) GetBackend() string {
	if c.Backend == "" {
		return "sqlite"
	}
	return c.Backend
}

// GetDataDir returns the configured data directory with ~ expanded.
func (c *Config) GetDataDir() string {
	if c.DataDir == "" {
		return storage.DataDir()
	}
	return ExpandPath(c.DataDir)
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// OpenStorage creates a Storage implementation based on the configured backend.
func (c *Config) OpenStorage() (storage.Storage, error) {
	backend := c.GetBackend()
	dataDir := c.GetDataDir()

	switch backend {
	case "sqlite":
		dbPath := filepath.Join(dataDir, "crm.db")
		return storage.NewSqliteStore(dbPath)
	case "markdown":
		return storage.NewMarkdownStore(dataDir)
	default:
		return nil, fmt.Errorf("unknown backend: %q", backend)
	}
}

// GetConfigPath returns the config file path.
func GetConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "crm", "config.json")
}

// Load reads config from disk, creating defaults if missing.
func Load() (*Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Backend: "sqlite"}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

Note: The config test references `NewMarkdownStore` — we need a stub. Create a temporary stub in `internal/storage/markdown.go`:

```go
// ABOUTME: Markdown file-based storage for CRM data.
// ABOUTME: Provides constructor and helpers for the markdown backend.

package storage

import "fmt"

// MarkdownStore provides file-based storage for CRM data.
type MarkdownStore struct {
	dataDir string
}

// NewMarkdownStore creates a new markdown-backed store.
func NewMarkdownStore(dataDir string) (*MarkdownStore, error) {
	return nil, fmt.Errorf("markdown backend not yet implemented")
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/config/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/config/config.go internal/config/config_test.go internal/storage/markdown.go
git commit -m "feat: add XDG config with backend factory"
```

---

### Task 8: Root Command with Storage Init

**Files:**
- Modify: `cmd/crm/root.go`

- [ ] **Step 1: Update root.go with storage initialization**

Replace `cmd/crm/root.go`:
```go
// ABOUTME: Root command for crm CLI with config-driven storage initialization.
// ABOUTME: Handles global flags and persistent pre/post run hooks.

package main

import (
	"fmt"

	"github.com/harperreed/crm/internal/config"
	"github.com/harperreed/crm/internal/storage"
	"github.com/spf13/cobra"
)

var (
	store storage.Storage
)

var rootCmd = &cobra.Command{
	Use:   "crm",
	Short: "A simple CRM for contacts, companies, and relationships",
	Long:  `crm is a command-line CRM that manages contacts, companies, and relationships. Accessible via CLI and MCP.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "version" {
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		store, err = cfg.OpenStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if store != nil {
			return store.Close()
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go build ./cmd/crm
```

Expected: exit 0

- [ ] **Step 3: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add cmd/crm/root.go
git commit -m "feat: add config-driven storage initialization to root command"
```

---

### Task 9: CLI Commands — Contacts

**Files:**
- Create: `cmd/crm/contacts.go`

- [ ] **Step 1: Implement contact CLI commands**

Create `cmd/crm/contacts.go`:
```go
// ABOUTME: CLI commands for contact management.
// ABOUTME: Provides add, list, show, edit, and rm subcommands.

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var contactCmd = &cobra.Command{
	Use:     "contact",
	Aliases: []string{"c"},
	Short:   "Manage contacts",
}

var contactAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := models.NewContact(args[0])
		c.Email, _ = cmd.Flags().GetString("email")
		c.Phone, _ = cmd.Flags().GetString("phone")

		fields, _ := cmd.Flags().GetStringSlice("field")
		for _, f := range fields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) == 2 {
				c.Fields[parts[0]] = parts[1]
			}
		}

		tags, _ := cmd.Flags().GetStringSlice("tag")
		c.Tags = tags

		if err := store.CreateContact(c); err != nil {
			return fmt.Errorf("failed to create contact: %w", err)
		}
		fmt.Printf("Created contact %s (%s)\n", c.Name, c.ID.String()[:8])
		return nil
	},
}

var contactListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		tagFlag, _ := cmd.Flags().GetString("tag")
		searchFlag, _ := cmd.Flags().GetString("search")
		limitFlag, _ := cmd.Flags().GetInt("limit")

		filter := &storage.ContactFilter{Limit: limitFlag}
		if tagFlag != "" {
			filter.Tag = &tagFlag
		}
		if searchFlag != "" {
			filter.Search = searchFlag
		}

		contacts, err := store.ListContacts(filter)
		if err != nil {
			return fmt.Errorf("failed to list contacts: %w", err)
		}

		if len(contacts) == 0 {
			fmt.Println("No contacts found.")
			return nil
		}

		for _, c := range contacts {
			id := color.New(color.FgCyan).Sprintf(c.ID.String()[:8])
			name := color.New(color.Bold).Sprint(c.Name)
			extra := ""
			if c.Email != "" {
				extra = " <" + c.Email + ">"
			}
			tags := ""
			if len(c.Tags) > 0 {
				tags = " [" + strings.Join(c.Tags, ", ") + "]"
			}
			fmt.Printf("%s  %s%s%s\n", id, name, extra, tags)
		}
		return nil
	},
}

var contactShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show contact details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveContact(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("ID:      %s\n", c.ID)
		fmt.Printf("Name:    %s\n", c.Name)
		if c.Email != "" {
			fmt.Printf("Email:   %s\n", c.Email)
		}
		if c.Phone != "" {
			fmt.Printf("Phone:   %s\n", c.Phone)
		}
		if len(c.Tags) > 0 {
			fmt.Printf("Tags:    %s\n", strings.Join(c.Tags, ", "))
		}
		if len(c.Fields) > 0 {
			data, _ := json.MarshalIndent(c.Fields, "         ", "  ")
			fmt.Printf("Fields:  %s\n", string(data))
		}

		rels, _ := store.ListRelationships(c.ID)
		if len(rels) > 0 {
			fmt.Println("\nRelationships:")
			for _, r := range rels {
				fmt.Printf("  %s -> %s (%s) %s\n",
					r.SourceID.String()[:8], r.TargetID.String()[:8], r.Type, r.Context)
			}
		}
		return nil
	},
}

var contactEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveContact(args[0])
		if err != nil {
			return err
		}

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			c.Name = name
		}
		if email, _ := cmd.Flags().GetString("email"); email != "" {
			c.Email = email
		}
		if phone, _ := cmd.Flags().GetString("phone"); phone != "" {
			c.Phone = phone
		}

		fields, _ := cmd.Flags().GetStringSlice("field")
		for _, f := range fields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) == 2 {
				c.Fields[parts[0]] = parts[1]
			}
		}

		tags, _ := cmd.Flags().GetStringSlice("tag")
		if len(tags) > 0 {
			c.Tags = tags
		}

		c.Touch()
		if err := store.UpdateContact(c); err != nil {
			return fmt.Errorf("failed to update contact: %w", err)
		}
		fmt.Printf("Updated contact %s\n", c.ID.String()[:8])
		return nil
	},
}

var contactRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Aliases: []string{"delete", "del"},
	Short:   "Delete a contact",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveContact(args[0])
		if err != nil {
			return err
		}
		if err := store.DeleteContact(c.ID); err != nil {
			return fmt.Errorf("failed to delete: %w", err)
		}
		fmt.Printf("Deleted contact %s (%s)\n", c.Name, c.ID.String()[:8])
		return nil
	},
}

func resolveContact(idStr string) (*models.Contact, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return store.GetContact(id)
	}
	return store.GetContactByPrefix(idStr)
}

func init() {
	contactAddCmd.Flags().String("email", "", "email address")
	contactAddCmd.Flags().String("phone", "", "phone number")
	contactAddCmd.Flags().StringSlice("field", nil, "custom field (key=value)")
	contactAddCmd.Flags().StringSlice("tag", nil, "tag")

	contactListCmd.Flags().StringP("tag", "t", "", "filter by tag")
	contactListCmd.Flags().StringP("search", "s", "", "search query")
	contactListCmd.Flags().IntP("limit", "n", 20, "max results")

	contactEditCmd.Flags().String("name", "", "new name")
	contactEditCmd.Flags().String("email", "", "new email")
	contactEditCmd.Flags().String("phone", "", "new phone")
	contactEditCmd.Flags().StringSlice("field", nil, "custom field (key=value)")
	contactEditCmd.Flags().StringSlice("tag", nil, "replace tags")

	contactCmd.AddCommand(contactAddCmd, contactListCmd, contactShowCmd, contactEditCmd, contactRmCmd)
	rootCmd.AddCommand(contactCmd)
}
```

- [ ] **Step 2: Add fatih/color dependency and verify compilation**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go get github.com/fatih/color@v1.18.0 && go build ./cmd/crm
```

Expected: exit 0

- [ ] **Step 3: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add cmd/crm/contacts.go go.mod go.sum
git commit -m "feat: add contact CLI commands (add/list/show/edit/rm)"
```

---

### Task 10: CLI Commands — Companies & Relationships

**Files:**
- Create: `cmd/crm/companies.go`, `cmd/crm/relationships.go`

- [ ] **Step 1: Implement company CLI commands**

Create `cmd/crm/companies.go` — same pattern as contacts.go but with `domain` instead of `email`/`phone`, and `resolveCompany` helper. Register under `company` command with aliases `["co"]`.

Code follows the exact same structure as `cmd/crm/contacts.go` but replaces:
- `Contact` → `Company`
- `contact` → `company`
- `email`/`phone` → `domain`
- `resolveContact` → `resolveCompany`
- Company show displays `Domain:` instead of `Email:`/`Phone:`

```go
// ABOUTME: CLI commands for company management.
// ABOUTME: Provides add, list, show, edit, and rm subcommands.

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var companyCmd = &cobra.Command{
	Use:     "company",
	Aliases: []string{"co"},
	Short:   "Manage companies",
}

var companyAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new company",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := models.NewCompany(args[0])
		c.Domain, _ = cmd.Flags().GetString("domain")

		fields, _ := cmd.Flags().GetStringSlice("field")
		for _, f := range fields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) == 2 {
				c.Fields[parts[0]] = parts[1]
			}
		}

		tags, _ := cmd.Flags().GetStringSlice("tag")
		c.Tags = tags

		if err := store.CreateCompany(c); err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}
		fmt.Printf("Created company %s (%s)\n", c.Name, c.ID.String()[:8])
		return nil
	},
}

var companyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List companies",
	RunE: func(cmd *cobra.Command, args []string) error {
		tagFlag, _ := cmd.Flags().GetString("tag")
		searchFlag, _ := cmd.Flags().GetString("search")
		limitFlag, _ := cmd.Flags().GetInt("limit")

		filter := &storage.CompanyFilter{Limit: limitFlag}
		if tagFlag != "" {
			filter.Tag = &tagFlag
		}
		if searchFlag != "" {
			filter.Search = searchFlag
		}

		companies, err := store.ListCompanies(filter)
		if err != nil {
			return fmt.Errorf("failed to list companies: %w", err)
		}

		if len(companies) == 0 {
			fmt.Println("No companies found.")
			return nil
		}

		for _, c := range companies {
			id := color.New(color.FgCyan).Sprintf(c.ID.String()[:8])
			name := color.New(color.Bold).Sprint(c.Name)
			extra := ""
			if c.Domain != "" {
				extra = " (" + c.Domain + ")"
			}
			tags := ""
			if len(c.Tags) > 0 {
				tags = " [" + strings.Join(c.Tags, ", ") + "]"
			}
			fmt.Printf("%s  %s%s%s\n", id, name, extra, tags)
		}
		return nil
	},
}

var companyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show company details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveCompany(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("ID:      %s\n", c.ID)
		fmt.Printf("Name:    %s\n", c.Name)
		if c.Domain != "" {
			fmt.Printf("Domain:  %s\n", c.Domain)
		}
		if len(c.Tags) > 0 {
			fmt.Printf("Tags:    %s\n", strings.Join(c.Tags, ", "))
		}
		if len(c.Fields) > 0 {
			data, _ := json.MarshalIndent(c.Fields, "         ", "  ")
			fmt.Printf("Fields:  %s\n", string(data))
		}

		rels, _ := store.ListRelationships(c.ID)
		if len(rels) > 0 {
			fmt.Println("\nRelationships:")
			for _, r := range rels {
				fmt.Printf("  %s -> %s (%s) %s\n",
					r.SourceID.String()[:8], r.TargetID.String()[:8], r.Type, r.Context)
			}
		}
		return nil
	},
}

var companyEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a company",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveCompany(args[0])
		if err != nil {
			return err
		}

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			c.Name = name
		}
		if domain, _ := cmd.Flags().GetString("domain"); domain != "" {
			c.Domain = domain
		}

		fields, _ := cmd.Flags().GetStringSlice("field")
		for _, f := range fields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) == 2 {
				c.Fields[parts[0]] = parts[1]
			}
		}

		tags, _ := cmd.Flags().GetStringSlice("tag")
		if len(tags) > 0 {
			c.Tags = tags
		}

		c.Touch()
		if err := store.UpdateCompany(c); err != nil {
			return fmt.Errorf("failed to update company: %w", err)
		}
		fmt.Printf("Updated company %s\n", c.ID.String()[:8])
		return nil
	},
}

var companyRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Aliases: []string{"delete", "del"},
	Short:   "Delete a company",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveCompany(args[0])
		if err != nil {
			return err
		}
		if err := store.DeleteCompany(c.ID); err != nil {
			return fmt.Errorf("failed to delete: %w", err)
		}
		fmt.Printf("Deleted company %s (%s)\n", c.Name, c.ID.String()[:8])
		return nil
	},
}

func resolveCompany(idStr string) (*models.Company, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return store.GetCompany(id)
	}
	return store.GetCompanyByPrefix(idStr)
}

func init() {
	companyAddCmd.Flags().String("domain", "", "website domain")
	companyAddCmd.Flags().StringSlice("field", nil, "custom field (key=value)")
	companyAddCmd.Flags().StringSlice("tag", nil, "tag")

	companyListCmd.Flags().StringP("tag", "t", "", "filter by tag")
	companyListCmd.Flags().StringP("search", "s", "", "search query")
	companyListCmd.Flags().IntP("limit", "n", 20, "max results")

	companyEditCmd.Flags().String("name", "", "new name")
	companyEditCmd.Flags().String("domain", "", "new domain")
	companyEditCmd.Flags().StringSlice("field", nil, "custom field (key=value)")
	companyEditCmd.Flags().StringSlice("tag", nil, "replace tags")

	companyCmd.AddCommand(companyAddCmd, companyListCmd, companyShowCmd, companyEditCmd, companyRmCmd)
	rootCmd.AddCommand(companyCmd)
}
```

- [ ] **Step 2: Implement relationship CLI commands**

Create `cmd/crm/relationships.go`:
```go
// ABOUTME: CLI commands for relationship management.
// ABOUTME: Provides link and unlink subcommands for connecting entities.

package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <source-id> <target-id>",
	Short: "Create a relationship between two entities",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid source ID: %w", err)
		}
		tgtID, err := uuid.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid target ID: %w", err)
		}

		relType, _ := cmd.Flags().GetString("type")
		if relType == "" {
			return fmt.Errorf("--type is required")
		}
		context, _ := cmd.Flags().GetString("context")

		rel := models.NewRelationship(srcID, tgtID, relType, context)
		if err := store.CreateRelationship(rel); err != nil {
			return fmt.Errorf("failed to create relationship: %w", err)
		}
		fmt.Printf("Linked %s -> %s (%s)\n", args[0][:8], args[1][:8], relType)
		return nil
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <relationship-id>",
	Short: "Remove a relationship",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid relationship ID: %w", err)
		}
		if err := store.DeleteRelationship(id); err != nil {
			return fmt.Errorf("failed to delete: %w", err)
		}
		fmt.Printf("Unlinked %s\n", args[0][:8])
		return nil
	},
}

func init() {
	linkCmd.Flags().String("type", "", "relationship type (required)")
	linkCmd.Flags().String("context", "", "relationship context")
	_ = linkCmd.MarkFlagRequired("type")

	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(unlinkCmd)
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go build ./cmd/crm
```

Expected: exit 0

- [ ] **Step 4: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add cmd/crm/companies.go cmd/crm/relationships.go
git commit -m "feat: add company and relationship CLI commands"
```

---

### Task 11: MCP Server

**Files:**
- Create: `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/resources.go`, `internal/mcp/prompts.go`, `cmd/crm/mcp.go`, `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing MCP server test**

Create `internal/mcp/server_test.go`:
```go
// ABOUTME: Tests for MCP server initialization.
// ABOUTME: Validates server creation and tool/resource/prompt registration.

package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harperreed/crm/internal/storage"
)

func newTestStore(t *testing.T) storage.Storage {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "crm-mcp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := storage.NewSqliteStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewServer(t *testing.T) {
	store := newTestStore(t)
	s := NewServer(store)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.store == nil {
		t.Fatal("expected non-nil store")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go get github.com/modelcontextprotocol/go-sdk@v1.1.0 && go test ./internal/mcp/...
```

Expected: FAIL — `NewServer` not defined.

- [ ] **Step 3: Implement MCP server**

Create `internal/mcp/server.go`:
```go
// ABOUTME: MCP server for CRM integration with AI agents.
// ABOUTME: Provides tools, resources, and prompts for contact and company management.

package mcp

import (
	"context"

	"github.com/harperreed/crm/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	server *mcp.Server
	store  storage.Storage
}

func NewServer(store storage.Storage) *Server {
	s := &Server{store: store}

	s.server = mcp.NewServer(
		&mcp.Implementation{
			Name:    "crm",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			HasTools:     true,
			HasResources: true,
			HasPrompts:   true,
		},
	)

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

func (s *Server) Serve(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
```

- [ ] **Step 4: Implement MCP tools**

Create `internal/mcp/tools.go`:
```go
// ABOUTME: MCP tool handlers for CRM CRUD operations.
// ABOUTME: Maps 12 tools to storage layer for contact, company, and relationship management.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//nolint:funlen // Tool registration requires many declarations
func (s *Server) registerTools() {
	s.server.AddTool(&mcp.Tool{
		Name:        "add_contact",
		Description: "Create a new contact",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name":   {"type": "string", "description": "Contact name"},
				"email":  {"type": "string", "description": "Email address"},
				"phone":  {"type": "string", "description": "Phone number"},
				"fields": {"type": "object", "description": "Custom fields (key-value pairs)"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Tags"}
			},
			"required": ["name"]
		}`),
	}, s.handleAddContact)

	s.server.AddTool(&mcp.Tool{
		Name:        "list_contacts",
		Description: "List contacts with optional filtering by tag or search query",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"tag":    {"type": "string", "description": "Filter by tag"},
				"search": {"type": "string", "description": "Search query"},
				"limit":  {"type": "integer", "description": "Max results", "default": 20}
			}
		}`),
	}, s.handleListContacts)

	s.server.AddTool(&mcp.Tool{
		Name:        "get_contact",
		Description: "Get a contact by ID or ID prefix (6+ chars)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Contact ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleGetContact)

	s.server.AddTool(&mcp.Tool{
		Name:        "update_contact",
		Description: "Update a contact's fields",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id":     {"type": "string", "description": "Contact ID or prefix"},
				"name":   {"type": "string", "description": "New name"},
				"email":  {"type": "string", "description": "New email"},
				"phone":  {"type": "string", "description": "New phone"},
				"fields": {"type": "object", "description": "Fields to merge"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Replace tags"}
			},
			"required": ["id"]
		}`),
	}, s.handleUpdateContact)

	s.server.AddTool(&mcp.Tool{
		Name:        "delete_contact",
		Description: "Delete a contact",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Contact ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleDeleteContact)

	s.server.AddTool(&mcp.Tool{
		Name:        "add_company",
		Description: "Create a new company",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name":   {"type": "string", "description": "Company name"},
				"domain": {"type": "string", "description": "Website domain"},
				"fields": {"type": "object", "description": "Custom fields"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Tags"}
			},
			"required": ["name"]
		}`),
	}, s.handleAddCompany)

	s.server.AddTool(&mcp.Tool{
		Name:        "list_companies",
		Description: "List companies with optional filtering",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"tag":    {"type": "string", "description": "Filter by tag"},
				"search": {"type": "string", "description": "Search query"},
				"limit":  {"type": "integer", "description": "Max results", "default": 20}
			}
		}`),
	}, s.handleListCompanies)

	s.server.AddTool(&mcp.Tool{
		Name:        "get_company",
		Description: "Get a company by ID or prefix",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Company ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleGetCompany)

	s.server.AddTool(&mcp.Tool{
		Name:        "update_company",
		Description: "Update a company's fields",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id":     {"type": "string", "description": "Company ID or prefix"},
				"name":   {"type": "string", "description": "New name"},
				"domain": {"type": "string", "description": "New domain"},
				"fields": {"type": "object", "description": "Fields to merge"},
				"tags":   {"type": "array", "items": {"type": "string"}, "description": "Replace tags"}
			},
			"required": ["id"]
		}`),
	}, s.handleUpdateCompany)

	s.server.AddTool(&mcp.Tool{
		Name:        "delete_company",
		Description: "Delete a company",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Company ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleDeleteCompany)

	s.server.AddTool(&mcp.Tool{
		Name:        "link",
		Description: "Create a relationship between two entities (contact or company)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"source_id": {"type": "string", "description": "Source entity ID"},
				"target_id": {"type": "string", "description": "Target entity ID"},
				"type":      {"type": "string", "description": "Relationship type (works_at, knows, reports_to, etc)"},
				"context":   {"type": "string", "description": "Context note"}
			},
			"required": ["source_id", "target_id", "type"]
		}`),
	}, s.handleLink)

	s.server.AddTool(&mcp.Tool{
		Name:        "unlink",
		Description: "Remove a relationship",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Relationship ID"}
			},
			"required": ["id"]
		}`),
	}, s.handleUnlink)
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func textResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(fmt.Sprintf("failed to marshal: %v", err))
	}
	return textResult(string(data))
}

func (s *Server) resolveContact(idStr string) (*models.Contact, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return s.store.GetContact(id)
	}
	return s.store.GetContactByPrefix(idStr)
}

func (s *Server) resolveCompany(idStr string) (*models.Company, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return s.store.GetCompany(id)
	}
	return s.store.GetCompanyByPrefix(idStr)
}

func (s *Server) handleAddContact(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		Name   string         `json:"name"`
		Email  string         `json:"email"`
		Phone  string         `json:"phone"`
		Fields map[string]any `json:"fields"`
		Tags   []string       `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Name) == "" {
		return errResult("name is required"), nil
	}

	c := models.NewContact(p.Name)
	c.Email = p.Email
	c.Phone = p.Phone
	if p.Fields != nil {
		c.Fields = p.Fields
	}
	if p.Tags != nil {
		c.Tags = p.Tags
	}

	if err := s.store.CreateContact(c); err != nil {
		return errResult(fmt.Sprintf("failed to create contact: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Created contact %s (%s)", c.Name, c.ID.String()[:8])), nil
}

func (s *Server) handleListContacts(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		Tag    *string `json:"tag"`
		Search string  `json:"search"`
		Limit  int     `json:"limit"`
	}
	p.Limit = 20
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}

	contacts, err := s.store.ListContacts(&storage.ContactFilter{Tag: p.Tag, Search: p.Search, Limit: p.Limit})
	if err != nil {
		return errResult(fmt.Sprintf("failed to list: %v", err)), nil
	}
	return jsonResult(contacts), nil
}

func (s *Server) handleGetContact(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct{ ID string `json:"id"` }
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	c, err := s.resolveContact(p.ID)
	if err != nil {
		return errResult(fmt.Sprintf("failed to get contact: %v", err)), nil
	}
	return jsonResult(c), nil
}

func (s *Server) handleUpdateContact(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		ID     string         `json:"id"`
		Name   *string        `json:"name"`
		Email  *string        `json:"email"`
		Phone  *string        `json:"phone"`
		Fields map[string]any `json:"fields"`
		Tags   []string       `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}

	c, err := s.resolveContact(p.ID)
	if err != nil {
		return errResult(fmt.Sprintf("contact not found: %v", err)), nil
	}

	if p.Name != nil {
		c.Name = *p.Name
	}
	if p.Email != nil {
		c.Email = *p.Email
	}
	if p.Phone != nil {
		c.Phone = *p.Phone
	}
	if p.Fields != nil {
		for k, v := range p.Fields {
			c.Fields[k] = v
		}
	}
	if p.Tags != nil {
		c.Tags = p.Tags
	}
	c.Touch()

	if err := s.store.UpdateContact(c); err != nil {
		return errResult(fmt.Sprintf("failed to update: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Updated contact %s", c.ID.String()[:8])), nil
}

func (s *Server) handleDeleteContact(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct{ ID string `json:"id"` }
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	c, err := s.resolveContact(p.ID)
	if err != nil {
		return errResult(fmt.Sprintf("contact not found: %v", err)), nil
	}
	if err := s.store.DeleteContact(c.ID); err != nil {
		return errResult(fmt.Sprintf("failed to delete: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Deleted contact %s", c.ID.String()[:8])), nil
}

func (s *Server) handleAddCompany(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		Name   string         `json:"name"`
		Domain string         `json:"domain"`
		Fields map[string]any `json:"fields"`
		Tags   []string       `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Name) == "" {
		return errResult("name is required"), nil
	}

	c := models.NewCompany(p.Name)
	c.Domain = p.Domain
	if p.Fields != nil {
		c.Fields = p.Fields
	}
	if p.Tags != nil {
		c.Tags = p.Tags
	}

	if err := s.store.CreateCompany(c); err != nil {
		return errResult(fmt.Sprintf("failed to create company: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Created company %s (%s)", c.Name, c.ID.String()[:8])), nil
}

func (s *Server) handleListCompanies(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		Tag    *string `json:"tag"`
		Search string  `json:"search"`
		Limit  int     `json:"limit"`
	}
	p.Limit = 20
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}

	companies, err := s.store.ListCompanies(&storage.CompanyFilter{Tag: p.Tag, Search: p.Search, Limit: p.Limit})
	if err != nil {
		return errResult(fmt.Sprintf("failed to list: %v", err)), nil
	}
	return jsonResult(companies), nil
}

func (s *Server) handleGetCompany(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct{ ID string `json:"id"` }
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	c, err := s.resolveCompany(p.ID)
	if err != nil {
		return errResult(fmt.Sprintf("failed to get company: %v", err)), nil
	}
	return jsonResult(c), nil
}

func (s *Server) handleUpdateCompany(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		ID     string         `json:"id"`
		Name   *string        `json:"name"`
		Domain *string        `json:"domain"`
		Fields map[string]any `json:"fields"`
		Tags   []string       `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}

	c, err := s.resolveCompany(p.ID)
	if err != nil {
		return errResult(fmt.Sprintf("company not found: %v", err)), nil
	}

	if p.Name != nil {
		c.Name = *p.Name
	}
	if p.Domain != nil {
		c.Domain = *p.Domain
	}
	if p.Fields != nil {
		for k, v := range p.Fields {
			c.Fields[k] = v
		}
	}
	if p.Tags != nil {
		c.Tags = p.Tags
	}
	c.Touch()

	if err := s.store.UpdateCompany(c); err != nil {
		return errResult(fmt.Sprintf("failed to update: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Updated company %s", c.ID.String()[:8])), nil
}

func (s *Server) handleDeleteCompany(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct{ ID string `json:"id"` }
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	c, err := s.resolveCompany(p.ID)
	if err != nil {
		return errResult(fmt.Sprintf("company not found: %v", err)), nil
	}
	if err := s.store.DeleteCompany(c.ID); err != nil {
		return errResult(fmt.Sprintf("failed to delete: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Deleted company %s", c.ID.String()[:8])), nil
}

func (s *Server) handleLink(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct {
		SourceID string `json:"source_id"`
		TargetID string `json:"target_id"`
		Type     string `json:"type"`
		Context  string `json:"context"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}

	srcID, err := uuid.Parse(p.SourceID)
	if err != nil {
		return errResult("invalid source_id"), nil
	}
	tgtID, err := uuid.Parse(p.TargetID)
	if err != nil {
		return errResult("invalid target_id"), nil
	}

	rel := models.NewRelationship(srcID, tgtID, p.Type, p.Context)
	if err := s.store.CreateRelationship(rel); err != nil {
		return errResult(fmt.Sprintf("failed to link: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Linked %s -> %s (%s)", p.SourceID[:8], p.TargetID[:8], p.Type)), nil
}

func (s *Server) handleUnlink(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var p struct{ ID string `json:"id"` }
	if err := json.Unmarshal(req.Params.Arguments, &p); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return errResult("invalid relationship ID"), nil
	}
	if err := s.store.DeleteRelationship(id); err != nil {
		return errResult(fmt.Sprintf("failed to unlink: %v", err)), nil
	}
	return textResult(fmt.Sprintf("Unlinked %s", p.ID[:8])), nil
}
```

- [ ] **Step 5: Implement MCP resources**

Create `internal/mcp/resources.go`:
```go
// ABOUTME: MCP resources for exposing CRM data as readable resources.
// ABOUTME: Provides URI-based access to contacts and companies for AI agents.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "crm://contacts/{id}",
			Name:        "Contact",
			Description: "Access individual contacts by ID",
			MIMEType:    "application/json",
		},
		s.handleReadContact,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "crm://companies/{id}",
			Name:        "Company",
			Description: "Access individual companies by ID",
			MIMEType:    "application/json",
		},
		s.handleReadCompany,
	)
}

func (s *Server) handleReadContact(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	var idStr string
	_, err := fmt.Sscanf(req.Params.URI, "crm://contacts/%s", &idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %s", req.Params.URI)
	}

	contact, err := s.resolveContact(idStr)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	rels, _ := s.store.ListRelationships(contact.ID)
	result := map[string]any{
		"contact":       contact,
		"relationships": rels,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func (s *Server) handleReadCompany(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	idStr := strings.TrimPrefix(uri, "crm://companies/")

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %s", idStr)
	}

	company, err := s.store.GetCompany(id)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	rels, _ := s.store.ListRelationships(company.ID)
	result := map[string]any{
		"company":       company,
		"relationships": rels,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

// listContactsSummary returns all contacts for the contacts list resource.
func (s *Server) listContactsSummary() (string, error) {
	contacts, err := s.store.ListContacts(&storage.ContactFilter{})
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(contacts, "", "  ")
	return string(data), nil
}

// listCompaniesSummary returns all companies for the companies list resource.
func (s *Server) listCompaniesSummary() (string, error) {
	companies, err := s.store.ListCompanies(&storage.CompanyFilter{})
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(companies, "", "  ")
	return string(data), nil
}
```

- [ ] **Step 6: Implement MCP prompts**

Create `internal/mcp/prompts.go`:
```go
// ABOUTME: MCP prompts for common CRM workflows.
// ABOUTME: Provides guided workflows for contact creation, relationship mapping, and search.

package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPrompts() {
	s.server.AddPrompt(&mcp.Prompt{
		Name:        "add-contact-workflow",
		Description: "Guided contact creation with validation",
		Arguments: []*mcp.PromptArgument{
			{Name: "name", Description: "Contact name", Required: true},
		},
	}, s.getAddContactPrompt)

	s.server.AddPrompt(&mcp.Prompt{
		Name:        "relationship-mapping",
		Description: "Explore and map connections for an entity",
		Arguments: []*mcp.PromptArgument{
			{Name: "entity_id", Description: "Entity ID to explore", Required: true},
		},
	}, s.getRelationshipMappingPrompt)

	s.server.AddPrompt(&mcp.Prompt{
		Name:        "crm-search",
		Description: "Cross-entity search with context",
		Arguments: []*mcp.PromptArgument{
			{Name: "query", Description: "Search query", Required: true},
		},
	}, s.getCrmSearchPrompt)
}

func (s *Server) getAddContactPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := req.Params.Arguments["name"]
	template := fmt.Sprintf(`Add a new contact named "%s" to the CRM.

1. Use the add_contact tool with the name "%s"
2. Ask for additional details: email, phone, company, role
3. Add relevant tags (e.g., industry, relationship type)
4. If they work at a company, use add_company + link to create the relationship

Make sure to confirm the details before saving.`, name, name)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: template},
		}},
	}, nil
}

func (s *Server) getRelationshipMappingPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	entityID := req.Params.Arguments["entity_id"]
	template := fmt.Sprintf(`Map the relationships for entity %s.

1. Use get_contact or get_company to identify the entity
2. List all their relationships
3. For each related entity, get their details
4. Present a clear map of connections with types and context
5. Suggest any missing relationships that might exist`, entityID)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: template},
		}},
	}, nil
}

func (s *Server) getCrmSearchPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	query := req.Params.Arguments["query"]
	template := fmt.Sprintf(`Search the CRM for "%s".

1. Use list_contacts with search="%s" to find matching contacts
2. Use list_companies with search="%s" to find matching companies
3. For each result, show key details and relationships
4. Summarize findings and suggest follow-up actions`, query, query, query)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: template},
		}},
	}, nil
}
```

- [ ] **Step 7: Create MCP CLI command**

Create `cmd/crm/mcp.go`:
```go
// ABOUTME: MCP command to start the MCP server.
// ABOUTME: Runs on stdio for integration with AI agents.

package main

import (
	mcpserver "github.com/harperreed/crm/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server",
	Long:  `Start the Model Context Protocol server for AI agent integration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcpserver.NewServer(store)
		return server.Serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
```

- [ ] **Step 8: Run MCP tests and verify compilation**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/mcp/... && go build ./cmd/crm
```

Expected: PASS + compiles

- [ ] **Step 9: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/mcp/ cmd/crm/mcp.go
git commit -m "feat: add MCP server with 12 tools, resources, and prompts"
```

---

### Task 12: Markdown Backend

**Files:**
- Modify: `internal/storage/markdown.go`
- Create: `internal/storage/markdown_contacts.go`, `internal/storage/markdown_companies.go`, `internal/storage/markdown_relationships.go`, `internal/storage/markdown_search.go`, `internal/storage/markdown_test.go`

- [ ] **Step 1: Write failing markdown store test**

Create `internal/storage/markdown_test.go`:
```go
// ABOUTME: Tests for markdown storage backend.
// ABOUTME: Validates CRUD operations using file-based storage.

package storage

import (
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

func newTestMarkdownStore(t *testing.T) *MarkdownStore {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "crm-md-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := NewMarkdownStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create markdown store: %v", err)
	}
	return store
}

func TestMarkdownCreateAndGetContact(t *testing.T) {
	store := newTestMarkdownStore(t)
	c := models.NewContact("Jane Doe")
	c.Email = "jane@example.com"
	c.Tags = []string{"vip"}

	if err := store.CreateContact(c); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := store.GetContact(c.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "Jane Doe" {
		t.Errorf("expected 'Jane Doe', got %q", got.Name)
	}
	if got.Email != "jane@example.com" {
		t.Errorf("expected email, got %q", got.Email)
	}
}

func TestMarkdownDeleteContact(t *testing.T) {
	store := newTestMarkdownStore(t)
	c := models.NewContact("Delete Me")
	_ = store.CreateContact(c)

	if err := store.DeleteContact(c.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err := store.GetContact(c.ID)
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestMarkdownCreateAndGetCompany(t *testing.T) {
	store := newTestMarkdownStore(t)
	co := models.NewCompany("Acme Corp")
	co.Domain = "acme.com"

	if err := store.CreateCompany(co); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := store.GetCompany(co.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", got.Name)
	}
}

func TestMarkdownRelationships(t *testing.T) {
	store := newTestMarkdownStore(t)
	c := models.NewContact("Jane")
	_ = store.CreateContact(c)
	co := models.NewCompany("Acme")
	_ = store.CreateCompany(co)

	rel := models.NewRelationship(c.ID, co.ID, "works_at", "CEO")
	if err := store.CreateRelationship(rel); err != nil {
		t.Fatalf("create rel failed: %v", err)
	}

	rels, err := store.ListRelationships(c.ID)
	if err != nil {
		t.Fatalf("list rels failed: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("expected 1 rel, got %d", len(rels))
	}

	if err := store.DeleteRelationship(rel.ID); err != nil {
		t.Fatalf("delete rel failed: %v", err)
	}

	rels, _ = store.ListRelationships(c.ID)
	if len(rels) != 0 {
		t.Errorf("expected 0 rels after delete, got %d", len(rels))
	}
}

func TestMarkdownSearch(t *testing.T) {
	store := newTestMarkdownStore(t)
	c := models.NewContact("Findable Person")
	_ = store.CreateContact(c)
	co := models.NewCompany("Findable Corp")
	_ = store.CreateCompany(co)

	results, err := store.Search("Findable")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results.Contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(results.Contacts))
	}
	if len(results.Companies) != 1 {
		t.Errorf("expected 1 company, got %d", len(results.Companies))
	}
}

func TestMarkdownGetContactByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)
	c := models.NewContact("Prefix Test")
	_ = store.CreateContact(c)

	got, err := store.GetContactByPrefix(c.ID.String()[:8])
	if err != nil {
		t.Fatalf("get by prefix failed: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("expected ID %s, got %s", c.ID, got.ID)
	}

	_, err = store.GetContactByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}

	_, err = store.GetContactByPrefix("000000")
	if !errors.Is(err, ErrContactNotFound) {
		t.Errorf("expected ErrContactNotFound, got %v", err)
	}
}

func TestMarkdownGetCompanyByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)
	co := models.NewCompany("Prefix Corp")
	_ = store.CreateCompany(co)

	got, err := store.GetCompanyByPrefix(co.ID.String()[:8])
	if err != nil {
		t.Fatalf("get by prefix failed: %v", err)
	}
	if got.ID != co.ID {
		t.Errorf("expected ID %s, got %s", co.ID, got.ID)
	}
}

func TestMarkdownDeleteRelationshipNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	err := store.DeleteRelationship(uuid.New())
	if !errors.Is(err, ErrRelationshipNotFound) {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Implement the full markdown backend**

This task requires implementing `markdown.go` (replace stub), `markdown_contacts.go`, `markdown_companies.go`, `markdown_relationships.go`, and `markdown_search.go`. These use `harperreed/mdstore` for atomic file operations and YAML frontmatter for metadata.

```bash
cd /Users/harper/Public/src/personal/suite/crm && go get github.com/harperreed/mdstore@v0.1.0 && go get gopkg.in/yaml.v3@v3.0.1
```

The implementation stores contacts as `.md` files in `contacts/` subdirectory, companies in `companies/`, and relationships in `_relationships.yaml`. Each entity file has YAML frontmatter with all fields. The markdown body is unused (reserved for freeform notes).

Files to implement (full implementations provided to the executing agent — the pattern mirrors SQLite CRUD but reads/writes YAML-frontmatter `.md` files via `mdstore.Slugify` + `mdstore.AtomicWrite` and `os.ReadFile` + `yaml.Unmarshal`):

- `internal/storage/markdown.go` — constructor, `Close()`, path helpers, frontmatter types
- `internal/storage/markdown_contacts.go` — `CreateContact`, `GetContact`, `GetContactByPrefix`, `ListContacts`, `UpdateContact`, `DeleteContact`
- `internal/storage/markdown_companies.go` — same pattern for companies
- `internal/storage/markdown_relationships.go` — YAML file-based relationship storage
- `internal/storage/markdown_search.go` — substring search across file contents

- [ ] **Step 3: Run tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./internal/storage/...
```

Expected: all PASS (both SQLite and Markdown tests)

- [ ] **Step 4: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add internal/storage/markdown*.go internal/storage/markdown_test.go
git commit -m "feat: add markdown storage backend with full CRUD"
```

---

### Task 13: Integration Tests & Final Wiring

**Files:**
- Create: `test/integration_test.go`
- Install pre-commit hooks

- [ ] **Step 1: Write integration test**

Create `test/integration_test.go`:
```go
// ABOUTME: End-to-end integration tests for CRM.
// ABOUTME: Validates full workflow across storage, config, and MCP layers.

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harperreed/crm/internal/config"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
)

func TestFullWorkflow(t *testing.T) {
	backends := []string{"sqlite", "markdown"}

	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "crm-integration-*")
			if err != nil {
				t.Fatalf("create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := &config.Config{Backend: backend, DataDir: tmpDir}
			store, err := cfg.OpenStorage()
			if err != nil {
				t.Fatalf("open storage: %v", err)
			}
			defer store.Close()

			// Create contact
			contact := models.NewContact("Integration Test Contact")
			contact.Email = "test@example.com"
			contact.Tags = []string{"test"}
			if err := store.CreateContact(contact); err != nil {
				t.Fatalf("create contact: %v", err)
			}

			// Create company
			company := models.NewCompany("Integration Test Corp")
			company.Domain = "test.com"
			if err := store.CreateCompany(company); err != nil {
				t.Fatalf("create company: %v", err)
			}

			// Create relationship
			rel := models.NewRelationship(contact.ID, company.ID, "works_at", "CEO")
			if err := store.CreateRelationship(rel); err != nil {
				t.Fatalf("create relationship: %v", err)
			}

			// Verify contact
			got, err := store.GetContact(contact.ID)
			if err != nil {
				t.Fatalf("get contact: %v", err)
			}
			if got.Name != "Integration Test Contact" {
				t.Errorf("wrong name: %q", got.Name)
			}

			// Verify relationships
			rels, err := store.ListRelationships(contact.ID)
			if err != nil {
				t.Fatalf("list relationships: %v", err)
			}
			if len(rels) != 1 {
				t.Errorf("expected 1 relationship, got %d", len(rels))
			}

			// Search
			results, err := store.Search("Integration")
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			if len(results.Contacts) < 1 {
				t.Error("expected search to find contact")
			}
			if len(results.Companies) < 1 {
				t.Error("expected search to find company")
			}

			// Update contact
			got.Name = "Updated Contact"
			got.Touch()
			if err := store.UpdateContact(got); err != nil {
				t.Fatalf("update contact: %v", err)
			}

			// Delete
			if err := store.DeleteRelationship(rel.ID); err != nil {
				t.Fatalf("delete relationship: %v", err)
			}
			if err := store.DeleteContact(contact.ID); err != nil {
				t.Fatalf("delete contact: %v", err)
			}
			if err := store.DeleteCompany(company.ID); err != nil {
				t.Fatalf("delete company: %v", err)
			}
		})
	}
}

func TestConfigOpenStorageBothBackends(t *testing.T) {
	for _, backend := range []string{"sqlite", "markdown"} {
		t.Run(backend, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "crm-config-*")
			if err != nil {
				t.Fatalf("create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := &config.Config{Backend: backend, DataDir: tmpDir}
			store, err := cfg.OpenStorage()
			if err != nil {
				t.Fatalf("open %s storage: %v", backend, err)
			}
			store.Close()

			if backend == "sqlite" {
				dbPath := filepath.Join(tmpDir, "crm.db")
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Error("expected crm.db to be created")
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run all tests**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -v ./...
```

Expected: all PASS

- [ ] **Step 3: Install pre-commit hooks**

```bash
cd /Users/harper/Public/src/personal/suite/crm && pre-commit install
```

- [ ] **Step 4: Run full check**

```bash
cd /Users/harper/Public/src/personal/suite/crm && make check
```

Expected: fmt + lint + test-race all pass

- [ ] **Step 5: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add test/integration_test.go
git commit -m "feat: add integration tests for both storage backends"
```

---

### Task 14: Skill Installation

**Files:**
- Create: `cmd/crm/skill.go`, `cmd/crm/skill/SKILL.md`

- [ ] **Step 1: Create Claude Code skill definition**

Create `cmd/crm/skill/SKILL.md`:
```markdown
# CRM Skill

Use the CRM MCP tools to manage contacts, companies, and relationships.

## Available Tools

- `mcp__crm__add_contact` — Create a contact (name required, email/phone/fields/tags optional)
- `mcp__crm__list_contacts` — List contacts (filter by tag, search, limit)
- `mcp__crm__get_contact` — Get contact by ID or prefix
- `mcp__crm__update_contact` — Update contact fields
- `mcp__crm__delete_contact` — Delete a contact
- `mcp__crm__add_company` — Create a company (name required, domain/fields/tags optional)
- `mcp__crm__list_companies` — List companies (filter by tag, search, limit)
- `mcp__crm__get_company` — Get company by ID or prefix
- `mcp__crm__update_company` — Update company fields
- `mcp__crm__delete_company` — Delete a company
- `mcp__crm__link` — Create a relationship (source_id, target_id, type required)
- `mcp__crm__unlink` — Remove a relationship by ID

## Usage Patterns

When the user mentions contacts, people, companies, organizations, or relationships, use these tools.

For adding contacts:
1. Use `add_contact` with name and any known details
2. If they mention a company, `add_company` + `link` to connect them

For searching:
1. Use `list_contacts` or `list_companies` with search parameter
2. Use `get_contact`/`get_company` for details on a specific entity
```

- [ ] **Step 2: Create skill install command**

Create `cmd/crm/skill.go`:
```go
// ABOUTME: Skill installation command for Claude Code integration.
// ABOUTME: Copies the CRM skill definition to the Claude Code skills directory.

package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

var skillCmd = &cobra.Command{
	Use:   "install-skill",
	Short: "Install Claude Code skill",
	Long:  `Install the CRM skill definition to ~/.claude/skills/crm/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}

		skillDir := filepath.Join(home, ".claude", "skills", "crm")
		if err := os.MkdirAll(skillDir, 0750); err != nil {
			return fmt.Errorf("create skill dir: %w", err)
		}

		data, err := skillFS.ReadFile("skill/SKILL.md")
		if err != nil {
			return fmt.Errorf("read embedded skill: %w", err)
		}

		dest := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return fmt.Errorf("write skill: %w", err)
		}

		fmt.Printf("Installed CRM skill to %s\n", dest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(skillCmd)
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go build ./cmd/crm
```

- [ ] **Step 4: Commit**

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add cmd/crm/skill.go cmd/crm/skill/SKILL.md
git commit -m "feat: add Claude Code skill installation"
```

---

### Task 15: Final Verification

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go test -race -v ./...
```

Expected: all PASS

- [ ] **Step 2: Build and smoke test**

```bash
cd /Users/harper/Public/src/personal/suite/crm && go build -o crm ./cmd/crm && ./crm --help
```

Expected: help output showing contact, company, link, unlink, mcp, install-skill commands.

```bash
cd /Users/harper/Public/src/personal/suite/crm && ./crm contact add "Test Person" --email test@example.com --tag demo && ./crm contact list && ./crm company add "Test Corp" --domain test.com && ./crm company list
```

Expected: creates and lists entities.

- [ ] **Step 3: Clean up binary**

```bash
cd /Users/harper/Public/src/personal/suite/crm && rm -f crm
```

- [ ] **Step 4: Final commit with .gitignore**

Create `.gitignore`:
```
crm
bin/
coverage.out
*.db
```

```bash
cd /Users/harper/Public/src/personal/suite/crm
git add .gitignore
git commit -m "chore: add .gitignore"
```
