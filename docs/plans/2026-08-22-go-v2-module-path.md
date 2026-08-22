<!-- ABOUTME: Gives the TDD steps for migrating CRM to its canonical Go v2 module path. -->
<!-- ABOUTME: Covers source-install version reporting, docs, review, verification, and release v2.3.0. -->

# Go v2 Module Path Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Publish CRM v2.3.0 as a valid Go v2 module that installs from the public `/v2` path and reports its installed version.

**Architecture:** Keep the module at the repository root, add `/v2` to its identity and every first-party import, and guard that identity with a repository integration test. Preserve GoReleaser's linker-injected version, but fall back only to checksummed Go module metadata used by a version-query install; keep checksum-empty checkout builds on `dev`.

**Tech Stack:** Go 1.25.5, Cobra, `runtime/debug`, Go modules, GoReleaser, GitHub Actions, Homebrew.

**Approved design:** `docs/superpowers/specs/2026-08-22-go-v2-module-path-design.md`

---

### Task 1: Report the embedded module version for source installs

**Files:**
- Create: `cmd/crm/build_version.go`
- Create: `cmd/crm/build_version_test.go`
- Modify: `cmd/crm/root.go:17-24`
- Modify: `cmd/crm/root_test.go:62-67`
- Modify: `cmd/crm/version.go:12-21`
- Create: `test/cli_version_e2e_test.go`

**Status:** Accepted. The first implementation landed in `0e2b7f0`; public-command and write-error coverage landed in `490133b` and `7d52894`. Review correction `9fed8a2` requires a nonempty `debug.BuildInfo.Main.Sum` before using the embedded module version, adds the checksum-empty pseudo-version unit case, and adds a real checkout end-to-end test for both public version commands.

**Step 1: Write the failing unit test**

Create `cmd/crm/build_version_test.go`:

```go
// ABOUTME: Verifies CRM version selection for releases, Go installs, and local builds.
// ABOUTME: Keeps linker values authoritative while normalizing embedded module versions.

package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionForBuild(t *testing.T) {
	tests := []struct {
		name          string
		linkerVersion string
		buildInfo     *debug.BuildInfo
		buildInfoOK   bool
		want          string
	}{
		{
			name:          "linker version wins",
			linkerVersion: "2.3.0",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}},
			buildInfoOK:   true,
			want:          "2.3.0",
		},
		{
			name:          "Go install version loses v prefix",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v2.3.0", Sum: "h1:installed"}},
			buildInfoOK:   true,
			want:          "2.3.0",
		},
		{
			name:          "checkout pseudo-version stays dev",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v2.2.1-0.20260822183510-ef91f365a575"}},
			buildInfoOK:   true,
			want:          "dev",
		},
		{
			name:          "local build stays dev",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			buildInfoOK:   true,
			want:          "dev",
		},
		{
			name:          "empty module version stays dev",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{},
			buildInfoOK:   true,
			want:          "dev",
		},
		{
			name:          "missing build info stays dev",
			linkerVersion: "dev",
			buildInfoOK:   false,
			want:          "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionForBuild(tt.linkerVersion, tt.buildInfo, tt.buildInfoOK); got != tt.want {
				t.Fatalf("versionForBuild() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run the test to verify RED**

Run:

```bash
env -u GOROOT go test ./cmd/crm -run '^TestVersionForBuild$' -count=1 -v
```

Expected: compilation fails because `versionForBuild` does not exist. A failure caused by the host toolchain is not the required RED result.

**Step 3: Implement the smallest version resolver**

Create `cmd/crm/build_version.go`:

```go
// ABOUTME: Selects the user-visible CRM version from linker or Go build metadata.
// ABOUTME: Lets tagged Go installs report their module version while local builds stay dev.

package main

import (
	"runtime/debug"
	"strings"
)

func displayVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	return versionForBuild(version, buildInfo, ok)
}

func versionForBuild(linkerVersion string, buildInfo *debug.BuildInfo, buildInfoOK bool) string {
	if linkerVersion != "dev" {
		return linkerVersion
	}
	if !buildInfoOK || buildInfo == nil || buildInfo.Main.Version == "" || buildInfo.Main.Version == "(devel)" {
		return linkerVersion
	}
	if buildInfo.Main.Sum == "" {
		return linkerVersion
	}
	return strings.TrimPrefix(buildInfo.Main.Version, "v")
}
```

Change `cmd/crm/root.go` so Cobra receives the resolved version:

```go
	Version: displayVersion(),
```

Change `cmd/crm/root_test.go` to assert against the command's configured version:

```go
	want := fmt.Sprintf("crm version %s\n", rootCmd.Version)
```

Change the first output line in `cmd/crm/version.go`:

```go
		fmt.Printf("crm version %s\n", displayVersion())
```

Do not change the `version = "dev"` declaration in `cmd/crm/main.go`; GoReleaser's `-X main.version=...` linker flag depends on that string variable.

**Step 4: Run focused and package tests to verify GREEN**

Run:

```bash
env -u GOROOT gofmt -w cmd/crm/build_version.go cmd/crm/build_version_test.go cmd/crm/root.go cmd/crm/root_test.go cmd/crm/version.go
env -u GOROOT go test ./cmd/crm -run '^(TestVersionForBuild|TestRootVersionFlag)$' -count=1 -v
env -u GOROOT go test ./cmd/crm -count=1
env -u GOROOT go test ./test -run '^TestCLICheckoutBuildReportsDev$' -count=1 -v
```

Expected: all commands exit 0. The unit test proves linker precedence, `v` removal for a checksummed module, checksum-empty checkout fallback, and missing-metadata fallback without replacing `debug.ReadBuildInfo` or mocking application behavior. The end-to-end test builds the real checkout and proves both `crm --version` and `crm version` report `dev`.

**Step 5: Run fresh-eyes review and commit**

Use `@fresh-eyes-review` on the six touched files. Pay special attention to linker compatibility, nil and checksum-empty build metadata, and agreement between `crm --version` and `crm version`. Fix each finding and rerun Step 4.

Then commit only this task:

```bash
git status --short
git add cmd/crm/build_version.go cmd/crm/build_version_test.go cmd/crm/root.go cmd/crm/root_test.go cmd/crm/version.go test/cli_version_e2e_test.go
git diff --cached --check
git commit -m "fix: report Go install module version"
```

---

### Task 2: Migrate the module identity and first-party imports

**Status:** Accepted at `f1bf321` with the module-path regression test and every current first-party Go reference migrated to `/v2`.

**Files:**
- Create: `test/module_path_test.go`
- Modify: `go.mod:1`
- Modify: `cmd/crm/companies.go`
- Modify: `cmd/crm/contacts.go`
- Modify: `cmd/crm/history.go`
- Modify: `cmd/crm/history_test.go`
- Modify: `cmd/crm/mcp.go`
- Modify: `cmd/crm/relationships.go`
- Modify: `cmd/crm/root.go`
- Modify: `cmd/crm/root_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/history/event_test.go`
- Modify: `internal/history/snapshot.go`
- Modify: `internal/history/snapshot_test.go`
- Modify: `internal/mcp/history_test.go`
- Modify: `internal/mcp/nil_collections_test.go`
- Modify: `internal/mcp/noop_update_test.go`
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/mcp/tools.go`
- Modify: `internal/mcp/update_response_test.go`
- Modify: `internal/storage/collections.go`
- Modify: `internal/storage/interface.go`
- Modify: `internal/storage/markdown.go`
- Modify: `internal/storage/markdown_companies.go`
- Modify: `internal/storage/markdown_contacts.go`
- Modify: `internal/storage/markdown_history.go`
- Modify: `internal/storage/markdown_history_test.go`
- Modify: `internal/storage/markdown_relationships.go`
- Modify: `internal/storage/markdown_search.go`
- Modify: `internal/storage/markdown_test.go`
- Modify: `internal/storage/sqlite.go`
- Modify: `internal/storage/sqlite_companies.go`
- Modify: `internal/storage/sqlite_companies_test.go`
- Modify: `internal/storage/sqlite_contacts.go`
- Modify: `internal/storage/sqlite_contacts_test.go`
- Modify: `internal/storage/sqlite_history.go`
- Modify: `internal/storage/sqlite_history_test.go`
- Modify: `internal/storage/sqlite_relationships.go`
- Modify: `internal/storage/sqlite_relationships_test.go`
- Modify: `internal/storage/sqlite_search.go`
- Modify: `internal/storage/sqlite_search_test.go`
- Modify: `internal/storage/sqlite_test.go`
- Modify: `test/cli_history_e2e_test.go`
- Modify: `test/history_integration_test.go`
- Modify: `test/integration_test.go`

**Step 1: Write the failing repository test**

Create `test/module_path_test.go`:

```go
// ABOUTME: Guards the repository's canonical major-version module identity.
// ABOUTME: Prevents future v2 tags from becoming invisible to the Go toolchain.

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModulePathMatchesMajorVersion(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	const want = "module github.com/harperreed/crm/v2"
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "module ") {
			continue
		}
		if line != want {
			t.Fatalf("module directive = %q, want %q", line, want)
		}
		return
	}
	t.Fatal("go.mod has no module directive")
}
```

**Step 2: Run the test to verify RED**

Run:

```bash
env -u GOROOT go test ./test -run '^TestModulePathMatchesMajorVersion$' -count=1 -v
```

Expected: `FAIL` with `module directive = "module github.com/harperreed/crm", want "module github.com/harperreed/crm/v2"`.

**Step 3: Change the module and imports**

Change the first line of `go.mod` to:

```text
module github.com/harperreed/crm/v2
```

In only the Go files listed above, replace each import prefix:

```text
github.com/harperreed/crm/
```

with:

```text
github.com/harperreed/crm/v2/
```

This is a bounded mechanical rewrite. Do not alter historical Markdown files, external dependency paths, storage formats, or user data.

Then collect the bounded import set and normalize only those files:

```bash
import_files=("${(@f)$(rg -l 'github\.com/harperreed/crm/v2/' cmd internal test --glob '*.go' | sort)}")
env -u GOROOT gofmt -w test/module_path_test.go "${import_files[@]}"
env -u GOROOT go mod tidy
```

Expected: `go.sum` and dependency versions do not change. The only `go.mod` change is the module directive.

**Step 4: Verify GREEN and search every reference class**

Run:

```bash
env -u GOROOT go test ./test -run '^TestModulePathMatchesMajorVersion$' -count=1 -v
env -u GOROOT go test ./... -count=1
rg --pcre2 -n '"github\.com/harperreed/crm/(?!v2/)' --glob '*.go'
rg -n '^module github\.com/harperreed/crm/v2$' go.mod
git diff -- go.sum
```

Expected: both tests exit 0; the negative import search and `go.sum` diff print nothing; the module search prints `go.mod:1`. Also inspect direct imports, tests, and any string literals separately:

```bash
rg -n 'github\.com/harperreed/crm/' cmd internal test --glob '*.go'
rg -n 'github\.com/harperreed/crm' --glob '*.go' --glob '*_test.go'
```

Expected: every current first-party Go reference starts with `github.com/harperreed/crm/v2/`.

**Step 5: Run fresh-eyes review and commit**

Use `@fresh-eyes-review` on `go.mod`, `test/module_path_test.go`, and the complete mechanical diff. Confirm no import gained two `/v2` segments and no historical document changed. Fix findings and rerun Step 4.

Then commit the exact migration set:

```bash
git status --short
import_files=("${(@f)$(rg -l 'github\.com/harperreed/crm/v2/' cmd internal test --glob '*.go' | sort)}")
git add go.mod test/module_path_test.go "${import_files[@]}"
git diff --cached --check
git commit -m "fix: use canonical Go v2 module path"
```

Before committing, inspect `git diff --cached --name-only` and unstage any file not listed in this task. Never use `git add -A`.

---

### Task 3: Document the supported Go install path

**Status:** Accepted at `ef91f36`. The claims and regression test were re-reviewed after `9fed8a2` corrected checkout detection; the README's tagged-install and local-`dev` statements remain accurate.

**Files:**
- Modify: `test/module_path_test.go`
- Modify: `README.md:8-28`
- Modify: `CLAUDE.md:53-60`
- Modify: `gotchas.md:6`

**Step 1: Write the failing documentation regression test**

Add this test to `test/module_path_test.go` and add no new imports:

```go
func TestReadmeUsesVersionedGoInstallPath(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	readme := string(contents)
	const want = "go install github.com/harperreed/crm/v2/cmd/crm@latest"
	if !strings.Contains(readme, want) {
		t.Fatalf("README.md does not contain %q", want)
	}
	const stale = "go install github.com/harperreed/crm/cmd/crm@"
	if strings.Contains(readme, stale) {
		t.Fatalf("README.md contains stale unversioned install path %q", stale)
	}
}
```

**Step 2: Run the test to verify RED**

Run:

```bash
env -u GOROOT go test ./test -run '^TestReadmeUsesVersionedGoInstallPath$' -count=1 -v
```

Expected: `FAIL` because README does not yet contain the `/v2` install command.

**Step 3: Update only current guidance**

Replace README's clone-only Go install block with:

````markdown
With Go 1.25.5 or newer:

```bash
go install github.com/harperreed/crm/v2/cmd/crm@latest
```

For a development checkout:

```bash
git clone https://github.com/harperreed/crm.git
cd crm
make install
```

Tagged `go install` builds and GoReleaser builds report their release version. Local checkout builds report `dev`.
````

In `CLAUDE.md`, replace the module convention with:

```markdown
- The module path is `github.com/harperreed/crm/v2`; the binary name is `crm`.
```

In `gotchas.md`, replace the obsolete broken-tag entry with:

```markdown
- CRM major version 2 requires `/v2` in the `go.mod` module path, every first-party Go import, and every `go install` path. Removing it makes v2 tags invisible to the Go toolchain.
```

Leave `docs/audits/`, prior plans, and prior design records unchanged; their old paths describe earlier repository state.

**Step 4: Verify GREEN and documentation scope**

Run:

```bash
env -u GOROOT go test ./test -run '^(TestModulePathMatchesMajorVersion|TestReadmeUsesVersionedGoInstallPath)$' -count=1 -v
rg -n 'github\.com/harperreed/crm' README.md CLAUDE.md gotchas.md
rg -n 'go install github\.com/harperreed/crm/cmd/crm@' README.md CLAUDE.md gotchas.md
git diff --name-only
git diff --check
```

Expected: tests pass; every current module or install reference uses `/v2`; the stale-path search prints nothing; only this task's four files are newly changed.

**Step 5: Run fresh-eyes review and commit**

Use `@fresh-eyes-review` on the four files. Check that source installs no longer claim all builds report `dev`, that the old path is absent from current guidance, and that historical records remain untouched. Fix findings and rerun Step 4.

Then commit:

```bash
git status --short
git add README.md CLAUDE.md gotchas.md test/module_path_test.go
git diff --cached --check
git commit -m "docs: document canonical Go install path"
```

---

### Task 4: Run whole-branch review and local release gates

**Files:**
- Review: every file changed from `main...fix/v2-module-path`
- Verify: `go.mod`, `go.sum`, `.goreleaser.yml`, generated `dist/` archives

**Step 1: Review the complete branch**

Use `@requesting-code-review` against the merge base with `main`. The review must cover:

- module-major correctness and all current import paths;
- both public version commands;
- linker flag compatibility in `.goreleaser.yml`;
- nil, empty, checksum-empty pseudo-version, `(devel)`, checksummed tagged, and linker-injected version cases;
- historical-document preservation;
- release and Homebrew compatibility.

Fix every critical or important finding with RED/GREEN coverage. Record any rejected finding with file-and-line evidence.

**Step 2: Run the mandatory fresh-eyes review**

Use `@fresh-eyes-review` on the whole branch. Fix findings, rerun affected focused tests, and do not commit until the review is clean.

If review causes changes, stage only their exact files and commit them with a concise conventional commit before continuing.

**Step 3: Run the canonical cold check**

The inherited `GOROOT` is inconsistent. Use the verified Go 1.27 toolchain and the locally rebuilt Go 1.27 `golangci-lint` with fresh caches:

```bash
set -euo pipefail
test -x /tmp/crm-history-lint.jHjlhN/golangci-lint
lint_cache=$(mktemp -d)
go_cache=$(mktemp -d)
test -n "$lint_cache"
test -n "$go_cache"
env -u GOROOT \
  PATH="/tmp/crm-history-lint.jHjlhN:/Users/harper/.cargo/bin:/Users/harper/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin" \
  GOLANGCI_LINT_CACHE="$lint_cache" \
  GOCACHE="$go_cache" \
  make check
```

Expected: formatting, import formatting, lint, and every Go test pass with no warning or error. If the temporary linter no longer exists, rebuild a Go 1.27-compatible linter and record its verified path; do not silently fall back to an incompatible binary.

**Step 4: Run vet, race, and real local binaries**

```bash
set -euo pipefail
go_cache=$(mktemp -d)
test -n "$go_cache"
env -u GOROOT GOCACHE="$go_cache" go vet ./...
env -u GOROOT GOCACHE="$go_cache" go test -race ./... -count=1
build_dir=$(mktemp -d)
test -n "$build_dir"
env -u GOROOT GOCACHE="$go_cache" go build -o "$build_dir/crm" ./cmd/crm
checkout_short_version=$("$build_dir/crm" --version)
checkout_long_version=$("$build_dir/crm" version)
checkout_long_first_line=${checkout_long_version%%$'\n'*}
test "$checkout_short_version" = "crm version dev"
test "$checkout_long_first_line" = "crm version dev"
```

Expected: vet and race exit 0; both binary commands exit 0 and report exactly `crm version dev` on their first line. Neither version command may initialize CRM storage.

**Step 5: Validate release configuration and snapshot artifacts**

First run:

```bash
goreleaser check
```

Expected: exit 2 only for the accepted deprecated `brews` property. Any other warning or error fails the gate.

Then run:

```bash
set -euo pipefail
go_cache=$(mktemp -d)
test -n "$go_cache"
env -u GOROOT GOCACHE="$go_cache" goreleaser release --snapshot --clean
archives=(dist/crm_*_*.tar.gz(N))
checksum_files=(dist/checksums.txt(N))
archive_count=${#archives[@]}
checksum_count=${#checksum_files[@]}
test "$archive_count" -eq 4
test "$checksum_count" -eq 1
for archive in "${archives[@]}"; do
  archive_listing=$(tar -tzf "$archive")
  archive_entries=("${(@f)archive_listing}")
  has_readme=false
  for archive_entry in "${archive_entries[@]}"; do
    if [[ "$archive_entry" = README.md || "$archive_entry" = */README.md ]]; then
      has_readme=true
      break
    fi
  done
  test "$has_readme" = true
done
git diff --exit-code HEAD -- go.mod go.sum
test -z "$(git status --porcelain)"
```

Expected: snapshot exits 0; exactly four archives exist for Darwin/Linux on amd64/arm64; checksums exist; every archive contains README; `go.mod` and `go.sum` remain unchanged. Remove generated `dist/` only if Git reports it as ignored output; do not delete tracked files.

**Step 6: Commit review fixes, if any**

Run fresh-eyes again on any review fix, then stage exact paths and make one concise conventional commit. Finish with:

```bash
set -euo pipefail
git status --short --branch
git log --oneline main..HEAD
test -z "$(git status --porcelain)"
```

Expected: a clean `fix/v2-module-path` worktree with only reviewed commits above `main`.

---

### Task 5: Prove the candidate module before public tagging

**Files:**
- Temporary bare repository and install root: created beneath `candidate_root=$(mktemp -d)`
- Remote branch: `origin/fix/v2-module-path`

This procedure succeeded against clean commit `9fed8a2` without creating a project or origin tag. Run it again at the final reviewed HEAD because this documentation correction changes the release SHA.

**Step 1: Prove and push one fixed reviewed SHA**

```bash
set -euo pipefail
test "$(git branch --show-current)" = fix/v2-module-path
test -z "$(git status --porcelain)"
test "$(git remote get-url --push origin)" = git@github.com:harperreed/crm.git
git fetch origin --prune --tags
if git show-ref --verify --quiet refs/tags/v2.3.0; then exit 1; fi
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
test -z "$remote_tag_ref"
release_sha=$(git rev-parse HEAD)
candidate_root=$(mktemp -d)
test -n "$candidate_root"
git clone --bare . "$candidate_root/crm.git"
git --git-dir="$candidate_root/crm.git" tag v2.3.0 "$release_sha"
candidate_tag_sha=$(git --git-dir="$candidate_root/crm.git" rev-parse v2.3.0^{commit})
test "$candidate_tag_sha" = "$release_sha"
git config --file "$candidate_root/gitconfig" url."file://$candidate_root/crm.git".insteadOf https://github.com/harperreed/crm
env -u GOROOT \
  GIT_CONFIG_GLOBAL="$candidate_root/gitconfig" \
  GIT_CONFIG_NOSYSTEM=1 \
  GIT_ALLOW_PROTOCOL=file:https \
  GOPROXY=direct \
  GONOSUMDB=github.com/harperreed/crm/v2 \
  GOBIN="$candidate_root/bin" \
  GOMODCACHE="$candidate_root/mod" \
  GOCACHE="$candidate_root/cache" \
  go install github.com/harperreed/crm/v2/cmd/crm@v2.3.0
candidate_short_version=$("$candidate_root/bin/crm" --version)
candidate_long_version=$("$candidate_root/bin/crm" version)
candidate_long_first_line=${candidate_long_version%%$'\n'*}
test "$candidate_short_version" = "crm version 2.3.0"
test "$candidate_long_first_line" = "crm version 2.3.0"
test "$(git rev-parse HEAD)" = "$release_sha"
if git show-ref --verify --quiet refs/tags/v2.3.0; then exit 1; fi
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
test -z "$remote_tag_ref"
git push origin "$release_sha:refs/heads/fix/v2-module-path"
remote_feature_ref=$(git ls-remote --heads origin refs/heads/fix/v2-module-path)
remote_feature_sha=${remote_feature_ref%%[[:space:]]*}
test "$remote_feature_sha" = "$release_sha"
test -z "$(git status --porcelain)"
```

Expected: the clean feature branch, temporary candidate tag, installed binary, and authoritative remote feature ref all use one `$release_sha` assigned once. Both version commands report `2.3.0`, and neither the project repository nor origin gains `v2.3.0`.

This replaces the direct exact-SHA version query. Before a valid `/v2` tag exists, that lookup loads deprecation metadata from the old `v2.2.0` `@latest` tag and fails because its `go.mod` declares the invalid unsuffixed module path. Do not create a candidate tag in the project repository or on origin.

---

### Task 6: Merge, tag, publish, and verify v2.3.0

**Files:**
- Local branch: `main`
- Remote ref: `origin/main`
- New immutable tag: `v2.3.0`
- GitHub release and Homebrew formula generated by the existing release workflow

**Step 1: Bind, merge, tag, and publish the authoritative reviewed SHA**

```bash
set -euo pipefail
test "$(git branch --show-current)" = fix/v2-module-path
test -z "$(git status --porcelain)"
test "$(git remote get-url --push origin)" = git@github.com:harperreed/crm.git
git fetch origin --prune --tags
if git show-ref --verify --quiet refs/tags/v2.3.0; then exit 1; fi
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
test -z "$remote_tag_ref"
remote_feature_ref=$(git ls-remote --heads origin refs/heads/fix/v2-module-path)
release_sha=${remote_feature_ref%%[[:space:]]*}
test -n "$release_sha"
test "$(git rev-parse refs/heads/fix/v2-module-path)" = "$release_sha"
remote_main_ref=$(git ls-remote --heads origin refs/heads/main)
remote_main_sha=${remote_main_ref%%[[:space:]]*}
test -n "$remote_main_sha"
git merge-base --is-ancestor "$remote_main_sha" "$release_sha"
git switch main
test "$(git branch --show-current)" = main
test -z "$(git status --porcelain)"
git merge --ff-only "$release_sha"
test "$(git rev-parse HEAD)" = "$release_sha"
git tag v2.3.0 "$release_sha"
test "$(git cat-file -t v2.3.0)" = commit
test "$(git rev-parse v2.3.0^{commit})" = "$release_sha"
git push --atomic origin "$release_sha:refs/heads/main" refs/tags/v2.3.0:refs/tags/v2.3.0
remote_main_ref=$(git ls-remote --heads origin refs/heads/main)
remote_main_sha=${remote_main_ref%%[[:space:]]*}
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
remote_tag_sha=${remote_tag_ref%%[[:space:]]*}
test "$remote_main_sha" = "$release_sha"
test "$remote_tag_sha" = "$release_sha"
```

Expected: the authoritative origin feature SHA equals the local reviewed branch, origin `main` is its ancestor, local `main` fast-forwards to that fixed SHA, and the lightweight tag points to it. The atomic explicit-SHA push publishes `main` and `v2.3.0` together, and authoritative remote queries confirm both refs. Never use `--force`.

**Step 2: Monitor the tag-triggered release**

```bash
set -euo pipefail
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
release_sha=${remote_tag_ref%%[[:space:]]*}
test -n "$release_sha"
release_run_id=
for attempt in {1..10}; do
  release_run_id=$(gh run list --workflow Release --commit "$release_sha" --limit 1 --json databaseId --jq '.[0].databaseId')
  if [[ -n "$release_run_id" ]]; then break; fi
  sleep 5
done
test -n "$release_run_id"
release_head_sha=$(gh run view "$release_run_id" --json headSha --jq .headSha)
test "$release_head_sha" = "$release_sha"
gh run view "$release_run_id" --json databaseId,status,conclusion,headSha,url
gh run watch "$release_run_id" --exit-status
release_conclusion=$(gh run view "$release_run_id" --json conclusion --jq .conclusion)
test "$release_conclusion" = success
release_assets=$(gh release view v2.3.0 --json assets --jq '.assets[].name')
release_asset_names=("${(@f)release_assets}")
release_asset_count=${#release_asset_names[@]}
test "$release_asset_count" -eq 5
release_asset_lines=$'\n'"$release_assets"$'\n'
for expected_asset in \
  crm_2.3.0_darwin_amd64.tar.gz \
  crm_2.3.0_darwin_arm64.tar.gz \
  crm_2.3.0_linux_amd64.tar.gz \
  crm_2.3.0_linux_arm64.tar.gz \
  checksums.txt; do
  [[ "$release_asset_lines" = *$'\n'"$expected_asset"$'\n'* ]]
done
```

Expected: the workflow concludes `success`; the release has four platform archives plus checksums and targets `v2.3.0`.

If the workflow fails, collect its logs with a self-contained lookup:

```bash
set -euo pipefail
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
release_sha=${remote_tag_ref%%[[:space:]]*}
test -n "$release_sha"
release_run_id=$(gh run list --workflow Release --commit "$release_sha" --limit 1 --json databaseId --jq '.[0].databaseId')
test -n "$release_run_id"
gh run view "$release_run_id" --log-failed
```

Then apply `@systematic-debugging` and repair the workflow without moving `v2.3.0`.

**Step 3: Verify the Homebrew formula**

```bash
set -euo pipefail
formula=$(gh api repos/harperreed/homebrew-tap/contents/Formula/crm.rb --jq .content | base64 --decode)
[[ "$formula" = *"https://github.com/harperreed/crm/releases/download/v2.3.0/"* ]]
[[ "$formula" = *"crm_2.3.0_"* ]]
for old_minor in 0 1 2; do
  if [[ "$formula" = *"/releases/download/v2.${old_minor}."* ]]; then exit 1; fi
done
```

Expected: the published formula contains a v2.3.0 release URL and artifact name and contains no v2.0, v2.1, or v2.2 release URL. The formula may omit an explicit `version` line when Homebrew derives it from the URL.

**Step 4: Verify the public tagged Go install**

```bash
set -euo pipefail
public_install_root=$(mktemp -d)
test -n "$public_install_root"
env -u GOROOT \
  GOPROXY=https://proxy.golang.org,direct \
  GOBIN="$public_install_root/bin" \
  GOMODCACHE="$public_install_root/mod" \
  GOCACHE="$public_install_root/cache" \
  go install github.com/harperreed/crm/v2/cmd/crm@v2.3.0
public_short_version=$("$public_install_root/bin/crm" --version)
public_long_version=$("$public_install_root/bin/crm" version)
public_long_first_line=${public_long_version%%$'\n'*}
test "$public_short_version" = "crm version 2.3.0"
test "$public_long_first_line" = "crm version 2.3.0"
```

Expected: install succeeds from a clean module cache, `--version` reports exactly `crm version 2.3.0`, and the detailed command's first line is the same. Retry only for observed proxy propagation; do not weaken the check to `GOPROXY=direct` as the final public proof.

**Step 5: Record final evidence**

```bash
set -euo pipefail
test "$(git branch --show-current)" = main
test -z "$(git status --porcelain)"
test "$(git remote get-url --push origin)" = git@github.com:harperreed/crm.git
git fetch origin --prune --tags
remote_tag_ref=$(git ls-remote --tags --refs origin refs/tags/v2.3.0)
release_sha=${remote_tag_ref%%[[:space:]]*}
test -n "$release_sha"
test "$(git rev-parse v2.3.0^{commit})" = "$release_sha"
test "$(git rev-parse main)" = "$release_sha"
remote_main_ref=$(git ls-remote --heads origin refs/heads/main)
remote_main_sha=${remote_main_ref%%[[:space:]]*}
test "$remote_main_sha" = "$release_sha"
release_assets=$(gh release view v2.3.0 --json assets --jq '.assets[].name')
release_asset_names=("${(@f)release_assets}")
release_asset_count=${#release_asset_names[@]}
test "$release_asset_count" -eq 5
release_url=$(gh release view v2.3.0 --json url --jq .url)
test -n "$release_url"
```

Expected: the worktree is clean; all three Git SHAs match `$release_sha`; the release URL and five expected release files are present. Report the canonical, vet, race, snapshot, isolated candidate-tag install, workflow, Homebrew, and public tagged-install results without claiming more than the commands proved.
