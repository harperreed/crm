// ABOUTME: Tests for activity object implementation
// ABOUTME: Covers activity creation, field extraction, and change tracking
package officeos

import (
	"testing"
	"time"
)

func TestNewActivityObject(t *testing.T) {
	actorID := "user-123"
	objectID := "contact-456"
	metadata := map[string]interface{}{
		"test": "value",
	}

	activity := NewActivityObject(actorID, VerbCreated, objectID, KindRecord, metadata)

	if activity.Kind != KindActivity {
		t.Errorf("Expected kind %s, got %s", KindActivity, activity.Kind)
	}

	if activity.CreatedBy != actorID {
		t.Errorf("Expected created_by %s, got %s", actorID, activity.CreatedBy)
	}

	fields, err := activity.GetFields()
	if err != nil {
		t.Fatalf("Failed to get fields: %v", err)
	}

	if fields.ActorID != actorID {
		t.Errorf("Expected actorId %s, got %s", actorID, fields.ActorID)
	}

	if fields.Verb != VerbCreated {
		t.Errorf("Expected verb %s, got %s", VerbCreated, fields.Verb)
	}

	if fields.ObjectID != objectID {
		t.Errorf("Expected objectId %s, got %s", objectID, fields.ObjectID)
	}

	if fields.ObjectKind != KindRecord {
		t.Errorf("Expected objectKind %s, got %s", KindRecord, fields.ObjectKind)
	}

	if fields.Metadata["test"] != "value" {
		t.Errorf("Expected metadata test=value, got %v", fields.Metadata["test"])
	}
}

func TestActivityFields(t *testing.T) {
	activity := NewActivityObject("actor-1", VerbUpdated, "obj-1", KindTask, nil)

	fields, err := activity.GetFields()
	if err != nil {
		t.Fatalf("Failed to get fields: %v", err)
	}

	if fields.ActorID != "actor-1" {
		t.Errorf("Expected actorId actor-1, got %s", fields.ActorID)
	}

	if fields.Verb != VerbUpdated {
		t.Errorf("Expected verb %s, got %s", VerbUpdated, fields.Verb)
	}
}

func TestActivityObjectACL(t *testing.T) {
	actorID := "user-123"
	activity := NewActivityObject(actorID, VerbCreated, "obj-1", KindRecord, nil)

	if len(activity.ACL) != 1 {
		t.Fatalf("Expected 1 ACL entry, got %d", len(activity.ACL))
	}

	if activity.ACL[0].ActorID != actorID {
		t.Errorf("Expected ACL actorId %s, got %s", actorID, activity.ACL[0].ActorID)
	}

	if activity.ACL[0].Role != "owner" {
		t.Errorf("Expected ACL role owner, got %s", activity.ACL[0].Role)
	}
}

func TestCalculateChanges_NoChanges(t *testing.T) {
	now := time.Now()
	obj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: now,
		UpdatedAt: now,
		Fields: map[string]interface{}{
			"name": "John",
		},
	}

	changes := calculateChanges(obj, obj)
	if len(changes) != 0 {
		t.Errorf("Expected no changes, got %v", changes)
	}
}

func TestCalculateChanges_FieldChanges(t *testing.T) {
	now := time.Now()
	oldObj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: now,
		UpdatedAt: now,
		Fields: map[string]interface{}{
			"title": "VP Engineering",
			"email": "test@example.com",
		},
	}

	newObj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
		Fields: map[string]interface{}{
			"title": "CTO",
			"email": "test@example.com",
		},
	}

	changes := calculateChanges(oldObj, newObj)
	if len(changes) == 0 {
		t.Fatal("Expected changes, got none")
	}

	titleChange, ok := changes["title"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected title change to be a map, got %T", changes["title"])
	}

	if titleChange["before"] != "VP Engineering" {
		t.Errorf("Expected before value 'VP Engineering', got %v", titleChange["before"])
	}

	if titleChange["after"] != "CTO" {
		t.Errorf("Expected after value 'CTO', got %v", titleChange["after"])
	}

	// Email should not be in changes
	if _, exists := changes["email"]; exists {
		t.Error("Email should not be in changes as it didn't change")
	}
}

func TestCalculateChanges_TagChanges(t *testing.T) {
	now := time.Now()
	oldObj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"tag1", "tag2"},
		Fields:    map[string]interface{}{},
	}

	newObj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
		Tags:      []string{"tag1", "tag3"},
		Fields:    map[string]interface{}{},
	}

	changes := calculateChanges(oldObj, newObj)
	if len(changes) == 0 {
		t.Fatal("Expected tag changes, got none")
	}

	tagChange, ok := changes["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tags change to be a map, got %T", changes["tags"])
	}

	if tagChange["before"] == nil {
		t.Error("Expected before tags to be set")
	}

	if tagChange["after"] == nil {
		t.Error("Expected after tags to be set")
	}
}

func TestStringSliceEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{"equal slices", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
		{"different values", []string{"a", "b"}, []string{"a", "c"}, false},
		{"empty slices", []string{}, []string{}, true},
		{"nil vs empty", nil, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringSliceEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for %v vs %v", tt.expected, result, tt.a, tt.b)
			}
		})
	}
}
