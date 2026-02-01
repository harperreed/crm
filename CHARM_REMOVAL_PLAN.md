# Pagen - Charm Removal Plan

## Overview

Remove all Charmbracelet dependencies, transition to web-only UI with local SQLite storage.

## Charmbracelet Dependencies

**Direct:**
- `github.com/charmbracelet/bubbles v0.21.0` - TUI components
- `github.com/charmbracelet/bubbletea v1.3.10` - TUI framework
- `github.com/charmbracelet/charm` (2389-research fork) - KV storage
- `github.com/charmbracelet/lipgloss v1.1.0` - Terminal styling

## Decision: REMOVE TUI ENTIRELY

The web UI at `/web` provides equivalent functionality and is more maintainable. Delete the entire `tui/` directory.

## Removal Strategy

### Phase 1: Create SQLite Repository Layer

The existing `models/types.go` already defines models matching `charm/models.go`. Create new `repository/` package:

- `repository/db.go` - Connection management
- `repository/contacts.go`
- `repository/companies.go`
- `repository/deals.go`
- `repository/relationships.go`
- `repository/interactions.go`
- `repository/cadence.go`
- `repository/filters.go`

Use existing `db/schema.go` schema.

### Phase 2: Add Export Commands

```bash
pagen export contacts [--format yaml|markdown]
pagen export companies [--format yaml|markdown]
pagen export deals [--format yaml|markdown]
pagen export all [--format yaml] [--output-dir dir]
```

**Markdown Format:**
```markdown
# Contact: John Smith

- **Email**: john@example.com
- **Phone**: 555-1234
- **Company**: Acme Corp

## Interaction History
- [2024-01-15] Email follow-up
- [2024-01-10] Initial meeting
```

**YAML Format:**
```yaml
contacts:
  - id: "uuid-here"
    name: "John Smith"
    email: "john@example.com"
    company_id: "company-uuid"
    company_name: "Acme Corp"
```

### Phase 3: Update All Consumers

**Files to modify:**
- `main.go` - Remove TUI launch, use repository
- `cli/contacts.go` - Use repository.DB
- `cli/companies.go` - Use repository.DB
- `cli/deals.go` - Use repository.DB
- `cli/followups.go` - Use repository.DB
- `cli/relationships.go` - Use repository.DB
- `cli/mcp.go` - Use repository.DB
- `handlers/*.go` - All handler files
- `web/server.go` - Use repository.DB
- `viz/viz.go` - Use repository.DB

### Phase 4: Delete TUI and Charm Packages

**Delete directories:**
- `tui/` - Entire directory (11 files)
- `charm/` - Entire directory (10 files)

**Modify:**
- `officeos/timeline_view.go` - Remove lipgloss (use plain text)

### Phase 5: Update go.mod

Remove:
```go
replace github.com/charmbracelet/charm => github.com/2389-research/charm v0.20.0

github.com/charmbracelet/bubbles v0.21.0
github.com/charmbracelet/bubbletea v1.3.10
github.com/charmbracelet/charm v0.0.0
github.com/charmbracelet/lipgloss v1.1.0
```

Run `go mod tidy`.

### Phase 6: Replace Sync Commands

- `pagen sync status` -> Show SQLite path and stats
- `pagen sync now` -> Remove
- `pagen sync link/unlink` -> Remove
- `pagen sync wipe` -> `pagen db reset`
- `pagen sync repair` -> `pagen db repair` (SQLite vacuum)

## Files Summary

### DELETE (21 files):
```
tui/tui.go
tui/list_view.go
tui/detail_view.go
tui/edit_view.go
tui/sync_view.go
tui/sync_view_test.go
tui/followup_view.go
tui/graph_view.go
tui/delete_view.go
tui/task_view.go
tui/task_view_test.go
charm/client.go
charm/cli.go
charm/config.go
charm/models.go
charm/repository.go
charm/filters.go
charm/keys.go
charm/cascades.go
charm/testhelper.go
charm/wal_test.go
```

### CREATE (9 files):
```
repository/db.go
repository/contacts.go
repository/companies.go
repository/deals.go
repository/relationships.go
repository/interactions.go
repository/cadence.go
repository/filters.go
cli/export.go
```

### MODIFY (20+ files):
```
main.go
cli/contacts.go
cli/companies.go
cli/deals.go
cli/followups.go
cli/relationships.go
cli/mcp.go
cli/sync.go
cli/vault_sync.go
handlers/contacts.go
handlers/companies.go
handlers/deals.go
handlers/relationships.go
handlers/query.go
handlers/resources.go
handlers/prompts.go
handlers/viz.go
handlers/followup_handlers.go
web/server.go
viz/viz.go
officeos/timeline_view.go
go.mod
```

## Implementation Order

1. Create repository package with SQLite
2. Add export commands (before destructive changes)
3. Create migration tool to export Charm data
4. Update handlers to use repository
5. Update CLI commands
6. Update web server
7. Update main.go, remove TUI
8. Delete `tui/` directory
9. Delete `charm/` directory
10. Clean up go.mod
11. Run tests

## Data Migration

Before removing Charm:
1. `pagen export all --format yaml --output-dir ./backup`
2. Verify backup contains all data
3. Create import command for YAML
4. Test import into new SQLite repository
