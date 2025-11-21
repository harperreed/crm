# Pagen - Personal Agent Toolkit

A personal agent toolkit with CRM capabilities. Works both as a Model Context Protocol (MCP) server for Claude Desktop AND as a standalone CLI for direct terminal use.

## Features

- **Contact Management** - Track people with full interaction history
- **Company Management** - Organize contacts by company with industry tracking
- **Deal Pipeline** - Manage sales from prospecting to closed
- **Relationship Tracking** - Map connections between contacts (colleagues, friends, etc.)
- **Universal Query** - Flexible searching across all entity types

## Installation

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/harperreed/pagen/releases)

### Build from Source

```bash
git clone https://github.com/harperreed/pagen.git
cd pagen
go build -o pagen
```

## Usage

Pagen can be used in two ways:

### 1. MCP Server for Claude Desktop

Configure Claude Desktop to use the MCP server.

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "pagen": {
      "command": "/path/to/pagen",
      "args": ["mcp"]
    }
  }
}
```

Then restart Claude Desktop and you'll have access to all 13 CRM tools through natural language.

### 2. CLI for Direct Terminal Use

Use the CLI directly for quick CRM operations:

```bash
# Add a company
pagen crm add-company --name "Acme Corp" --industry "Software" --domain "acme.com"

# Add a contact
pagen crm add-contact --name "John Smith" --email "john@acme.com" --company "Acme Corp"

# List contacts
pagen crm list-contacts

# List contacts at specific company
pagen crm list-contacts --company "Acme Corp"

# Add a deal
pagen crm add-deal --title "Enterprise License" --company "Acme Corp" --amount 5000000 --stage "negotiation"

# List deals
pagen crm list-deals

# List deals in specific stage
pagen crm list-deals --stage "negotiation"

# List companies
pagen crm list-companies
```

### Global Flags

- `--version` - Show version and exit
- `--db-path <path>` - Use custom database path (default: `~/.local/share/crm/crm.db`)
- `--init` - Initialize database and exit (use with `crm` command)

### Available Commands

**Top Level:**
- `pagen mcp` - Start MCP server (for Claude Desktop)
- `pagen crm` - CRM management commands

**CRM Commands:**
- `pagen crm add-company` - Create a new company
- `pagen crm list-companies` - List/search companies
- `pagen crm add-contact` - Create a new contact
- `pagen crm list-contacts` - List/search contacts
- `pagen crm add-deal` - Create a new deal
- `pagen crm list-deals` - List/search deals

Run `pagen` without arguments for full help.

## Database

The server uses SQLite and stores data at:
- **Default:** `~/.local/share/crm/crm.db` (XDG data directory)
- **Custom:** Specify with `--db-path` flag

### Database Schema

- **contacts** - People with names, emails, phone numbers, company associations
- **companies** - Organizations with industry and domain information
- **deals** - Sales pipeline with stages, amounts, and expected close dates
- **deal_notes** - Activity logs on deals
- **relationships** - Bidirectional connections between contacts

## MCP Tools (13 Total)

### Contact Operations (4 tools)
- `add_contact` - Create new contacts with optional company linking
- `find_contacts` - Search by name, email, or company
- `update_contact` - Modify contact information
- `log_contact_interaction` - Record interactions with timestamp tracking

### Company Operations (2 tools)
- `add_company` - Create companies with industry/domain metadata
- `find_companies` - Search by name or domain

### Deal Operations (3 tools)
- `create_deal` - Create deals with company and contact associations
- `update_deal` - Modify deal details including stage and amount
- `add_deal_note` - Add activity notes to deals

### Relationship Operations (3 tools)
- `link_contacts` - Create relationships between contacts
- `find_contact_relationships` - Find all connections for a contact
- `remove_relationship` - Delete relationship links

### Query Operations (1 tool)
- `query_crm` - Universal query across all entity types with flexible filtering

## Example Usage

### In Claude Desktop (via MCP)

Once configured, try these natural language commands:

```
Add a company called Acme Corp in the software industry

Add a contact named John Smith with email john@acme.com at Acme Corp

Create a deal titled "Enterprise License" with Acme Corp for $50,000

Link John Smith and Jane Doe as colleagues who met at a conference

Show me all my contacts at Acme Corp

Query all deals in negotiation stage above $10,000
```

### In Terminal (via CLI)

Quick commands for direct CRM management:

```bash
# Start fresh
pagen --db-path ~/my-crm.db --init crm

# Add companies
pagen crm add-company --name "Acme Corp" --industry "Software"
pagen crm add-company --name "TechStart" --industry "SaaS"

# Add contacts
pagen crm add-contact --name "Alice" --email "alice@acme.com" --company "Acme Corp"
pagen crm add-contact --name "Bob" --email "bob@techstart.io" --company "TechStart"

# Create deals
pagen crm add-deal --title "Enterprise" --company "Acme Corp" --amount 5000000

# Query data
pagen crm list-contacts --company "Acme Corp"
pagen crm list-deals --stage negotiation
```

## Development

### Running Tests

```bash
go test ./...
```

### Test Coverage

```bash
go test ./... -cover
```

## Version

Current version: 0.1.0

## Architecture

- **Language:** Go 1.21+
- **Database:** SQLite with WAL mode
- **MCP SDK:** github.com/modelcontextprotocol/go-sdk v1.1.0
- **Storage:** XDG Base Directory specification

## License

[Add your license here]
