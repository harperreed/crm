<!-- ABOUTME: Defines read-only CRM history for contacts, companies, and relationships. -->
<!-- ABOUTME: Specifies immutable snapshots, backend persistence, recovery, CLI/MCP access, and tests. -->

# CRM History Design

## Goal

Add trustworthy, read-only history without replacing the current-state CRM model. A person or agent must be able to see when a contact, company, or relationship changed and inspect the record before and after that change.

The first release records new mutations only. It does not infer older events, restore old states, purge events, or make events the source of truth.

Estimated implementation size: 900–1,300 lines including unit, integration, and end-to-end tests.

## Product decisions

- Keep current contact, company, and relationship records authoritative for normal reads.
- Record successful creates, meaningful updates, and deletes after this feature ships.
- Cover SQLite and Markdown storage.
- Expose history through the CLI and MCP.
- Keep history until a later explicit purge feature is designed.
- Record a coarse mutation source: `cli` or `mcp`. Do not add user identity.
- Include relationship link/unlink events in the timelines of both endpoint entities.
- Provide timeline and event inspection only. Undo and restore are out of scope.
- Preserve deleted records in history. Deleting CRM data is not an erasure of its historical snapshots.

## Event model

Add an internal history package with stable domain and wire types. A committed event has:

```text
HistoryEvent
  id                  UUID
  schema_version      integer, initially 1
  entity_type         contact | company | relationship
  entity_id           UUID of the event's subject
  related_entity_ids  subject ID, plus both endpoints for relationships
  action              create | update | delete
  source              cli | mcp
  occurred_at          UTC timestamp
  before               versioned JSON snapshot or null
  after                versioned JSON snapshot or null
```

Create events have a null `before`; delete events have a null `after`. Update events have both. Event IDs and event contents never change after commit.

Snapshots use dedicated snake-case DTOs rather than adding JSON tags to the current models. This keeps the history schema stable and avoids changing existing MCP response shapes. The version-1 snapshots contain every persisted field:

- Contact: ID, name, email, phone, fields, tags, created time, and updated time.
- Company: ID, name, domain, fields, tags, created time, and updated time.
- Relationship: ID, source ID, target ID, type, context, and created time.

The history package compares snapshots and reports changed top-level domain fields for timeline summaries. It omits bookkeeping-only timestamp differences from the summary. If an update changes no domain field, storage leaves the current record unchanged and writes no event.

## Mutation source

The root command chooses one source when it opens storage:

- `crm mcp` opens storage with source `mcp`.
- Every other storage-backed command opens storage with source `cli`.

The source flows through configuration into both backend constructors and stays fixed for that store instance. This avoids mutable global attribution and avoids widening every CRUD method with an actor argument.

## Storage contract

Keep existing CRUD call shapes. Extend the storage contract with read-only history operations:

```text
ListHistory(entityIDOrPrefix, limit) -> newest-first event summaries
GetHistoryEvent(eventIDOrPrefix) -> full event
```

Both methods accept full UUIDs or prefixes of at least six characters. Prefixes resolve against committed history, so deleted entities and relationships remain addressable. Multiple matching IDs return the existing ambiguity error. `ListHistory` returns an empty list when no committed event matches; this is also how legacy entities with no recorded history appear. A missing event in `GetHistoryEvent` uses a history-specific not-found error.

Limits default to 20 at the public interfaces and may not exceed 100. Storage orders events by occurrence time descending, then event ID for deterministic ties.

`ListHistory` matches any `related_entity_id`. A contact or company timeline therefore includes relationship events where that entity was the source or target. A relationship timeline is available through the relationship's own ID.

## SQLite persistence

Add two tables through the existing idempotent schema initialization:

```text
history_events
  id primary key
  schema_version
  entity_type
  entity_id
  action
  source
  occurred_at
  before_json nullable
  after_json nullable

history_event_entities
  event_id
  entity_id
  primary key (event_id, entity_id)
```

Index `history_event_entities.entity_id` and the event ordering columns. Related rows reference their event and are inserted in the same transaction.

Each CRUD mutation becomes one SQLite transaction:

1. Read and validate the prior state when required.
2. Build canonical before/after snapshots.
3. Detect and return from a semantic no-op update.
4. Write the current-state mutation.
5. Insert the history event and related entity rows.
6. Commit.

FTS triggers continue to run inside that transaction. Any current-state or history failure rolls back both. Update and delete paths must read their prior state through the same transaction rather than through the store's non-transactional getter.

## Markdown persistence

Store history beneath the Markdown data directory:

```text
_history/
  events/
    <event-uuid>.json
  pending/
    <event-uuid>.json
```

One immutable file per event avoids a partially appended shared log. Timeline reads scan committed event files, select matching related IDs, sort, and apply the limit. This is adequate for a single-user CRM; an index is not part of this release.

A process-local mutex serializes Markdown mutation and recovery work. It does not claim to make the existing Markdown backend safe for concurrent writers in separate processes.

Each mutation follows a small write-ahead protocol:

1. Read and validate the current record.
2. Build the exact before/after event.
3. Detect and return from a semantic no-op update.
4. Atomically write the complete event to `_history/pending/`.
5. Apply the intended current-state write or delete using atomic file operations where applicable.
6. Atomically rename the pending event into `_history/events/`.

Relationship-list writes must use an atomic replacement rather than a direct rewrite. Contact and company name changes remain recoverable even if the old filename was removed before the replacement was written because the pending event contains the complete intended state.

### Startup recovery

`NewMarkdownStore` creates the history directories and processes every pending event before returning:

- If current state equals `before`, apply `after` idempotently and commit the event.
- If current state already equals `after`, commit the event without another mutation.
- For a pending create, absence is the `before` state.
- For a pending delete, absence is the `after` state.
- If current state matches neither snapshot, stop with a history-conflict error containing the event ID and pending path.

Recovery never guesses past an unexplained manual edit. Corrupt pending or committed event files return errors rather than disappearing from results.

If finalizing the event fails after current state changes, the mutation returns an error and leaves the pending event for startup recovery. The caller must not receive a false success.

## CLI

Add a root `history` command:

```text
crm history <entity-id-or-prefix> [--limit 20]
crm history show <event-id-or-prefix>
```

The timeline prints newest first. Each row includes:

- UTC timestamp
- action and entity type
- source
- event-ID prefix
- changed top-level fields for updates

Create, delete, link, and unlink rows use concise action summaries. `history show` prints event metadata followed by formatted before and after JSON snapshots. A timeline with no events prints `No history found.` and exits successfully.

CLI history lookup does not resolve through current contact or company records; it queries history directly so deleted entities work.

## MCP

Register two read-only tools, bringing the tool count from 12 to 14:

- `list_history`: required `entity_id`, optional `limit`; returns structured summaries.
- `get_history_event`: required `event_id`; returns the full structured event.

Both accept full UUIDs or six-character prefixes. Validation and storage errors use the server's existing error-result convention. The tools expose no restore, mutation, or purge operation.

All existing MCP mutations run against a store opened with source `mcp`; CLI mutations use `cli`.

## Error behavior

- Failed creates, updates, and deletes produce no committed history.
- SQLite rolls back both state and history on every error.
- Markdown may leave a pending operation after an error; recovery resolves only known before/after states.
- History prefix validation distinguishes too-short and ambiguous identifiers. Timeline lookups with no committed match return an empty list; event lookups with no match return a history-specific error.
- Invalid event JSON, unknown schema versions, and recovery conflicts are explicit errors.
- Limits below one use the public default; limits above 100 fail validation rather than allocating an unbounded result.

## Migration and compatibility

SQLite uses `CREATE TABLE IF NOT EXISTS`, so opening an existing database adds empty history tables without touching existing records. Markdown creates empty `_history/events/` and `_history/pending/` directories. Neither backend emits baseline events.

Existing CRUD commands and MCP tools keep their public inputs and successful outputs. Current record formats stay readable. The storage constructors and configuration plumbing change internally to require a mutation source.

The implementation must not absorb or rewrite the uncommitted SQLite-driver and MCP security POC files currently in Doctor Biz's working tree. History work will run in an isolated branch/workspace after planning. Any later integration conflict with those files requires explicit review.

## Security and privacy

History snapshots contain the same personal data as current CRM records and preserve it after deletion. History uses the same local trust boundary and storage permissions as its backend. MCP clients that can access CRM tools can also read history.

This feature does not add authentication, encryption, redaction, retention expiry, or secure erasure. Documentation must state that delete removes the current record but not its historical snapshots.

## Testing

### Unit tests

- Version-1 snapshot encoding and decoding for every entity type.
- Stable snake-case JSON without changing current model serialization.
- Semantic equality and changed-field summaries.
- Prefix and limit validation.
- Markdown recovery decisions for before, after, and conflict states.

### Backend contract tests

Run the same history behavior against real SQLite and Markdown stores:

- Create, meaningful update, no-op update, and delete events.
- Relationship create/delete events visible through the relationship and both endpoints.
- Deleted entity and event lookup by full ID and prefix.
- Newest-first deterministic ordering and limits.
- Failed mutations do not create committed events.
- Source and before/after snapshots match exactly.

SQLite-specific tests prove rollback when event persistence fails. Markdown-specific tests create real pending files and prove replay, finalize-only recovery, conflict refusal, corrupt-event errors, and atomic relationship-file replacement.

### Integration tests

- Open pre-history SQLite and Markdown fixtures and confirm an empty timeline.
- Mutate existing records and confirm the first event contains the real pre-feature state and new state.
- Exercise a full contact/company/link/unlink/delete timeline across both backends.

### End-to-end tests

- Build and run the real CLI with temporary XDG config and data directories for both backends.
- Verify timeline and event inspection output, source `cli`, relationship aggregation, and deleted-record lookup.
- Use a real MCP transport and real storage to mutate data, then verify source `mcp` through both history tools. Do not add mocks or an application mock mode.

The canonical completion gate remains `make check`, followed by `go vet ./...`, `go test -race ./...`, and real CLI/MCP usage. Existing release verification and its documented `brews` exception remain unchanged.

## Documentation

Update README and contributor guidance with:

- Both CLI history commands.
- Both MCP history tools and the new total tool count.
- Future-only recording and indefinite retention.
- Source attribution.
- Relationship aggregation.
- The fact that delete does not erase historical snapshots.
- Markdown crash recovery and its existing single-process-writer limit.

## Acceptance criteria

The feature is ready when both backends produce the same observable history for the same mutation sequence, interrupted Markdown mutations recover without silent data loss, SQLite state and history remain atomic, CLI and MCP can inspect deleted states, all test layers pass without mocks, and no unrelated POC changes enter history commits.
