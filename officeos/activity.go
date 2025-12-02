// ABOUTME: Activity object implementation for timeline tracking
// ABOUTME: Defines ActivityObject and activity generation logic
package officeos

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ActivityVerb represents the action performed on an object.
type ActivityVerb string

const (
	VerbCreated ActivityVerb = "created"
	VerbUpdated ActivityVerb = "updated"
	VerbDeleted ActivityVerb = "deleted"
	VerbViewed  ActivityVerb = "viewed"
	VerbShared  ActivityVerb = "shared"
)

// ActivityFields contains the activity-specific fields.
type ActivityFields struct {
	ActorID    string                 `json:"actorId"`
	Verb       ActivityVerb           `json:"verb"`
	ObjectID   string                 `json:"objectId"`
	ObjectKind ObjectKind             `json:"objectKind"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ActivityObject represents an activity in the timeline.
type ActivityObject struct {
	BaseObject
}

// NewActivityObject creates a new activity object.
func NewActivityObject(actorID string, verb ActivityVerb, objectID string, objectKind ObjectKind, metadata map[string]interface{}) *ActivityObject {
	now := time.Now()
	fields := ActivityFields{
		ActorID:    actorID,
		Verb:       verb,
		ObjectID:   objectID,
		ObjectKind: objectKind,
		Metadata:   metadata,
	}

	return &ActivityObject{
		BaseObject: BaseObject{
			ID:        uuid.New().String(),
			Kind:      KindActivity,
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: actorID,
			ACL: []ACLEntry{
				{ActorID: actorID, Role: "owner"},
			},
			Fields: fields,
		},
	}
}

// GetFields returns the activity-specific fields.
func (a *ActivityObject) GetFields() (*ActivityFields, error) {
	bytes, err := json.Marshal(a.Fields)
	if err != nil {
		return nil, err
	}

	var fields ActivityFields
	if err := json.Unmarshal(bytes, &fields); err != nil {
		return nil, err
	}

	return &fields, nil
}

// ActivityGenerator implements the ActivityHooks interface.
type ActivityGenerator struct {
	store ObjectStore
}

// NewActivityGenerator creates a new activity generator.
func NewActivityGenerator(store ObjectStore) *ActivityGenerator {
	return &ActivityGenerator{store: store}
}

// OnCreate generates an activity when an object is created.
func (ag *ActivityGenerator) OnCreate(obj *BaseObject) error {
	activity := NewActivityObject(
		obj.CreatedBy,
		VerbCreated,
		obj.ID,
		obj.Kind,
		map[string]interface{}{
			"timestamp": obj.CreatedAt.Format(time.RFC3339),
		},
	)

	return ag.store.Create(&activity.BaseObject)
}

// OnUpdate generates an activity when an object is updated.
func (ag *ActivityGenerator) OnUpdate(oldObj, newObj *BaseObject) error {
	// Calculate changes
	changes := calculateChanges(oldObj, newObj)
	if len(changes) == 0 {
		// No changes to record
		return nil
	}

	activity := NewActivityObject(
		newObj.CreatedBy, // Using the updater as actor
		VerbUpdated,
		newObj.ID,
		newObj.Kind,
		map[string]interface{}{
			"timestamp": newObj.UpdatedAt.Format(time.RFC3339),
			"changes":   changes,
		},
	)

	return ag.store.Create(&activity.BaseObject)
}

// OnDelete generates an activity when an object is deleted.
func (ag *ActivityGenerator) OnDelete(obj *BaseObject) error {
	activity := NewActivityObject(
		obj.CreatedBy,
		VerbDeleted,
		obj.ID,
		obj.Kind,
		map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	)

	return ag.store.Create(&activity.BaseObject)
}

// calculateChanges compares two objects and returns a map of changes.
func calculateChanges(oldObj, newObj *BaseObject) map[string]interface{} {
	changes := make(map[string]interface{})

	// Compare basic fields
	if oldObj.UpdatedAt != newObj.UpdatedAt {
		// Fields likely changed, marshal both and compare
		oldFields, err1 := json.Marshal(oldObj.Fields)
		newFields, err2 := json.Marshal(newObj.Fields)

		if err1 == nil && err2 == nil && string(oldFields) != string(newFields) {
			// Parse as maps for detailed comparison
			var oldMap, newMap map[string]interface{}
			if err1 := json.Unmarshal(oldFields, &oldMap); err1 == nil {
				if err2 := json.Unmarshal(newFields, &newMap); err2 == nil {
					for key, newVal := range newMap {
						oldVal, exists := oldMap[key]
						if !exists || !deepEqual(oldVal, newVal) {
							changes[key] = map[string]interface{}{
								"before": oldVal,
								"after":  newVal,
							}
						}
					}
				}
			}
		}
	}

	// Compare tags
	if !stringSliceEqual(oldObj.Tags, newObj.Tags) {
		changes["tags"] = map[string]interface{}{
			"before": oldObj.Tags,
			"after":  newObj.Tags,
		}
	}

	return changes
}

// deepEqual performs a simple deep comparison of values.
func deepEqual(a, b interface{}) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// stringSliceEqual checks if two string slices are equal.
func stringSliceEqual(a, b []string) bool {
	// Handle nil vs empty slice distinction
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
