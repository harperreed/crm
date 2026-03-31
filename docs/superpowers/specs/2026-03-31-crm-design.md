# CRM Design Spec

A super simple CRM accessible via MCP (for agents) and CLI (for humans). Follows the proven patterns from toki, memo, and pulse in the suite.

## Core Entities

**Contacts** - People you interact with.
- Required: name
- Optional: email, phone
- Flexible: `fields` JSON map for anything else (title, address, social links, notes)
- Tags for categorization

**Companies** - Organizations.
- Required: name
- Optional: domain (website)
- Flexible: `fields` JSON map (industry, size, address, etc.)
- Tags for categorization

**Relationships** - Typed links between any two entities (contact-to-contact, contact-to-company, company-to-company).
- Type: freeform string ("works_at", "knows", "reports_to", "partner_of", etc.)
- Context: freeform note about the relationship

## Architecture

```
crm/
├── cmd/crm/
│   ├── main.go              # Entry point
│   ├── root.go              # Cobra root + PersistentPreRunE storage init
│   ├── mcp.go               # MCP server command
│   ├── contacts.go          # add/list/show/edit/rm contact commands
│   ├── companies.go         # add/list/show/edit/rm company commands
│   ├── relationships.go     # link/unlink/list relationship commands
│   └── skill.go             # Claude Code skill installation
│
├── internal/
│   ├── models/
│   │   ├── contact.go       # Contact struct
│   │   ├── company.go       # Company struct
│   │   └── relationship.go  # Relationship struct
│   │
│   ├── storage/
│   │   ├── interface.go     # Storage interface (~15 methods)
│   │   ├── filter.go        # Filter objects for queries
│   │   ├── sqlite.go        # SQLite implementation (modernc.org/sqlite)
│   │   ├── sqlite_schema.go # Schema + FTS5
│   │   ├── markdown.go      # Markdown implementation (harperreed/mdstore)
│   │   └── migrate.go       # Backend migration utility
│   │
│   ├── mcp/
│   │   ├── server.go        # MCP server setup
│   │   ├── tools.go         # Tool handlers (12 tools)
│   │   ├── resources.go     # Resource endpoints (4 resources)
│   │   └── prompts.go       # Workflow prompts (3 prompts)
│   │
│   └── config/
│       └── config.go        # XDG config + backend factory
│
├── go.mod
├── Makefile
└── .pre-commit-config.yaml
```

## Data Model

### Contact

```go
type Contact struct {
    ID        uuid.UUID
    Name      string            // required
    Email     string            // optional
    Phone     string            // optional
    Fields    map[string]any    // flexible key-value pairs
    Tags      []string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Company

```go
type Company struct {
    ID        uuid.UUID
    Name      string            // required
    Domain    string            // optional (website)
    Fields    map[string]any    // flexible key-value pairs
    Tags      []string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Relationship

```go
type Relationship struct {
    ID        uuid.UUID
    SourceID  uuid.UUID         // contact or company
    TargetID  uuid.UUID         // contact or company
    Type      string            // "works_at", "knows", "reports_to", etc.
    Context   string            // freeform note
    CreatedAt time.Time
}
```

## Storage Interface

```go
type Storage interface {
    // Contacts
    CreateContact(contact *models.Contact) error
    GetContact(id uuid.UUID) (*models.Contact, error)
    ListContacts(filter *ContactFilter) ([]*models.Contact, error)
    UpdateContact(contact *models.Contact) error
    DeleteContact(id uuid.UUID) error

    // Companies
    CreateCompany(company *models.Company) error
    GetCompany(id uuid.UUID) (*models.Company, error)
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
```

### Filter Objects

```go
type ContactFilter struct {
    Tag    *string
    Search string
    Limit  int
}

type CompanyFilter struct {
    Tag    *string
    Search string
    Limit  int
}
```

### Search Results

```go
type SearchResults struct {
    Contacts  []*models.Contact
    Companies []*models.Company
}
```

## Storage Backends

### SQLite (default for existing users)

- Driver: `modernc.org/sqlite` (pure Go)
- WAL mode, foreign keys enabled
- FTS5 for full-text search on name, email, and fields
- XDG data directory: `~/.local/share/crm/crm.db`

**Schema:**

```sql
CREATE TABLE contacts (
    rowid INTEGER PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    email TEXT DEFAULT '',
    phone TEXT DEFAULT '',
    fields TEXT DEFAULT '{}',
    tags TEXT DEFAULT '[]',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE companies (
    rowid INTEGER PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    domain TEXT DEFAULT '',
    fields TEXT DEFAULT '{}',
    tags TEXT DEFAULT '[]',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE relationships (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    type TEXT NOT NULL,
    context TEXT DEFAULT '',
    created_at DATETIME NOT NULL
);

-- FTS5 for contacts
CREATE VIRTUAL TABLE contacts_fts USING fts5(
    name, email, fields,
    content=contacts, content_rowid=rowid
);

-- FTS5 for companies
CREATE VIRTUAL TABLE companies_fts USING fts5(
    name, domain, fields,
    content=companies, content_rowid=rowid
);
```

### Markdown (default for new users)

- Uses `harperreed/mdstore` for atomic file operations
- Contacts: `~/.local/share/crm/contacts/<slug>.md` with YAML frontmatter
- Companies: `~/.local/share/crm/companies/<slug>.md` with YAML frontmatter
- Relationships: `~/.local/share/crm/_relationships.yaml` index file

**Contact markdown example:**

```markdown
---
id: 01JTEST1234
name: Jane Doe
email: jane@example.com
phone: "+1-555-0123"
fields:
  title: VP Engineering
  linkedin: linkedin.com/in/janedoe
tags:
  - engineering
  - vip
created_at: 2026-03-31T10:00:00Z
updated_at: 2026-03-31T10:00:00Z
---

Notes about Jane go here as freeform markdown content.
```

## MCP Server

### Tools (12)

| Tool | Input | Description |
|------|-------|-------------|
| `add_contact` | name (req), email, phone, fields, tags | Create contact |
| `list_contacts` | tag, search, limit | List/filter contacts |
| `get_contact` | id | Get contact by ID (prefix match) |
| `update_contact` | id (req), name, email, phone, fields, tags | Update contact |
| `delete_contact` | id | Delete contact |
| `add_company` | name (req), domain, fields, tags | Create company |
| `list_companies` | tag, search, limit | List/filter companies |
| `get_company` | id | Get company by ID (prefix match) |
| `update_company` | id (req), name, domain, fields, tags | Update company |
| `delete_company` | id | Delete company |
| `link` | source_id, target_id, type (req), context | Create relationship |
| `unlink` | id | Delete relationship |

### Resources (4)

| URI | Description |
|-----|-------------|
| `crm://contacts` | All contacts summary (name, email, tags) |
| `crm://companies` | All companies summary (name, domain, tags) |
| `crm://contacts/{id}` | Full contact with relationships |
| `crm://companies/{id}` | Full company with relationships |

### Prompts (3)

| Prompt | Description |
|--------|-------------|
| `add-contact-workflow` | Guided contact creation with validation |
| `relationship-mapping` | Explore and visualize connections for an entity |
| `crm-search` | Cross-entity search with context |

## CLI Commands

```
crm contact add <name> [--email EMAIL] [--phone PHONE] [--field KEY=VALUE]... [--tag TAG]...
crm contact list [--tag TAG] [--search QUERY] [--limit N]
crm contact show <id>
crm contact edit <id> [--name NAME] [--email EMAIL] [--field KEY=VALUE]...
crm contact rm <id>

crm company add <name> [--domain DOMAIN] [--field KEY=VALUE]... [--tag TAG]...
crm company list [--tag TAG] [--search QUERY] [--limit N]
crm company show <id>
crm company edit <id> [--name NAME] [--domain DOMAIN] [--field KEY=VALUE]...
crm company rm <id>

crm link <source-id> <target-id> --type TYPE [--context CONTEXT]
crm unlink <relationship-id>

crm mcp                    # Start MCP server (stdio)
crm install-skill          # Install Claude Code skill
```

## Configuration

XDG-compliant config at `~/.config/crm/config.json`:

```json
{
    "storage_backend": "sqlite",
    "data_dir": ""
}
```

- `storage_backend`: "sqlite" or "markdown"
- `data_dir`: override for data directory (default: `~/.local/share/crm/`)

## Dependencies

- `modernc.org/sqlite` - Pure Go SQLite
- `harperreed/mdstore` - Markdown file storage
- `modelcontextprotocol/go-sdk` - MCP server
- `spf13/cobra` - CLI framework
- `google/uuid` - UUID generation
- `fatih/color` - Terminal colors
- `gopkg.in/yaml.v3` - YAML parsing
- `stretchr/testify` - Testing assertions

## Testing Strategy

- Storage interface tests run against both SQLite and Markdown backends
- CLI command tests use temp storage (like memo's setupTestStore pattern)
- MCP tool tests verify JSON schemas and handler behavior
- Pre-commit hooks: go fmt, goimports, golangci-lint, unit tests, go vet

## What's NOT Included (by design)

- No web UI (MCP + CLI only)
- No Google sync (add later if needed)
- No follow-up/cadence tracking
- No deal pipeline
- No visualization
- No Office OS schema (simple dedicated tables instead)
