# CRM Skill

A lightweight CRM for managing contacts, companies, and relationships. Access via MCP tools.

## Available Tools

### Contacts
- `mcp__crm__add_contact` — Add a contact. Required: `name`. Optional: `email`, `phone`, `fields` (object), `tags` (string array).
- `mcp__crm__list_contacts` — List contacts. Optional: `tag`, `search`, `limit` (default 20).
- `mcp__crm__get_contact` — Get a contact by full UUID or prefix (min 6 chars). Required: `id`.
- `mcp__crm__update_contact` — Update a contact. Required: `id`. Optional: `name`, `email`, `phone`, `fields` (merged), `tags` (replaced).
- `mcp__crm__delete_contact` — Delete a contact. Required: `id`.

### Companies
- `mcp__crm__add_company` — Add a company. Required: `name`. Optional: `domain`, `fields` (object), `tags` (string array).
- `mcp__crm__list_companies` — List companies. Optional: `tag`, `search`, `limit` (default 20).
- `mcp__crm__get_company` — Get a company by full UUID or prefix (min 6 chars). Required: `id`.
- `mcp__crm__update_company` — Update a company. Required: `id`. Optional: `name`, `domain`, `fields` (merged), `tags` (replaced).
- `mcp__crm__delete_company` — Delete a company. Required: `id`.

### Relationships
- `mcp__crm__link` — Create a relationship. Required: `source_id`, `target_id`, `type`. Optional: `context`.
- `mcp__crm__unlink` — Delete a relationship. Required: `id`.

### History
- `mcp__crm__list_history` — List an entity's history newest first. Required: `entity_id` (full UUID or prefix of at least 6 characters). Optional: `limit` (default 20, maximum 100). Contact and company timelines include their link and unlink events.
- `mcp__crm__get_history_event` — Get one event with full before and after snapshots. Required: `event_id` (full UUID or prefix of at least 6 characters).

History records whether each mutation came from `cli` or `mcp`. It starts with changes made after CRM was upgraded to a history-capable version, so an empty or short timeline does not describe changes made before then. Deleted records remain visible in immutable history indefinitely, including any personal data in their snapshots.

## Usage Patterns

### Add a contact and link to a company
```
1. mcp__crm__add_contact(name: "Jane Doe", email: "jane@acme.com", tags: ["engineering"])
2. mcp__crm__add_company(name: "Acme Corp", domain: "acme.com")
3. mcp__crm__link(source_id: "<contact_id>", target_id: "<company_id>", type: "works_at")
```

### Search and retrieve
```
1. mcp__crm__list_contacts(search: "jane")
2. mcp__crm__get_contact(id: "<uuid_or_prefix>")
```

### Update and clean up
```
1. mcp__crm__update_contact(id: "<id>", email: "jane.new@acme.com")
2. mcp__crm__unlink(id: "<relationship_id>")
3. mcp__crm__delete_contact(id: "<id>")
```

### Inspect changes
```
1. mcp__crm__list_history(entity_id: "<entity_id_or_prefix>")
2. mcp__crm__get_history_event(event_id: "<event_id_or_prefix>")
```

## MCP Server Configuration

Add to your Claude Code MCP config:

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
