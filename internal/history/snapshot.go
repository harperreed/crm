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
	"github.com/harperreed/crm/v2/internal/models"
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
	beforeFields, err := strictSnapshotObject(entityType, before)
	if err != nil {
		return nil, fmt.Errorf("decode before snapshot: %w", err)
	}
	afterFields, err := strictSnapshotObject(entityType, after)
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
	requiredKeys, err := snapshotKeys(destination)
	if err != nil {
		return err
	}
	if err := validateSnapshotKeys(snapshot, requiredKeys); err != nil {
		return err
	}
	decoder := newSnapshotDecoder(snapshot)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	return validateSnapshotStructure(destination)
}

func snapshotKeys(destination any) ([]string, error) {
	switch destination.(type) {
	case *contactSnapshot:
		return []string{"id", "name", "email", "phone", "fields", "tags", "created_at", "updated_at"}, nil
	case *companySnapshot:
		return []string{"id", "name", "domain", "fields", "tags", "created_at", "updated_at"}, nil
	case *relationshipSnapshot:
		return []string{"id", "source_id", "target_id", "type", "context", "created_at"}, nil
	default:
		return nil, fmt.Errorf("unsupported history snapshot destination %T", destination)
	}
}

func validateSnapshotKeys(snapshot json.RawMessage, required []string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(snapshot, &object); err != nil {
		return err
	}
	if object == nil {
		return errors.New("history snapshot must be a JSON object")
	}
	allowed := make(map[string]struct{}, len(required))
	for _, key := range required {
		allowed[key] = struct{}{}
		if _, ok := object[key]; !ok {
			return fmt.Errorf("history snapshot is missing field %q", key)
		}
	}
	for key := range object {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("history snapshot contains unknown field %q", key)
		}
	}
	return nil
}

func validateSnapshotStructure(destination any) error {
	switch snapshot := destination.(type) {
	case *contactSnapshot:
		if snapshot.ID == uuid.Nil || snapshot.CreatedAt.IsZero() || snapshot.UpdatedAt.IsZero() {
			return errors.New("contact snapshot requires ID, created_at, and updated_at")
		}
	case *companySnapshot:
		if snapshot.ID == uuid.Nil || snapshot.CreatedAt.IsZero() || snapshot.UpdatedAt.IsZero() {
			return errors.New("company snapshot requires ID, created_at, and updated_at")
		}
	case *relationshipSnapshot:
		if snapshot.ID == uuid.Nil || snapshot.SourceID == uuid.Nil || snapshot.TargetID == uuid.Nil || snapshot.CreatedAt.IsZero() {
			return errors.New("relationship snapshot requires ID, source_id, target_id, and created_at")
		}
	default:
		return fmt.Errorf("unsupported history snapshot destination %T", destination)
	}
	return nil
}

func snapshotIdentityAndRelatedIDs(entityType EntityType, snapshot json.RawMessage) (uuid.UUID, []uuid.UUID, error) {
	switch entityType {
	case EntityContact:
		contact, err := ContactFromSnapshot(snapshot)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return contact.ID, []uuid.UUID{contact.ID}, nil
	case EntityCompany:
		company, err := CompanyFromSnapshot(snapshot)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return company.ID, []uuid.UUID{company.ID}, nil
	case EntityRelationship:
		relationship, err := RelationshipFromSnapshot(snapshot)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return relationship.ID, canonicalRelatedIDs([]uuid.UUID{
			relationship.ID,
			relationship.SourceID,
			relationship.TargetID,
		}), nil
	default:
		return uuid.Nil, nil, fmt.Errorf("invalid history entity type %q", entityType)
	}
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
	if err := newSnapshotDecoder(snapshot).Decode(&object); err != nil {
		return nil, err
	}
	if object == nil {
		return make(map[string]any), nil
	}
	return object, nil
}

func newSnapshotDecoder(snapshot json.RawMessage) *json.Decoder {
	decoder := json.NewDecoder(bytes.NewReader(snapshot))
	decoder.UseNumber()
	return decoder
}

func strictSnapshotObject(entityType EntityType, snapshot json.RawMessage) (map[string]any, error) {
	if isNullSnapshot(snapshot) {
		return make(map[string]any), nil
	}
	canonical, err := canonicalSnapshot(entityType, snapshot)
	if err != nil {
		return nil, err
	}
	return snapshotObject(canonical)
}
