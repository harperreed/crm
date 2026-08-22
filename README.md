<!-- ABOUTME: User guide for installing, configuring, and operating CRM. -->
<!-- ABOUTME: Documents the CLI, storage backends, MCP server, and development checks. -->

# CRM

CRM is a small local-first relationship manager for contacts, companies, and the links between them. Humans use its command-line interface; agents use the same data through its MCP server.

## Install

With Homebrew:

```bash
brew install harperreed/tap/crm
```

With Go 1.25.5 or newer:

```bash
git clone https://github.com/harperreed/crm.git
cd crm
make install
```

Source installs do not use GoReleaser's linker flags, so `crm --version` reports `dev`.

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

Inspect an entity's timeline, then show one event's full before and after snapshots:

```bash
crm history <entity-id-or-prefix>
crm history show <event-id-or-prefix>
```

`crm history` lists up to 20 events by default; pass `--limit` to change the count. Contact and company timelines include link and unlink events involving that entity. Each event records whether the change came from the CLI (`cli`) or MCP (`mcp`).

History starts with changes made after upgrading; CRM does not create baseline events for existing records. Events are immutable and kept indefinitely. Deleting a contact, company, or relationship removes the current record but keeps its historical snapshots, including any personal data they contain.

Run `crm help` to browse commands. Add `--help` to the exact command you plan to run, such as `crm contact add --help`, for its full flags and usage.

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

For project-scoped Claude Code setup, save this configuration as `.mcp.json` in the project root:

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

The server exposes 14 tools for contact and company CRUD, linking and unlinking, and read-only history. Use `list_history` for an entity timeline and `get_history_event` for a full event. It also exposes contact and company resource templates and three prompts for contact creation, relationship mapping, and cross-entity search.

Install the bundled Claude Code skill with:

Warning: this command overwrites an existing `~/.claude/skills/crm/SKILL.md` without a backup.

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
```

`goreleaser check` currently exits non-zero only because the accepted Homebrew formula configuration uses the deprecated `brews` field.

The project uses Go 1.25.5. SQLite uses the pure-Go `modernc.org/sqlite` driver, so release builds do not require CGO.
