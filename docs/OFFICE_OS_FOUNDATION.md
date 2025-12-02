# Office OS Foundation

## Overview

The Office OS Foundation is a flexible, schema-less database layer built on two core primitives: **Objects** and **Relationships**. This approach replaces the rigid, table-specific schema with a universal data model that can represent any domain.

## Core Concepts

### Objects

Objects are the fundamental entities in Office OS. Each object has:

- **ID**: Unique identifier (auto-generated UUID)
- **Type**: Category of the object (e.g., "Person", "Company", "Task")
- **Name**: Human-readable name
- **Metadata**: Flexible JSON object for domain-specific data
- **Timestamps**: CreatedAt and UpdatedAt for audit trail

```go
type Object struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Name      string                 `json:"name"`
    Metadata  map[string]interface{} `json:"metadata"`
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
}
```

### Relationships

Relationships connect objects together, forming a graph structure. Each relationship has:

- **ID**: Unique identifier
- **SourceID**: The object this relationship originates from
- **TargetID**: The object this relationship points to
- **Type**: The nature of the relationship (e.g., "works_at", "manages")
- **Metadata**: Additional context about the relationship
- **Timestamps**: CreatedAt and UpdatedAt

```go
type Relationship struct {
    ID         string                 `json:"id"`
    SourceID   string                 `json:"source_id"`
    TargetID   string                 `json:"target_id"`
    Type       string                 `json:"type"`
    Metadata   map[string]interface{} `json:"metadata"`
    CreatedAt  time.Time              `json:"created_at"`
    UpdatedAt  time.Time              `json:"updated_at"`
}
```

## Repository Pattern

The foundation provides two repository interfaces:

### ObjectsRepository

CRUD operations for objects:

- `Create(ctx, object)` - Create new object with auto-generated ID
- `Get(ctx, id)` - Retrieve object by ID
- `Update(ctx, object)` - Update existing object
- `Delete(ctx, id)` - Delete object (cascades to relationships)
- `List(ctx, type)` - List all objects, optionally filtered by type

### RelationshipsRepository

CRUD and graph query operations:

- `Create(ctx, relationship)` - Create new relationship
- `Get(ctx, id)` - Retrieve relationship by ID
- `Update(ctx, relationship)` - Update existing relationship
- `Delete(ctx, id)` - Delete relationship
- `List(ctx, type)` - List all relationships, optionally filtered by type
- `FindBySource(ctx, sourceID, type)` - Find relationships from a source
- `FindByTarget(ctx, targetID, type)` - Find relationships to a target
- `FindBetween(ctx, id1, id2)` - Find all relationships between two objects

## Schema

The database schema consists of two tables:

### objects

```sql
CREATE TABLE objects (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX idx_objects_type ON objects(type);
CREATE INDEX idx_objects_created_at ON objects(created_at);
```

### relationships

```sql
CREATE TABLE relationships (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    type TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (source_id) REFERENCES objects(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES objects(id) ON DELETE CASCADE
);

CREATE INDEX idx_relationships_source ON relationships(source_id);
CREATE INDEX idx_relationships_target ON relationships(target_id);
CREATE INDEX idx_relationships_type ON relationships(type);
```

## Use Cases

### CRM System

```go
// Create a company
company := &Object{
    Type: "Company",
    Name: "Acme Corp",
    Metadata: map[string]interface{}{
        "domain": "acme.com",
        "industry": "Technology",
    },
}

// Create a person
person := &Object{
    Type: "Person",
    Name: "Alice",
    Metadata: map[string]interface{}{
        "email": "alice@acme.com",
        "title": "VP Engineering",
    },
}

// Connect them
relationship := &Relationship{
    SourceID: person.ID,
    TargetID: company.ID,
    Type: "works_at",
    Metadata: map[string]interface{}{
        "start_date": "2024-01-01",
    },
}
```

### Project Management

```go
// Create project and tasks
project := &Object{Type: "Project", Name: "Website Redesign"}
task := &Object{Type: "Task", Name: "Design mockups"}

// Link task to project
taskRel := &Relationship{
    SourceID: task.ID,
    TargetID: project.ID,
    Type: "belongs_to",
}

// Assign person to task
assignment := &Relationship{
    SourceID: person.ID,
    TargetID: task.ID,
    Type: "assigned_to",
}
```

### Knowledge Graph

```go
// Create concepts
ai := &Object{Type: "Concept", Name: "Artificial Intelligence"}
ml := &Object{Type: "Concept", Name: "Machine Learning"}

// Create hierarchy
hierarchy := &Relationship{
    SourceID: ml.ID,
    TargetID: ai.ID,
    Type: "is_part_of",
}
```

## Migration from Legacy Schema

The `cmd/migrate` utility helps transition from the old table-based schema:

```bash
# Dry run to see what would happen
go run cmd/migrate/main.go -db path/to/db.sqlite -dry-run

# Create backup and migrate
go run cmd/migrate/main.go -db path/to/db.sqlite -force

# Skip backup (not recommended)
go run cmd/migrate/main.go -db path/to/db.sqlite -force -backup=false
```

### Migration Process

1. Creates backup of database (unless disabled)
2. Drops legacy tables (companies, contacts, deals, etc.)
3. Creates new Office OS foundation tables
4. Enables foreign key constraints

**Warning**: Migration is destructive. Legacy data is not transferred to the new schema. Backup your data before migrating.

## Benefits

### Flexibility

- No schema changes needed to add new entity types
- Metadata can evolve without migrations
- Relationships can be added dynamically

### Simplicity

- Two tables instead of many
- Universal query patterns
- Consistent API across all entity types

### Power

- Graph queries out of the box
- Multi-type relationships supported
- Cascade deletes maintain referential integrity

## Testing

The foundation includes comprehensive tests:

```bash
# Run all foundation tests
go test ./db -run "TestObjects|TestRelationships"

# Run integration tests
go test ./db -run "TestCRM|TestProject|TestKnowledge"

# Run example
go run examples/basic_usage.go
```

## Performance Considerations

### Indexes

The schema includes strategic indexes:
- `objects.type` - Fast filtering by entity type
- `relationships.source_id` - Fast source lookups
- `relationships.target_id` - Fast target lookups
- `relationships.type` - Fast relationship type filtering

### Query Patterns

Efficient patterns:
- List objects by type: O(log n) with index
- Find relationships by source/target: O(log n) with index
- Get object by ID: O(1) primary key lookup

Less efficient patterns:
- Full metadata search (requires application-level filtering)
- Complex graph traversals (may need caching layer)

## Future Extensions

Possible enhancements:
- Full-text search on metadata
- Graph traversal helpers
- Relationship validation rules
- Versioning/audit trail
- Soft deletes

## Examples

See `examples/basic_usage.go` for a comprehensive walkthrough of all core features.

See `db/integration_test.go` for real-world scenario tests.
