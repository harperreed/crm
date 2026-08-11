<!-- ABOUTME: Plans the TDD implementation of the approved CRM release and documentation repairs. -->
<!-- ABOUTME: Specifies exact edits, red-green checks, commits, and final release verification. -->

# Release and Documentation Repair Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make release checks clean, support both version entry points, and replace stale or missing user documentation with current guidance.

**Architecture:** Keep the existing Cobra command tree and storage behavior intact. Enable Cobra's native version flag from the existing linker-injected version variable, migrate GoReleaser in place to its current archive and Homebrew cask schema, annotate two verified static-analysis false positives at their exact lines, and make README/CLAUDE/gotchas read from current code as the source of truth.

**Tech Stack:** Go 1.25.5, Cobra, GitHub Actions, GoReleaser 2.17, golangci-lint/gosec, Markdown.

**Estimated change:** About 25 lines of Go and tests, 15 lines of release/CI YAML, and 140–180 lines of documentation.

---

### Task 1: Support `crm --version`

**Files:**

- Create: `cmd/crm/root_test.go`
- Modify: `cmd/crm/root.go:16-20`
- Modify: `.github/workflows/ci.yml:81-85`

**Step 1: Write the failing CLI test**

Create `cmd/crm/root_test.go`:

```go
// ABOUTME: Tests for root CRM command behavior.
// ABOUTME: Verifies public flags that bypass storage initialization.

package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestRootVersionFlag(t *testing.T) {
	var output bytes.Buffer
	rootCmd.SetArgs([]string{"--version"})
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	if err := Execute(); err != nil {
		t.Fatalf("Execute --version: %v", err)
	}

	want := fmt.Sprintf("crm version %s\n", version)
	if got := output.String(); got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
```

**Step 2: Run the test and verify RED**

Run:

```bash
go test ./cmd/crm -run '^TestRootVersionFlag$' -count=1 -v
```

Expected: FAIL because `--version` is an unknown flag. Confirm the failure is the missing public behavior, not test setup.

**Step 3: Enable Cobra's native version flag**

Add the existing version value to the root command in `cmd/crm/root.go`:

```go
var rootCmd = &cobra.Command{
	Use:     "crm",
	Short:   "A simple CRM for contacts, companies, and relationships",
	Long:    "CRM is a lightweight contact relationship manager accessible via CLI and MCP.",
	Version: version,
```

Do not change `cmd/crm/version.go`; its detailed command remains public.

**Step 4: Run the focused test and verify GREEN**

Run:

```bash
go test ./cmd/crm -run '^TestRootVersionFlag$' -count=1 -v
```

Expected: PASS with no storage files created.

**Step 5: Extend CI's real binary check**

Replace the single command in `.github/workflows/ci.yml` with:

```yaml
      - name: Verify version commands
        run: |
          ./crm version
          ./crm --version
```

**Step 6: Exercise both paths on a real binary**

Run:

```bash
make build
./crm version
./crm --version
```

Expected: both commands exit 0 and report `dev`; the subcommand also reports commit and build date.

**Step 7: Commit**

Run `git status`, then:

```bash
git add cmd/crm/root_test.go cmd/crm/root.go .github/workflows/ci.yml
git commit -m "feat: support version flag"
```

Do not bypass hooks.

### Task 2: Migrate GoReleaser to archives and Homebrew casks

**Files:**

- Modify: `.goreleaser.yml:27-45`

**Step 1: Reproduce the current schema failure**

Run:

```bash
goreleaser check
```

Expected: exit 2 with deprecations for `archives.format` and `brews`.

**Step 2: Replace the deprecated configuration**

Replace the `archives` and `brews` sections with:

```yaml
archives:
  - formats:
      - tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - README*

homebrew_casks:
  - repository:
      owner: harperreed
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://github.com/harperreed/crm"
    description: "Lightweight CRM with MCP server and CLI for contacts, companies, and relationships"
    binaries:
      - crm
```

Do not add `directory: Formula`, formula `install`/`test` blocks, or a license pattern.

**Step 3: Verify GREEN against the installed schema**

Run:

```bash
goreleaser check
```

Expected: exit 0 with no deprecation warning.

**Step 4: Commit**

Run `git status`, then:

```bash
git add .goreleaser.yml
git commit -m "fix: migrate release config to Homebrew cask"
```

Do not bypass hooks.

### Task 3: Resolve the verified G202 false positives

**Files:**

- Modify: `internal/storage/sqlite_contacts.go:95`
- Modify: `internal/storage/sqlite_companies.go:95`

**Step 1: Reproduce the lint failure**

Run:

```bash
golangci-lint run ./...
```

Expected: exactly two G202 findings at the documented query-concatenation lines.

**Step 2: Reconfirm the security invariant**

Read each full list function and verify:

- Every member of `clauses` is a fixed string literal in source.
- Filter values are appended only to `args`.
- `s.db.Query(query, args...)` sends values separately from SQL text.

If any invariant is false, stop and fix the query construction instead of suppressing G202.

**Step 3: Add narrow, justified annotations**

Change each reported line to:

```go
		query += " WHERE " + strings.Join(clauses, " AND ") // #nosec G202 -- clauses are fixed SQL; filter values remain parameterized.
```

The annotation must stay on the reported line, name only G202, and keep the justification.

**Step 4: Verify behavior and lint GREEN**

Run:

```bash
go test ./internal/storage -run '^(TestListContacts|TestListCompanies)$' -count=1 -v
golangci-lint run ./...
```

Expected: both storage tests pass and lint exits 0 with no warnings.

**Step 5: Commit**

Run `git status`, then:

```bash
git add internal/storage/sqlite_contacts.go internal/storage/sqlite_companies.go
git commit -m "fix: document safe SQL clause construction"
```

Do not bypass hooks.

### Task 4: Add the user README

**Files:**

- Create: `README.md`

**Step 1: Create the README from current behavior**

Create `README.md` with this content:

````markdown
<!-- ABOUTME: User guide for installing, configuring, and operating CRM. -->
<!-- ABOUTME: Documents the CLI, storage backends, MCP server, and development checks. -->

# CRM

CRM is a small local-first relationship manager for contacts, companies, and the links between them. Humans use its command-line interface; agents use the same data through its MCP server.

## Install

With Homebrew:

```bash
brew install --cask harperreed/tap/crm
```

With Go 1.25.5 or newer:

```bash
go install github.com/harperreed/crm/cmd/crm@latest
```

Check the installed build with either public version entry point:

```bash
crm --version
crm version
```

## Use the CLI

Add and inspect a contact:

```bash
crm contact add "Jane Doe" \
  --email jane@example.com \
  --phone +1-555-0123 \
  --tag engineering \
  --field title="VP Engineering"
crm contact list --tag engineering
crm contact show <contact-id-or-prefix>
```

Add a company and connect it to the contact:

```bash
crm company add "Acme Corp" --domain acme.com --tag customer
crm link <contact-id-or-prefix> <company-id-or-prefix> \
  --type works_at \
  --context "VP Engineering"
```

Contacts and companies support `add`, `list`, `show`, `edit`, and `rm`. List commands accept `--tag`, `--search`, and `--limit`. IDs shown by the CLI may be shortened to a unique prefix of at least six characters.

Run `crm help`, `crm contact --help`, or `crm company --help` for the full command reference.

## Configure storage

CRM uses SQLite by default. With no overrides, it stores the database at:

```text
~/.local/share/crm/crm.db
```

`$XDG_DATA_HOME/crm/crm.db` takes precedence when `XDG_DATA_HOME` is set.

Configuration is optional. CRM reads JSON from `$XDG_CONFIG_HOME/crm/config.json`, or `~/.config/crm/config.json` when `XDG_CONFIG_HOME` is unset:

```json
{
  "backend": "markdown",
  "data_dir": "~/.local/share/crm"
}
```

`backend` accepts `sqlite` or `markdown`. `data_dir` overrides the normal XDG data directory. The Markdown backend writes contacts and companies as Markdown files with YAML frontmatter and stores links in `_relationships.yaml`.

## Connect an MCP client

Start the stdio server with:

```bash
crm mcp
```

Claude Code project configuration:

```json
{
  "mcpServers": {
    "crm": {
      "command": "crm",
      "args": ["mcp"]
    }
  }
}
```

The server exposes 12 tools for contact and company CRUD plus linking and unlinking. It also exposes contact and company resource templates and three prompts for contact creation, relationship mapping, and cross-entity search.

Install the bundled Claude Code skill with:

```bash
crm install-skill
```

This writes the skill to `~/.claude/skills/crm/SKILL.md`.

## Develop

```bash
make build          # Build ./crm
make test           # Run unit and integration tests
make test-race      # Run tests with the race detector
make lint           # Run golangci-lint
make check          # Format, lint, and test
goreleaser check    # Validate release configuration
```

The project uses Go 1.25.5. SQLite uses the pure-Go `modernc.org/sqlite` driver, so release builds do not require CGO.
````

**Step 2: Verify every documented command and configuration key**

Run:

```bash
go run ./cmd/crm --help
go run ./cmd/crm contact add --help
go run ./cmd/crm contact list --help
go run ./cmd/crm company add --help
go run ./cmd/crm link --help
go run ./cmd/crm mcp --help
```

Then compare the config example with `internal/config/config.go` and the MCP counts with `internal/mcp/tools.go`, `resources.go`, and `prompts.go`. Fix the README if code and prose differ; do not change code to fit prose.

**Step 3: Commit**

Run `git status`, then:

```bash
git add README.md
git commit -m "docs: add CRM user guide"
```

Do not bypass hooks.

### Task 5: Correct contributor docs and shared memory

**Files:**

- Modify: `CLAUDE.md:1-49`
- Modify: `gotchas.md:1-5`

**Step 1: Replace stale contributor guidance**

Replace `CLAUDE.md` with:

```markdown
<!-- ABOUTME: Contributor guide for the CRM codebase and its canonical development commands. -->
<!-- ABOUTME: Summarizes project structure, storage conventions, dependencies, and checks. -->

# CRM Project

## Overview

A lightweight CRM for contacts, companies, and relationships. Humans use the CLI; agents use the MCP server. See `README.md` for installation and usage.

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
make install         # Install with go install to the configured Go bin directory
make clean           # Remove build and coverage artifacts
goreleaser check     # Validate release configuration
```

## Conventions

- Hand-written Go files start with two `// ABOUTME:` lines.
- Module path: `github.com/harperreed/crm`.
- Binary name: `crm`.
- SQLite is the default backend and uses `modernc.org/sqlite`.
- The default database is `$XDG_DATA_HOME/crm/crm.db` when set, otherwise `~/.local/share/crm/crm.db`.
- XDG paths are handled directly in `internal/config` and `internal/storage`.
- Cobra provides the CLI. `crm --version` and `crm version` are both public.
- Pre-commit hooks enforce formatting, linting, tests, and vet for applicable files.

## Direct Dependencies

- `github.com/fatih/color` — terminal output
- `github.com/google/uuid` — entity IDs
- `github.com/harperreed/mdstore` — Markdown file operations
- `github.com/modelcontextprotocol/go-sdk` — MCP server
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML encoding
- `modernc.org/sqlite` — pure-Go SQLite driver
```

**Step 2: Replace stale gotchas with durable facts**

Replace `gotchas.md` with:

```markdown
# Project gotchas

- `crm --version` and `crm version` are both public contracts; CI must exercise both.
- GoReleaser publishes CRM as a Homebrew cask in `harperreed/homebrew-tap`, not as a generated binary formula.
- The SQLite list-query builders concatenate only fixed SQL clause strings. Filter values remain parameterized; keep the two G202 annotations narrow and justified.
```

**Step 3: Re-run the false-claim expansion search**

Run:

```bash
rg -n 'mattn/go-sqlite3|adrg/xdg|Install to GOPATH/bin|LICENSE\*|^brews:|format: tar\.gz' README.md CLAUDE.md gotchas.md .goreleaser.yml
```

Expected: no matches.

**Step 4: Commit**

Run `git status`, then:

```bash
git add CLAUDE.md gotchas.md
git commit -m "docs: align contributor guidance with CRM"
```

Do not bypass hooks.

### Task 6: Run full release and repository verification

**Files:**

- Verify only; modify the smallest in-scope file if a check exposes a defect.

**Step 1: Run canonical checks**

Run:

```bash
make check
go vet ./...
go test -race ./...
goreleaser check
```

Expected: every command exits 0 with no warnings or errors.

**Step 2: Build a snapshot release**

Record the current module hashes:

```bash
shasum go.mod go.sum
```

Run:

```bash
goreleaser release --snapshot --clean
```

Expected: exit 0; four macOS/Linux AMD64/ARM64 archives are built and a Homebrew cask artifact is generated without publishing.

Inspect the artifact inventory:

```bash
jq -r '.[] | [.type, .name, .goos, .goarch] | @tsv' dist/artifacts.json
```

Re-run `shasum go.mod go.sum` and confirm both hashes are unchanged after the GoReleaser before-hook.

**Step 3: Verify the real binary contract**

Run:

```bash
make build
./crm version
./crm --version
```

Expected: both commands exit 0 and contain the same version value.

**Step 4: Check tracked state and prose**

Run:

```bash
git diff --check
git status --short --branch
```

Expected: the tracked working tree is clean. Ignored `crm` and `dist/` artifacts may remain.

**Step 5: Run mandatory review gates**

Invoke `fresh-eyes-review` across every changed file. Fix in-scope findings, add or rerun the relevant regression check, and then invoke `verification-before-completion` with the full commands above.

If review produces changes, run `git status`, stage only those exact files, and commit with a concise conventional message. Never bypass hooks.

**Step 6: Record the outcome**

Append a project journal entry with the final test, lint, snapshot, and release results. Update `gotchas.md` only if verification uncovered a durable fact not already captured.
