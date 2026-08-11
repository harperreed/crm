<!-- ABOUTME: Defines the approved repair of CRM release checks and current user documentation. -->
<!-- ABOUTME: Covers version compatibility, GoReleaser migration, lint findings, README, and verification. -->

# Release and Documentation Repair Design

Date: 2026-08-11

> **Decision update — 2026-08-11:** Doctor Biz superseded this design's Homebrew cask migration and retained the generated Homebrew formula. See the [execution plan](../../plans/2026-08-11-release-and-documentation-repair.md) for the implemented decision.
>
> `goreleaser check` is expected to exit 2 only for the deprecated `brews` field; `goreleaser release --snapshot --clean` is the passing release validation. The unmatched `LICENSE*` archive glob also remains. Sections below that prescribe a cask migration or a clean `goreleaser check` are historical design records, not the accepted result.

## Goal

Restore clean release and repository checks, add conventional `crm --version` support without removing `crm version`, and make the user-facing documentation match the shipped program.

## Scope

This change will:

- Add `crm --version` as a supported public entry point while preserving the existing `crm version` command.
- Migrate the GoReleaser archive and Homebrew configuration away from fields deprecated by GoReleaser 2.17.
- Keep Homebrew distribution through the existing `harperreed/homebrew-tap` repository, using a cask instead of a generated binary formula.
- Resolve the two G202 lint findings in the SQLite list-query builders without changing query behavior.
- Add a root README with installation, configuration, CLI, storage, MCP, and development guidance.
- Correct stale claims in `CLAUDE.md` and update `gotchas.md` so shared project memory reflects the repaired state.
- Extend CI verification to exercise both supported version entry points.

## Non-goals

- Do not remove or rename `crm version`.
- Do not choose or add a software license. That is a separate legal decision.
- Do not change CRM data models, storage formats, search behavior, MCP schemas, or other CLI commands.
- Do not replace Homebrew distribution with a new release service or custom publishing script.
- Do not rewrite the historical audit report. It records the state of commit `48dcf97` and remains valid as an audit artifact.

## CLI Version Contract

Cobra's root command will receive the existing build-time `version` value. This enables its native `--version` flag while leaving the current version subcommand untouched.

Expected development-build behavior:

```text
$ crm --version
crm version dev

$ crm version
crm version dev
  commit: none
  built:  unknown
```

Release builds will continue to inject version, commit, and build date through the existing linker flags. The short flag reports the version; the subcommand retains the detailed build metadata.

## Release Configuration

The GoReleaser migration will follow the current upstream schema:

- Replace `archives.format: tar.gz` with `archives.formats: [tar.gz]`.
- Replace `brews` with `homebrew_casks`.
- Remove the formula-only `install` and `test` blocks.
- Declare `crm` in the cask's `binaries` list.
- Let the cask use its default `Casks` directory rather than the old `Formula` directory.
- Preserve the tap repository, token, homepage, and description.
- Include `README*` in release archives and remove the unmatched `LICENSE*` pattern.

The README will document cask installation as:

```bash
brew install --cask harperreed/tap/crm
```

The release workflow will still use GoReleaser on `v*` tags. CI will build the binary and run both `./crm version` and `./crm --version`, so the compatibility contract fails before a release if either path breaks.

## Lint Findings

Both G202 findings are false positives. The query builders select SQL clause strings from fixed constants in source code; every filter value remains a positional argument passed separately to `database/sql`.

The smallest honest fix is a narrow `#nosec G202` annotation at each concatenation site with a comment that states this invariant. No query logic or filtering behavior will change. Existing SQLite filter and search tests remain the behavioral guard.

## Documentation

The new README will cover:

- What CRM manages and why it exposes both CLI and MCP interfaces.
- Homebrew and `go install` installation.
- Contact, company, and relationship command examples.
- SQLite defaults, Markdown opt-in, config file location, config keys, and data paths.
- MCP server configuration plus the available tools, resources, and prompts.
- Canonical development and verification commands.

`CLAUDE.md` will remain a compact contributor guide. It will name `modernc.org/sqlite`, describe built-in XDG handling, state the correct data-path fallback, and describe `make install` without assuming `GOBIN` is unset. Its dependency list will match the direct requirements in `go.mod`.

`gotchas.md` will stop describing repaired failures as current. It will retain only durable facts that could surprise later contributors, including the dual version contract, the Homebrew cask migration, and the reason for the narrow G202 annotations.

## Testing and Verification

Implementation will use these red-green checks:

1. Add a CLI test that invokes the root command with `--version`; observe the current unknown-flag failure, then enable Cobra version support and observe the expected output.
2. Run `goreleaser check`; retain the current deprecation failure as the red state, migrate the configuration, and require a clean exit.
3. Run `golangci-lint run ./...`; retain the current two G202 findings as the red state, add the justified annotations, and require a clean exit.

Final verification will run:

```bash
make check
go vet ./...
go test -race ./...
goreleaser check
goreleaser release --snapshot --clean
make build
./crm version
./crm --version
git diff --check
```

The snapshot release must produce archives and a Homebrew cask without publishing. Verification will also confirm that `go mod tidy` in the GoReleaser before-hook leaves `go.mod` and `go.sum` unchanged.

## Success Criteria

- Both version entry points succeed and report the same version value.
- `make check`, race tests, vet, and GoReleaser validation pass without new warnings or errors.
- A snapshot release builds all configured macOS and Linux AMD64/ARM64 artifacts and generates the cask.
- README commands and configuration examples match executable help and source.
- `CLAUDE.md` contains none of the five false claims recorded in the 2026-08-11 audit.
- No CRM behavior outside version reporting changes.
