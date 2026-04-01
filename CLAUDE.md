# CRM Project

## Overview

A lightweight CRM for contacts, companies, and relationships. Accessible via CLI and MCP.

## Project Structure

```
crm/
├── cmd/crm/         # CLI entry point and Cobra commands
├── internal/
│   ├── models/      # Data types (Contact, Company, Relationship)
│   ├── storage/     # Storage interface and implementations (SQLite, Markdown)
│   ├── mcp/         # MCP server, tools, resources, prompts
│   └── config/      # XDG config and backend factory
├── go.mod
├── Makefile
└── CLAUDE.md
```

## Build & Test

```bash
make build           # Build binary
make test            # Run tests
make test-race       # Run tests with race detector
make test-coverage   # Generate coverage report
make lint            # Run golangci-lint
make fmt             # Format code
make check           # fmt + lint + test
make install         # Install to GOPATH/bin
make clean           # Remove artifacts
```

## Code Conventions

- All code files start with two `// ABOUTME:` comment lines describing the file.
- Module path: `github.com/harperreed/crm`
- Binary name: `crm`
- Storage backend: SQLite via `mattn/go-sqlite3` with `XDG_DATA_HOME/crm/crm.db` default.
- CLI framework: `github.com/spf13/cobra`
- Pre-commit hooks enforce formatting, linting, and tests.

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/mattn/go-sqlite3` - SQLite driver
- `github.com/adrg/xdg` - XDG directory paths
