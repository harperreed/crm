# Task Management System - Implementation Demo

## Overview

Complete implementation of the Office OS task management system as specified in `docs/plans/2025-12-01-office-os-migration-design.md`.

## Features Implemented

### 1. Core Object Model
- **BaseObject**: Generic Office OS object with ID, kind, timestamps, ACL, tags, and JSONB fields
- **TaskObject**: Specialized task implementation extending BaseObject
- Full JSON serialization/deserialization support

### 2. Task Management
- Task creation with title, assignee, due date, and related records
- Status transitions: `todo`, `in_progress`, `done`, `cancelled`
- Automatic completion timestamp tracking
- Related record linking for tasks associated with contacts/deals
- Due date tracking with overdue and "due soon" helpers

### 3. Database Layer
- CRUD operations: CreateTask, GetTask, UpdateTask, DeleteTask
- Advanced queries:
  - ListTasks with filtering (status, assignee, due date)
  - ListOverdueTasks for tasks past their due date
  - ListDueSoonTasks for upcoming tasks (within N days)
  - ListTasksByRelatedRecord for object-linked tasks
- Full SQLite JSON support using `json_extract` functions

### 4. TUI Views
- **Task Board View**: Kanban-style board with columns for each status
- **Task Cards**: Visual cards showing title, due date, and priority indicators
- **Visual Indicators**:
  - ⚠ Red warning for overdue tasks
  - ◷ Yellow clock for tasks due soon
  - Gray clock for tasks with future due dates

## Example Usage

```go
// Create a new task
createdBy := uuid.New()
assigneeID := uuid.New()
dueAt := time.Now().Add(7 * 24 * time.Hour)

task := objects.NewTaskObject(
    createdBy,
    "Follow up with Sarah about Q1 planning",
    assigneeID,
    &dueAt,
)

// Link to related records
contactID := uuid.Parse("contact-sarah-uuid")
dealID := uuid.Parse("opp-acme-q1-uuid")
task.AddRelatedRecord(contactID)
task.AddRelatedRecord(dealID)

// Save to database
err := db.CreateTask(sqlDB, task)

// Transition status
err = task.TransitionStatus(objects.TaskStatusInProgress)
err = db.UpdateTask(sqlDB, task)

// Query overdue tasks
overdueTasks, err := db.ListOverdueTasks(sqlDB)

// Query tasks due within 7 days
dueSoonTasks, err := db.ListDueSoonTasks(sqlDB, 7)

// Query tasks for a specific contact
contactTasks, err := db.ListTasksByRelatedRecord(sqlDB, contactID)
```

## Task Object Structure

```json
{
  "id": "task-uuid",
  "kind": "task",
  "created_at": "2025-12-01T20:00:00Z",
  "updated_at": "2025-12-01T20:15:00Z",
  "created_by": "user-uuid",
  "acl": [
    {"actorId": "user-uuid", "role": "owner"}
  ],
  "tags": ["urgent", "q1"],
  "fields": {
    "title": "Follow up with Sarah about Q1 planning",
    "status": "in_progress",
    "assigneeId": "user-uuid",
    "dueAt": "2025-12-15T09:00:00Z",
    "relatedRecordIds": [
      "contact-sarah-uuid",
      "opp-acme-q1-uuid"
    ]
  }
}
```

## Database Schema

```sql
CREATE TABLE objects (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  created_by TEXT NOT NULL,
  acl TEXT NOT NULL,
  tags TEXT,
  fields TEXT NOT NULL
);

CREATE INDEX idx_objects_kind ON objects(kind);
CREATE INDEX idx_objects_created_by ON objects(created_by);
CREATE INDEX idx_objects_created_at ON objects(created_at);
```

## Test Coverage

### Unit Tests
- BaseObject JSON serialization
- TaskObject creation and field management
- Status transition validation
- Due date logic (overdue, due soon)
- Related record linking
- Completion tracking

### Database Tests
- All CRUD operations
- Filter queries by status, assignee, due date
- Overdue task queries
- Due soon task queries
- Related record queries

### Integration Tests
- Complete task lifecycle workflows
- Task-to-record linking
- Status validation
- Overdue detection
- Completion cycle tracking

### TUI Tests
- Task board rendering
- Task card display
- Visual indicators (overdue, due soon)
- Duration formatting

## Task Board Example Output

```
┌─────────────────────────────┐  ┌─────────────────────────────┐  ┌─────────────────────────────┐
│         TODO                │  │      IN PROGRESS            │  │          DONE               │
│         (2 tasks)           │  │      (1 tasks)              │  │         (3 tasks)           │
│                             │  │                             │  │                             │
│ ┌─────────────────────────┐ │  │ ┌─────────────────────────┐ │  │ ┌─────────────────────────┐ │
│ │ Follow up with Sarah    │ │  │ │ Complete Q1 planning    │ │  │ │ Review budget proposal  │ │
│ │ ⚠ Dec 1, 2025          │ │  │ │ ◷ Dec 8, 2025          │ │  │ │ ◷ Nov 30, 2025         │ │
│ └─────────────────────────┘ │  │ └─────────────────────────┘ │  │ └─────────────────────────┘ │
│                             │  │                             │  │                             │
│ ┌─────────────────────────┐ │  │                             │  │ ┌─────────────────────────┐ │
│ │ Send proposal to client │ │  │                             │  │ │ Update CRM contacts     │ │
│ │ ◷ Dec 5, 2025          │ │  │                             │  │ │                         │ │
│ └─────────────────────────┘ │  │                             │  │ └─────────────────────────┘ │
└─────────────────────────────┘  └─────────────────────────────┘  └─────────────────────────────┘
```

## Status Transitions

```
     ┌─────────┐
     │  todo   │◄────────────┐
     └────┬────┘             │
          │                  │
          ▼                  │
   ┌─────────────┐           │
   │ in_progress │───────────┤
   └──────┬──────┘           │
          │                  │
          ▼                  │
     ┌────────┐              │
     │  done  │──────────────┘
     └────────┘

     ┌───────────┐
     │ cancelled │ (from any state)
     └───────────┘
```

## Performance Considerations

### SQLite JSON Queries
- Uses `json_extract()` for efficient field access
- Indexed on `kind` for fast task filtering
- Related record queries use `json_each()` for array matching

### Query Examples
```sql
-- List overdue tasks
SELECT * FROM objects
WHERE kind = 'task'
  AND json_extract(fields, '$.dueAt') < datetime('now')
  AND json_extract(fields, '$.status') NOT IN ('done', 'cancelled');

-- List tasks by related record
SELECT * FROM objects, json_each(json_extract(fields, '$.relatedRecordIds'))
WHERE kind = 'task' AND json_each.value = ?;
```

## Next Steps (Post-Merge)

1. **CLI Integration**: Add task commands to main CLI
2. **TUI Integration**: Add "Tasks" tab to main TUI navigation
3. **Sync Integration**: Create tasks from calendar events automatically
4. **Notifications**: Due date reminders
5. **Advanced Filtering**: Tags, search, custom filters
6. **Task Templates**: Quick task creation from templates

## Conclusion

The task management system is complete and ready for integration. All tests pass, the foundation is solid, and the implementation follows the Office OS specification exactly.
