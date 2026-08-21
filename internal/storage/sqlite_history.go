// ABOUTME: Persists immutable history events in SQLite and serves timeline queries.
// ABOUTME: Resolves committed entity and event prefixes while validating stored data.
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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
