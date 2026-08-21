// ABOUTME: Defines immutable CRM history events and validates their wire contract.
// ABOUTME: Normalizes event IDs, related entities, timestamps, and query limits.
package history

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100

	currentSchemaVersion = 1
)

type EntityType string

const (
	EntityContact      EntityType = "contact"
	EntityCompany      EntityType = "company"
	EntityRelationship EntityType = "relationship"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

type Source string

const (
	SourceCLI Source = "cli"
	SourceMCP Source = "mcp"
)

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

func NewEvent(
	entityType EntityType,
	entityID uuid.UUID,
	related []uuid.UUID,
	action Action,
	source Source,
	before json.RawMessage,
	after json.RawMessage,
	occurredAt time.Time,
) (*Event, error) {
	event := &Event{
		ID:               uuid.New(),
		SchemaVersion:    currentSchemaVersion,
		EntityType:       entityType,
		EntityID:         entityID,
		RelatedEntityIDs: canonicalRelatedIDs(related),
		Action:           action,
		Source:           source,
		OccurredAt:       occurredAt.UTC(),
		Before:           before,
		After:            after,
	}
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return event, nil
}

func (e *Event) Validate() error {
	if e == nil {
		return errors.New("history event is nil")
	}
	if e.ID == uuid.Nil {
		return errors.New("history event ID is required")
	}
	if e.SchemaVersion != currentSchemaVersion {
		return fmt.Errorf("unsupported history schema version %d", e.SchemaVersion)
	}
	if !validEntityType(e.EntityType) {
		return fmt.Errorf("invalid history entity type %q", e.EntityType)
	}
	if e.EntityID == uuid.Nil {
		return errors.New("history entity ID is required")
	}
	if !slices.Contains(e.RelatedEntityIDs, e.EntityID) {
		return errors.New("history related entity IDs must include the subject")
	}
	if !validAction(e.Action) {
		return fmt.Errorf("invalid history action %q", e.Action)
	}
	if !validSource(e.Source) {
		return fmt.Errorf("invalid history source %q", e.Source)
	}
	if e.OccurredAt.IsZero() {
		return errors.New("history occurrence time is required")
	}
	if err := validateSnapshots(e.Action, e.Before, e.After); err != nil {
		return err
	}
	return nil
}

func NormalizeLimit(limit int) (int, error) {
	if limit <= 0 {
		return DefaultLimit, nil
	}
	if limit > MaxLimit {
		return 0, fmt.Errorf("history limit must not exceed %d", MaxLimit)
	}
	return limit, nil
}

func canonicalRelatedIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	canonical := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		canonical = append(canonical, id)
	}
	slices.SortFunc(canonical, func(left, right uuid.UUID) int {
		return bytes.Compare(left[:], right[:])
	})
	return canonical
}

func validEntityType(entityType EntityType) bool {
	return entityType == EntityContact || entityType == EntityCompany || entityType == EntityRelationship
}

func validAction(action Action) bool {
	return action == ActionCreate || action == ActionUpdate || action == ActionDelete
}

func validSource(source Source) bool {
	return source == SourceCLI || source == SourceMCP
}

func validateSnapshots(action Action, before, after json.RawMessage) error {
	beforeIsNull := isNullSnapshot(before)
	afterIsNull := isNullSnapshot(after)

	switch action {
	case ActionCreate:
		if !beforeIsNull || afterIsNull {
			return errors.New("create event requires a null before and a non-null after snapshot")
		}
	case ActionUpdate:
		if beforeIsNull || afterIsNull {
			return errors.New("update event requires before and after snapshots")
		}
	case ActionDelete:
		if beforeIsNull || !afterIsNull {
			return errors.New("delete event requires a non-null before and a null after snapshot")
		}
	}

	if !beforeIsNull && !json.Valid(before) {
		return errors.New("history before snapshot is invalid JSON")
	}
	if !afterIsNull && !json.Valid(after) {
		return errors.New("history after snapshot is invalid JSON")
	}
	return nil
}

func isNullSnapshot(snapshot json.RawMessage) bool {
	trimmed := bytes.TrimSpace(snapshot)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
