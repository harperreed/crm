# Office OS Migration Design

**Date:** 2025-12-01
**Status:** Approved
**Timeline:** 4 weeks (aggressive)
**Approach:** Parallel worktrees with subagents

## Executive Summary

Migrate pagen from specialized CRM tables to a unified "Office OS" data model with generic object storage. This enables:

1. **Activity Timeline** - Complete history for any object (emails, meetings, changes)
2. **Task Management** - Follow-ups as real actionable items
3. **Unified Search** - Search across all entity types
4. **Extensibility** - Add new object types without schema migrations

## Strategic Decisions

### Migration Strategy: Fresh Start
- **Decision:** Nuke existing database, re-sync from Google
- **Rationale:** Simplest migration path, existing data can be recreated from Google
- **Impact:** Lose any hand-curated notes/relationships (acceptable for aggressive timeline)

### Storage: Single Objects Table (Pure Office OS)
- **Decision:** One physical `objects` table with JSONB fields
- **Rationale:** Maximum flexibility, true to Office OS spec
- **Trade-off:** Slightly slower queries, but SQLite JSON functions are fast enough

### Implementation: Parallel Worktrees
- **Decision:** Three concurrent workstreams using git worktrees + subagents
- **Rationale:** 3x faster development, validate architecture early
- **Risk:** Integration conflicts (mitigated by clear interface contracts)

### MCP Integration: Break Temporarily
- **Decision:** Don't maintain MCP compatibility during migration
- **Rationale:** Faster development, rebuild MCP handlers after schema stabilizes
- **Impact:** No Claude integration for 2-3 weeks (acceptable for personal tool)

## Architecture

### Core Schema

```sql
CREATE TABLE objects (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,  -- 'user', 'record', 'task', 'event', 'message', 'activity', 'notification'
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  created_by TEXT NOT NULL,
  acl TEXT NOT NULL,   -- JSON: [{"actorId": "user-id", "role": "owner"}]
  tags TEXT,           -- JSON: ["crm", "urgent"]
  fields TEXT NOT NULL -- JSONB: schema-specific fields
);

CREATE INDEX idx_objects_kind ON objects(kind);
CREATE INDEX idx_objects_created_by ON objects(created_by);
CREATE INDEX idx_objects_created_at ON objects(created_at);
CREATE INDEX idx_objects_schema ON objects(json_extract(fields, '$.schemaId'));
```

### Object Kinds

**Core Primitives:**
- `user` - Identity
- `record` - Generic structured data (CRM entities, KB articles)
- `task` - Work units
- `event` - Time-bound coordination
- `message` - Communication (email, chat, comments)
- `activity` - Audit trail
- `notification` - User alerts

**Schema Conventions:**
- Contacts: `kind='record'`, `fields.schemaId='schema:crm_contact'`
- Companies: `kind='record'`, `fields.schemaId='schema:crm_account'`
- Opportunities: `kind='record'`, `fields.schemaId='schema:crm_opportunity'`

### Example Objects

**Contact (RecordObject):**
```json
{
  "id": "contact-uuid",
  "kind": "record",
  "created_at": "2025-12-01T20:00:00Z",
  "updated_at": "2025-12-01T20:00:00Z",
  "created_by": "you",
  "acl": [{"actorId": "you", "role": "owner"}],
  "tags": ["vip", "technical"],
  "fields": {
    "schemaId": "schema:crm_contact",
    "firstName": "Sarah",
    "lastName": "Chen",
    "email": "sarah@acme.com",
    "phone": "+1-555-0100",
    "title": "CTO",
    "accountId": "account-uuid",
    "lifecycleStage": "customer"
  }
}
```

**Activity (Timeline Entry):**
```json
{
  "id": "activity-uuid",
  "kind": "activity",
  "created_at": "2025-12-01T20:15:00Z",
  "created_by": "you",
  "fields": {
    "actorId": "you",
    "verb": "updated",
    "objectId": "contact-sarah-uuid",
    "objectKind": "record",
    "metadata": {
      "changes": {
        "title": {"before": "VP Engineering", "after": "CTO"}
      }
    }
  }
}
```

**Task (Follow-up):**
```json
{
  "id": "task-uuid",
  "kind": "task",
  "created_at": "2025-12-01T20:00:00Z",
  "created_by": "you",
  "fields": {
    "title": "Follow up with Sarah about Q1 planning",
    "status": "todo",
    "assigneeId": "you",
    "dueAt": "2025-12-15T09:00:00Z",
    "relatedRecordIds": ["contact-sarah-uuid", "opp-acme-q1-uuid"]
  }
}
```

**Email (MessageObject):**
```json
{
  "id": "message-uuid",
  "kind": "message",
  "created_at": "2025-12-01T14:30:00Z",
  "created_by": "contact-sarah-uuid",
  "fields": {
    "channelKind": "inbox",
    "threadRootId": "gmail-thread-id",
    "authorId": "contact-sarah-uuid",
    "bodyText": "Re: Q1 Planning meeting...",
    "contextObjectId": "opp-acme-q1-uuid",
    "gmailMessageId": "msg_12345"
  }
}
```

**Calendar Event (EventObject):**
```json
{
  "id": "event-uuid",
  "kind": "event",
  "created_at": "2025-12-01T20:00:00Z",
  "created_by": "you",
  "fields": {
    "title": "Q1 Planning with Sarah",
    "startAt": "2025-12-15T14:00:00Z",
    "endAt": "2025-12-15T15:00:00Z",
    "organizerId": "you",
    "attendeeIds": ["contact-sarah-uuid"],
    "relatedRecordIds": ["opp-acme-q1-uuid"],
    "gcalEventId": "evt_67890"
  }
}
```

## Implementation Plan

### Parallel Worktree Strategy

**Worktree 1: Foundation** (`feature/office-os-foundation`)
- Core `objects` table schema
- BaseObject struct and interfaces
- Generic CRUD operations
- JSON field helpers for SQLite
- Migration script to nuke old DB
- Basic test coverage

**Worktree 2: Activity System** (`feature/activity-timeline`)
- ActivityObject implementation
- Activity generation hooks (onCreate, onUpdate, onDelete)
- Timeline query logic
- TUI timeline view component
- Integration tests

**Worktree 3: Task System** (`feature/task-management`)
- TaskObject implementation
- Task CRUD operations
- Task status transitions
- Due date reminder logic
- TUI task list/board views
- Integration tests

### Integration Sequence

**Week 1:** Parallel development
- All three worktrees progress independently
- Foundation defines interfaces for Activity + Task systems
- Daily sync meetings to align on interface contracts

**Week 2:** Foundation merge
- Merge `feature/office-os-foundation` → main
- **BREAKING CHANGE:** Fresh database, old data gone
- Re-sync from Google to populate new schema
- Verify basic CRUD works

**Week 3:** Activity system merge
- Merge `feature/activity-timeline` → main
- Activity hooks fire on all object changes
- Timeline view shows complete history
- Verify with real Google sync data

**Week 4:** Task system merge + Polish
- Merge `feature/task-management` → main
- Task creation from TUI
- Task-to-object linking
- Due date reminders
- Full integration testing
- Documentation updates

## Feature Delivery

### Week 1-2: Foundation
**What works:**
- Create/read/update/delete any object
- Query by kind/schemaId
- Fresh Google sync populates contacts/companies
- Basic TUI list views

**What doesn't:**
- No activity history yet
- No tasks yet
- No timeline views
- MCP broken

### Week 3: Activity Timeline
**What works:**
- Complete history for any object
- See all emails/meetings/changes for a contact
- TUI timeline tab on detail views
- Activity auto-generated on changes

**What doesn't:**
- Still no tasks
- MCP still broken

### Week 4: Task Management + Complete
**What works:**
- Create tasks with due dates
- Link tasks to contacts/deals
- Task board view (kanban)
- Task list with filtering
- **Complete Office OS foundation ready**

**What doesn't:**
- MCP still broken (rebuild in Week 5+)
- No notification system yet (future)
- No knowledge base yet (future)

## Google Sync Integration

### Mapping to Office OS

**Gmail Sync:**
- Each email → `MessageObject`
- `fields.gmailMessageId` preserves Google ID
- `fields.authorId` links to contact (via email matching)
- `fields.contextObjectId` links to deal if detected
- High-signal filtering unchanged (replied-to + starred)

**Calendar Sync:**
- Each event → `EventObject`
- `fields.gcalEventId` preserves Google ID
- `fields.attendeeIds` links to contacts
- `fields.relatedRecordIds` links to deals if detected
- Incremental sync via existing sync_state table

**Contacts Sync:**
- Each contact → `RecordObject` with `schemaId='schema:crm_contact'`
- Company population unchanged (domain matching)
- Email/phone matching unchanged

**Sync State:**
- Keep existing `sync_state` table (orthogonal to Office OS)
- historyId tracking unchanged
- Daemon mode unchanged

## TUI Updates

### Current Views → New Implementation

**List View:**
- **Before:** Separate lists for contacts/companies/deals
- **After:** Generic object list filtered by `kind` + `schemaId`
- Query: `SELECT * FROM objects WHERE kind='record' AND json_extract(fields, '$.schemaId')='schema:crm_contact'`

**Detail View:**
- **Before:** Entity-specific detail panel
- **After:** Generic object detail + **NEW: Timeline tab**
- Timeline shows: activities, messages, events, tasks for this object
- Query: `SELECT * FROM objects WHERE kind IN ('activity','message','event','task') AND ... references this object`

**Edit View:**
- **Before:** Form with typed fields
- **After:** Generic JSONB field editor
- Validates against schema conventions

**Sync View:**
- **Before:** Shows Google sync status
- **After:** Unchanged (still uses sync_state table)

**Followup View:**
- **Before:** Cadence-based follow-up list
- **After:** **Task list view** filtered by status/due date
- Shows all tasks, sortable/filterable
- Quick create task from any object

**Graph View:**
- **Before:** Contact relationship visualization
- **After:** Updated to query objects table for relationships
- May add new relationship types (task dependencies, etc.)

**NEW: Task Board View:**
- Kanban board grouped by status
- Drag-and-drop status changes
- Filter by assignee/due date/related objects

**NEW: Timeline View:**
- Chronological activity stream
- Filter by object type, actor, date range
- Shows complete context for any entity

## Testing Strategy

### Foundation Worktree
- Unit tests for CRUD operations
- JSON field extraction tests
- Schema validation tests
- SQLite query performance tests

### Activity Worktree
- Activity generation tests (onCreate, onUpdate, onDelete)
- Timeline query tests
- Activity filtering tests
- Integration tests with foundation

### Task Worktree
- Task status transition tests
- Due date logic tests
- Task-to-object linking tests
- Integration tests with foundation

### Integration Testing (Post-merge)
- Full Google sync → Office OS object creation
- TUI rendering with real data
- Timeline view correctness
- Task creation and completion flows
- Performance tests with realistic data volumes

## Risk Mitigation

**Risk: Integration conflicts between worktrees**
- Mitigation: Foundation defines clear interfaces upfront
- Daily sync meetings to align
- Shared interface contract document

**Risk: JSONB query performance**
- Mitigation: Add indexes on frequently queried fields
- Benchmark with realistic data volumes
- Fall back to hybrid schema if needed

**Risk: Data loss from fresh start**
- Mitigation: Export current DB to JSON before nuking
- Google sync recreates most data
- Acceptable trade-off for speed

**Risk: MCP broken for 3-4 weeks**
- Mitigation: You're the only user, can tolerate downtime
- Rebuild MCP handlers after schema stabilizes
- Document new MCP API alongside migration

**Risk: Aggressive timeline causes quality issues**
- Mitigation: Each worktree has tests
- Integration testing after each merge
- Week 4 buffer for polish/debugging

## Success Criteria

**Minimum Viable Office OS (End of Week 4):**
- ✅ All objects stored in unified `objects` table
- ✅ Contacts, companies, opportunities migrated to RecordObject
- ✅ Google sync populates new schema
- ✅ Activity timeline shows complete history for any object
- ✅ Tasks can be created and linked to objects
- ✅ TUI timeline view works
- ✅ TUI task list/board works
- ✅ Basic search across object types
- ✅ Foundation ready for knowledge base, notifications, etc.

**Known Incomplete (Week 5+):**
- ❌ MCP server integration (rebuild required)
- ❌ Notification system (future)
- ❌ Knowledge base (future)
- ❌ Full ACL enforcement (stubbed for now)
- ❌ Advanced search (FTS5 integration)

## Future Extensions (Post-Week 4)

Once foundation is stable:

1. **Rebuild MCP Server** (Week 5-6)
   - New handlers for Office OS objects
   - Maintain Claude integration
   - Enhanced prompts with timeline context

2. **Notification System** (Week 7)
   - Task due date reminders
   - Activity-based alerts
   - Email integration

3. **Knowledge Base** (Week 8+)
   - Articles as RecordObjects
   - Full-text search integration
   - Auto-suggestions based on context

4. **Advanced Search** (Week 8+)
   - SQLite FTS5 integration
   - Unified search across all object types
   - Semantic search with embeddings

5. **Multi-user Support** (Future)
   - Enforce ACLs
   - User management
   - Collaboration features

## Conclusion

This aggressive 4-week migration transforms pagen from a specialized CRM into a flexible "Office OS" platform. The parallel worktree strategy accelerates development while the fresh-start approach eliminates migration complexity.

Key deliverables:
- **Week 2:** Foundation + fresh Google sync
- **Week 3:** Activity timeline
- **Week 4:** Task management

The unified object model unlocks future capabilities (KB, notifications, advanced search) without requiring further schema migrations.

**Ready to build.**
