// ABOUTME: Persists immutable history events in SQLite and serves timeline queries.
// ABOUTME: Resolves committed entity and event prefixes while validating stored data.
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
)

const sqliteHistoryColumns = `
	history_events.id, history_events.schema_version, history_events.entity_type,
	history_events.entity_id, history_events.action, history_events.source,
	history_events.occurred_at, history_events.before_json, history_events.after_json`

type rowScanner interface {
	Scan(dest ...any) error
}

func decodeSQLiteJSON(value string, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader([]byte(value)))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if fields, ok := destination.(*map[string]any); ok {
		for key, field := range *fields {
			(*fields)[key] = compatibleSQLiteJSONNumber(field)
		}
	}
	return nil
}

func compatibleSQLiteJSONNumber(value any) any {
	switch value := value.(type) {
	case json.Number:
		parsed, err := strconv.ParseFloat(value.String(), 64)
		if err != nil {
			return value
		}
		encoded, err := json.Marshal(parsed)
		if err == nil && string(encoded) == value.String() {
			return parsed
		}
		return value
	case []any:
		for index, item := range value {
			value[index] = compatibleSQLiteJSONNumber(item)
		}
		return value
	case map[string]any:
		for key, item := range value {
			value[key] = compatibleSQLiteJSONNumber(item)
		}
		return value
	default:
		return value
	}
}

func (s *SqliteStore) commitHistoryEvent(event *history.Event) error {
	tx, err := s.db.Begin()
	if err != nil {
		return wrapSQLiteHistoryError("begin history transaction", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate history event: %w", err)
	}
	current, err := currentSQLiteSnapshot(tx, event)
	if err != nil {
		return err
	}
	equal, err := history.EqualSnapshots(event.EntityType, current, event.Before)
	if err != nil {
		return fmt.Errorf("compare current %s snapshot: %w", event.EntityType, err)
	}
	if !equal {
		return fmt.Errorf("%w: current %s %s does not match event before snapshot", ErrHistoryConflict, event.EntityType, event.EntityID)
	}
	if err := applySQLiteHistoryEvent(tx, event); err != nil {
		return normalizeSQLiteHistoryConflict(err)
	}
	if err := insertSQLiteHistoryEvent(tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrapSQLiteHistoryError("commit history transaction", err)
	}
	return nil
}

// sqliteCodeError matches modernc.org/sqlite.Error without coupling history logic to driver internals.
type sqliteCodeError interface {
	error
	Code() int
}

func wrapSQLiteHistoryError(operation string, err error) error {
	return normalizeSQLiteHistoryConflict(fmt.Errorf("%s: %w", operation, err))
}

func normalizeSQLiteHistoryConflict(err error) error {
	const sqliteBusyPrimaryCode = 5
	var sqliteError sqliteCodeError
	if errors.As(err, &sqliteError) && sqliteError.Code()&0xff == sqliteBusyPrimaryCode {
		return fmt.Errorf("%w: %w", ErrHistoryConflict, err)
	}
	return err
}

func currentSQLiteSnapshot(tx *sql.Tx, event *history.Event) (json.RawMessage, error) {
	switch event.EntityType {
	case history.EntityContact:
		contact, err := scanContact(tx.QueryRow(`
			SELECT id, name, email, phone, fields, tags, created_at, updated_at
			FROM contacts WHERE id = ?`, event.EntityID.String()))
		if errors.Is(err, ErrContactNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read current contact snapshot: %w", err)
		}
		snapshot, err := sqliteContactSnapshot(contact)
		if err != nil {
			return nil, fmt.Errorf("snapshot current contact: %w", err)
		}
		return snapshot, nil
	case history.EntityCompany:
		company, err := scanCompany(tx.QueryRow(`
			SELECT id, name, domain, fields, tags, created_at, updated_at
			FROM companies WHERE id = ?`, event.EntityID.String()))
		if errors.Is(err, ErrCompanyNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read current company snapshot: %w", err)
		}
		snapshot, err := sqliteCompanySnapshot(company)
		if err != nil {
			return nil, fmt.Errorf("snapshot current company: %w", err)
		}
		return snapshot, nil
	case history.EntityRelationship:
		relationship, err := scanRelationship(tx.QueryRow(`
			SELECT id, source_id, target_id, type, context, created_at
			FROM relationships WHERE id = ?`, event.EntityID.String()))
		if errors.Is(err, ErrRelationshipNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read current relationship snapshot: %w", err)
		}
		snapshot, err := sqliteRelationshipSnapshot(relationship)
		if err != nil {
			return nil, fmt.Errorf("snapshot current relationship: %w", err)
		}
		return snapshot, nil
	default:
		return nil, fmt.Errorf("unsupported history entity type %q", event.EntityType)
	}
}

func applySQLiteHistoryEvent(tx *sql.Tx, event *history.Event) error {
	var err error
	switch event.EntityType {
	case history.EntityContact:
		err = applySQLiteContactHistoryEvent(tx, event)
	case history.EntityCompany:
		err = applySQLiteCompanyHistoryEvent(tx, event)
	case history.EntityRelationship:
		err = applySQLiteRelationshipHistoryEvent(tx, event)
	default:
		return fmt.Errorf("unsupported history entity type %q", event.EntityType)
	}
	if err != nil {
		return fmt.Errorf("apply %s %s %s: %w", event.Action, event.EntityType, event.EntityID, err)
	}
	return nil
}

func applySQLiteContactHistoryEvent(tx *sql.Tx, event *history.Event) error {
	if event.Action == history.ActionDelete {
		return execSQLiteHistoryMutation(
			tx,
			"DELETE FROM contacts WHERE id = ?",
			ErrContactNotFound,
			event.EntityID.String(),
		)
	}
	contact, err := history.ContactFromSnapshot(event.After)
	if err != nil {
		return fmt.Errorf("decode contact history snapshot: %w", err)
	}
	fields, err := json.Marshal(contact.Fields)
	if err != nil {
		return fmt.Errorf("marshal contact fields: %w", err)
	}
	tags, err := json.Marshal(contact.Tags)
	if err != nil {
		return fmt.Errorf("marshal contact tags: %w", err)
	}
	if event.Action == history.ActionCreate {
		_, err = tx.Exec(`
			INSERT INTO contacts (id, name, email, phone, fields, tags, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			contact.ID.String(), contact.Name, contact.Email, contact.Phone,
			string(fields), string(tags), contact.CreatedAt.UTC(), contact.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert contact: %w", err)
		}
		return nil
	}
	return execSQLiteHistoryMutation(
		tx,
		`UPDATE contacts SET name=?, email=?, phone=?, fields=?, tags=?, updated_at=? WHERE id=?`,
		ErrContactNotFound,
		contact.Name, contact.Email, contact.Phone, string(fields), string(tags),
		contact.UpdatedAt.UTC(), contact.ID.String(),
	)
}

func applySQLiteCompanyHistoryEvent(tx *sql.Tx, event *history.Event) error {
	if event.Action == history.ActionDelete {
		return execSQLiteHistoryMutation(
			tx,
			"DELETE FROM companies WHERE id = ?",
			ErrCompanyNotFound,
			event.EntityID.String(),
		)
	}
	company, err := history.CompanyFromSnapshot(event.After)
	if err != nil {
		return fmt.Errorf("decode company history snapshot: %w", err)
	}
	fields, err := json.Marshal(company.Fields)
	if err != nil {
		return fmt.Errorf("marshal company fields: %w", err)
	}
	tags, err := json.Marshal(company.Tags)
	if err != nil {
		return fmt.Errorf("marshal company tags: %w", err)
	}
	if event.Action == history.ActionCreate {
		_, err = tx.Exec(`
			INSERT INTO companies (id, name, domain, fields, tags, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			company.ID.String(), company.Name, company.Domain, string(fields), string(tags),
			company.CreatedAt.UTC(), company.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert company: %w", err)
		}
		return nil
	}
	return execSQLiteHistoryMutation(
		tx,
		`UPDATE companies SET name=?, domain=?, fields=?, tags=?, updated_at=? WHERE id=?`,
		ErrCompanyNotFound,
		company.Name, company.Domain, string(fields), string(tags), company.UpdatedAt.UTC(), company.ID.String(),
	)
}

func applySQLiteRelationshipHistoryEvent(tx *sql.Tx, event *history.Event) error {
	if event.Action == history.ActionDelete {
		return execSQLiteHistoryMutation(
			tx,
			"DELETE FROM relationships WHERE id = ?",
			ErrRelationshipNotFound,
			event.EntityID.String(),
		)
	}
	if event.Action != history.ActionCreate {
		return fmt.Errorf("unsupported relationship history action %q", event.Action)
	}
	relationship, err := history.RelationshipFromSnapshot(event.After)
	if err != nil {
		return fmt.Errorf("decode relationship history snapshot: %w", err)
	}
	_, err = tx.Exec(`
		INSERT INTO relationships (id, source_id, target_id, type, context, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		relationship.ID.String(), relationship.SourceID.String(), relationship.TargetID.String(),
		relationship.Type, relationship.Context, relationship.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert relationship: %w", err)
	}
	return nil
}

func execSQLiteHistoryMutation(
	tx *sql.Tx,
	query string,
	notFound error,
	args ...any,
) error {
	result, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if count == 0 {
		return notFound
	}
	return nil
}

func insertSQLiteHistoryEvent(tx *sql.Tx, event *history.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate history event: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO history_events (
			id, schema_version, entity_type, entity_id, action, source,
			occurred_at, before_json, after_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID.String(), event.SchemaVersion, event.EntityType, event.EntityID.String(),
		event.Action, event.Source, event.OccurredAt.UTC(), nullableHistoryJSON(event.Before),
		nullableHistoryJSON(event.After),
	); err != nil {
		return fmt.Errorf("insert history event: %w", err)
	}
	for _, entityID := range event.RelatedEntityIDs {
		if _, err := tx.Exec(
			"INSERT INTO history_event_entities (event_id, entity_id) VALUES (?, ?)",
			event.ID.String(),
			entityID.String(),
		); err != nil {
			return fmt.Errorf("insert history event entity: %w", err)
		}
	}
	return nil
}

func nullableHistoryJSON(snapshot json.RawMessage) any {
	trimmed := bytes.TrimSpace(snapshot)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	return string(snapshot)
}

func scanSQLiteHistoryEvent(row rowScanner) (*history.Event, error) {
	var (
		event                      history.Event
		id, entityID               string
		entityType, action, source string
		beforeJSON, afterJSON      sql.NullString
	)
	if err := row.Scan(
		&id,
		&event.SchemaVersion,
		&entityType,
		&entityID,
		&action,
		&source,
		&event.OccurredAt,
		&beforeJSON,
		&afterJSON,
	); err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse history event ID %q: %w", id, err)
	}
	parsedEntityID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("parse history entity ID %q: %w", entityID, err)
	}
	event.ID = parsedID
	event.EntityType = history.EntityType(entityType)
	event.EntityID = parsedEntityID
	event.Action = history.Action(action)
	event.Source = history.Source(source)
	if beforeJSON.Valid {
		event.Before = json.RawMessage(beforeJSON.String)
	}
	if afterJSON.Valid {
		event.After = json.RawMessage(afterJSON.String)
	}
	return &event, nil
}

func (s *SqliteStore) relatedEntityIDs(eventID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.db.Query(`
		SELECT entity_id
		FROM history_event_entities
		WHERE event_id = ?
		ORDER BY entity_id`, eventID.String())
	if err != nil {
		return nil, fmt.Errorf("query history event entities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []uuid.UUID
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			return nil, fmt.Errorf("scan history event entity: %w", err)
		}
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("%w: parse history event entity ID %q: %w", ErrHistoryCorrupt, rawID, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history event entities: %w", err)
	}
	return ids, nil
}

func (s *SqliteStore) resolveHistoryEntityID(prefix string) (uuid.UUID, error) {
	if len(prefix) < 6 {
		return uuid.Nil, ErrPrefixTooShort
	}
	prefix = normalizeHistoryIDPrefix(prefix)
	rows, err := s.db.Query(`
		SELECT DISTINCT entity_id
		FROM history_event_entities
		WHERE substr(entity_id, 1, length(?)) = ?
		ORDER BY entity_id
		LIMIT 2`, prefix, prefix)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve history entity prefix: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return resolveSQLiteHistoryPrefix(rows, false)
}

func (s *SqliteStore) resolveHistoryEventID(prefix string) (uuid.UUID, error) {
	if len(prefix) < 6 {
		return uuid.Nil, ErrPrefixTooShort
	}
	if id, err := uuid.Parse(prefix); err == nil {
		prefix = id.String()
	} else {
		prefix = normalizeHistoryIDPrefix(prefix)
	}
	rows, err := s.db.Query(`
		SELECT id
		FROM history_events
		WHERE substr(id, 1, length(?)) = ?
		ORDER BY id
		LIMIT 2`, prefix, prefix)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve history event prefix: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return resolveSQLiteHistoryPrefix(rows, true)
}

func resolveSQLiteHistoryPrefix(rows *sql.Rows, event bool) (uuid.UUID, error) {
	var matches []string
	for rows.Next() {
		var match string
		if err := rows.Scan(&match); err != nil {
			return uuid.Nil, err
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return uuid.Nil, err
	}
	if len(matches) == 0 {
		if event {
			return uuid.Nil, ErrHistoryNotFound
		}
		return uuid.Nil, nil
	}
	if len(matches) > 1 {
		return uuid.Nil, ErrAmbiguousPrefix
	}
	id, err := uuid.Parse(matches[0])
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: parse stored history ID %q: %w", ErrHistoryCorrupt, matches[0], err)
	}
	return id, nil
}

func (s *SqliteStore) ListHistory(entityIDOrPrefix string, limit int) ([]*history.Summary, error) {
	normalizedLimit, err := history.NormalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	entityID, err := uuid.Parse(entityIDOrPrefix)
	if err != nil {
		entityID, err = s.resolveHistoryEntityID(entityIDOrPrefix)
		if err != nil {
			return nil, err
		}
		if entityID == uuid.Nil {
			return []*history.Summary{}, nil
		}
	}

	rows, err := s.db.Query(`
		SELECT `+sqliteHistoryColumns+`
		FROM history_events
		JOIN history_event_entities ON history_event_entities.event_id = history_events.id
		WHERE history_event_entities.entity_id = ?
		ORDER BY occurred_at DESC, history_events.id
		LIMIT ?`, entityID.String(), normalizedLimit)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}

	var events []*history.Event
	for rows.Next() {
		event, scanErr := scanSQLiteHistoryEvent(rows)
		if scanErr != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("%w: scan history event: %w", ErrHistoryCorrupt, scanErr)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate history events: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close history events: %w", err)
	}

	summaries := make([]*history.Summary, 0, len(events))
	for _, event := range events {
		if err := s.loadAndValidateSQLiteHistoryEvent(event); err != nil {
			return nil, err
		}
		summary, err := event.Summary()
		if err != nil {
			return nil, fmt.Errorf("%w: summarize history event %s: %w", ErrHistoryCorrupt, event.ID, err)
		}
		summaries = append(summaries, &summary)
	}
	return summaries, nil
}

func (s *SqliteStore) GetHistoryEvent(eventIDOrPrefix string) (*history.Event, error) {
	eventID, err := s.resolveHistoryEventID(eventIDOrPrefix)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRow(`SELECT `+sqliteHistoryColumns+` FROM history_events WHERE id = ?`, eventID.String())
	event, err := scanSQLiteHistoryEvent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHistoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: scan history event: %w", ErrHistoryCorrupt, err)
	}
	if err := s.loadAndValidateSQLiteHistoryEvent(event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *SqliteStore) loadAndValidateSQLiteHistoryEvent(event *history.Event) error {
	related, err := s.relatedEntityIDs(event.ID)
	if err != nil {
		return fmt.Errorf("load related entities for history event %s: %w", event.ID, err)
	}
	event.RelatedEntityIDs = related
	if err := event.Validate(); err != nil {
		return fmt.Errorf("%w: validate history event %s: %w", ErrHistoryCorrupt, event.ID, err)
	}
	return nil
}
