<!-- ABOUTME: Contributor guide for the CRM codebase and its canonical development commands. -->
<!-- ABOUTME: Summarizes project structure, storage conventions, dependencies, and checks. -->

# CRM Project

## Overview

CRM is a lightweight manager for contacts, companies, and relationships. Humans use the CLI; agents use the MCP server. See `README.md` for installation and usage.

## Project Structure

```text
crm/
├── cmd/crm/         # CLI entry point and Cobra commands
├── internal/
│   ├── models/      # Contact, Company, and Relationship types
│   ├── storage/     # Storage interface plus SQLite and Markdown backends
│   ├── mcp/         # MCP tools, resource templates, and prompts
│   └── config/      # XDG config and backend factory
├── test/            # Cross-backend integration tests
├── README.md
├── go.mod
├── Makefile
└── CLAUDE.md
```

## Build and Test

```bash
make build           # Build ./crm
make test            # Run tests
make test-race       # Run tests with the race detector
make test-coverage   # Generate coverage.out and coverage.html
make lint            # Run golangci-lint
make fmt             # Format Go code
make check           # Format, lint, and test
make install         # Install with go install to GOBIN or Go's default bin directory
make clean           # Remove build and coverage artifacts
```

## Release Validation

`goreleaser check` exits 2 only because the accepted Homebrew formula uses GoReleaser's deprecated `brews` field. Until the project can ship a signed, notarized cask and migrate the tap at the same time, use the passing snapshot command:

```bash
goreleaser release --snapshot --clean
```

The snapshot command replaces `dist/` with generated release archives, checksums, metadata, binaries, and a Homebrew formula. It does not publish them.

## Conventions

- Hand-written Go files start with two `// ABOUTME:` lines.
- The module path is `github.com/harperreed/crm`; the binary name is `crm`.
- SQLite is the default backend and uses `modernc.org/sqlite`.
- The default database is `$XDG_DATA_HOME/crm/crm.db` when set, otherwise `~/.local/share/crm/crm.db`.
- `internal/config` and `internal/storage` handle XDG paths directly.
- Cobra provides the CLI. `crm --version` and `crm version` are both public.
- History records new mutations only. CLI mutations use event source `cli`; mutations handled by `crm mcp` use `mcp`.
- Pre-commit hooks enforce formatting, linting, tests, and vet for applicable files.

## History Storage

SQLite stores events in `history_events` and their timeline memberships in `history_event_entities`. Each mutation writes current state and history in one transaction, so both changes commit or both roll back.

The Markdown backend stores immutable event JSON in `_history/events/`. Its write-ahead log writes each mutation to `_history/pending/` before changing current state. Opening the store replays or finalizes pending events when current state matches the recorded before or after snapshot; it rejects unexplained conflicts. A process-local mutex serializes Markdown writes and recovery, but there is no cross-process writer lock. Run only one Markdown-writing CRM process at a time.

## Direct Dependencies

- `github.com/fatih/color` — terminal output
- `github.com/google/uuid` — entity IDs
- `github.com/harperreed/mdstore` — Markdown file operations
- `github.com/modelcontextprotocol/go-sdk` — MCP server
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML encoding
- `modernc.org/sqlite` — pure-Go SQLite driver
