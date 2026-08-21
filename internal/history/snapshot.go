// ABOUTME: Converts CRM models to stable version-one history snapshots.
// ABOUTME: Compares canonical snapshots and reports changed domain fields.
package history

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

type contactSnapshot struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Phone     string         `json:"phone"`
	Fields    map[string]any `json:"fields"`
	Tags      []string       `json:"tags"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type companySnapshot struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Domain    string         `json:"domain"`
	Fields    map[string]any `json:"fields"`
	Tags      []string       `json:"tags"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type relationshipSnapshot struct {
	ID        uuid.UUID `json:"id"`
	SourceID  uuid.UUID `json:"source_id"`
	TargetID  uuid.UUID `json:"target_id"`
	Type      string    `json:"type"`
	Context   string    `json:"context"`
	CreatedAt time.Time `json:"created_at"`
}

func SnapshotContact(contact *models.Contact) (json.RawMessage, error) {
	if contact == nil {
		return nil, errors.New("cannot snapshot a nil contact")
	}
	return marshalSnapshot(contactSnapshot{
		ID:        contact.ID,
		Name:      contact.Name,
		Email:     contact.Email,
		Phone:     contact.Phone,
		Fields:    contact.Fields,
		Tags:      contact.Tags,
		CreatedAt: contact.CreatedAt,
		UpdatedAt: contact.UpdatedAt,
	})
}

func ContactFromSnapshot(snapshot json.RawMessage) (*models.Contact, error) {
	var dto contactSnapshot
	if err := unmarshalSnapshot(snapshot, &dto); err != nil {
		return nil, fmt.Errorf("decode contact snapshot: %w", err)
	}
	return &models.Contact{
		ID:        dto.ID,
		Name:      dto.Name,
		Email:     dto.Email,
		Phone:     dto.Phone,
		Fields:    dto.Fields,
		Tags:      dto.Tags,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}, nil
}

func SnapshotCompany(company *models.Company) (json.RawMessage, error) {
	if company == nil {
		return nil, errors.New("cannot snapshot a nil company")
	}
	return marshalSnapshot(companySnapshot{
		ID:        company.ID,
		Name:      company.Name,
		Domain:    company.Domain,
		Fields:    company.Fields,
		Tags:      company.Tags,
		CreatedAt: company.CreatedAt,
		UpdatedAt: company.UpdatedAt,
	})
}

func CompanyFromSnapshot(snapshot json.RawMessage) (*models.Company, error) {
	var dto companySnapshot
	if err := unmarshalSnapshot(snapshot, &dto); err != nil {
		return nil, fmt.Errorf("decode company snapshot: %w", err)
	}
	return &models.Company{
		ID:        dto.ID,
		Name:      dto.Name,
		Domain:    dto.Domain,
		Fields:    dto.Fields,
		Tags:      dto.Tags,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}, nil
}

func SnapshotRelationship(relationship *models.Relationship) (json.RawMessage, error) {
	if relationship == nil {
		return nil, errors.New("cannot snapshot a nil relationship")
	}
	return marshalSnapshot(relationshipSnapshot{
		ID:        relationship.ID,
		SourceID:  relationship.SourceID,
		TargetID:  relationship.TargetID,
		Type:      relationship.Type,
		Context:   relationship.Context,
		CreatedAt: relationship.CreatedAt,
	})
}

func RelationshipFromSnapshot(snapshot json.RawMessage) (*models.Relationship, error) {
	var dto relationshipSnapshot
	if err := unmarshalSnapshot(snapshot, &dto); err != nil {
		return nil, fmt.Errorf("decode relationship snapshot: %w", err)
	}
	return &models.Relationship{
		ID:        dto.ID,
		SourceID:  dto.SourceID,
		TargetID:  dto.TargetID,
		Type:      dto.Type,
		Context:   dto.Context,
		CreatedAt: dto.CreatedAt,
	}, nil
}

func EqualSnapshots(entityType EntityType, left, right json.RawMessage) (bool, error) {
	if !validEntityType(entityType) {
		return false, fmt.Errorf("invalid history entity type %q", entityType)
	}
	leftIsNull := isNullSnapshot(left)
	rightIsNull := isNullSnapshot(right)
	if leftIsNull && rightIsNull {
		return true, nil
	}
	if leftIsNull || rightIsNull {
		nonNull := left
		if leftIsNull {
			nonNull = right
		}
		if _, err := canonicalSnapshot(entityType, nonNull); err != nil {
			return false, err
		}
		return false, nil
	}

	canonicalLeft, err := canonicalSnapshot(entityType, left)
	if err != nil {
		return false, err
	}
	canonicalRight, err := canonicalSnapshot(entityType, right)
	if err != nil {
		return false, err
	}
	return bytes.Equal(canonicalLeft, canonicalRight), nil
}

func ChangedFields(entityType EntityType, before, after json.RawMessage) ([]string, error) {
	if !validEntityType(entityType) {
		return nil, fmt.Errorf("invalid history entity type %q", entityType)
	}
	beforeFields, err := snapshotObject(before)
	if err != nil {
		return nil, fmt.Errorf("decode before snapshot: %w", err)
	}
	afterFields, err := snapshotObject(after)
	if err != nil {
		return nil, fmt.Errorf("decode after snapshot: %w", err)
	}
	delete(beforeFields, "updated_at")
	delete(afterFields, "updated_at")

	keys := make(map[string]struct{}, len(beforeFields)+len(afterFields))
	for key := range beforeFields {
		keys[key] = struct{}{}
	}
	for key := range afterFields {
		keys[key] = struct{}{}
	}

	changed := make([]string, 0, len(keys))
	for key := range keys {
		if !reflect.DeepEqual(beforeFields[key], afterFields[key]) {
			changed = append(changed, key)
		}
	}
	slices.Sort(changed)
	return changed, nil
}

func (e *Event) Summary() (Summary, error) {
	if err := e.Validate(); err != nil {
		return Summary{}, err
	}
	changed, err := ChangedFields(e.EntityType, e.Before, e.After)
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		ID:            e.ID,
		EntityType:    e.EntityType,
		EntityID:      e.EntityID,
		Action:        e.Action,
		Source:        e.Source,
		OccurredAt:    e.OccurredAt,
		ChangedFields: changed,
	}, nil
}

func marshalSnapshot(snapshot any) (json.RawMessage, error) {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("encode history snapshot: %w", err)
	}
	return json.RawMessage(encoded), nil
}

func unmarshalSnapshot(snapshot json.RawMessage, destination any) error {
	if isNullSnapshot(snapshot) {
		return errors.New("history snapshot is null")
	}
	if err := json.Unmarshal(snapshot, destination); err != nil {
		return err
	}
	return nil
}

func canonicalSnapshot(entityType EntityType, snapshot json.RawMessage) (json.RawMessage, error) {
	switch entityType {
	case EntityContact:
		contact, err := ContactFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		return SnapshotContact(contact)
	case EntityCompany:
		company, err := CompanyFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		return SnapshotCompany(company)
	case EntityRelationship:
		relationship, err := RelationshipFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		return SnapshotRelationship(relationship)
	default:
		return nil, fmt.Errorf("invalid history entity type %q", entityType)
	}
}

func snapshotObject(snapshot json.RawMessage) (map[string]any, error) {
	if isNullSnapshot(snapshot) {
		return make(map[string]any), nil
	}
	var object map[string]any
	if err := json.Unmarshal(snapshot, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return make(map[string]any), nil
	}
	return object, nil
}
