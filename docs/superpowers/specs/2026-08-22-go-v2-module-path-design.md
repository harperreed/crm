<!-- ABOUTME: Defines the migration to the canonical Go v2 module and install path. -->
<!-- ABOUTME: Covers source imports, regression checks, documentation, and the v2.3.0 release gate. -->

# Go v2 Module Path Migration Design

## Goal

Make the published v2 module installable with the Go toolchain. After release, this command must build the CRM from the public repository:

```bash
go install github.com/harperreed/crm/v2/cmd/crm@v2.3.0
```

The migration changes module identity, first-party import paths, and source-install version reporting. It does not change storage formats, CLI commands, MCP contracts, or release artifacts.

Estimated implementation size: 90–130 changed lines across roughly 50 source and current-documentation files, plus focused module-path, documentation, and version-resolution tests.

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

Keep the linker-injected version used by GoReleaser as the first choice for version output. When that value remains `dev`, use the Go toolchain's embedded module version only when `debug.BuildInfo.Main.Sum` is nonempty, reject empty or `(devel)` versions, and remove the leading `v`. A downloaded version-query install has a nonempty `h1:` checksum. Go 1.24 and newer may embed a VCS pseudo-version in a local checkout build, but its main-module checksum is empty, so local developer builds continue to report `dev`.

Historical design, plan, and audit documents remain unchanged when they describe the old path or old failure. They are records of the state at the time. Current guidance in `README.md`, `CLAUDE.md`, and `gotchas.md` must describe the new path.

## Regression protection

Add a focused integration test that reads the repository's `go.mod` and requires the exact module directive `github.com/harperreed/crm/v2`. The normal package build then verifies that all first-party imports agree with that identity. Add a README regression check for the exact versioned install command, plus unit tests for linker-version precedence, checksummed module-version normalization, and checksum-empty local-build fallback. A real checkout end-to-end test must build the binary and require both `crm --version` and `crm version` to report `dev`.

Each regression test must fail for its intended reason before its production or documentation change is made. Tests must use real values or files and the Go test runner, with no mocks.

The canonical `make check`, `go vet ./...`, and `go test -race ./...` gates remain required. A GoReleaser snapshot must build all configured targets, include `README.md`, and leave `go.mod` and `go.sum` unchanged.

## Documentation

Add this supported source-install command to the README:

```bash
go install github.com/harperreed/crm/v2/cmd/crm@latest
```

Keep Homebrew and release archives as the primary installation paths. Update contributor guidance to name the `/v2` module. Replace the obsolete gotcha about broken v2 tags with a durable warning that every first-party import and Go install path must retain `/v2` for the lifetime of major version 2.

Do not claim that the public install works until the `v2.3.0` tag exists and the public tagged-install check passes.

## Release procedure

Implementation runs on `fix/v2-module-path` from `main` and follows TDD. After task review, whole-branch review, fresh-eyes review, and local verification:

1. Require a clean reviewed branch and record its exact commit.
2. Clone that checkout to a temporary bare repository, create `v2.3.0` only in the bare clone, and redirect the GitHub module URL to it through a temporary Git configuration.
3. With direct module lookup and isolated binary, module, and build caches, install `github.com/harperreed/crm/v2/cmd/crm@v2.3.0` from the temporary candidate and require both version commands to report `2.3.0`.
4. Confirm the real checkout and origin still have no `v2.3.0` tag.
5. Push the reviewed branch commit and verify the remote branch points to the exact reviewed SHA.
6. Fast-forward local `main` to the reviewed commit, create the repository's conventional lightweight `v2.3.0` tag at that commit, and atomically push `main` with only that new tag.
7. Monitor the tag-triggered release workflow through completion.
8. Verify the GitHub release assets, remote tag and `main` SHA, Homebrew formula version, and the public `@v2.3.0` Go install path.

If the isolated candidate install fails, stop before merging or tagging. If the release workflow fails after the immutable tag is pushed, diagnose and repair the workflow without moving the tag.

The isolated gate was validated against clean commit `9fed8a2`: the temporary bare repository alone carried `v2.3.0`, the real version-query install succeeded with isolated caches, and both version commands reported `2.3.0`. The project checkout and origin gained no tag. Run the gate again at the final reviewed SHA before publishing.

## Compatibility and risks

- Existing Homebrew installs, downloaded archives, CRM data, and command behavior do not change.
- Anyone importing repository packages must switch to `/v2`. All reusable application packages are under `internal`, so supported external library use does not exist.
- The old `github.com/harperreed/crm/cmd/crm@latest` path remains unsupported. Documentation must show only the versioned path.
- Before the first valid `/v2` tag exists, `go install` with an exact commit SHA is not a sound pre-publication gate. Go loads deprecation metadata from `@latest`; the observed lookup starts with `loading deprecation for github.com/harperreed/crm/v2` and rejects `v2.2.0` because that tag declares the invalid unsuffixed module path. The isolated candidate-tag install proves the reviewed source without changing project or origin refs.
- Go module proxies may cache published tags. No existing v2 tag may be deleted or retargeted.
- The current release workflow's documented GoReleaser deprecation warning is not part of this migration.

## Acceptance criteria

- `go.mod` declares `module github.com/harperreed/crm/v2`.
- All current first-party Go imports use `github.com/harperreed/crm/v2/`.
- GoReleaser linker values take precedence, tagged `go install` builds use a checksummed embedded module version without the leading `v`, and checksum-empty checkout builds report `dev` through both public version commands.
- Historical documents remain historically accurate; current guidance contains no stale unversioned module or install path.
- The regression test fails before the migration and passes afterward.
- Canonical, vet, race, and snapshot-release gates pass with clean output except the accepted `brews` deprecation from `goreleaser check`.
- An isolated `v2.3.0` candidate tag at the exact reviewed SHA installs through the `/v2` path and both version commands report `2.3.0` before any public tag is created.
- `main`, tag `v2.3.0`, the GitHub release, and the Homebrew formula all point to the reviewed commit.
- A clean external `go install github.com/harperreed/crm/v2/cmd/crm@v2.3.0` succeeds and the installed binary reports version `2.3.0`.
