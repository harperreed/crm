# Pagen - Personal Agent Toolkit

A personal agent toolkit with CRM capabilities. Works both as a Model Context Protocol (MCP) server for Claude Desktop AND as a standalone CLI for direct terminal use.

## Features

- **Contact Management** - Track people with full interaction history
- **Company Management** - Organize contacts by company with industry tracking
- **Deal Pipeline** - Manage sales from prospecting to closed
- **Relationship Tracking** - Map connections between contacts (colleagues, friends, etc.)
- **Follow-Up Tracking** - Never lose touch with your network through smart cadence tracking
- **Universal Query** - Flexible searching across all entity types

## Installation

### Homebrew (macOS)

```bash
# Add the tap
brew tap harperreed/tap

# Install pagen
brew install pagen
```

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/harperreed/pagen/releases)

### Build from Source

```bash
git clone https://github.com/harperreed/pagen.git
cd pagen
CGO_ENABLED=1 go build -o pagen
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

Then restart Claude Desktop and you'll have access to all 19 CRM tools through natural language.

### 2. Interactive TUI (Default)

Launch the full-screen interactive terminal interface:

```bash
pagen
```

Features:
- **Tab** - Switch between Contacts/Companies/Deals
- **Arrow keys** - Navigate rows
- **Enter** - View details
- **n** - Create new entity
- **e** - Edit selected entity
- **d** - Delete selected entity
- **g** - View graph for entity
- **/** - Search/filter
- **q** - Quit

### 3. CLI for Direct Terminal Use

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

## Visualization Features

### Terminal Dashboard

View a static ASCII dashboard with pipeline overview, stats, and alerts:

```bash
pagen viz
```

Output:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  PAGEN CRM DASHBOARD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PIPELINE OVERVIEW
  prospecting    ████░░░░░░  12 ($45K)
  qualification  ██████░░░░  18 ($120K)
  negotiation    ████████░░   8 ($250K)

STATS
  📇 45 contacts  🏢 12 companies  💼 43 deals

NEEDS ATTENTION
  ⚠️  8 contacts - no contact in 30+ days
  ⚠️  3 deals - stale (no activity in 14+ days)
```

### Read-Only Web UI

Start the web dashboard server:

```bash
pagen web [--port 8080]
```

Visit `http://localhost:8080` in your browser.

Pages:
- `/` - Dashboard with stats and pipeline
- `/contacts` - Searchable contacts table
- `/companies` - Companies with org charts
- `/deals` - Deals with stage filtering
- `/graphs` - Interactive graph generation

All pages use HTMX for partial updates (no full page reloads).

### GraphViz Visualizations

Generate relationship graphs in DOT format:

```bash
# All contact relationships
pagen viz graph contacts

# Specific contact's network
pagen viz graph contacts <contact-id>

# Company org chart
pagen viz graph company <company-id-or-name>

# Deal pipeline flow
pagen viz graph pipeline
```

Save to file:
```bash
pagen viz graph contacts > graph.dot
dot -Tsvg graph.dot -o graph.svg
```

## Updated Command Structure

```
pagen                          # Launch interactive TUI (default)
pagen crm <command> [args]     # CLI commands for scripting
pagen mcp                      # MCP server for Claude Desktop
pagen viz                      # Terminal dashboard
pagen viz graph <type> [args]  # Generate GraphViz graphs
pagen web [--port 8080]        # Web UI server
```

### Global Flags

- `--version` - Show version and exit
- `--db-path <path>` - Use custom database path (default: `~/.local/share/crm/crm.db`)
- `--init` - Initialize database and exit (use with `crm` command)

### Available Commands

**Top Level:**
- `pagen` - Launch interactive TUI (default)
- `pagen mcp` - Start MCP server (for Claude Desktop)
- `pagen crm` - CRM management commands
- `pagen viz` - Terminal dashboard
- `pagen viz graph` - Generate GraphViz visualizations
- `pagen web` - Start web UI server

Run `pagen --help` for full help.

## Complete CRUD Operations

### Contacts

```bash
pagen crm add-contact --name "Alice" --email "alice@example.com" [--phone "555-1234"] [--company "CompanyName"] [--notes "Notes"]
pagen crm find-contacts [--query "search"] [--company-id <uuid>]
pagen crm update-contact <id> [--name "New Name"] [--email "new@email.com"] [--phone "555-5678"] [--company "NewCompany"] [--notes "Updated notes"]
pagen crm delete-contact <id>
pagen crm log-interaction --contact <name-or-id> [--note "Met for coffee"]
```

### Companies

```bash
pagen crm add-company --name "Acme Corp" [--domain "acme.com"] [--industry "Software"] [--notes "Notes"]
pagen crm find-companies [--query "search"]
pagen crm update-company <id> [--name "New Name"] [--domain "newdomain.com"] [--industry "NewIndustry"] [--notes "Updated notes"]
pagen crm delete-company <id>  # Fails if company has active deals
```

### Deals

```bash
pagen crm add-deal --title "Enterprise License" --company "Acme Corp" [--contact "Alice"] [--amount 500000] [--currency USD] [--stage prospecting] [--note "Initial outreach"]
pagen crm find-deals [--query "search"]
pagen crm update-deal <id> [--title "New Title"] [--stage negotiation] [--amount 600000]
pagen crm delete-deal <id>  # Cascades to deal notes
pagen crm add-deal-note --deal <id> --note "Follow-up completed"
```

### Relationships

```bash
pagen crm link-contacts --contact1 <id-or-name> --contact2 <id-or-name> [--type "colleague"] [--context "Work together"]
pagen crm find-relationships --contact <id> [--type "colleague"]
pagen crm update-relationship <id> [--type "friend"] [--context "Updated context"]
pagen crm delete-relationship <id>
```

### Query (MCP-style)

```bash
pagen crm query --entity-type <contact|company|deal|relationship> [--query "search"] [--limit 50]
```

### Follow-Up Commands

```bash
# List contacts needing follow-up
pagen followups list [--overdue-only] [--strength weak|medium|strong] [--limit 10]

# Log an interaction
pagen followups log --contact "Alice" --type meeting --notes "Coffee chat"

# Set follow-up cadence
pagen followups set-cadence --contact "Bob" --days 14 --strength strong

# View network health stats
pagen followups stats

# Generate daily digest
pagen followups digest [--format text|json|html]
```

### Follow-Up in TUI

Press `f` to view the Follow-Ups tab showing:
- Prioritized list of contacts needing attention
- Visual indicators (🔴 overdue, 🟡 due soon, 🟢 on track)
- Quick interaction logging with `l`
- Cadence adjustment with `c`

### Follow-Up in Web UI

Visit `/followups` for:
- Filterable table of contacts needing follow-up
- One-click interaction logging via HTMX
- Priority-based sorting

## Google Sync

Pagen syncs both **Contacts** and **Calendar** data from Google into your local CRM database, creating a unified contact and interaction management experience.

### Features

**Contacts Sync:**
- **One-Time OAuth Setup** - Authenticate with Google using industry-standard OAuth 2.0 flow
- **Selective Import** - Import all contacts or just recent ones (last 6 months)
- **Smart Mapping** - Maps Google contact fields to pagen's schema (names, emails, phones, companies)
- **Duplicate Prevention** - Won't create duplicate contacts

**Calendar Sync:**
- **Incremental Sync** - Only fetches new/changed events using sync tokens
- **Automatic Interaction Logging** - Meeting attendees become contacts with logged interactions
- **Smart Filtering** - Skips irrelevant events (all-day, declined, solo, cancelled)
- **Cadence Tracking** - Auto-updates contact follow-up cadences based on interactions
- **Initial Sync** - Import last 6 months of calendar history

**Shared Features:**
- **XDG-Compliant Storage** - Credentials stored securely in `~/.local/share/pagen/`
- **Combined Sync** - Run both contacts and calendar in one command

### Setup Instructions

#### 1. Create Google Cloud Project

1. Visit the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select an existing one)
3. Enable the required APIs:
   - Navigate to "APIs & Services" > "Library"
   - Search for and enable **Google People API** (for contacts)
   - Search for and enable **Google Calendar API** (for calendar)

#### 2. Create OAuth 2.0 Credentials

1. Go to "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "OAuth client ID"
3. Select application type: **Desktop app**
4. Give it a name (e.g., "Pagen Desktop")
5. Click "Create"
6. Download the JSON credentials (optional - you just need the Client ID and Secret)

#### 3. Configure Environment Variables

Set your OAuth credentials as environment variables:

```bash
export GOOGLE_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="your-client-secret"
```

Add these to your shell profile (`~/.zshrc`, `~/.bashrc`, etc.) to make them permanent.

### Usage

#### Initialize Google OAuth (One-Time)

Run the init command to authenticate with Google:

```bash
pagen sync init
```

This will:
1. Open your default browser to Google's OAuth consent screen
2. Ask you to select your Google account and grant permissions
3. Save the access token to `~/.local/share/pagen/google-credentials.json`
4. Display a success message

**Note:** The OAuth flow uses `http://localhost:8080` as the redirect URI. Make sure this port is available.

#### Sync Contacts

After initialization, import your Google Contacts:

```bash
# Import all contacts
pagen sync contacts

# Initial sync - only import contacts modified in last 6 months
pagen sync contacts --initial
```

The import will:
- Fetch contacts from Google People API
- Map fields to pagen's schema (names → name, emails → email, phones → phone)
- Extract company names from organization fields
- Create company records if they don't exist
- Link contacts to companies
- Skip contacts that already exist (based on name matching)

#### Sync Calendar

Sync your Google Calendar to automatically log interactions with contacts:

```bash
# Incremental sync - fetch only new/changed events since last sync
pagen sync calendar

# Initial sync - import last 6 months of calendar history
pagen sync calendar --initial
```

Example output (initial sync):

```
Syncing Google Calendar (last 6 months)...
  → Fetching events...
  ✓ Fetched 156 events
  ✓ Skipped 23 all-day events
  ✓ Skipped 8 declined events
  ✓ Skipped 12 solo events

  → Processing 113 meetings...
  ✓ Created 47 new contacts from attendees
  ✓ Logged 113 interactions

Summary:
  New contacts: 47
  Interactions: 113 logged

Sync token saved. Next sync will be incremental.
```

Example output (incremental sync):

```
Syncing Google Calendar...
  ✓ 3 new events since last sync
  ✓ 1 updated event
  ✓ Logged 4 interactions

All up to date!
```

**Event Filtering Rules:**

Calendar sync intelligently filters events to focus on meaningful interactions:

- **Skips All-Day Events** - Daily events like holidays, birthdays, reminders
- **Skips Declined Events** - Meetings you declined won't create interactions
- **Skips Solo Events** - Events with 0-1 attendees (you alone)
- **Skips Cancelled Events** - Cancelled meetings are ignored

Only multi-person meetings you accepted or tentatively accepted are imported.

**What Happens During Sync:**

1. **Fetch Events** - Downloads events from Google Calendar API
2. **Filter Events** - Applies filtering rules (see above)
3. **Create Contacts** - Meeting attendees become contacts if they don't exist
4. **Log Interactions** - Creates interaction records with meeting timestamps
5. **Update Cadences** - Adjusts follow-up schedules based on interaction history
6. **Save Sync Token** - Stores incremental sync state for next run

#### Combined Sync

Sync both contacts and calendar in one command:

```bash
pagen sync
```

Example output:

```
Syncing Google Contacts...
  ✓ No changes since last sync

Syncing Google Calendar...
  ✓ 3 new events
  ✓ Logged 3 interactions

Summary:
  Interactions: 3 logged

All up to date!
```

### Storage Locations

- **OAuth Tokens:** `~/.local/share/pagen/google-credentials.json`
- **CRM Database:** `~/.local/share/pagen/pagen.db`
- **Sync State:** Stored in database `sync_state` table

All paths follow XDG Base Directory specifications.

### Current Limitations

This is the foundation layer for Google integration. Current limitations:

**Contacts:**
- **Import Only** - No bidirectional sync (updates in pagen don't sync back to Google)
- **No Update Detection** - Won't update existing contacts with changes from Google
- **Simple Matching** - Uses name-based duplicate detection

**Calendar:**
- **Read-Only** - Calendar events are imported, not modified in Google
- **Manual Sync** - No automatic background sync (run commands manually)
- **Basic Filtering** - Simple event type filtering (no content analysis)

Future phases will add bidirectional sync, automated updates, relationship syncing, deal detection, and more sophisticated conflict resolution.

### Troubleshooting

#### General Issues

**"Missing environment variables" error:**
- Verify `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` are set: `env | grep GOOGLE`

**OAuth flow doesn't open browser:**
- The URL will be printed to the terminal - copy and paste it manually

**Port 8080 already in use:**
- Stop any services using port 8080, or wait for the OAuth flow to use a random available port

#### Contacts Issues

**"API not enabled" error:**
- Ensure Google People API is enabled in your Google Cloud Console project
- Visit: https://console.cloud.google.com/apis/library/people.googleapis.com

#### Calendar Issues

**"Google Calendar API not enabled" error:**
- Calendar API must be enabled in Google Cloud Console
- Visit: https://console.cloud.google.com/apis/library/calendar-json.googleapis.com
- Click "Enable" and try syncing again

**"Sync token expired" warning:**
- This happens if you deleted events or haven't synced in a long time
- The sync will automatically fall back to time-based incremental sync
- No action needed - the sync will continue normally

**"No events imported" (all events skipped):**
- Check the event filtering rules above
- Verify you have multi-person meetings (not solo events)
- Ensure events aren't all-day, declined, or cancelled
- Try `--initial` flag to see detailed filtering stats

**"Failed to fetch calendar events" network error:**
- Check your internet connection
- The sync will automatically retry (3 attempts)
- If it persists, check Google Cloud Console API quotas

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

## MCP Tools

Total: **22 tools** for Claude Desktop integration

### Contact Operations (5 tools)
- `add_contact` - Create new contacts with optional company linking
- `find_contacts` - Search by name, email, or company
- `update_contact` - Modify contact information
- `delete_contact` - Delete a contact and all associated relationships
- `log_contact_interaction` - Record interactions with timestamp tracking

### Company Operations (4 tools)
- `add_company` - Create companies with industry/domain metadata
- `find_companies` - Search by name or domain
- `update_company` - Modify company information
- `delete_company` - Delete a company (must have no active deals)

### Deal Operations (4 tools)
- `create_deal` - Create deals with company and contact associations
- `update_deal` - Modify deal details including stage and amount
- `delete_deal` - Delete a deal and all associated notes
- `add_deal_note` - Add activity notes to deals

### Relationship Operations (4 tools)
- `link_contacts` - Create relationships between contacts
- `find_contact_relationships` - Find all connections for a contact
- `update_relationship` - Update a relationship's type and context
- `remove_relationship` - Delete relationship links

### Follow-Up Operations (3 tools)
- `get_followup_list` - Get prioritized follow-up suggestions
- `log_interaction` - Log interactions and update tracking
- `set_cadence` - Configure follow-up frequency per contact

### Query Operations (1 tool)
- `query_crm` - Universal query across all entity types with flexible filtering

### Visualization Operations (1 tool)
- `generate_graph` - Generate GraphViz DOT for contact networks, company org charts, or deal pipelines

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

## Architecture

**Single Binary Distribution:**
- All templates embedded via `go:embed`
- No external files required
- Pure Go dependencies only

**Technology Stack:**
- Database: SQLite with CGO
- TUI: bubbletea + lipgloss
- Web: Go templates + HTMX (CDN) + Tailwind (CDN)
- GraphViz: goccy/go-graphviz (pure Go)
- MCP: Claude Agent SDK

## Testing

Run all scenario tests:

```bash
make test
./.scratch/test_all_visualization.sh
./.scratch/test_integration.sh
```

Manual testing:
- TUI: `./.scratch/test_tui_manual.sh`
- Web: `./.scratch/test_web_manual.sh`

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

## License

[Add your license here]
