# Project gotchas

- `crm --version` and `crm version` are both public contracts; CI must exercise both.
- CRM still publishes a generated Homebrew formula through `harperreed/homebrew-tap`. Keep the deprecated GoReleaser `brews` field until a signed, notarized cask and a coordinated tap migration are ready; `goreleaser check` exits 2 for this deprecation, but snapshot releases succeed.
- The release archive still includes an unmatched `LICENSE*` glob. Do not add or choose a license as a side effect of release maintenance.
- `go install github.com/harperreed/crm/cmd/crm@latest` resolves `v1.5.1`, which does not contain `cmd/crm`; the repository's `v2.0.0` tag is not valid for a module path without `/v2`. Use the README's clone-and-`make install` flow until the module tags are repaired.
- The SQLite list-query builders concatenate only fixed SQL clause strings. Filter values remain parameterized; keep the two G202 annotations narrow and justified.
