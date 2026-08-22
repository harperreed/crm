<!-- ABOUTME: Defines the migration to the canonical Go v2 module and install path. -->
<!-- ABOUTME: Covers source imports, regression checks, documentation, and the v2.3.0 release gate. -->

# Go v2 Module Path Migration Design

## Goal

Make the published v2 module installable with the Go toolchain. After release, this command must build the CRM from the public repository:

```bash
go install github.com/harperreed/crm/v2/cmd/crm@v2.3.0
```

The migration changes module identity and first-party import paths only. It does not change CRM runtime behavior, storage formats, CLI commands, MCP contracts, or release artifacts.

Estimated implementation size: 60–100 changed lines across roughly 50 source and current-documentation files, plus one focused regression test.

## Current fault

The repository publishes v2 tags while `go.mod` declares:

```text
module github.com/harperreed/crm
```

Go requires modules released at v2 or later to include the matching `/v2` suffix. The Go command therefore ignores the repository's v2 tags for the unversioned module path. The old install path resolves the v1 line, where `cmd/crm` does not exist.

The published `v2.2.0` tag is immutable. This migration lands in `v2.3.0`; it does not delete, move, or replace an existing tag.

## Decision

Keep the module at the repository root and change its canonical path to:

```text
github.com/harperreed/crm/v2
```

Update every current first-party Go import to use that prefix. The repository contains only the executable, internal packages, and tests; it exposes no reusable public library package. The import change is therefore mechanical and has a bounded compatibility cost.

Do not create a duplicate `v2/` source tree or a nested CLI module. Those layouts add maintenance without serving another supported major line. Do not resume v1 releases as a workaround.

## Source changes

Change the `module` directive in `go.mod` and every current Go source or test import beginning with `github.com/harperreed/crm/`.

Run `go mod tidy` after the path change. It must not add dependencies or change dependency versions. Formatting and import grouping must continue to come from the project's canonical tools.

Historical design, plan, and audit documents remain unchanged when they describe the old path or old failure. They are records of the state at the time. Current guidance in `README.md`, `CLAUDE.md`, and `gotchas.md` must describe the new path.

## Regression protection

Add a focused integration test that reads the repository's `go.mod` and requires the exact module directive `github.com/harperreed/crm/v2`. The normal package build then verifies that all first-party imports agree with that identity.

The test must fail against `v2.2.0` for the missing `/v2` suffix before any production or documentation change is made. It must use real files and the Go test runner, with no mocks.

The canonical `make check`, `go vet ./...`, and `go test -race ./...` gates remain required. A GoReleaser snapshot must build all configured targets, include `README.md`, and leave `go.mod` and `go.sum` unchanged.

## Documentation

Add this supported source-install command to the README:

```bash
go install github.com/harperreed/crm/v2/cmd/crm@latest
```

Keep Homebrew and release archives as the primary installation paths. Update contributor guidance to name the `/v2` module. Replace the obsolete gotcha about broken v2 tags with a durable warning that every first-party import and Go install path must retain `/v2` for the lifetime of major version 2.

Do not claim that the public install works until the `v2.3.0` tag exists and the remote install check passes.

## Release procedure

Implementation runs on `fix/v2-module-path` from `main` and follows TDD. After task review, whole-branch review, fresh-eyes review, and local verification:

1. Push the reviewed branch commit.
2. Install the exact remote commit with `GOBIN` and module caches isolated in a temporary directory:

   ```bash
   go install github.com/harperreed/crm/v2/cmd/crm@<commit-sha>
   ```

3. Run the installed binary's `--version` command.
4. Fast-forward local `main` to the reviewed commit.
5. Create the repository's conventional lightweight `v2.3.0` tag at that commit.
6. Atomically push `main` and only that new tag so current documentation and the valid module release become public together.
7. Monitor the tag-triggered release workflow through completion.
8. Verify the GitHub release assets, remote tag and `main` SHA, Homebrew formula version, and the public `@v2.3.0` Go install path.

If the remote commit install fails, stop before merging or tagging. If the release workflow fails after the immutable tag is pushed, diagnose and repair the workflow without moving the tag.

## Compatibility and risks

- Existing Homebrew installs, downloaded archives, CRM data, and command behavior do not change.
- Anyone importing repository packages must switch to `/v2`. All reusable application packages are under `internal`, so supported external library use does not exist.
- The old `github.com/harperreed/crm/cmd/crm@latest` path remains unsupported. Documentation must show only the versioned path.
- Go module proxies may cache published tags. No existing v2 tag may be deleted or retargeted.
- The current release workflow's documented GoReleaser deprecation warning is not part of this migration.

## Acceptance criteria

- `go.mod` declares `module github.com/harperreed/crm/v2`.
- All current first-party Go imports use `github.com/harperreed/crm/v2/`.
- Historical documents remain historically accurate; current guidance contains no stale unversioned module or install path.
- The regression test fails before the migration and passes afterward.
- Canonical, vet, race, and snapshot-release gates pass with clean output except the accepted `brews` deprecation from `goreleaser check`.
- The reviewed remote commit installs and runs through the `/v2` path before tagging.
- `main`, tag `v2.3.0`, the GitHub release, and the Homebrew formula all point to the reviewed commit.
- A clean external `go install github.com/harperreed/crm/v2/cmd/crm@v2.3.0` succeeds and the installed binary reports version `2.3.0`.
