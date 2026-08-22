<!-- ABOUTME: TDD implementation plan for immutable CRM history across SQLite and Markdown. -->
<!-- ABOUTME: Breaks history models, persistence, recovery, CLI/MCP access, tests, and docs into reviewed commits. -->

# CRM History Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add read-only, future-only history for contact, company, and relationship mutations through both the CLI and MCP.

**Architecture:** Current records remain authoritative. Each mutation creates a versioned immutable before/after event; SQLite commits state and history in one transaction, while Markdown uses a pending-event write-ahead protocol with startup recovery. History queries aggregate relationship events through indexed related entity IDs.

**Tech Stack:** Go 1.25.5, Cobra, `database/sql`, modernc SQLite, Markdown/YAML files, JSON snapshots, MCP Go SDK v1.4.1.

**State:** Implementation, documentation audit, review, and final verification are complete. The branch is awaiting Doctor Biz's disposition choice.

**Next step:** Choose whether to merge, open a pull request, keep the branch, or discard it.

## Global Constraints

- Implement the approved design in `docs/superpowers/specs/2026-08-21-crm-history-design.md`; deviations require Doctor Biz's approval.
- Record future mutations only. Do not create baseline events for existing records.
- Keep events indefinitely. Do not add restore, undo, purge, authentication, encryption, or secure-erasure behavior.
- Cover contacts, companies, relationships, SQLite, Markdown, CLI, and MCP.
- Store source as exactly `cli` or `mcp`; do not add user identity.
- Contact/company timelines include relationship events through both endpoint IDs.
- Limits default to 20 and reject values above 100.
- Existing CRUD inputs and successful outputs remain unchanged.
- Do not add dependencies.
- All new hand-written Go files start with two `// ABOUTME:` lines.
- Use real stores and transports. Do not use mocks or add application mock modes.
- Preserve Doctor Biz's uncommitted `go.mod`, `go.mod.bak`, `internal/poc/`, `internal/mcp/poc_test.go`, and `internal/storage/dsn_poc_test.go`; they are not part of history commits.
- Execute from an isolated worktree created by `superpowers:using-git-worktrees` after returning the dirty primary checkout to `feat/crm-implementation`.
- Machine preflight: the inherited `GOROOT` currently points at a missing mise path. Until that local setting is repaired, run Go commands as `env -u GOROOT go ...` and Make targets as `env -u GOROOT make ...`; do not encode this machine workaround in project files.

---

## File map

**History domain**

- Create `internal/history/event.go`: event enums, event/summary types, validation, limit normalization, and summary construction.
- Create `internal/history/event_test.go`: event validation, prefix-independent behavior, limits, ordering fields, and summaries.
- Create `internal/history/snapshot.go`: stable version-1 DTOs, encode/decode functions, semantic equality, and changed-field detection.
- Create `internal/history/snapshot_test.go`: round trips and stable snake-case JSON for all entities.

**Storage contract and persistence**

- Modify `internal/storage/interface.go`: history errors and read methods.
- Modify `internal/config/config.go`: pass history source into storage.
- Modify `cmd/crm/root.go`: choose `cli` or `mcp` source.
- Modify tracked constructor call sites in config, storage, MCP, and integration tests.
- Modify `internal/storage/sqlite.go`: source/clock fields and history schema.
- Create `internal/storage/sqlite_history.go`: event queries, inserts, transaction verification, and event application.
- Create `internal/storage/sqlite_history_test.go`: schema, query, prefixes, aggregation, transaction rollback, and mutation tests.
- Modify SQLite CRUD files so all six mutations commit through history.
- Modify `internal/storage/markdown.go`: source/clock/mutex fields, history directories, and startup recovery.
- Create `internal/storage/markdown_history.go`: committed-event queries, pending protocol, recovery, state comparison, and event application.
- Create `internal/storage/markdown_history_test.go`: event queries, mutations, recovery states, corruption, and conflicts.
- Modify Markdown CRUD files to prepare history events and make relationship writes atomic.

**Public interfaces**

- Create `cmd/crm/history.go`: timeline and event-inspection commands plus formatting.
- Create `cmd/crm/history_test.go`: command formatting and validation with real stores.
- Create `test/cli_history_e2e_test.go`: built-binary workflows with temporary XDG directories for both backends.
- Create `internal/mcp/history.go`: two tool definitions and handlers.
- Create `internal/mcp/history_test.go`: real MCP client/server transport workflow.
- Modify `internal/mcp/tools.go`: register 14 tools and update the count comment.

**Cross-backend contract and docs**

- Create `test/history_integration_test.go`: equivalent history behavior and legacy-data migration for both backends.
- Modify `README.md`, `CLAUDE.md`, `gotchas.md`, and `cmd/crm/skill/SKILL.md`: document history and deletion retention.

---

### Task 1: Define immutable history events and snapshots

**Files:**

- Create: `internal/history/event.go`
- Create: `internal/history/event_test.go`
- Create: `internal/history/snapshot.go`
- Create: `internal/history/snapshot_test.go`

**Interfaces:**

- Produces:
  - `type EntityType string` with `EntityContact`, `EntityCompany`, `EntityRelationship`.
  - `type Action string` with `ActionCreate`, `ActionUpdate`, `ActionDelete`.
  - `type Source string` with `SourceCLI`, `SourceMCP`.
  - `type Event` and `type Summary` using the JSON names from the design.
  - `func NewEvent(entityType EntityType, entityID uuid.UUID, related []uuid.UUID, action Action, source Source, before, after json.RawMessage, occurredAt time.Time) (*Event, error)`.
  - `func (e *Event) Validate() error`.
  - `func (e *Event) Summary() (Summary, error)`.
  - `func NormalizeLimit(limit int) (int, error)` with constants `DefaultLimit = 20`, `MaxLimit = 100`.
  - `func SnapshotContact(*models.Contact) (json.RawMessage, error)` and `func ContactFromSnapshot(json.RawMessage) (*models.Contact, error)`.
  - `func SnapshotCompany(*models.Company) (json.RawMessage, error)` and `func CompanyFromSnapshot(json.RawMessage) (*models.Company, error)`.
  - `func SnapshotRelationship(*models.Relationship) (json.RawMessage, error)` and `func RelationshipFromSnapshot(json.RawMessage) (*models.Relationship, error)`.
  - `func EqualSnapshots(entityType EntityType, left, right json.RawMessage) (bool, error)`.
  - `func ChangedFields(entityType EntityType, before, after json.RawMessage) ([]string, error)`.

- [x] **Step 1: Write failing event tests**

Create table-driven tests that require exact enum validation, action-specific null snapshots, related-ID de-duplication, UTC timestamps, and limit behavior. Include these concrete cases:

```go
func TestNormalizeLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{name: "default zero", input: 0, want: DefaultLimit},
		{name: "default negative", input: -1, want: DefaultLimit},
		{name: "explicit", input: 42, want: 42},
		{name: "maximum", input: MaxLimit, want: MaxLimit},
		{name: "over maximum", input: MaxLimit + 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeLimit(%d) error = %v", tt.input, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("NormalizeLimit(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
```

`TestNewEventValidation` must cover create (`before=nil`), update (both present), delete (`after=nil`), invalid source, invalid entity type, missing subject in related IDs, invalid snapshot JSON, and zero timestamp.

- [x] **Step 2: Run event tests and verify RED**

Run:

```bash
env -u GOROOT go test ./internal/history -run 'TestNormalizeLimit|TestNewEventValidation' -count=1 -v
```

Expected: FAIL because `internal/history` and its exported types do not exist.

- [x] **Step 3: Implement event types and validation**

Use this exact public shape:

```go
type Event struct {
	ID               uuid.UUID       `json:"id"`
	SchemaVersion    int             `json:"schema_version"`
	EntityType       EntityType      `json:"entity_type"`
	EntityID         uuid.UUID       `json:"entity_id"`
	RelatedEntityIDs []uuid.UUID     `json:"related_entity_ids"`
	Action           Action          `json:"action"`
	Source           Source          `json:"source"`
	OccurredAt       time.Time       `json:"occurred_at"`
	Before           json.RawMessage `json:"before"`
	After            json.RawMessage `json:"after"`
}

type Summary struct {
	ID            uuid.UUID  `json:"id"`
	EntityType    EntityType `json:"entity_type"`
	EntityID      uuid.UUID  `json:"entity_id"`
	Action        Action     `json:"action"`
	Source        Source     `json:"source"`
	OccurredAt    time.Time  `json:"occurred_at"`
	ChangedFields []string   `json:"changed_fields"`
}
```

`NewEvent` generates a UUID, sets schema version 1, canonicalizes related IDs by sorting and de-duplicating them, converts time to UTC, and calls `Validate` before returning.

- [x] **Step 4: Write failing snapshot tests**

Round-trip one fully populated instance of each current model. Assert the JSON contains snake-case keys and never contains `"ID"`, `"CreatedAt"`, or `"UpdatedAt"`. For contacts, require:

```go
wantChanged := []string{"email", "fields", "phone", "tags"}
```

after changing those four fields. Assert that changing only `updated_at` produces an empty changed-field list.

- [x] **Step 5: Run snapshot tests and verify RED**

Run:

```bash
env -u GOROOT go test ./internal/history -run 'Snapshot|ChangedFields' -count=1 -v
```

Expected: FAIL because the snapshot functions do not exist.

- [x] **Step 6: Implement stable version-1 snapshot DTOs**

Use private DTO structs with explicit JSON tags and the exported conversion functions above. Preserve exact UUIDs and timestamps. `EqualSnapshots` decodes and re-encodes both sides through the entity-specific DTO, treats two null sides as equal, and compares the canonical JSON bytes. `ChangedFields` decodes JSON objects, removes `updated_at`, compares remaining top-level values with `reflect.DeepEqual`, sorts field names, and returns an error for invalid JSON or an unknown entity type.

Do not add tags to `models.Contact`, `models.Company`, or `models.Relationship`.

- [x] **Step 7: Run Task 1 verification**

Run:

```bash
env -u GOROOT go test ./internal/history -count=1 -v
env -u GOROOT go test ./... -count=1
git diff --check
```

Expected: all commands exit 0. The pre-existing POC files are absent from the isolated worktree.

- [x] **Step 8: Commit Task 1**

```bash
git status --short
git add internal/history/event.go internal/history/event_test.go internal/history/snapshot.go internal/history/snapshot_test.go
git commit -m "feat: define CRM history events"
```

Do not bypass hooks.

---

### Task 2: Add history reads and source plumbing to both stores

**Files:**

- Modify: `internal/storage/interface.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `cmd/crm/root.go`
- Modify: `cmd/crm/root_test.go`
- Modify: `internal/storage/sqlite.go`
- Create: `internal/storage/sqlite_history.go`
- Create: `internal/storage/sqlite_history_test.go`
- Modify: `internal/storage/markdown.go`
- Create: `internal/storage/markdown_history.go`
- Create: `internal/storage/markdown_history_test.go`
- Modify: tracked constructor calls in `internal/storage/*_test.go`, `internal/mcp/server_test.go`, and `test/integration_test.go`

**Interfaces:**

- Consumes: Task 1 `history.Event`, `history.Summary`, `history.Source`, and `history.NormalizeLimit`.
- Produces:

```go
ListHistory(entityIDOrPrefix string, limit int) ([]*history.Summary, error)
GetHistoryEvent(eventIDOrPrefix string) (*history.Event, error)
```

- Produces constructors:

```go
NewSqliteStore(dbPath string, source history.Source) (*SqliteStore, error)
NewMarkdownStore(dataDir string, source history.Source) (*MarkdownStore, error)
func (c *Config) OpenStorage(source history.Source) (storage.Storage, error)
```

- [x] **Step 1: Write failing source-plumbing tests**

Add `TestHistorySourceForCommand` around a pure helper in `cmd/crm/root.go`:

```go
func historySourceForCommand(cmd *cobra.Command) history.Source {
	if cmd.Name() == "mcp" {
		return history.SourceMCP
	}
	return history.SourceCLI
}
```

Update config tests to call `cfg.OpenStorage(history.SourceCLI)` and assert the concrete stores retain that source from same-package storage tests.

- [x] **Step 2: Run source tests and verify RED**

```bash
env -u GOROOT go test ./cmd/crm ./internal/config ./internal/storage -run 'HistorySource|OpenStorage' -count=1 -v
```

Expected: FAIL because constructors and `OpenStorage` do not accept a source.

- [x] **Step 3: Add source fields and update every tracked call site**

Add `source history.Source` and `now func() time.Time` to both stores. Constructors validate `source`, set `now: time.Now`, and keep all existing initialization. `root.go` passes `historySourceForCommand(cmd)` to `cfg.OpenStorage`.

Use `history.SourceCLI` in existing storage/config/integration tests and `history.SourceMCP` in MCP tests. Do not modify Doctor Biz's uncommitted POC files.

- [x] **Step 4: Write failing schema and history-query tests**

SQLite tests must assert both tables and these indexes exist:

```text
idx_history_event_entities_entity_id
idx_history_events_occurred_at
```

Write committed fixture events through unexported backend helpers, then require:

- full-ID and six-character entity-prefix lookup;
- relationship events returned for source and target IDs;
- event-prefix lookup;
- too-short and ambiguous prefix errors;
- an empty list for an entity ID with no committed events;
- `ErrHistoryNotFound` for a missing event lookup;
- newest-first ordering with event-ID tie breaking;
- default, explicit, and over-100 limits.

Run the same read cases against Markdown fixture event files.

- [x] **Step 5: Run query tests and verify RED**

```bash
env -u GOROOT go test ./internal/storage -run 'HistorySchema|ListHistory|GetHistoryEvent' -count=1 -v
```

Expected: FAIL because schema, files, and query methods do not exist.

- [x] **Step 6: Extend the storage contract and errors**

Add the two read methods to `Storage` and these sentinels:

```go
ErrHistoryNotFound  = errors.New("history not found")
ErrHistoryConflict  = errors.New("history recovery conflict")
ErrHistoryCorrupt   = errors.New("history event is corrupt")
```

Reuse `ErrPrefixTooShort` and `ErrAmbiguousPrefix`.

- [x] **Step 7: Implement SQLite history schema and reads**

Add the two tables from the design with a foreign key from `history_event_entities.event_id` to `history_events.id`. Keep related IDs normalized; do not duplicate them as JSON in the event table.

Implement these focused helpers in `sqlite_history.go`:

```go
type rowScanner interface {
	Scan(dest ...any) error
}

func insertSQLiteHistoryEvent(tx *sql.Tx, event *history.Event) error
func scanSQLiteHistoryEvent(row rowScanner) (*history.Event, error)
func (s *SqliteStore) relatedEntityIDs(eventID uuid.UUID) ([]uuid.UUID, error)
func (s *SqliteStore) resolveHistoryEntityID(prefix string) (uuid.UUID, error)
func (s *SqliteStore) resolveHistoryEventID(prefix string) (uuid.UUID, error)
```

For a full entity UUID, `ListHistory` queries it directly and returns an empty slice when no event matches. For a prefix, resolve exactly one related entity ID; no match also returns an empty slice. Query ordered events with the normalized limit, load related IDs, and return summaries. `GetHistoryEvent` resolves the event and returns the full snapshots; no match returns `ErrHistoryNotFound`.

- [x] **Step 8: Implement Markdown history directories and reads**

Create `_history/events` and `_history/pending` with the existing directory permission convention. Implement:

```go
func (s *MarkdownStore) historyEventsDir() string
func (s *MarkdownStore) historyPendingDir() string
func (s *MarkdownStore) readCommittedHistoryEvents() ([]*history.Event, error)
func (s *MarkdownStore) writeCommittedHistoryEvent(event *history.Event) error
```

Read every `.json` file strictly. Validate each event and wrap malformed JSON or unknown schemas with `ErrHistoryCorrupt` and the file path. Resolve prefixes across committed events, sort by `OccurredAt DESC` then ID, and apply the normalized limit. Match SQLite semantics: no entity match returns an empty list, while no event match returns `ErrHistoryNotFound`.

Constructor startup recovery is added in Task 4; at this task there are no pending files created by product mutations.

- [x] **Step 9: Run Task 2 verification**

```bash
env -u GOROOT go test ./cmd/crm ./internal/config ./internal/storage -count=1
env -u GOROOT go test ./... -count=1
git diff --check
```

Expected: all exit 0.

- [x] **Step 10: Commit Task 2**

Stage only the files listed in Task 2 after inspecting `git status`, then:

```bash
git commit -m "feat: add CRM history storage contract"
```

Do not bypass hooks.

---

### Task 3: Make SQLite mutations and history atomic

**Files:**

- Modify: `internal/storage/sqlite_history.go`
- Modify: `internal/storage/sqlite_contacts.go`
- Modify: `internal/storage/sqlite_companies.go`
- Modify: `internal/storage/sqlite_relationships.go`
- Modify: `internal/storage/sqlite_history_test.go`

**Interfaces:**

- Consumes: Task 1 snapshot functions and Task 2 SQLite schema.
- Produces:

```go
func (s *SqliteStore) commitHistoryEvent(event *history.Event) error
func applySQLiteHistoryEvent(tx *sql.Tx, event *history.Event) error
func currentSQLiteSnapshot(tx *sql.Tx, event *history.Event) (json.RawMessage, error)
```

- [x] **Step 1: Write failing mutation-history tests**

For every entity type, exercise create, meaningful update where supported, and delete. Assert exact before/after snapshots, source, related IDs, and actions. Relationship create/delete must appear under the relationship, source, and target timelines.

Add a contact no-op test:

```go
before, err := store.GetContact(contact.ID)
if err != nil { t.Fatal(err) }
before.Touch()
if err := store.UpdateContact(before); err != nil { t.Fatal(err) }
events, err := store.ListHistory(contact.ID.String(), 0)
if err != nil { t.Fatal(err) }
if len(events) != 1 { // create only
	t.Fatalf("history count = %d, want 1", len(events))
}
```

Fetch the contact again and assert its stored `updated_at` still equals the pre-update value.

- [x] **Step 2: Run SQLite mutation tests and verify RED**

```bash
env -u GOROOT go test ./internal/storage -run 'TestSqliteHistoryMutation|TestSqliteHistoryNoOp' -count=1 -v
```

Expected: FAIL because current CRUD writes no events.

- [x] **Step 3: Centralize SQLite event application**

`commitHistoryEvent` must:

1. Begin a transaction and defer rollback.
2. Read the transaction's current snapshot.
3. Require it to equal the event's `before` state, including absence for create.
4. Apply `after` or deletion by entity type.
5. Insert the event and all related IDs.
6. Commit.

`applySQLiteHistoryEvent` decodes the canonical DTO and uses parameterized INSERT, UPDATE, or DELETE statements. It returns existing entity-specific not-found errors where applicable. Do not construct SQL from event data.

- [x] **Step 4: Route all SQLite mutations through events**

Each public mutation performs only preparation and delegation:

```go
func (s *SqliteStore) UpdateContact(contact *models.Contact) error {
	current, err := s.GetContact(contact.ID)
	if err != nil { return err }
	before, err := history.SnapshotContact(current)
	if err != nil { return fmt.Errorf("snapshot current contact: %w", err) }
	after, err := history.SnapshotContact(contact)
	if err != nil { return fmt.Errorf("snapshot updated contact: %w", err) }
	changed, err := history.ChangedFields(history.EntityContact, before, after)
	if err != nil { return err }
	if len(changed) == 0 { return nil }
	event, err := history.NewEvent(
		history.EntityContact, contact.ID, []uuid.UUID{contact.ID},
		history.ActionUpdate, s.source, before, after, s.now(),
	)
	if err != nil { return err }
	return s.commitHistoryEvent(event)
}
```

Implement create/delete contact, create/update/delete company, and create/delete relationship with their exact entity type, action, snapshot function, and related IDs. Add an internal relationship lookup for delete preparation; do not add a public CRUD method.

- [x] **Step 5: Write failing rollback tests**

Create a real SQLite trigger that rejects history inserts:

```sql
CREATE TRIGGER fail_history_insert
BEFORE INSERT ON history_events
BEGIN
  SELECT RAISE(ABORT, 'history insert failed');
END;
```

Assert a contact create leaves no contact, an update preserves the old state, and a delete preserves the current state. Assert no event rows commit.

- [x] **Step 6: Run rollback tests and verify GREEN after transaction implementation**

```bash
env -u GOROOT go test ./internal/storage -run 'TestSqliteHistoryRollback' -count=1 -v
env -u GOROOT go test ./internal/storage -count=1
```

Expected: all tests pass and no new lint warnings appear.

- [x] **Step 7: Commit Task 3**

```bash
git status --short
git add internal/storage/sqlite_history.go internal/storage/sqlite_history_test.go internal/storage/sqlite_contacts.go internal/storage/sqlite_companies.go internal/storage/sqlite_relationships.go
git commit -m "feat: record atomic SQLite history"
```

Do not bypass hooks.

---

### Task 4: Add crash-recoverable Markdown history mutations

**Files:**

- Modify: `internal/storage/markdown.go`
- Modify: `internal/storage/markdown_history.go`
- Modify: `internal/storage/markdown_history_test.go`
- Modify: `internal/storage/markdown_contacts.go`
- Modify: `internal/storage/markdown_companies.go`
- Modify: `internal/storage/markdown_relationships.go`

**Interfaces:**

- Produces:

```go
func (s *MarkdownStore) commitHistoryEvent(event *history.Event) error
func (s *MarkdownStore) recoverPendingHistory() error
func (s *MarkdownStore) currentHistorySnapshot(event *history.Event) (json.RawMessage, error)
func (s *MarkdownStore) applyHistoryEvent(event *history.Event) error
func (s *MarkdownStore) finalizePendingHistory(event *history.Event) error
```

- [x] **Step 1: Write failing Markdown mutation tests**

Repeat the Task 3 event assertions against `MarkdownStore`: all create/update/delete events, semantic no-op, relationship aggregation, source, and exact snapshots. Also assert `_relationships.yaml` remains valid after each link/unlink.

- [x] **Step 2: Run Markdown mutation tests and verify RED**

```bash
env -u GOROOT go test ./internal/storage -run 'TestMarkdownHistoryMutation|TestMarkdownHistoryNoOp' -count=1 -v
```

Expected: FAIL because Markdown CRUD writes no pending or committed events.

- [x] **Step 3: Implement the pending-event protocol**

Add `mu sync.Mutex` to `MarkdownStore`. Public mutations lock once, prepare the exact event, and call `commitHistoryEvent`. That function must atomically write indented JSON to `_history/pending/<event-id>.json`, apply the event idempotently, and rename it to `_history/events/<event-id>.json`.

If the final event already exists, compare the validated event contents. Remove an identical pending duplicate; return `ErrHistoryConflict` for different contents.

Use private current-state helpers so recovery never calls public CRUD and never creates a second event.

- [x] **Step 4: Make Markdown relationship writes atomic**

Replace direct `mdstore.WriteYAML` use with YAML marshaling plus `mdstore.AtomicWrite` to the trusted `_relationships.yaml` path. Preserve the existing on-disk YAML shape.

- [x] **Step 5: Route all Markdown mutations through events**

Use the same event preparation rules as SQLite. `applyHistoryEvent` decodes the after snapshot for create/update, writes the exact state, and removes current state for delete. For relationship changes, replace or remove only the matching relationship ID while preserving unrelated entries.

Strict recovery lookups must return parse errors from current entity files. Do not use the existing list behavior that skips malformed Markdown files.

- [x] **Step 6: Write failing recovery tests**

Create real pending event files for these cases:

- current equals `before`: apply `after`, commit event;
- current equals `after`: finalize only;
- pending create with current absent;
- pending delete with current absent;
- current equals neither: constructor returns `ErrHistoryConflict` with event ID and path;
- malformed pending JSON: constructor returns `ErrHistoryCorrupt`;
- malformed committed JSON: history read returns `ErrHistoryCorrupt`.

- [x] **Step 7: Implement startup recovery**

After current directories and `_history` directories exist, `NewMarkdownStore` calls `recoverPendingHistory` before returning. Sort pending filenames for deterministic recovery. For each event, compare current state with `history.EqualSnapshots`.

Never overwrite a state that matches neither side of the event.

- [x] **Step 8: Run Task 4 verification**

```bash
env -u GOROOT go test ./internal/storage -run 'TestMarkdownHistory|TestMarkdownRecovery' -count=1 -v
env -u GOROOT go test ./internal/storage -count=1
git diff --check
```

Expected: all pass.

- [x] **Step 9: Commit Task 4**

```bash
git status --short
git add internal/storage/markdown.go internal/storage/markdown_history.go internal/storage/markdown_history_test.go internal/storage/markdown_contacts.go internal/storage/markdown_companies.go internal/storage/markdown_relationships.go
git commit -m "feat: record recoverable Markdown history"
```

Do not bypass hooks.

---

### Task 5: Prove cross-backend history parity and migration

**Files:**

- Create: `test/history_integration_test.go`
- Modify only if the new tests expose a backend defect: files from Tasks 1–4

**Interfaces:**

- Consumes the complete storage history contract.
- Produces a reusable scenario that both backends must pass unchanged.

- [x] **Step 1: Write the cross-backend scenario test**

Use a factory table for SQLite and Markdown stores, both with `history.SourceCLI`. Run this sequence with fixed entity UUIDs and explicit timestamps:

1. Create contact and company.
2. Link them.
3. Change the contact phone and company domain.
4. Perform timestamp-only no-op updates.
5. Unlink them.
6. Delete contact and company.

Assert contact and company timelines each contain their own mutations plus both relationship events. Assert relationship history survives unlink. Fetch every event by six-character prefix and verify before/after states.

- [x] **Step 2: Run parity test and verify failures are backend-specific**

```bash
env -u GOROOT go test ./test -run 'TestHistoryParity' -count=1 -v
```

Expected before fixes: any failure names the backend and mismatched event. Fix the smallest backend defect, then rerun until PASS.

- [x] **Step 3: Write legacy-data migration tests**

For SQLite, create an old-format database with current tables and one contact but no history tables, then open `NewSqliteStore`. For Markdown, write one existing contact frontmatter file before opening `NewMarkdownStore`.

Assert:

- opening creates history storage but no events;
- `ListHistory(fullID, 0)` returns an empty slice without error;
- the first meaningful update creates one update event whose `before` is the seeded record;
- no create event is fabricated.

- [x] **Step 4: Run integration verification**

```bash
env -u GOROOT go test ./test -run 'TestHistoryParity|TestHistoryLegacyData' -count=1 -v
env -u GOROOT go test ./... -count=1
```

Expected: PASS for both backends.

- [x] **Step 5: Commit Task 5**

```bash
git status --short
git add test/history_integration_test.go
# Stage a backend file only if the new contract test required a focused repair.
git commit -m "test: verify history across storage backends"
```

Do not bypass hooks.

---

### Task 6: Add CLI history inspection

**Files:**

- Create: `cmd/crm/history.go`
- Create: `cmd/crm/history_test.go`
- Create: `test/cli_history_e2e_test.go`

**Interfaces:**

- Produces:

```text
crm history <entity-id-or-prefix> [--limit 20]
crm history show <event-id-or-prefix>
```

- [x] **Step 1: Write failing command tests**

With a real temporary store, require timeline output to contain UTC RFC3339 timestamps, uppercase action, entity type, `[cli]`, an eight-character event prefix, and sorted changed fields. Require `history show` to contain event metadata plus `Before:` and `After:` pretty JSON blocks.

Test zero events prints exactly:

```text
No history found.
```

- [x] **Step 2: Run command tests and verify RED**

```bash
env -u GOROOT go test ./cmd/crm -run 'TestHistory' -count=1 -v
```

Expected: FAIL because the command is not registered.

- [x] **Step 3: Implement the Cobra commands**

Register `historyCmd` on the root and `historyShowCmd` beneath it. Use `history.NormalizeLimit`; do not duplicate limit rules. Resolve identifiers only through storage history methods.

Use pure formatting helpers so exact output can be unit tested:

```go
func formatHistorySummary(summary *history.Summary) string
func formatHistoryEvent(event *history.Event) (string, error)
```

The timeline sorts changed fields before joining them. Full snapshots use `json.Indent`; null sides print `null`.

- [x] **Step 4: Write the built-binary end-to-end test**

Build `./cmd/crm` once into `t.TempDir()`. For each backend, create temporary `XDG_CONFIG_HOME` and `XDG_DATA_HOME`, write a real config JSON, then invoke the binary to add, edit, link, unlink, delete, list history, and show one event. Parse UUIDs from command output; do not inject model data directly.

Assert history works after deletion and every CLI event source is `cli`.

- [x] **Step 5: Run CLI verification**

```bash
env -u GOROOT go test ./cmd/crm ./test -run 'TestHistory|TestCLIHistory' -count=1 -v
```

Expected: PASS for SQLite and Markdown.

- [x] **Step 6: Commit Task 6**

```bash
git status --short
git add cmd/crm/history.go cmd/crm/history_test.go test/cli_history_e2e_test.go
git commit -m "feat: add CRM history commands"
```

Do not bypass hooks.

---

### Task 7: Add MCP history tools over a real transport

**Files:**

- Create: `internal/mcp/history.go`
- Create: `internal/mcp/history_test.go`
- Modify: `internal/mcp/tools.go`

**Interfaces:**

- Produces tools:
  - `list_history` with required `entity_id` and optional integer `limit`.
  - `get_history_event` with required `event_id`.

- [x] **Step 1: Write failing tool-list and transport tests**

Update the tool-list expectation from 12 to 14 and require both names.

Create a real client/server session using the installed SDK API:

```go
serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
serverSession, err := server.server.Connect(ctx, serverTransport, nil)
if err != nil { t.Fatal(err) }
defer serverSession.Close()

client := sdkmcp.NewClient(
	&sdkmcp.Implementation{Name: "crm-history-test", Version: "1.0.0"},
	nil,
)
clientSession, err := client.Connect(ctx, clientTransport, nil)
if err != nil { t.Fatal(err) }
defer clientSession.Close()
```

Call `add_contact`, `update_contact`, `list_history`, and `get_history_event` through `ClientSession.CallTool`. Use a real SQLite store opened with `history.SourceMCP`. Assert returned events use source `mcp` and carry exact snapshots.

- [x] **Step 2: Run MCP tests and verify RED**

```bash
env -u GOROOT go test ./internal/mcp -run 'TestServerListTools|TestHistoryToolsTransport' -count=1 -v
```

Expected: FAIL because the tools are absent and the count is 12.

- [x] **Step 3: Implement definitions and handlers**

Use the existing raw JSON schema and `errResult`/`jsonResult` conventions. `list_history` normalizes the limit through storage/history behavior and returns summaries. `get_history_event` returns the full event. Empty identifiers return MCP error results before storage calls.

Register both tools after the existing 12 tools and update the ABOUTME/count comment to 14.

- [x] **Step 4: Run MCP verification**

```bash
env -u GOROOT go test ./internal/mcp -count=1 -v
env -u GOROOT go test -race ./internal/mcp -count=1
```

Expected: PASS with a clean race run.

- [x] **Step 5: Commit Task 7**

```bash
git status --short
git add internal/mcp/history.go internal/mcp/history_test.go internal/mcp/tools.go
git commit -m "feat: expose CRM history through MCP"
```

Do not bypass hooks.

---

### Task 8: Document history, retention, and local limits

**Files:**

- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `gotchas.md`
- Modify: `cmd/crm/skill/SKILL.md`

**Interfaces:**

- Documents the exact CLI commands, 14 MCP tools, future-only history, indefinite retention, CLI/MCP source, relationship aggregation, and deletion retention.

- [x] **Step 1: Verify live help before writing prose**

```bash
env -u GOROOT go run ./cmd/crm history --help
env -u GOROOT go run ./cmd/crm history show --help
```

Expected: both exit 0 and match Task 6 flags and arguments.

- [x] **Step 2: Update README usage and MCP sections**

Add two copyable examples:

```bash
crm history <entity-id-or-prefix>
crm history show <event-id-or-prefix>
```

State that contact/company timelines include relationship changes, history begins after upgrade, and deletes retain snapshots. Change the MCP count from 12 to 14 and name both history tools.

- [x] **Step 3: Update contributor guidance and bundled skill**

Add storage layout and recovery facts to `CLAUDE.md`. Add durable gotchas for immutable PII retention and Markdown's single-process writer boundary. Teach the bundled skill when to call `list_history` and `get_history_event`; do not claim restore or purge exists.

- [x] **Step 4: Audit prose against code**

```bash
rg -n '12 tools|restore|undo|purge|history' README.md CLAUDE.md gotchas.md cmd/crm/skill/SKILL.md
git diff --check
```

Expected: no stale 12-tool claim; restore/undo/purge appear only as explicit non-features if mentioned.

- [x] **Step 5: Commit Task 8**

```bash
git status --short
git add README.md CLAUDE.md gotchas.md cmd/crm/skill/SKILL.md
git commit -m "docs: document CRM history"
```

Do not bypass hooks.

---

### Task 9: Run final verification and review gates

**Files:**

- Verify all changed files.
- Modify only the smallest in-scope file if a check exposes a defect.

- [x] **Step 1: Run canonical checks**

```bash
env -u GOROOT make check
env -u GOROOT go vet ./...
env -u GOROOT go test -race ./...
```

Expected: all exit 0 with no new warnings or errors.

- [x] **Step 2: Run focused history tests uncached**

```bash
env -u GOROOT go test ./internal/history ./internal/storage ./internal/mcp ./cmd/crm ./test -count=1 -v
```

Expected: all history unit, backend, integration, CLI E2E, and MCP transport tests pass.

- [x] **Step 3: Verify real CLI behavior for both backends**

Use temporary XDG directories and real commands to create, edit, link, unlink, delete, list history, and inspect an event. Confirm sources, relationship aggregation, and deleted-state inspection. Save command output to a temporary log and remove it afterward.

- [x] **Step 4: Verify release behavior**

```bash
goreleaser check
env -u GOROOT goreleaser release --snapshot --clean
```

Expected: `goreleaser check` exits 2 only for the documented deprecated `brews` field. Snapshot release exits 0, archives contain README, and `go.mod`/`go.sum` hashes remain unchanged. Remove generated `crm` and `dist/` afterward.

- [x] **Step 5: Run mandatory review gates**

Invoke `superpowers:requesting-code-review` for the cumulative diff, fix every critical or important finding, and rerun affected tests. Invoke `fresh-eyes-review` after fixes. Finally invoke `superpowers:verification-before-completion` and repeat Steps 1–4 at the final commit.

- [x] **Step 6: Verify repository hygiene**

```bash
git diff --check
git status --short --branch
git log --oneline --decorate -15
```

Expected: the isolated history worktree is clean. No Doctor Biz POC file appears in `git diff <history-base>..HEAD`.

- [ ] **Step 7: Finish the branch**

Invoke `superpowers:finishing-a-development-branch`. Present the standard merge/PR/keep/discard choices. Do not merge into the dirty primary checkout until Doctor Biz chooses and the POC overlap is reviewed.
