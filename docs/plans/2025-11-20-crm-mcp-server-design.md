# CRM MCP Server Design

**Date:** 2025-11-20
**Status:** Approved

## Purpose

Build an MCP server that manages CRM data (contacts, companies, deals) via stdio, storing data in SQLite at the XDG data directory.

## Architecture

### Components

```
crm-mcp/
├── main.go              # Server initialization & stdio transport
├── db/
│   ├── schema.go        # SQLite schema & migrations
│   └── queries.go       # Database operations
├── handlers/
│   ├── contacts.go      # Contact tool handlers
│   ├── companies.go     # Company tool handlers
│   ├── deals.go         # Deal tool handlers
│   ├── relationships.go # Relationship tool handlers
│   └── query.go         # Universal query handler
└── models/
    └── types.go         # Go structs for entities
```

### Data Flow

1. Claude Desktop launches binary via stdio
2. Server initializes SQLite database (creates if missing)
3. MCP tools register with server
4. Claude calls tools via JSON-RPC over stdio
5. Handlers execute SQLite queries
6. Server returns structured responses

### Technology Stack

- **Language:** Go 1.21+
- **MCP SDK:** `github.com/modelcontextprotocol/go-sdk`
- **Database:** SQLite via `github.com/mattn/go-sqlite3`
- **UUID:** `github.com/google/uuid`
- **XDG Support:** `github.com/adrg/xdg`
- **Storage:** `$XDG_DATA_HOME/crm/crm.db` (~/.local/share/crm/crm.db)

## Data Models

### Contact

- `id` (UUID, primary key)
- `name` (text, required)
- `email` (text, indexed)
- `phone` (text)
- `company_id` (UUID, foreign key to companies)
- `notes` (text)
- `last_contacted_at` (timestamp)
- `created_at`, `updated_at` (timestamps)

### Company

- `id` (UUID, primary key)
- `name` (text, required, indexed)
- `domain` (text, e.g., "acme.com")
- `industry` (text)
- `notes` (text)
- `created_at`, `updated_at` (timestamps)

### Deal

- `id` (UUID, primary key)
- `title` (text, required)
- `amount` (integer, cents)
- `currency` (text, default "USD")
- `stage` (text: prospecting, qualification, proposal, negotiation, closed_won, closed_lost)
- `company_id` (UUID, foreign key, required)
- `contact_id` (UUID, foreign key)
- `expected_close_date` (date)
- `created_at`, `updated_at`, `last_activity_at` (timestamps)

### Deal Note

- `id` (UUID, primary key)
- `deal_id` (UUID, foreign key, required)
- `content` (text, required)
- `created_at` (timestamp)

### Relationship

- `id` (UUID, primary key)
- `contact_id_1` (UUID, foreign key to contacts, required)
- `contact_id_2` (UUID, foreign key to contacts, required)
- `relationship_type` (text, e.g., "colleague", "friend", "saw_together", "introduced_by")
- `context` (text, description of how they're connected)
- `created_at`, `updated_at` (timestamps)

**Note:** Relationships are bidirectional. When querying, search both contact_id_1 and contact_id_2.

## MCP Tools

### Contact Operations

**`add_contact`**
- Input: `name` (required), `email`, `phone`, `company_name`, `notes`
- Creates contact, looks up or creates company if `company_name` provided
- Returns: Contact with ID

**`find_contacts`**
- Input: `query` (searches name/email), `company_id`, `limit` (default 10)
- Returns: Matching contacts array

**`update_contact`**
- Input: `id` (required), fields to update
- Returns: Updated contact

**`log_contact_interaction`**
- Input: `contact_id` (required), `note`, `interaction_date` (defaults to now)
- Updates `last_contacted_at`, appends note
- Returns: Updated contact

### Company Operations

**`add_company`**
- Input: `name` (required), `domain`, `industry`, `notes`
- Returns: Company with ID

**`find_companies`**
- Input: `query` (searches name/domain), `limit` (default 10)
- Returns: Matching companies array

### Deal Operations

**`create_deal`**
- Input: `title` (required), `amount`, `currency`, `stage`, `company_name` (required), `contact_name`, `expected_close_date`, `initial_note`
- Looks up or creates company and contact
- Returns: Deal with ID

**`update_deal`**
- Input: `id` (required), fields to update
- Updates `last_activity_at` automatically
- Returns: Updated deal

**`add_deal_note`**
- Input: `deal_id` (required), `content` (required)
- Creates note, updates deal's `last_activity_at`
- Updates associated contact's `last_contacted_at`
- Returns: Note with timestamp

### Relationship Operations

**`link_contacts`**
- Input: `contact_id_1` (required), `contact_id_2` (required), `relationship_type`, `context`
- Links two contacts with a relationship
- Example types: "colleague", "friend", "saw_together", "introduced_by", "spouse", "business_partner"
- Returns: Relationship with ID

**`find_contact_relationships`**
- Input: `contact_id` (required), `relationship_type` (optional)
- Finds all contacts related to the given contact
- Returns: Array of relationships with full contact details

**`remove_relationship`**
- Input: `relationship_id` (required)
- Removes a relationship link
- Returns: Success confirmation

### Query Operations

**`query_crm`**
- Input: `entity_type` (contact/company/deal/relationship), `filters` (JSON), `limit`
- Executes flexible queries with related data
- Example: Find deals in "negotiation" with amount > $10,000
- Example: Find all "colleague" relationships
- Returns: Matching entities with related data

## Error Handling

### Validation

- Check required fields before database operations
- Validate email format (basic regex)
- Validate UUID format for IDs
- Validate deal stage against allowed values
- Ensure amount is non-negative

### Error Messages

- Invalid input: Clear message explaining the problem
- Not found: "Contact with ID X not found"
- Database errors: Generic message without internal details

### Relationship Handling

When `company_name` appears in `add_contact` or `create_deal`:
1. Search for company by name (case-insensitive)
2. Create company if not found
3. Link to contact or deal

When deleting company with contacts or deals: Return error listing dependent records.

## Database Configuration

- **Mode:** WAL (write-ahead logging)
- **Permissions:** 0600 (owner read/write only)
- **Initialization:** Create schema automatically on first run
- **Indexes:** email, company name, deal stage, relationship contact IDs

## Testing Strategy

### Unit Tests

**Database layer (`db/`):**
- Schema creation
- CRUD operations per entity
- Query filters and joins
- Relationship handling
- Use `:memory:` SQLite for speed

**Handlers (`handlers/`):**
- Each MCP tool with valid inputs
- Validation errors
- Not-found scenarios
- Mock database layer

### Integration Tests

- Full server startup with test database
- Execute MCP tool calls via JSON-RPC interface
- Verify end-to-end flow

### Test Coverage

- >80% coverage on business logic
- 100% coverage on exported handler functions

### TDD Approach

1. Write failing test
2. Implement minimum code to pass
3. Refactor while keeping tests green

## Claude Desktop Integration

Add to Claude Desktop configuration:

```json
{
  "mcpServers": {
    "crm": {
      "command": "/path/to/crm-mcp"
    }
  }
}
```

## Development

**Build:** `go build -o crm-mcp`
**Run:** `go run main.go`
**Test:** `go test ./...`
**Module:** `github.com/harperreed/crm-mcp`

## Next Steps

1. Initialize Go module and project structure
2. Implement database schema and migrations
3. Build MCP tool handlers (TDD approach)
4. Add integration tests
5. Test with Claude Desktop
