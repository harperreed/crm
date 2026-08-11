<!-- ABOUTME: Records the 2026-08-11 audit of user-facing CRM documentation against the codebase. -->
<!-- ABOUTME: Lists false claims, missing coverage, verified surfaces, and adjacent release risks. -->

# Documentation Audit Report

Generated: 2026-08-11 | Commit: `48dcf97`

## Scope

This audit covers the two current user-facing documents:

- `CLAUDE.md`
- `cmd/crm/skill/SKILL.md`

The audit excludes `docs/superpowers/plans/` and `docs/superpowers/specs/`. Those files record historical planning and design work, so treating them as current user documentation would produce false alarms. The repository has no `README.md`.

The audit used two passes. Pass 1 extracted claims from each target and checked them against source, configuration, command help, and tests. Pass 2 searched for related drift and compared the documented surface with the actual CLI, configuration, storage, MCP, and release surfaces.

## Executive Summary

| Metric | Count |
|---|---:|
| Documents scanned | 2 |
| Claims verified | 55 |
| Verified true | 50 (90.9%) |
| **Verified false** | **5 (9.1%)** |
| Needs human review | 0 |
| Documentation gap groups | 8 |

The embedded CRM skill is accurate: all 22 checked claims match the MCP server. All five false claims are in `CLAUDE.md`. They come from a driver change, removal of the XDG helper dependency, and path wording that no longer matches the current environment or fallback behavior.

## False Claims Requiring Fixes

### `CLAUDE.md`

| Line | Claim | Reality | Evidence | Fix |
|---:|---|---|---|---|
| 32 | `make install` installs to `GOPATH/bin` | `go install` uses `GOBIN` when set, then falls back to `GOPATH/bin`. This machine has `GOBIN` set. | `Makefile:19-20`; `go env GOBIN GOPATH` | Say that the target installs with `go install`, to `GOBIN` or the Go default bin directory. |
| 41 | SQLite uses `mattn/go-sqlite3` | The store imports the pure-Go `modernc.org/sqlite` driver. | `internal/storage/sqlite.go:12`; `go.mod:12` | Replace the driver name. |
| 41 | The default database path is `XDG_DATA_HOME/crm/crm.db` | That path applies when `XDG_DATA_HOME` is set. The unset fallback is `~/.local/share/crm/crm.db`. | `internal/storage/sqlite.go:137-145`; `internal/config/config.go:53-56` | Document both branches. |
| 48 | `github.com/mattn/go-sqlite3` is a dependency | It is absent from `go.mod`; `modernc.org/sqlite` is the direct dependency. | `go.mod:5-12` | Replace the dependency entry. |
| 49 | `github.com/adrg/xdg` handles XDG paths | It is absent from `go.mod`. The config and storage packages read XDG environment variables directly. | `internal/config/config.go:64-70`; `internal/storage/sqlite.go:137-145` | Remove the dependency entry and describe the built-in path handling. |

## Verified Claim Groups

| Document | Claim group | True | Evidence |
|---|---|---:|---|
| `CLAUDE.md` | Project purpose and CLI/MCP access | 2 | `internal/models/*.go`; `cmd/crm/root.go:16-19`; `cmd/crm/mcp.go:10-21` |
| `CLAUDE.md` | Project tree and component roles | 10 | Listed paths exist; `internal/storage/interface.go:20-43`; `internal/mcp/server.go:19-31` |
| `CLAUDE.md` | Build, test, lint, format, check, and cleanup targets | 8 | `Makefile:4-33` |
| `CLAUDE.md` | ABOUTME convention | 1 | All 40 Go files start with two `// ABOUTME:` lines. |
| `CLAUDE.md` | Module, binary, backend, XDG environment, Cobra, and hooks | 7 | `go.mod:1,10`; `Makefile:7`; `internal/config/config.go:21-25`; `.pre-commit-config.yaml:20-38` |
| CRM skill | Contact tools and schemas | 5 | `internal/mcp/tools.go:76-153,267-411` |
| CRM skill | Company tools and schemas | 5 | `internal/mcp/tools.go:156-231,414-552` |
| CRM skill | Relationship tools and schemas | 2 | `internal/mcp/tools.go:234-263,555-604` |
| CRM skill | Usage examples | 8 | Each example uses registered tool and parameter names in `internal/mcp/tools.go`. |
| CRM skill | Claude Code stdio configuration | 1 | `cmd/crm/mcp.go:10-17`; current Claude Code MCP configuration accepts `command` and `args` under `mcpServers`. |

The `mcp__crm__*` names in the skill depend on the client-side server name remaining `crm`. The server itself advertises bare names such as `add_contact`; Claude Code adds the configured server prefix.

## Pass 2A: Pattern Expansion

| Pattern | Count | Root cause |
|---|---:|---|
| Removed or replaced dependencies | 3 | SQLite moved to `modernc.org/sqlite`, and XDG handling moved into project code. |
| Incomplete path or install defaults | 2 | The docs describe one environment branch as universal. |

A repository-wide search outside historical plans and specs found no further uses of `mattn/go-sqlite3`, `adrg/xdg`, the incomplete database path, or the `GOPATH/bin` claim.

## Pass 2B: Documentation Gaps

| Severity | Gap | Actual surface |
|---|---|---|
| High | No README | The repository has no entry document for installation, configuration, CLI use, MCP setup, or releases. |
| High | No configuration reference | JSON keys are `backend` and `data_dir`; config lives at `$XDG_CONFIG_HOME/crm/config.json` or `~/.config/crm/config.json`. See `internal/config/config.go:15-89`. |
| Medium | No CLI reference | Contacts and companies each have add, list, show, edit, and remove commands. Root commands also include link, unlink, MCP, skill installation, and version output. See `cmd/crm/*.go`. |
| Medium | Storage behavior is thinly documented | The docs omit backend selection, Markdown file layout, WAL and foreign-key setup, and the different search implementations. See `internal/storage/`. |
| Medium | MCP resources are omitted | The server exposes `crm://contacts/{id}` and `crm://companies/{id}`. See `internal/mcp/resources.go:15-35`. |
| Medium | MCP prompts are omitted | The server exposes `add-contact-workflow`, `relationship-mapping`, and `crm-search`. See `internal/mcp/prompts.go:13-53`. |
| Medium | Release flow is omitted | Tag pushes run GoReleaser and target macOS/Linux on AMD64/ARM64 plus a Homebrew tap. See `.github/workflows/release.yml` and `.goreleaser.yml`. |
| Low | Runtime and dependency requirements are incomplete | Go `1.25.5` and six direct dependencies beyond Cobra do not appear in the docs. See `go.mod:3-12`. |

## Adjacent Repository Risks

These are code or release configuration defects found while verifying documentation. They are not false documentation claims.

| Severity | Finding | Evidence | Recommended action |
|---|---|---|---|
| High | The Homebrew formula test runs `crm --version`, but the CLI implements `crm version`. The configured command exits 1. | `.goreleaser.yml:42-45`; `cmd/crm/version.go:12-24`; direct command run | Change the formula test or add an approved CLI flag contract. |
| High | The current GoReleaser rejects `.goreleaser.yml` because `archives.format` and `brews` are deprecated. | `goreleaser check` exits 2 | Update the config to the installed GoReleaser schema before the next release. |
| Medium | Archive patterns name `README*` and `LICENSE*`, but neither file exists. | `.goreleaser.yml:27-32`; repository inventory | Add the intended files or remove the patterns after checking GoReleaser's missing-file behavior. |
| Medium | The canonical linter currently reports two `gosec` G202 findings. | `golangci-lint run ./...`; `internal/storage/sqlite_contacts.go:95`; `internal/storage/sqlite_companies.go:95` | Confirm that only fixed SQL fragments are concatenated, then refactor or add narrow, justified annotations. |

Unit tests and `go vet ./...` passed during the audit. The linter failures mean `make check` is currently red.

## Human Review Queue

No extracted documentation claim remains unverified. Product decisions remain for the scope and location of the missing README and whether MCP resources and prompts belong in the installed skill.

## External Verification

The Claude Code MCP JSON shape was checked against Anthropic's current official documentation: <https://code.claude.com/docs/en/mcp>.
