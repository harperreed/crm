# Project gotchas

- `crm --version` and `crm version` are both public contracts; CI must exercise both.
- CRM still publishes a generated Homebrew formula through `harperreed/homebrew-tap`. Keep the deprecated GoReleaser `brews` field until a signed, notarized cask and a coordinated tap migration are ready; `goreleaser check` exits 2 for this deprecation, but snapshot releases succeed.
- The release archive still includes an unmatched `LICENSE*` glob. Do not add or choose a license as a side effect of release maintenance.
- CRM major version 2 requires `/v2` in the `go.mod` module path, every first-party Go import, and every `go install` path; removing it makes v2 tags invisible to the Go toolchain.
- Go 1.24 and newer can give a checkout build a VCS pseudo-version while leaving `debug.BuildInfo.Main.Sum` empty. CRM reports an embedded module version only when that checksum is nonempty; checkout builds remain `dev`.
- Before the first valid `/v2` tag, `go install` with an exact commit SHA can fail while loading deprecation data from the invalid old `@latest` tag. Prove a release candidate with an isolated temporary bare-repository tag instead; never create or move a project or origin tag for this gate.
- The SQLite list-query builders concatenate only fixed SQL clause strings. Filter values remain parameterized; keep the two G202 annotations narrow and justified.
- History events are immutable and have no expiry or purge path. Deleting a current record leaves its snapshots, including personal data, in history indefinitely.
- The Markdown history mutex protects one process only. Do not run concurrent Markdown writers in separate CRM processes; startup recovery handles interrupted pending writes, not cross-process races.
- SQLite history lists fetch related entity IDs once per returned event, capped at 100 events. Batch that lookup if history browsing becomes a measured bottleneck.
