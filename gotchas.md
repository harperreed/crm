# Project gotchas

- `CLAUDE.md` still names `mattn/go-sqlite3` and `adrg/xdg`; the code uses `modernc.org/sqlite` and handles XDG paths directly. See `docs/audits/AUDIT_REPORT_2026-08-11.md`.
- Release checks are not clean: the Homebrew test calls unsupported `crm --version`, current GoReleaser rejects deprecated fields, and the archive expects missing README and license files.
- `make check` is red because `golangci-lint` reports G202 in both SQLite list-query builders.
