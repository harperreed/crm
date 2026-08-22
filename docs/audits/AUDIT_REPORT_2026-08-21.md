<!-- ABOUTME: Records the 2026-08-21 audit of current CRM documentation against the codebase. -->
<!-- ABOUTME: Lists verified claims, corrections made during the audit, inventory checks, and review limits. -->

# Documentation Audit Report

Generated: 2026-08-21 | Commit: `b2b7c64`

## Scope

This audit covers the four current project documents changed or relied on by CRM history:

- `README.md`
- `CLAUDE.md`
- `gotchas.md`
- `cmd/crm/skill/SKILL.md`

The audit excludes `docs/superpowers/`, `docs/plans/`, and prior reports in `docs/audits/`. Those files preserve plans, designs, or past audit state rather than current usage claims.

Pass 1 extracted file, configuration, command, schema, version, and behavior claims from each document and checked them against source, live help, tests, local release configuration, and the Go module proxy. Pass 2 expanded the false-claim pattern and compared documented CLI, MCP, configuration, dependency, storage, and release surfaces with the repository inventory.

## Executive Summary

| Metric | Count |
|---|---:|
| Documents scanned | 4 |
| Atomic claims checked | 255 |
| Verified true after corrections | 249 (97.6%) |
| **Verified false after corrections** | **0** |
| Needs human or environment review | 6 (2.4%) |
| Corrections committed during audit | 2 |

The history documentation matches the implementation. The CLI examples, 14-tool MCP inventory, event sources, relationship aggregation, future-only recording, deletion retention, indefinite retention, SQLite transaction boundary, and Markdown recovery and writer limits all have direct code or test evidence.

## Corrections Made During the Audit

| Document | Original claim or gap | Evidence | Correction |
|---|---|---|---|
| `README.md:69` | Group help commands were called the “full command reference,” but they omit operation flags. | Live `crm contact --help` lists subcommands; `crm contact add --help` lists `--email`, `--phone`, `--field`, and `--tag`. | Direct readers to browse with `crm help` and add `--help` to the exact command they plan to run. |
| `gotchas.md:6` | The invalid-major-version warning named only `v2.0.0`, while `v2.1.0` now has the same module-path defect. | Local tags include both versions; `go.mod:1` has no `/v2`; the Go proxy still resolves `@latest` to `v1.5.1`. | Describe the current `v2.x` tags as incompatible with the unsuffixed module path. |

Both corrections are in commit `b2b7c64`.

## Verified Claim Groups

| Document | Claim group | Atomic claims | Evidence |
|---|---:|---:|---|
| `README.md` | Install, version, CLI, configuration, storage, MCP, history, and development commands | 53 | `Makefile`; `cmd/crm/`; `internal/config/config.go`; `internal/storage/`; `internal/mcp/`; live command help; focused CLI, MCP, and integration tests |
| `CLAUDE.md` | Project tree, canonical commands, dependencies, release configuration, storage transactions, and Markdown recovery | 67 | Repository inventory; `Makefile`; `go.mod`; `.goreleaser.yml`; `internal/storage/sqlite_history.go`; `internal/storage/markdown_history.go` |
| `gotchas.md` | Version contracts, release limits, module tags, SQL annotations, retention, and Markdown concurrency | 30 | CI workflow; Go module proxy; repository tags; `.goreleaser.yml`; storage source and tests |
| Bundled skill | All MCP tool names, schemas, defaults, examples, source labels, and retention behavior | 105 | `internal/mcp/tools.go`; `internal/mcp/history.go`; storage interfaces; transport and parity tests |

### History evidence

- `cmd/crm/history.go` defines the documented timeline and event commands, default limit, and output.
- `internal/mcp/tools.go` registers exactly 14 tools; `internal/mcp/history.go` defines `list_history` and `get_history_event`.
- `test/history_integration_test.go` proves relationship aggregation, deletion retention, exact snapshots, and no fabricated baseline events for legacy data on both backends.
- `test/cli_history_e2e_test.go` proves deleted entities remain inspectable through the built CLI and that CLI events use source `cli`.
- `internal/mcp/history_test.go` proves source `mcp`, exact snapshots, required IDs, default and maximum limits, and full event transport.
- `internal/storage/sqlite_history.go` writes current state and history in one SQLite transaction.
- `internal/storage/markdown_history.go` publishes pending events before state changes and recovers only explained before/after states.

## Pass 2A: Pattern Expansion

| Pattern | Result | Action |
|---|---|---|
| Completeness words such as “full,” “all,” and “only” | One false README claim; other matches were supported. | Corrected the help wording and rechecked every match. |
| Exact release or module versions | `@latest` remains `v1.5.1`; both current `v2.x` tags have the unsuffixed module path. | Broadened the gotcha from one tag to current `v2.x` tags. |
| History limits, sources, retention, and baseline wording | Repeated claims in all four documents agree with code and tests. | No change. Keep these claims synchronized in later work. |
| MCP tool counts and names | Registry and bundled skill both contain the same 14 tools; README states 14 and names both history tools. | No change. |
| Release-success claims | Local `goreleaser check` still reports only deprecated `brews`; a snapshot build could not finish because the host ran out of disk. | Kept in the human/environment review queue rather than marking them false. |

## Pass 2B: Inventory Comparison

| Surface | Code inventory | Documentation result |
|---|---:|---|
| MCP tools | 14 | All 14 appear in the bundled skill; README states 14 and names both history tools. |
| MCP resource templates | 2 | README documents contact and company templates. |
| MCP prompts | 3 | README documents contact creation, relationship mapping, and search prompts. |
| History CLI | `history`, `history show`, `--limit` | README documents both commands and points to exact-command help for flags. |
| Configuration keys | `backend`, `data_dir` | README documents both keys, accepted backends, XDG paths, and tilde expansion. |
| Direct dependencies | 7 | `CLAUDE.md` lists all seven with accurate roles. |
| History storage | SQLite tables plus Markdown `_history/events` and `_history/pending` | Contributor and user docs cover the relevant transaction, recovery, retention, and writer boundaries. |

No undocumented history interface or documented-but-missing history interface was found.

## Human and Environment Review Queue

- [ ] Confirm that a signed, notarized cask and coordinated tap migration remain the required product policy before replacing GoReleaser's deprecated `brews` field (`CLAUDE.md:43`, `gotchas.md:4`).
- [ ] Re-run `goreleaser release --snapshot --clean` on a machine with enough free disk to verify the complete archive, checksum, metadata, binary, and formula set (`CLAUDE.md:46-49`, `gotchas.md:4`). The audit run stopped with `ENOSPC`, so it does not show configuration drift.
- [ ] Confirm the team policy that release maintenance must not choose or add a license as a side effect (`gotchas.md:5`).

## Adjacent Findings

- The MCP discovery test locks the 14 tool names and count, while transport tests cover the new history schemas. A comprehensive schema-contract test for all tools would catch future parameter drift earlier.
- The host data volume fell below 500 MiB free during the audit. Optional artifact-heavy checks were stopped after the disposable snapshot clone was removed.

## Verification

- Live CLI help passed for history, contact, company, MCP, and skill commands.
- Focused CLI, MCP transport, parity, legacy-data, and built-binary history tests passed during the documentation work and claim extraction.
- `goreleaser check` exited 2 with one issue: the deprecated `brews` property.
- `go list -m -json github.com/harperreed/crm@latest` returned `v1.5.1`.
- Pattern searches found no stale live `12 tools` claim outside excluded historical material.
- `git diff --check` passed for the documentation commits.
