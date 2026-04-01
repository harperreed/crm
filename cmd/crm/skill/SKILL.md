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
