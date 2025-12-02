# Office OS - Activity Timeline System

This package implements the Activity Timeline system for Office OS, providing complete audit trails and activity tracking for all objects.

## Features

### Core Components

1. **BaseObject** - Foundation for all Office OS objects with unified schema
2. **ActivityObject** - Activity tracking with actor, verb, object, and metadata
3. **Timeline** - Query and filter activities chronologically
4. **TimelineView** - TUI component for rendering timelines with color coding
5. **MemoryStore** - In-memory object storage for testing

### Activity Verbs

- `created` - Object creation
- `updated` - Object modification with change tracking
- `deleted` - Object deletion
- `viewed` - Object access
- `shared` - Object sharing

### Activity Generation

Activities are automatically generated through hooks:

```go
store := NewMemoryStore()
hooks := NewActivityGenerator(store)

// Automatically create activity on object creation
hooks.OnCreate(object)

// Track changes on update
hooks.OnUpdate(oldObject, newObject)

// Record deletion
hooks.OnDelete(object)
```

### Timeline Queries

Filter and retrieve activities:

```go
timeline := NewTimeline(store)

// Get all activities for an object
activities, _ := timeline.GetTimelineForObject("object-123")

// Filter by actor
activities, _ := timeline.GetActivitiesByActor("user-1", 10)

// Filter by verb
activities, _ := timeline.GetActivitiesByVerb(VerbUpdated, 10)

// Custom filtering
activities, _ := timeline.GetTimeline(TimelineFilter{
    ObjectID:   "contact-456",
    ActorID:    "user-1",
    Verb:       VerbUpdated,
    StartDate:  &startTime,
    EndDate:    &endTime,
    ObjectKind: KindRecord,
    Limit:      20,
})
```

### Timeline Rendering

Display timelines in TUI:

```go
view := NewTimelineView(timeline, 80, 40)

// Render timeline for an object
output := view.Render("contact-123")
fmt.Println(output)

// Custom filtered view
output = view.RenderFiltered(TimelineFilter{ActorID: "user-1"})
```

## Example

See the complete workflow in action:

```go
// Create store and components
store := NewMemoryStore()
hooks := NewActivityGenerator(store)
timeline := NewTimeline(store)

// Create a contact
contact := &BaseObject{
    ID:        "contact-sarah",
    Kind:      KindRecord,
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
    CreatedBy: "you",
    Fields: map[string]interface{}{
        "firstName": "Sarah",
        "lastName":  "Chen",
        "title":     "VP Engineering",
    },
}

store.Create(contact)
hooks.OnCreate(contact)

// Update the contact
updatedContact := *contact
updatedContact.Fields = map[string]interface{}{
    "firstName": "Sarah",
    "lastName":  "Chen",
    "title":     "CTO",  // Changed!
}
updatedContact.UpdatedAt = time.Now()

hooks.OnUpdate(contact, &updatedContact)

// Query timeline
activities, _ := timeline.GetTimelineForObject(contact.ID)
// Returns: [updated activity, created activity]
```

## Testing

Run all tests:

```bash
go test ./officeos/... -v
```

All tests include:
- Unit tests for ActivityObject
- Integration tests for activity generation
- Timeline query and filtering tests
- Timeline view rendering tests
- Example demonstrations

## Design

Based on the Office OS migration design doc (`docs/plans/2025-12-01-office-os-migration-design.md`):

- **Unified Objects Table** - All objects stored in single table with JSONB fields
- **Activity Tracking** - Complete audit trail for every object
- **Flexible Schema** - No migrations needed for new object types
- **Timeline Views** - Chronological activity display in TUI

## Status

✅ Complete implementation including:
- BaseObject and ActivityObject structs
- Activity generation hooks (onCreate, onUpdate, onDelete)
- Timeline query logic with filtering
- TUI timeline view component
- Comprehensive test coverage
- Example demonstrations

Ready for integration with the foundation and task worktrees.
